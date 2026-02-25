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

func TestBuildWorkspaceFullStateUsesInteractionMode(t *testing.T) {
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
	ws := workspaceInfo{
		Directory: workspace,
		DirName:   "test-project",
	}
	detail := srv.buildWorkspaceFullState(ws, nil)
	if !detail.InteractiveAuto {
		t.Fatal("workspace full state should report interactive auto when mode is self-drive")
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

func TestHandleAPIStateUsesInteractionMode(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 1)
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(filepath.Join(workspace, ".jj", "repo"), 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSaveState := state.Save(statePath(workspace), state.Workflow{InteractionMode: state.ModeSelfDrive}); errSaveState != nil {
		t.Fatal(errSaveState)
	}

	srv := NewServer(rootDir)
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/state", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", resp.Code, http.StatusOK)
	}

	var got apiFactoryState
	if errDecode := json.NewDecoder(resp.Body).Decode(&got); errDecode != nil {
		t.Fatalf("failed to decode response: %v", errDecode)
	}
	if len(got.Workspaces) == 0 {
		t.Fatal("expected at least one workspace in state response")
	}
	if !got.Workspaces[0].InteractiveAuto {
		t.Fatal("state response should report interactive auto when mode is self-drive")
	}
}
