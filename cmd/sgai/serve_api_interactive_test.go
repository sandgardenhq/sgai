package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestBuildWorkspaceDetailUsesWorkflowAutoLock(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSaveState := state.Save(statePath(workspace), state.Workflow{InteractiveAutoLock: true}); errSaveState != nil {
		t.Fatal(errSaveState)
	}

	srv := NewServer(rootDir)
	detail := srv.buildWorkspaceDetail(workspace)
	if !detail.InteractiveAuto {
		t.Fatal("workspace detail should report interactive auto when workflow lock is enabled")
	}
}

func TestHandleAPIWorkspaceSessionUsesWorkflowAutoLock(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSaveState := state.Save(statePath(workspace), state.Workflow{InteractiveAutoLock: true}); errSaveState != nil {
		t.Fatal(errSaveState)
	}

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/test-project/session", nil)
	req.SetPathValue("name", "test-project")
	resp := httptest.NewRecorder()

	srv.handleAPIWorkspaceSession(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
	}

	var got apiSessionResponse
	if errDecode := json.NewDecoder(resp.Body).Decode(&got); errDecode != nil {
		t.Fatalf("failed to decode response: %v", errDecode)
	}
	if !got.InteractiveAuto {
		t.Fatal("session response should report interactive auto when workflow lock is enabled")
	}
}
