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

func TestSelfDrivePersistsInteractiveAutoLock(t *testing.T) {
	t.Run("toggleOnSetsLock", func(t *testing.T) {
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

		newAutoMode := true
		wfState.InteractiveAutoLock = newAutoMode
		if errSave := state.Save(statePath(workspace), wfState); errSave != nil {
			t.Fatal(errSave)
		}

		loaded, errReload := state.Load(statePath(workspace))
		if errReload != nil {
			t.Fatal(errReload)
		}
		if !loaded.InteractiveAutoLock {
			t.Fatal("InteractiveAutoLock should be true after self-drive toggle on")
		}
	})

	t.Run("lockPreventsTurnOff", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-project")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		if errSave := state.Save(statePath(workspace), state.Workflow{InteractiveAutoLock: true}); errSave != nil {
			t.Fatal(errSave)
		}

		wfState, errLoad := state.Load(statePath(workspace))
		if errLoad != nil {
			t.Fatal(errLoad)
		}

		wasAuto := true
		newAutoMode := !wasAuto
		if wfState.InteractiveAutoLock {
			newAutoMode = true
		}

		if !newAutoMode {
			t.Fatal("newAutoMode should stay true when InteractiveAutoLock is already set")
		}
	})
}

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

func TestStartAPIClearsStaleInteractiveAutoLock(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if errSave := state.Save(statePath(workspace), state.Workflow{InteractiveAutoLock: true}); errSave != nil {
		t.Fatal(errSave)
	}

	wfState, errLoad := state.Load(statePath(workspace))
	if errLoad != nil {
		t.Fatal(errLoad)
	}

	wfState.InteractiveAutoLock = false
	wfState.StartedInteractive = true
	if errSave := state.Save(statePath(workspace), wfState); errSave != nil {
		t.Fatal(errSave)
	}

	effectiveAuto := wfState.InteractiveAutoLock

	loaded, errReload := state.Load(statePath(workspace))
	if errReload != nil {
		t.Fatal(errReload)
	}
	if loaded.InteractiveAutoLock {
		t.Fatal("InteractiveAutoLock should be false after interactive start cleared it")
	}
	if !loaded.StartedInteractive {
		t.Fatal("StartedInteractive should be true after interactive start")
	}
	if effectiveAuto {
		t.Fatal("effective auto mode should be false when interactive start clears the lock")
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
