package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func resetComposerSessions(t *testing.T) {
	t.Helper()
	composerSessionsMu.Lock()
	composerSessions = make(map[string]*composerSession)
	composerSessionsMu.Unlock()
}

func setupComposeTestWorkspace(t *testing.T) (string, *Server) {
	t.Helper()
	resetComposerSessions(t)
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".sgai", "agent"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".sgai", "agent", "coordinator.md"), []byte("---\ndescription: coordinates work\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(rootDir)
	return workspace, srv
}

func TestHandleAPIComposeState(t *testing.T) {
	workspace, srv := setupComposeTestWorkspace(t)
	workspaceName := filepath.Base(workspace)

	t.Run("returnsDefaultState", func(t *testing.T) {
		resetComposerSessions(t)
		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/compose?workspace="+workspaceName, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiComposeStateResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.Workspace != workspaceName {
			t.Errorf("workspace = %q; want %q", result.Workspace, workspaceName)
		}
		if len(result.TechStackItems) == 0 {
			t.Error("techStackItems should not be empty")
		}
	})

	t.Run("returnsExistingGoalState", func(t *testing.T) {
		resetComposerSessions(t)
		goalContent := `---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
models:
  "coordinator": "anthropic/claude-opus-4-6"
  "backend-go-developer": "anthropic/claude-opus-4-6"
completionGateScript: make test
---

Build a REST API

## Tasks

- Build user endpoint
`
		if err := os.WriteFile(filepath.Join(workspace, "GOAL.md"), []byte(goalContent), 0644); err != nil {
			t.Fatal(err)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/compose?workspace="+workspaceName, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiComposeStateResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.State.CompletionGate != "make test" {
			t.Errorf("completionGate = %q; want %q", result.State.CompletionGate, "make test")
		}
		if result.State.Flow == "" {
			t.Error("flow should not be empty")
		}
		if len(result.State.Agents) < 2 {
			t.Errorf("agents count = %d; want >= 2", len(result.State.Agents))
		}
	})

	t.Run("notFoundWorkspace", func(t *testing.T) {
		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/compose?workspace=nonexistent", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})
}

func TestHandleAPIComposeTemplates(t *testing.T) {
	_, srv := setupComposeTestWorkspace(t)

	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compose/templates", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
	}

	var result apiComposeTemplatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Templates) != len(workflowTemplates) {
		t.Fatalf("templates count = %d; want %d", len(result.Templates), len(workflowTemplates))
	}

	foundBackend := false
	for _, tmpl := range result.Templates {
		if tmpl.ID == "backend" {
			foundBackend = true
			if tmpl.Name != "Backend Development" {
				t.Errorf("backend template name = %q; want %q", tmpl.Name, "Backend Development")
			}
			if len(tmpl.Agents) == 0 {
				t.Error("backend template should have agents")
			}
		}
	}
	if !foundBackend {
		t.Error("backend template not found in response")
	}
}

func TestHandleAPIComposePreview(t *testing.T) {
	workspace, srv := setupComposeTestWorkspace(t)
	workspaceName := filepath.Base(workspace)

	t.Run("generatesPreview", func(t *testing.T) {
		resetComposerSessions(t)

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		cs.state.Description = "A test project"
		cs.state.Tasks = "- Task 1\n- Task 2"
		cs.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/compose/preview?workspace="+workspaceName, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiComposePreviewResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !strings.Contains(result.Content, "A test project") {
			t.Error("preview should contain description")
		}
		if !strings.Contains(result.Content, "Task 1") {
			t.Error("preview should contain tasks")
		}
		if result.Etag == "" {
			t.Error("etag should not be empty")
		}
	})

	t.Run("returnsFlowError", func(t *testing.T) {
		resetComposerSessions(t)

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		cs.state.Flow = `invalid DOT {{{{ syntax >>>>`
		cs.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/compose/preview?workspace="+workspaceName, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiComposePreviewResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.FlowError == "" {
			t.Error("flowError should not be empty for invalid flow")
		}
	})
}

