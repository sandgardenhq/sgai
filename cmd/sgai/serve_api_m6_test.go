package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func setupM6TestWorkspace(t *testing.T) (string, string, *Server) {
	t.Helper()
	installFakeJJWithWorkspaceList(t, 2)
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(filepath.Join(workspace, ".jj", "repo"), 0755); err != nil {
		t.Fatal(err)
	}
	createsgaiDir(t, workspace)
	srv := NewServer(rootDir)
	return rootDir, workspace, srv
}

func setupM6ForkWorkspace(t *testing.T) (string, string, *Server) {
	t.Helper()
	installFakeJJWithWorkspaceList(t, 2)
	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(filepath.Join(rootPath, ".jj", "repo"), 0755); err != nil {
		t.Fatal(err)
	}
	createsgaiDir(t, rootPath)

	forkPath := filepath.Join(rootDir, "my-fork")
	if err := os.MkdirAll(forkPath, 0755); err != nil {
		t.Fatal(err)
	}
	jjDir := filepath.Join(forkPath, ".jj")
	if err := os.MkdirAll(jjDir, 0755); err != nil {
		t.Fatal(err)
	}
	repoLink := filepath.Join(rootPath, ".jj", "repo")
	if err := os.WriteFile(filepath.Join(jjDir, "repo"), []byte(repoLink), 0644); err != nil {
		t.Fatal(err)
	}
	createsgaiDir(t, forkPath)

	srv := NewServer(rootDir)
	return rootDir, forkPath, srv
}

func TestGenerateRandomForkName(t *testing.T) {
	t.Run("matchesExpectedPattern", func(t *testing.T) {
		const allowedSuffixChars = "0123456789aeiou"
		isLowercaseAlpha := func(s string) bool {
			for _, ch := range s {
				if ch < 'a' || ch > 'z' {
					return false
				}
			}
			return len(s) > 0
		}
		for range 100 {
			name := generateRandomForkName()
			parts := strings.SplitN(name, "-", 3)
			if len(parts) != 3 {
				t.Fatalf("name %q does not have 3 dash-separated parts", name)
			}
			if !isLowercaseAlpha(parts[0]) {
				t.Errorf("adjective %q is not lowercase alpha", parts[0])
			}
			if !isLowercaseAlpha(parts[1]) {
				t.Errorf("color %q is not lowercase alpha", parts[1])
			}
			if len(parts[2]) != 4 {
				t.Errorf("suffix %q should be 4 characters", parts[2])
			}
			for _, ch := range parts[2] {
				if !strings.ContainsRune(allowedSuffixChars, ch) {
					t.Errorf("suffix character %q not in allowed set", string(ch))
				}
			}
		}
	})

	t.Run("passesWorkspaceNameValidation", func(t *testing.T) {
		for range 50 {
			name := generateRandomForkName()
			if errMsg := validateWorkspaceName(name); errMsg != "" {
				t.Errorf("generated name %q failed validation: %s", name, errMsg)
			}
		}
	})

	t.Run("producesVariety", func(t *testing.T) {
		seen := make(map[string]bool)
		for range 50 {
			seen[generateRandomForkName()] = true
		}
		if len(seen) < 40 {
			t.Errorf("expected at least 40 unique names out of 50, got %d", len(seen))
		}
	})
}

func TestGoalContentBodyIsEmpty(t *testing.T) {
	t.Run("emptyString", func(t *testing.T) {
		if !goalContentBodyIsEmpty("") {
			t.Error("empty string should be considered empty")
		}
	})

	t.Run("whitespaceOnly", func(t *testing.T) {
		if !goalContentBodyIsEmpty("   \n\t  ") {
			t.Error("whitespace-only should be considered empty")
		}
	})

	t.Run("frontmatterOnly", func(t *testing.T) {
		content := "---\nflow: |\n  a -> b\n---\n"
		if !goalContentBodyIsEmpty(content) {
			t.Error("frontmatter-only content should be considered empty")
		}
	})

	t.Run("frontmatterWithWhitespaceBody", func(t *testing.T) {
		content := "---\nflow: |\n  a -> b\n---\n   \n  \n"
		if !goalContentBodyIsEmpty(content) {
			t.Error("frontmatter with whitespace body should be considered empty")
		}
	})

	t.Run("contentWithBody", func(t *testing.T) {
		content := "Build a web app"
		if goalContentBodyIsEmpty(content) {
			t.Error("content with body text should not be considered empty")
		}
	})

	t.Run("frontmatterWithBody", func(t *testing.T) {
		content := "---\nflow: |\n  a -> b\n---\nBuild a web app"
		if goalContentBodyIsEmpty(content) {
			t.Error("frontmatter with body should not be considered empty")
		}
	})
}

