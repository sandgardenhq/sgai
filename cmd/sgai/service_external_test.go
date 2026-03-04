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
)

func newTestServerForExternal(t *testing.T) (*Server, string) {
	t.Helper()
	rootDir := t.TempDir()
	configDir := t.TempDir()
	srv := &Server{
		sessions:           make(map[string]*session),
		everStartedDirs:    make(map[string]bool),
		pinnedDirs:         make(map[string]bool),
		pinnedConfigDir:    configDir,
		externalDirs:       make(map[string]bool),
		externalConfigDir:  configDir,
		rootDir:            rootDir,
		workspaceScanCache: newTTLCache[string, []workspaceGroup](0),
		classifyCache:      newTTLCache[string, workspaceKind](0),
		signals:            newSignalBroker(),
		stateCache:         newTTLCache[string, apiFactoryState](0),
	}
	return srv, rootDir
}

func TestLoadExternalDirs(t *testing.T) {
	t.Run("missingFile", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		if err := srv.loadExternalDirs(); err != nil {
			t.Fatalf("loadExternalDirs() unexpected error: %v", err)
		}
		if len(srv.externalDirs) != 0 {
			t.Errorf("externalDirs should be empty; got %d entries", len(srv.externalDirs))
		}
	})

	t.Run("emptyArray", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		if err := os.WriteFile(srv.externalFilePath(), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := srv.loadExternalDirs(); err != nil {
			t.Fatalf("loadExternalDirs() unexpected error: %v", err)
		}
		if len(srv.externalDirs) != 0 {
			t.Errorf("externalDirs should be empty; got %d entries", len(srv.externalDirs))
		}
	})

	t.Run("withPaths", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		dirA := t.TempDir()
		dirB := t.TempDir()
		data, _ := json.Marshal([]string{dirA, dirB})
		if err := os.WriteFile(srv.externalFilePath(), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := srv.loadExternalDirs(); err != nil {
			t.Fatalf("loadExternalDirs() unexpected error: %v", err)
		}
		if len(srv.externalDirs) != 2 {
			t.Fatalf("externalDirs should have 2 entries; got %d", len(srv.externalDirs))
		}
	})

	t.Run("invalidJSON", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		if err := os.WriteFile(srv.externalFilePath(), []byte("{invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := srv.loadExternalDirs(); err == nil {
			t.Error("loadExternalDirs() should return error for invalid JSON")
		}
	})

	t.Run("prunesNonexistentDirectories", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		realDir := t.TempDir()
		data, _ := json.Marshal([]string{realDir, "/nonexistent/path/that/does/not/exist"})
		if err := os.WriteFile(srv.externalFilePath(), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := srv.loadExternalDirs(); err != nil {
			t.Fatalf("loadExternalDirs() unexpected error: %v", err)
		}
		if len(srv.externalDirs) != 1 {
			t.Fatalf("externalDirs should have 1 entry after pruning; got %d", len(srv.externalDirs))
		}
	})
}

func TestSaveExternalDirs(t *testing.T) {
	t.Run("createsDirectoryAndFile", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "sgai")
		srv := &Server{
			externalDirs:      map[string]bool{"/path/to/a": true, "/path/to/b": true},
			externalConfigDir: configDir,
		}
		if err := srv.saveExternalDirs(); err != nil {
			t.Fatalf("saveExternalDirs() unexpected error: %v", err)
		}
		data, err := os.ReadFile(srv.externalFilePath())
		if err != nil {
			t.Fatalf("failed to read external.json: %v", err)
		}
		var dirs []string
		if err := json.Unmarshal(data, &dirs); err != nil {
			t.Fatalf("failed to parse external.json: %v", err)
		}
		if len(dirs) != 2 {
			t.Fatalf("expected 2 paths; got %d", len(dirs))
		}
	})

	t.Run("emptyDirs", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "sgai")
		srv := &Server{
			externalDirs:      make(map[string]bool),
			externalConfigDir: configDir,
		}
		if err := srv.saveExternalDirs(); err != nil {
			t.Fatalf("saveExternalDirs() unexpected error: %v", err)
		}
		data, err := os.ReadFile(srv.externalFilePath())
		if err != nil {
			t.Fatalf("failed to read external.json: %v", err)
		}
		var dirs []string
		if err := json.Unmarshal(data, &dirs); err != nil {
			t.Fatalf("failed to parse external.json: %v", err)
		}
		if len(dirs) != 0 {
			t.Errorf("expected empty array; got %v", dirs)
		}
	})
}

