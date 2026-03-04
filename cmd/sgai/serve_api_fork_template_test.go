package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupRootWithFakeJJ(t *testing.T, rootDir string) string {
	t.Helper()
	installFakeJJWithWorkspaceList(t, 2)
	workspace := filepath.Join(rootDir, "root-workspace")
	if errMkdir := os.MkdirAll(filepath.Join(workspace, ".jj", "repo"), 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)
	return workspace
}

func setupForkDir(t *testing.T, rootDir, forkName, rootWorkspace string) string {
	t.Helper()
	forkDir := filepath.Join(rootDir, forkName)
	if errMkdir := os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	repoPointer := filepath.Join(rootWorkspace, ".jj", "repo")
	if errWrite := os.WriteFile(filepath.Join(forkDir, ".jj", "repo"), []byte(repoPointer), 0644); errWrite != nil {
		t.Fatal(errWrite)
	}
	createsgaiDir(t, forkDir)
	return forkDir
}

func TestHandleAPIForkTemplate(t *testing.T) {
	t.Run("returnsGoalExampleWhenNoForks", func(t *testing.T) {
		rootDir := t.TempDir()
		setupRootWithFakeJJ(t, rootDir)

		srv := NewServer(rootDir)
		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiForkTemplateResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		if result.Content != goalExampleContent {
			t.Errorf("content should be goalExampleContent when no forks exist")
		}
	})

	t.Run("returnsLastForkGoalWhenForksExist", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := setupRootWithFakeJJ(t, rootDir)
		forkDir := setupForkDir(t, rootDir, "fork-one", workspace)

		forkGoalContent := "---\nflow: test\n---\n\n# Fork Goal\n"
		if errWrite := os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte(forkGoalContent), 0644); errWrite != nil {
			t.Fatal(errWrite)
		}

		srv := NewServer(rootDir)
		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiForkTemplateResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		if result.Content != forkGoalContent {
			t.Errorf("content = %q; want %q", result.Content, forkGoalContent)
		}
	})

	t.Run("returns404ForNonExistentWorkspace", func(t *testing.T) {
		rootDir := t.TempDir()
		srv := NewServer(rootDir)

		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/nonexistent/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("returnsBadRequestForNonRootWorkspace", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := setupRootWithFakeJJ(t, rootDir)
		setupForkDir(t, rootDir, "fork-ws", workspace)

		srv := NewServer(rootDir)
		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/fork-ws/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusBadRequest, resp.Body.String())
		}
	})

	t.Run("fallsBackToExampleWhenForkGoalEmpty", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := setupRootWithFakeJJ(t, rootDir)
		forkDir := setupForkDir(t, rootDir, "fork-empty", workspace)

		if errWrite := os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte("   \n  \n"), 0644); errWrite != nil {
			t.Fatal(errWrite)
		}

		srv := NewServer(rootDir)
		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiForkTemplateResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		if result.Content != goalExampleContent {
			t.Errorf("content should fall back to goalExampleContent when fork GOAL.md is empty")
		}
	})

	t.Run("fallsBackToExampleWhenForkGoalMissing", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := setupRootWithFakeJJ(t, rootDir)
		setupForkDir(t, rootDir, "fork-no-goal", workspace)

		srv := NewServer(rootDir)
		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		var result apiForkTemplateResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}

		if result.Content != goalExampleContent {
			t.Errorf("content should fall back to goalExampleContent when fork has no GOAL.md")
		}
	})

	t.Run("responseContentTypeIsJSON", func(t *testing.T) {
		rootDir := t.TempDir()
		setupRootWithFakeJJ(t, rootDir)

		srv := NewServer(rootDir)
		mux := serverMux(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/root-workspace/fork-template", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
		}

		ct := resp.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q; want application/json", ct)
		}
	})
}