func TestHandleAPIForkWorkspace(t *testing.T) {
	t.Run("successfulFork", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"goalContent":"Build a web app with authentication"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
		}

		var result apiForkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.Name == "" {
			t.Error("name should be auto-generated, not empty")
		}
		if result.Parent != "root-workspace" {
			t.Errorf("parent = %q; want %q", result.Parent, "root-workspace")
		}
		if result.CreatedAt == "" {
			t.Error("createdAt should not be empty")
		}
	})

	t.Run("writesGoalContent", func(t *testing.T) {
		rootDir, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		goalText := "Build a REST API for user management"
		body := strings.NewReader(`{"goalContent":"` + goalText + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
		}

		var result apiForkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		goalPath := filepath.Join(rootDir, result.Name, "GOAL.md")
		content, errRead := os.ReadFile(goalPath)
		if errRead != nil {
			t.Fatalf("failed to read GOAL.md: %v", errRead)
		}
		if string(content) != goalText {
			t.Errorf("GOAL.md content = %q; want %q", string(content), goalText)
		}
	})

	t.Run("rejectsEmptyGoalContent", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"goalContent":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusBadRequest, resp.Body.String())
		}
	})

	t.Run("rejectsFrontmatterOnlyGoalContent", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"goalContent":"---\nflow: |\n  a -> b\n---\n"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejectsForkOfFork", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"goalContent":"Build something"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"goalContent":"Build something"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/nonexistent/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("autoGeneratesUniqueName", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"goalContent":"First goal"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("first call: status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
		}

		var result1 apiForkResponse
		if err := json.NewDecoder(resp.Body).Decode(&result1); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		body2 := strings.NewReader(`{"goalContent":"Second goal"}`)
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body2)
		req2.Header.Set("Content-Type", "application/json")
		resp2 := httptest.NewRecorder()
		mux.ServeHTTP(resp2, req2)

		if resp2.Code != http.StatusCreated {
			t.Fatalf("second call: status = %d; want %d; body = %s", resp2.Code, http.StatusCreated, resp2.Body.String())
		}

		var result2 apiForkResponse
		if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result1.Name == result2.Name {
			t.Errorf("two forks should have different auto-generated names, both got %q", result1.Name)
		}
	})
}

func TestHandleAPIRenameWorkspace(t *testing.T) {
	t.Run("successfulRename", func(t *testing.T) {
		rootDir, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"name":"renamed-fork"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/rename", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiRenameResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.Name != "renamed-fork" {
			t.Errorf("name = %q; want %q", result.Name, "renamed-fork")
		}
		if result.OldName != forkName {
			t.Errorf("oldName = %q; want %q", result.OldName, forkName)
		}

		if _, err := os.Stat(filepath.Join(rootDir, "renamed-fork")); err != nil {
			t.Errorf("renamed directory should exist: %v", err)
		}
		if _, err := os.Stat(forkPath); !os.IsNotExist(err) {
			t.Errorf("old directory should not exist")
		}
	})

	t.Run("rejectsNonFork", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"name":"new-name"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/rename", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejectsInvalidName", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"name":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/rename", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejectsRunningSession", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		srv.mu.Lock()
		srv.sessions[forkPath] = &session{running: true}
		srv.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"name":"renamed-fork"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/rename", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusConflict {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusConflict)
		}
	})

	t.Run("conflictsWithExistingDir", func(t *testing.T) {
		rootDir, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		existingDir := filepath.Join(rootDir, "existing-name")
		if err := os.MkdirAll(existingDir, 0755); err != nil {
			t.Fatal(err)
		}

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"name":"existing-name"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/rename", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusConflict {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusConflict)
		}
	})

	t.Run("rekeysSessionMap", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		srv.mu.Lock()
		srv.sessions[forkPath] = &session{running: false}
		srv.everStartedDirs[forkPath] = true
		srv.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"name":"rekeyed-fork"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/rename", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		srv.mu.Lock()
		_, oldKeyExists := srv.sessions[forkPath]
		newDir := filepath.Dir(forkPath)
		newPath := filepath.Join(newDir, "rekeyed-fork")
		_, newKeyExists := srv.sessions[newPath]
		_, oldStarted := srv.everStartedDirs[forkPath]
		_, newStarted := srv.everStartedDirs[newPath]
		srv.mu.Unlock()

		if oldKeyExists {
			t.Error("old session key should not exist")
		}
		if !newKeyExists {
			t.Error("new session key should exist")
		}
		if oldStarted {
			t.Error("old everStartedDirs key should not exist")
		}
		if !newStarted {
			t.Error("new everStartedDirs key should exist")
		}
	})
}