func TestAttachExternalWorkspaceService(t *testing.T) {
	t.Run("happyPath", func(t *testing.T) {
		srv, rootDir := newTestServerForExternal(t)
		externalDir := t.TempDir()
		_ = rootDir

		result, err := srv.attachExternalWorkspaceService(externalDir)
		if err != nil {
			t.Fatalf("attachExternalWorkspaceService() unexpected error: %v", err)
		}
		if result.Dir != externalDir {
			t.Errorf("Dir = %q; want %q", result.Dir, externalDir)
		}
		if result.Name != filepath.Base(externalDir) {
			t.Errorf("Name = %q; want %q", result.Name, filepath.Base(externalDir))
		}
		canonical := resolveSymlinks(externalDir)
		srv.mu.Lock()
		attached := srv.externalDirs[canonical]
		srv.mu.Unlock()
		if !attached {
			t.Error("externalDirs should contain the attached dir")
		}
	})

	t.Run("withGoalMD", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(externalDir, "GOAL.md"), []byte("# Goal\n"), 0644); err != nil {
			t.Fatal(err)
		}

		result, err := srv.attachExternalWorkspaceService(externalDir)
		if err != nil {
			t.Fatalf("attachExternalWorkspaceService() unexpected error: %v", err)
		}
		if !result.HasGoal {
			t.Error("HasGoal should be true when GOAL.md exists")
		}
	})

	t.Run("relativePathRejected", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		_, err := srv.attachExternalWorkspaceService("relative/path")
		if err == nil {
			t.Error("expected error for relative path")
		}
		if !strings.Contains(err.Error(), "absolute") {
			t.Errorf("error should mention absolute; got %q", err.Error())
		}
	})

	t.Run("nonExistentDirRejected", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		_, err := srv.attachExternalWorkspaceService("/nonexistent/path/xyz")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("underRootDirRejected", func(t *testing.T) {
		srv, rootDir := newTestServerForExternal(t)
		subDir := filepath.Join(rootDir, "myworkspace")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		_, err := srv.attachExternalWorkspaceService(subDir)
		if err == nil {
			t.Error("expected error for path under rootDir")
		}
		if !strings.Contains(err.Error(), "root directory") {
			t.Errorf("error should mention root directory; got %q", err.Error())
		}
	})

	t.Run("alreadyAttachedRejected", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		if _, err := srv.attachExternalWorkspaceService(externalDir); err != nil {
			t.Fatalf("first attach should succeed: %v", err)
		}
		_, err := srv.attachExternalWorkspaceService(externalDir)
		if err == nil {
			t.Error("expected error for already-attached directory")
		}
		if !strings.Contains(err.Error(), "already attached") {
			t.Errorf("error should mention already attached; got %q", err.Error())
		}
	})

	t.Run("persistsToDisk", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		if _, err := srv.attachExternalWorkspaceService(externalDir); err != nil {
			t.Fatalf("attachExternalWorkspaceService() unexpected error: %v", err)
		}

		srv2 := &Server{
			externalDirs:      make(map[string]bool),
			externalConfigDir: srv.externalConfigDir,
		}
		if err := srv2.loadExternalDirs(); err != nil {
			t.Fatalf("loadExternalDirs() unexpected error: %v", err)
		}
		canonical := resolveSymlinks(externalDir)
		if !srv2.externalDirs[canonical] {
			t.Error("attached external dir should persist across server instances")
		}
	})
}

func TestDetachExternalWorkspaceService(t *testing.T) {
	t.Run("happyPath", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		if _, err := srv.attachExternalWorkspaceService(externalDir); err != nil {
			t.Fatalf("attach failed: %v", err)
		}

		result, err := srv.detachExternalWorkspaceService(externalDir)
		if err != nil {
			t.Fatalf("detachExternalWorkspaceService() unexpected error: %v", err)
		}
		if !result.Detached {
			t.Error("Detached should be true")
		}

		canonical := resolveSymlinks(externalDir)
		srv.mu.Lock()
		stillAttached := srv.externalDirs[canonical]
		srv.mu.Unlock()
		if stillAttached {
			t.Error("externalDirs should not contain the detached dir")
		}

		if _, errStat := os.Stat(externalDir); errStat != nil {
			t.Errorf("detach should not delete the directory: %v", errStat)
		}
	})

	t.Run("notAttachedReturnsError", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		_, err := srv.detachExternalWorkspaceService(externalDir)
		if err == nil {
			t.Error("expected error for directory not attached")
		}
		if !strings.Contains(err.Error(), "not attached") {
			t.Errorf("error should mention not attached; got %q", err.Error())
		}
	})
}

