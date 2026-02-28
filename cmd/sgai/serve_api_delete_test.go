package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupStandaloneWorkspace(t *testing.T) (string, *Server) {
	t.Helper()
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "standalone-ws")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	createsgaiDir(t, workspace)
	srv := NewServer(rootDir)
	return workspace, srv
}

func TestCreateWorkspacePinsByDefault(t *testing.T) {
	rootDir := t.TempDir()
	srv := NewServer(rootDir)

	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	body := strings.NewReader(`{"name":"new-project"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	workspacePath := filepath.Join(rootDir, "new-project")
	if !srv.isPinned(workspacePath) {
		t.Error("newly created workspace should be pinned by default")
	}
}

func TestForkWorkspacePinsByDefault(t *testing.T) {
	_, _, srv := setupM6TestWorkspace(t)

	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	body := strings.NewReader(`{"name":"pinned-fork"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/fork", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var result apiForkResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
		t.Fatalf("failed to decode response: %v", errDecode)
	}

	if !srv.isPinned(result.Dir) {
		t.Error("newly created fork should be pinned by default")
	}
}

func TestHandleAPIDeleteWorkspace(t *testing.T) {
	t.Run("successfulDelete", func(t *testing.T) {
		workspace, srv := setupStandaloneWorkspace(t)
		wsName := filepath.Base(workspace)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsName+"/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		var result apiDeleteWorkspaceResponse
		if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
			t.Fatalf("failed to decode response: %v", errDecode)
		}
		if !result.Deleted {
			t.Error("deleted should be true")
		}

		if _, errStat := os.Stat(workspace); !os.IsNotExist(errStat) {
			t.Error("workspace directory should not exist after deletion")
		}
	})

	t.Run("rejectsMissingConfirm", func(t *testing.T) {
		workspace, srv := setupStandaloneWorkspace(t)
		wsName := filepath.Base(workspace)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"confirm":false}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsName+"/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejectsRootWorkspace", func(t *testing.T) {
		_, _, srv := setupM6TestWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/root-workspace/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusBadRequest, resp.Body.String())
		}
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		_, srv := setupStandaloneWorkspace(t)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/nonexistent/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusNotFound)
		}
	})

	t.Run("invalidJSON", func(t *testing.T) {
		workspace, srv := setupStandaloneWorkspace(t)
		wsName := filepath.Base(workspace)

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`not json`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsName+"/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; want %d", resp.Code, http.StatusBadRequest)
		}
	})

	t.Run("cleansPinnedState", func(t *testing.T) {
		workspace, srv := setupStandaloneWorkspace(t)
		wsName := filepath.Base(workspace)

		srv.mu.Lock()
		srv.pinnedDirs[workspace] = true
		srv.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsName+"/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		if srv.isPinned(workspace) {
			t.Error("workspace should no longer be pinned after deletion")
		}
	})

	t.Run("stopsRunningSession", func(t *testing.T) {
		workspace, srv := setupStandaloneWorkspace(t)
		wsName := filepath.Base(workspace)

		srv.mu.Lock()
		srv.sessions[workspace] = &session{running: true}
		srv.mu.Unlock()

		mux := http.NewServeMux()
		srv.registerAPIRoutes(mux)

		body := strings.NewReader(`{"confirm":true}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+wsName+"/delete", body)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d; body = %s", resp.Code, http.StatusOK, resp.Body.String())
		}

		srv.mu.Lock()
		sess := srv.sessions[workspace]
		srv.mu.Unlock()

		if sess != nil && sess.running {
			t.Error("session should have been stopped")
		}
	})
}