func TestHandleAPIUpdateGoal(t *testing.T) {
	t.Run("successfulUpdate", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"content":"# New Goal\n\nBuild something great"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/root-workspace/goal", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiUpdateGoalResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !result.Updated {
			t.Error("updated should be true")
		}

		goalPath := filepath.Join(workspace, "GOAL.md")
		content, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatalf("failed to read GOAL.md: %v", err)
		}
		if !strings.Contains(string(content), "New Goal") {
			t.Errorf("GOAL.md content = %q; want containing 'New Goal'", string(content))
		}
	})

	t.Run("rejectsEmptyContent", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"content":""}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/root-workspace/goal", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"content":"content"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/nonexistent/goal", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		goalContent := `{"content":"# Goal v2\n\nSame content"}`

		for range 3 {
			body := strings.NewReader(goalContent)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/root-workspace/goal", body)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
			}
		}

		goalPath := filepath.Join(workspace, "GOAL.md")
		content, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatalf("failed to read GOAL.md: %v", err)
		}
		if !strings.Contains(string(content), "Goal v2") {
			t.Errorf("GOAL.md content = %q; want containing 'Goal v2'", string(content))
		}
	})
}

func TestHandleAPIAdhoc(t *testing.T) {
	t.Run("rejectsEmptyPrompt", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"prompt":"","model":"gpt-4"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/adhoc", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejectsEmptyModel", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"prompt":"do something","model":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/adhoc", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"prompt":"do something","model":"gpt-4"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/nonexistent/adhoc", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("idempotentWhenRunning", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		st := srv.getAdhocState(workspace)
		st.mu.Lock()
		st.running = true
		st.output.WriteString("partial output")
		st.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"prompt":"do something","model":"gpt-4"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/adhoc", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiAdhocResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !result.Running {
			t.Error("running should be true")
		}
		if result.Output != "partial output" {
			t.Errorf("output = %q; want %q", result.Output, "partial output")
		}
		if result.Message != "ad-hoc prompt already running" {
			t.Errorf("message = %q; want %q", result.Message, "ad-hoc prompt already running")
		}
	})

	t.Run("invalidJSON", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`not json`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/adhoc", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})
}

