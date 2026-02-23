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

func TestSelfDriveSetsInteractionMode(t *testing.T) {
	t.Run("selfDriveSetsMode", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-project")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		wfState, errLoad := state.Load(statePath(workspace))
		if errLoad != nil && !os.IsNotExist(errLoad) {
			t.Fatal(errLoad)
		}

		wfState.InteractionMode = state.ModeSelfDrive
		if errSave := state.Save(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		loaded, errReload := state.Load(statePath(workspace))
		if errReload != nil {
			t.Fatal(errReload)
		}
		if loaded.InteractionMode != state.ModeSelfDrive {
			t.Fatalf("InteractionMode should be %q after self-drive, got %q", state.ModeSelfDrive, loaded.InteractionMode)
		}
	})

	t.Run("selfDriveSticky", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-project")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		if errSave := state.Save(statePath(workspace), state.Workflow{InteractionMode: state.ModeSelfDrive}); errSave != nil {
			t.Fatal(errSave)
		}

		loaded, errLoad := state.Load(statePath(workspace))
		if errLoad != nil {
			t.Fatal(errLoad)
		}

		if loaded.InteractionMode != state.ModeSelfDrive {
			t.Fatal("self-drive mode should have InteractionMode == ModeSelfDrive")
		}
		if loaded.ToolsAllowed() {
			t.Fatal("self-drive mode should not allow tools")
		}
	})
}

func TestBuildWorkspaceDetailUsesInteractionMode(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSaveState := state.Save(statePath(workspace), state.Workflow{InteractionMode: state.ModeSelfDrive}); errSaveState != nil {
		t.Fatal(errSaveState)
	}

	srv := NewServer(rootDir)
	detail := srv.buildWorkspaceDetail(workspace)
	if !detail.InteractiveAuto {
		t.Fatal("workspace detail should report interactive auto when mode is self-drive")
	}
}

func TestStartAPISetsModeBrainstorming(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSave := state.Save(statePath(workspace), state.Workflow{InteractionMode: state.ModeSelfDrive}); errSave != nil {
		t.Fatal(errSave)
	}

	wfState, errLoad := state.Load(statePath(workspace))
	if errLoad != nil {
		t.Fatal(errLoad)
	}

	wfState.InteractionMode = state.ModeBrainstorming
	if errSave := state.Save(statePath(workspace), wfState); errSave != nil {
		t.Fatal(errSave)
	}

	loaded, errReload := state.Load(statePath(workspace))
	if errReload != nil {
		t.Fatal(errReload)
	}
	if loaded.InteractionMode != state.ModeBrainstorming {
		t.Fatalf("InteractionMode should be %q after interactive start, got %q", state.ModeBrainstorming, loaded.InteractionMode)
	}
	if loaded.InteractionMode == state.ModeSelfDrive {
		t.Fatal("brainstorming mode should not be self-drive")
	}
	if !loaded.ToolsAllowed() {
		t.Fatal("brainstorming mode should allow tools")
	}
}

func TestHandleAPIWorkspaceSessionUsesInteractionMode(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSaveState := state.Save(statePath(workspace), state.Workflow{InteractionMode: state.ModeSelfDrive}); errSaveState != nil {
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
		t.Fatal("session response should report interactive auto when mode is self-drive")
	}
}