func TestBrowseDirectoriesService(t *testing.T) {
	t.Run("listsSubdirectories", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "alpha"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "beta"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		entries, err := browseDirectoriesService(dir)
		if err != nil {
			t.Fatalf("browseDirectoriesService() unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries; got %d", len(entries))
		}
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name
		}
		if !slices.Contains(names, "alpha") {
			t.Errorf("expected alpha in entries; got %v", names)
		}
		if !slices.Contains(names, "beta") {
			t.Errorf("expected beta in entries; got %v", names)
		}
	})

	t.Run("filtersHiddenDirs", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".hidden"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "visible"), 0755); err != nil {
			t.Fatal(err)
		}

		entries, err := browseDirectoriesService(dir)
		if err != nil {
			t.Fatalf("browseDirectoriesService() unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry (hidden filtered); got %d", len(entries))
		}
		if entries[0].Name != "visible" {
			t.Errorf("expected visible; got %q", entries[0].Name)
		}
	})

	t.Run("nonExistentPathReturnsError", func(t *testing.T) {
		_, err := browseDirectoriesService("/nonexistent/path/xyz")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("emptyPathUsesHomeDir", func(t *testing.T) {
		entries, err := browseDirectoriesService("")
		if err != nil {
			t.Fatalf("browseDirectoriesService(\"\") unexpected error: %v", err)
		}
		_ = entries
	})

	t.Run("returnsIsDir", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755); err != nil {
			t.Fatal(err)
		}

		entries, err := browseDirectoriesService(dir)
		if err != nil {
			t.Fatalf("browseDirectoriesService() unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry; got %d", len(entries))
		}
		if !entries[0].IsDir {
			t.Error("IsDir should be true for directory entries")
		}
	})
}

func TestAPIBrowseDirectories(t *testing.T) {
	srv, _ := newTestServerForExternal(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "mysubdir"), 0755); err != nil {
		t.Fatal(err)
	}

	mux := serverMux(t, srv)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/browse-directories?path="+dir, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d: %s", w.Code, w.Body.String())
	}

	var resp apiBrowseDirectoriesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry; got %d", len(resp.Entries))
	}
	if resp.Entries[0].Name != "mysubdir" {
		t.Errorf("expected mysubdir; got %q", resp.Entries[0].Name)
	}
}

func TestAPIAttachWorkspace(t *testing.T) {
	t.Run("happyPath", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		mux := serverMux(t, srv)

		body := strings.NewReader(`{"path":"` + externalDir + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/attach", body)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201; got %d: %s", w.Code, w.Body.String())
		}

		var resp apiAttachWorkspaceResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Dir != externalDir {
			t.Errorf("Dir = %q; want %q", resp.Dir, externalDir)
		}
	})

	t.Run("alreadyAttachedReturnsConflict", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		if _, err := srv.attachExternalWorkspaceService(externalDir); err != nil {
			t.Fatalf("first attach failed: %v", err)
		}

		mux := serverMux(t, srv)

		body := strings.NewReader(`{"path":"` + externalDir + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/attach", body)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409; got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("relativePathReturnsBadRequest", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)

		mux := serverMux(t, srv)

		body := strings.NewReader(`{"path":"relative/path"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/attach", body)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400; got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestAPIDetachWorkspace(t *testing.T) {
	t.Run("happyPath", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)
		externalDir := t.TempDir()

		if _, err := srv.attachExternalWorkspaceService(externalDir); err != nil {
			t.Fatalf("attach failed: %v", err)
		}

		mux := serverMux(t, srv)

		body := strings.NewReader(`{"path":"` + externalDir + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/detach", body)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200; got %d: %s", w.Code, w.Body.String())
		}

		var resp apiDetachWorkspaceResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Detached {
			t.Error("Detached should be true")
		}
	})

	t.Run("notAttachedReturnsNotFound", func(t *testing.T) {
		srv, _ := newTestServerForExternal(t)

		mux := serverMux(t, srv)

		body := strings.NewReader(`{"path":"/some/not/attached/path"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/detach", body)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404; got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestExternalWorkspacesAppearInScan(t *testing.T) {
	rootDir := t.TempDir()
	configDir := t.TempDir()

	externalDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(externalDir, ".sgai"), 0755); err != nil {
		t.Fatal(err)
	}

	srv := &Server{
		sessions:           make(map[string]*session),
		everStartedDirs:    make(map[string]bool),
		pinnedDirs:         make(map[string]bool),
		pinnedConfigDir:    configDir,
		externalDirs:       map[string]bool{resolveSymlinks(externalDir): true},
		externalConfigDir:  configDir,
		rootDir:            rootDir,
		workspaceScanCache: newTTLCache[string, []workspaceGroup](0),
		classifyCache:      newTTLCache[string, workspaceKind](0),
		signals:            newSignalBroker(),
	}

	groups, err := srv.doScanWorkspaceGroups()
	if err != nil {
		t.Fatalf("doScanWorkspaceGroups() unexpected error: %v", err)
	}

	found := false
	for _, grp := range groups {
		if grp.Root.Directory == externalDir || grp.Root.Directory == resolveSymlinks(externalDir) {
			if !grp.Root.External {
				t.Error("External should be true for external workspace")
			}
			found = true
			break
		}
	}
	if !found {
		t.Errorf("external workspace %q should appear in scan results", externalDir)
	}
}