func installFakeOpencode(t *testing.T) {
	t.Helper()
	fakeBinDir := t.TempDir()
	fakeOpencode := filepath.Join(fakeBinDir, "opencode")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(fakeOpencode, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake opencode: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestHandleAPIAdhocLogsCommandAndPrompt(t *testing.T) {
	t.Run("outputContainsCommandAndPrompt", func(t *testing.T) {
		installFakeOpencode(t)
		_, workspace, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"prompt":"create a new feature","model":"anthropic/claude-opus-4-6 (max)"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/adhoc", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		st := srv.getAdhocState(workspace)

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			st.mu.Lock()
			running := st.running
			st.mu.Unlock()
			if !running {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		st.mu.Lock()
		running := st.running
		output := st.output.String()
		st.mu.Unlock()
		if running {
			t.Fatal("adhoc command did not finish within deadline")
		}

		if !strings.Contains(output, "$ opencode run -m anthropic/claude-opus-4-6") {
			t.Errorf("output should contain CLI command, got: %q", output)
		}
		if !strings.Contains(output, "prompt: create a new feature") {
			t.Errorf("output should contain prompt text, got: %q", output)
		}
	})
}

func TestBuildAdhocArgs(t *testing.T) {
	t.Run("modelWithoutVariant", func(t *testing.T) {
		args := buildAdhocArgs("anthropic/claude-opus-4-6")
		want := []string{"run", "-m", "anthropic/claude-opus-4-6", "--agent", "build", "--title", "adhoc [anthropic/claude-opus-4-6]"}
		if !slices.Equal(args, want) {
			t.Errorf("buildAdhocArgs(%q) = %v; want %v", "anthropic/claude-opus-4-6", args, want)
		}
	})

	t.Run("modelWithVariant", func(t *testing.T) {
		args := buildAdhocArgs("anthropic/claude-opus-4-6 (max)")
		want := []string{"run", "-m", "anthropic/claude-opus-4-6", "--agent", "build", "--title", "adhoc [anthropic/claude-opus-4-6 (max)]", "--variant", "max"}
		if !slices.Equal(args, want) {
			t.Errorf("buildAdhocArgs(%q) = %v; want %v", "anthropic/claude-opus-4-6 (max)", args, want)
		}
	})

	t.Run("modelWithMultiWordVariant", func(t *testing.T) {
		args := buildAdhocArgs("openai/gpt-4o (high quality)")
		want := []string{"run", "-m", "openai/gpt-4o", "--agent", "build", "--title", "adhoc [openai/gpt-4o (high quality)]", "--variant", "high quality"}
		if !slices.Equal(args, want) {
			t.Errorf("buildAdhocArgs(%q) = %v; want %v", "openai/gpt-4o (high quality)", args, want)
		}
	})
}

func TestHandleAPIDeleteFork(t *testing.T) {
	t.Run("successfulDelete", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)

		rootPath := getRootWorkspacePath(forkPath)
		rootName := filepath.Base(rootPath)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"forkDir":"` + forkPath + `","confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+rootName+"/delete-fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiDeleteForkResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}
		if !result.Deleted {
			t.Error("deleted should be true")
		}
		if result.Message != "fork deleted successfully" {
			t.Errorf("message = %q; want %q", result.Message, "fork deleted successfully")
		}

		if _, errStat := os.Stat(forkPath); !os.IsNotExist(errStat) {
			t.Error("fork directory should not exist after deletion")
		}
	})

	t.Run("rejectsNonRoot", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)
		forkName := filepath.Base(forkPath)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"forkDir":"/some/dir","confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+forkName+"/delete-fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejectsMissingConfirm", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)

		rootPath := getRootWorkspacePath(forkPath)
		rootName := filepath.Base(rootPath)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"forkDir":"` + forkPath + `","confirm":false}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+rootName+"/delete-fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"forkDir":"/some/dir","confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/nonexistent/delete-fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("invalidJSON", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`not json`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/delete-fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("stopsRunningSession", func(t *testing.T) {
		_, forkPath, srv := setupM6ForkWorkspace(t)

		rootPath := getRootWorkspacePath(forkPath)
		rootName := filepath.Base(rootPath)

		srv.mu.Lock()
		srv.sessions[forkPath] = &session{running: true}
		srv.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"forkDir":"` + forkPath + `","confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+rootName+"/delete-fork", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		srv.mu.Lock()
		sess := srv.sessions[forkPath]
		srv.mu.Unlock()

		if sess != nil && sess.running {
			t.Error("session should have been stopped")
		}
	})
}

func TestHandleAPIAdhocStatus(t *testing.T) {
	t.Run("returnsIdleState", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/adhoc", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiAdhocResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.Running {
			t.Error("running should be false")
		}
		if result.Output != "" {
			t.Errorf("output = %q; want empty", result.Output)
		}
	})

	t.Run("returnsRunningState", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		st := srv.getAdhocState(workspace)
		st.mu.Lock()
		st.running = true
		st.output.WriteString("partial output")
		st.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/adhoc", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiAdhocResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !result.Running {
			t.Error("running should be true")
		}
		if result.Output != "partial output" {
			t.Errorf("output = %q; want %q", result.Output, "partial output")
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/nonexistent/adhoc", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})
}