func TestHandleAPIComposeSave(t *testing.T) {
	workspace, srv := setupComposeTestWorkspace(t)
	workspaceName := filepath.Base(workspace)

	t.Run("savesGoalMd", func(t *testing.T) {
		resetComposerSessions(t)

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		cs.state.Description = "Saved project"
		cs.state.Tasks = "- Saved task"
		cs.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/compose?workspace="+workspaceName, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
		}

		var result apiComposeSaveResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !result.Saved {
			t.Error("saved should be true")
		}

		goalPath := filepath.Join(workspace, "GOAL.md")
		content, errRead := os.ReadFile(goalPath)
		if errRead != nil {
			t.Fatalf("failed to read GOAL.md: %v", errRead)
		}

		if !strings.Contains(string(content), "Saved project") {
			t.Error("GOAL.md should contain the description")
		}
		if !strings.Contains(string(content), "Saved task") {
			t.Error("GOAL.md should contain the tasks")
		}
	})

	t.Run("etagConflict", func(t *testing.T) {
		resetComposerSessions(t)

		goalPath := filepath.Join(workspace, "GOAL.md")
		if err := os.WriteFile(goalPath, []byte("original content"), 0644); err != nil {
			t.Fatal(err)
		}

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		cs.state.Description = "New content"
		cs.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/compose?workspace="+workspaceName, nil)
		req.Header.Set("If-Match", `"stale-etag-value"`)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusPreconditionFailed {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusPreconditionFailed)
		}
	})

	t.Run("etagMatchAllowsSave", func(t *testing.T) {
		resetComposerSessions(t)

		goalPath := filepath.Join(workspace, "GOAL.md")
		originalContent := []byte("original content for etag test")
		if err := os.WriteFile(goalPath, originalContent, 0644); err != nil {
			t.Fatal(err)
		}

		currentEtag := computeEtag(originalContent)

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		cs.state.Description = "Updated with valid etag"
		cs.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/compose?workspace="+workspaceName, nil)
		req.Header.Set("If-Match", currentEtag)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
		}
	})
}

func TestHandleAPIComposeDraft(t *testing.T) {
	workspace, srv := setupComposeTestWorkspace(t)
	workspaceName := filepath.Base(workspace)

	t.Run("savesDraftToSession", func(t *testing.T) {
		resetComposerSessions(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		draftBody := `{
			"state": {
				"description": "Draft description",
				"completionGate": "make test",
				"agents": [{"name": "coordinator", "selected": true, "model": "anthropic/claude-opus-4-6"}],
				"flow": "",
				"tasks": "- Draft task"
			},
			"wizard": {
				"currentStep": 2,
				"techStack": ["go", "react"],
				"safetyAnalysis": true
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/compose/draft?workspace="+workspaceName, strings.NewReader(draftBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiComposeDraftResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !result.Saved {
			t.Error("saved should be true")
		}

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		desc := cs.state.Description
		step := cs.wizard.CurrentStep
		safetyAnalysis := cs.wizard.SafetyAnalysis
		cs.mu.Unlock()

		if desc != "Draft description" {
			t.Errorf("state.description = %q; want %q", desc, "Draft description")
		}
		if step != 2 {
			t.Errorf("wizard.currentStep = %d; want %d", step, 2)
		}
		if !safetyAnalysis {
			t.Error("wizard.safetyAnalysis should be true")
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		resetComposerSessions(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		draftBody := `{
			"state": {
				"description": "Idempotent draft",
				"agents": [],
				"flow": "",
				"tasks": ""
			},
			"wizard": {
				"currentStep": 1,
				"techStack": [],
				"safetyAnalysis": false
			}
		}`

		for range 3 {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/compose/draft?workspace="+workspaceName, strings.NewReader(draftBody))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
			}
		}

		cs := getComposerSession(workspace)
		cs.mu.Lock()
		desc := cs.state.Description
		cs.mu.Unlock()

		if desc != "Idempotent draft" {
			t.Errorf("description = %q; want %q", desc, "Idempotent draft")
		}
	})

	t.Run("invalidBody", func(t *testing.T) {
		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/compose/draft?workspace="+workspaceName, strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})
}

func TestComputeEtag(t *testing.T) {
	content := []byte("test content")
	etag := computeEtag(content)

	if !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
		t.Errorf("etag should be quoted: %q", etag)
	}

	etag2 := computeEtag(content)
	if etag != etag2 {
		t.Errorf("same content should produce same etag: %q != %q", etag, etag2)
	}

	differentEtag := computeEtag([]byte("different content"))
	if etag == differentEtag {
		t.Error("different content should produce different etag")
	}

	nilEtag := computeEtag(nil)
	emptyEtag := computeEtag([]byte{})
	if nilEtag != emptyEtag {
		t.Errorf("nil and empty should produce same etag: %q != %q", nilEtag, emptyEtag)
	}
}

func TestHandleAPIComposeSaveConcurrentDrafts(t *testing.T) {
	workspace, srv := setupComposeTestWorkspace(t)
	workspaceName := filepath.Base(workspace)

	resetComposerSessions(t)

	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			draftBody := `{
				"state": {
					"description": "concurrent draft",
					"agents": [],
					"flow": "",
					"tasks": ""
				},
				"wizard": {
					"currentStep": 1,
					"techStack": [],
					"safetyAnalysis": false
				}
			}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/compose/draft?workspace="+workspaceName, strings.NewReader(draftBody))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Errorf("goroutine %d: status = %d; want %d", n, resp.Code, http.StatusOK)
			}
		}(i)
	}
	wg.Wait()
}