func TestHandleAPIAdhocStop(t *testing.T) {
	t.Run("stopsRunningAdhoc", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		st := srv.getAdhocState(workspace)
		st.mu.Lock()
		st.running = true
		st.output.WriteString("partial output")
		st.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/root-workspace/adhoc", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiAdhocResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.Running {
			t.Error("running should be false after stop")
		}
		if !strings.Contains(result.Output, "partial output") {
			t.Errorf("output = %q; want containing 'partial output'", result.Output)
		}
		if !strings.Contains(result.Output, "[stopped by user]") {
			t.Errorf("output = %q; want containing '[stopped by user]'", result.Output)
		}
		if result.Message != "ad-hoc stopped" {
			t.Errorf("message = %q; want %q", result.Message, "ad-hoc stopped")
		}

		st.mu.Lock()
		running := st.running
		st.mu.Unlock()
		if running {
			t.Error("adhoc state running should be false after stop")
		}
	})

	t.Run("stopNonRunningAdhoc", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/root-workspace/adhoc", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiAdhocResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.Running {
			t.Error("running should be false")
		}
		if result.Message != "ad-hoc stopped" {
			t.Errorf("message = %q; want %q", result.Message, "ad-hoc stopped")
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/nonexistent/adhoc", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("idempotentStop", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		st := srv.getAdhocState(workspace)
		st.mu.Lock()
		st.running = true
		st.output.WriteString("output")
		st.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		for range 3 {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/root-workspace/adhoc", nil)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
			}
		}

		st.mu.Lock()
		output := st.output.String()
		running := st.running
		st.mu.Unlock()

		if running {
			t.Error("running should be false")
		}
		if strings.Count(output, "[stopped by user]") != 1 {
			t.Errorf("output should contain exactly one '[stopped by user]', got: %q", output)
		}
	})
}

func TestHandleAPIOpenEditorGoal(t *testing.T) {
	t.Run("opensGoalFile", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		goalPath := filepath.Join(workspace, "GOAL.md")
		if err := os.WriteFile(goalPath, []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}

		var opened string
		srv.editor = &mockEditor{openFn: func(path string) error {
			opened = path
			return nil
		}}
		srv.editorAvailable = true

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/open-editor/goal", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}
		if opened != goalPath {
			t.Errorf("opened = %q; want %q", opened, goalPath)
		}
	})

	t.Run("missingGoalFile", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)
		srv.editor = &mockEditor{openFn: func(_ string) error { return nil }}
		srv.editorAvailable = true

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/open-editor/goal", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("editorNotAvailable", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		goalPath := filepath.Join(workspace, "GOAL.md")
		if err := os.WriteFile(goalPath, []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}
		srv.editorAvailable = false

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/open-editor/goal", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusServiceUnavailable)
		}
	})
}

func TestHandleAPIOpenEditorProjectManagement(t *testing.T) {
	t.Run("opensProjectManagementFile", func(t *testing.T) {
		_, workspace, srv := setupM6TestWorkspace(t)

		pmPath := filepath.Join(workspace, ".sgai", "PROJECT_MANAGEMENT.md")
		if err := os.WriteFile(pmPath, []byte("# PM"), 0644); err != nil {
			t.Fatal(err)
		}

		var opened string
		srv.editor = &mockEditor{openFn: func(path string) error {
			opened = path
			return nil
		}}
		srv.editorAvailable = true

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/open-editor/project-management", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}
		if opened != pmPath {
			t.Errorf("opened = %q; want %q", opened, pmPath)
		}
	})

	t.Run("missingProjectManagementFile", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)
		srv.editor = &mockEditor{openFn: func(_ string) error { return nil }}
		srv.editorAvailable = true

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/open-editor/project-management", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})
}

type mockEditor struct {
	openFn func(path string) error
}

func (m *mockEditor) open(path string) error {
	return m.openFn(path)
}
