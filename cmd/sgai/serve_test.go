package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func createsgaiDir(t *testing.T, projectDir string) {
	t.Helper()
	sgaiDir := filepath.Join(projectDir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatalf("Failed to create .sgai dir: %v", err)
	}
}

type mockEditorOpener struct {
	calls []string
	err   error
}

func (m *mockEditorOpener) open(path string) error {
	m.calls = append(m.calls, path)
	return m.err
}

// TestScanForProjects tests that scanForProjects returns only directories with .sgai.
func TestScanForProjects(t *testing.T) {
	rootDir := t.TempDir()

	projectWithsgai := filepath.Join(rootDir, "project-with-sgai")
	if err := os.MkdirAll(projectWithsgai, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, projectWithsgai)

	projectWithGoalOnly := filepath.Join(rootDir, "project-with-goal-only")
	if err := os.MkdirAll(projectWithGoalOnly, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectWithGoalOnly, "GOAL.md"), []byte("# Goal"), 0644); err != nil {
		t.Fatalf("Failed to create GOAL.md: %v", err)
	}

	emptyDir := filepath.Join(rootDir, "empty-dir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty dir: %v", err)
	}

	regularFile := filepath.Join(rootDir, "regular-file.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	projects, err := scanForProjects(rootDir)
	if err != nil {
		t.Fatalf("scanForProjects() error: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("scanForProjects() returned %d projects, want 3", len(projects))
	}

	var withWorkspace []string
	var withoutWorkspace []string
	for _, p := range projects {
		if p.HasWorkspace {
			withWorkspace = append(withWorkspace, p.DirName)
		} else {
			withoutWorkspace = append(withoutWorkspace, p.DirName)
		}
	}

	if len(withWorkspace) != 1 || withWorkspace[0] != "project-with-sgai" {
		t.Errorf("expected 1 project with workspace (project-with-sgai), got %v", withWorkspace)
	}

	if len(withoutWorkspace) != 2 {
		t.Errorf("expected 2 projects without workspace, got %v", withoutWorkspace)
	}
}

func TestScanForProjectsIncludesGoalMDOnly(t *testing.T) {
	rootDir := t.TempDir()

	projectWithGoalOnly := filepath.Join(rootDir, "legacy-project")
	if err := os.MkdirAll(projectWithGoalOnly, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectWithGoalOnly, "GOAL.md"), []byte("# Goal"), 0644); err != nil {
		t.Fatalf("Failed to create GOAL.md: %v", err)
	}

	projects, err := scanForProjects(rootDir)
	if err != nil {
		t.Fatalf("scanForProjects() error: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("scanForProjects() returned %d projects, want 1", len(projects))
	}

	if len(projects) > 0 && projects[0].HasWorkspace {
		t.Errorf("expected HasWorkspace=false for GOAL.md-only directory")
	}
}

// TestScanForProjectsMultiple tests scanning with multiple valid projects.
func TestScanForProjectsMultiple(t *testing.T) {
	rootDir := t.TempDir()

	projectNames := []string{"alpha", "beta", "gamma"}
	for _, name := range projectNames {
		projectDir := filepath.Join(rootDir, name)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}
		createsgaiDir(t, projectDir)
	}

	projects, err := scanForProjects(rootDir)
	if err != nil {
		t.Fatalf("scanForProjects() error: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("scanForProjects() returned %d projects, want 3", len(projects))
	}

	gotNames := make([]string, len(projects))
	for i, p := range projects {
		gotNames[i] = p.DirName
	}

	for _, want := range projectNames {
		if !slices.Contains(gotNames, want) {
			t.Errorf("scanForProjects() missing project %q, got %v", want, gotNames)
		}
	}
}

// TestValidateDirectory tests the security validation of directory paths.
func TestValidateDirectory(t *testing.T) {
	rootDir := t.TempDir()

	validProject := filepath.Join(rootDir, "valid-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create valid project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	nestedProject := filepath.Join(rootDir, "nested", "project")
	if err := os.MkdirAll(nestedProject, 0755); err != nil {
		t.Fatalf("Failed to create nested project dir: %v", err)
	}
	createsgaiDir(t, nestedProject)

	nosgaiProject := filepath.Join(rootDir, "no-sgai")
	if err := os.MkdirAll(nosgaiProject, 0755); err != nil {
		t.Fatalf("Failed to create no-sgai dir: %v", err)
	}

	legacyGoalProject := filepath.Join(rootDir, "legacy-goal")
	if err := os.MkdirAll(legacyGoalProject, 0755); err != nil {
		t.Fatalf("Failed to create legacy-goal dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyGoalProject, "GOAL.md"), []byte("# Goal"), 0644); err != nil {
		t.Fatalf("Failed to create GOAL.md: %v", err)
	}

	srv := NewServer(rootDir)

	tests := []struct {
		name      string
		dir       string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid project path",
			dir:     validProject,
			wantErr: false,
		},
		{
			name:    "valid nested project path",
			dir:     nestedProject,
			wantErr: false,
		},
		{
			name:      "path traversal with ..",
			dir:       filepath.Join(rootDir, "..", "outside"),
			wantErr:   true,
			errSubstr: "path traversal",
		},
		{
			name:      "path traversal in valid prefix",
			dir:       filepath.Join(validProject, "..", "..", "outside"),
			wantErr:   true,
			errSubstr: "path traversal",
		},
		{
			name:      "absolute path outside root",
			dir:       "/etc/passwd",
			wantErr:   true,
			errSubstr: "path traversal",
		},
		{
			name:      "empty directory",
			dir:       "",
			wantErr:   true,
			errSubstr: "directory is required",
		},
		{
			name:    "directory without .sgai",
			dir:     nosgaiProject,
			wantErr: false,
		},
		{
			name:    "legacy directory with GOAL.md only",
			dir:     legacyGoalProject,
			wantErr: false,
		},
		{
			name:    "non-existent directory within root",
			dir:     filepath.Join(rootDir, "does-not-exist"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := srv.validateDirectory(tt.dir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateDirectory() expected error containing %q, got nil", tt.errSubstr)
					return
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("validateDirectory() error = %q, want error containing %q", err.Error(), tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("validateDirectory() unexpected error: %v", err)
					return
				}
				if result == "" {
					t.Error("validateDirectory() returned empty path for valid directory")
				}
			}
		})
	}
}

// TestValidateDirectoryTraversalVariants tests various path traversal attack vectors.
func TestValidateDirectoryTraversalVariants(t *testing.T) {
	rootDir := t.TempDir()

	validProject := filepath.Join(rootDir, "valid-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create valid project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)

	attackVectors := []string{
		"../../../etc/passwd",
		"valid-project/../../../etc/passwd",
		"./valid-project/../../../etc/passwd",
		"/etc/passwd",
		"/tmp/malicious",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"....//....//....//etc/passwd",
	}

	for _, attack := range attackVectors {
		t.Run(attack, func(t *testing.T) {
			_, err := srv.validateDirectory(attack)
			if err == nil {
				t.Errorf("validateDirectory(%q) should have been rejected, but was accepted", attack)
			}
		})
	}
}

func TestIsLocalRequest(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		{"IPv4 localhost", "127.0.0.1:54321", true},
		{"IPv6 localhost bracketed", "[::1]:54321", true},
		{"External IPv4", "192.168.1.100:54321", false},
		{"External IPv6 bracketed", "[2001:db8::1]:54321", false},
		{"No port separator", "127.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			got := isLocalRequest(req)
			if got != tt.want {
				t.Errorf("isLocalRequest() with RemoteAddr %q = %v, want %v", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

func TestHandleWorkspaceOpenVSCodeMethodNotAllowed(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/test-project/open-vscode", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleWorkspaceOpenVSCode(rec, req, validProject)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET request should return 405 Method Not Allowed, got %d", rec.Code)
	}
}

func TestHandleWorkspaceOpenVSCodeForbiddenForRemote(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)
	srv.editorAvailable = true

	req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/open-vscode", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	rec := httptest.NewRecorder()

	srv.handleWorkspaceOpenVSCode(rec, req, validProject)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Remote request should return 403 Forbidden, got %d", rec.Code)
	}
}

func TestHandleWorkspaceOpenVSCodeUnavailable(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)
	srv.editorAvailable = false

	req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/open-vscode", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleWorkspaceOpenVSCode(rec, req, validProject)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Request with code unavailable should return 503 Service Unavailable, got %d", rec.Code)
	}
}

func TestHandleWorkspaceOpenVSCodeInvalidFile(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)
	srv.editorAvailable = true

	req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/open-vscode?file=../../../etc/passwd", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleWorkspaceOpenVSCode(rec, req, validProject)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Request with invalid file should return 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandleWorkspaceOpenVSCodeAllowedFiles(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	mock := &mockEditorOpener{}
	srv := NewServer(rootDir)
	srv.editorAvailable = true
	srv.editor = mock

	allowedFiles := []string{"GOAL.md", "PROJECT_MANAGEMENT.md"}
	disallowedFiles := []string{"../etc/passwd", "../../config", "random.txt", ".sgai/state.json"}

	for _, file := range allowedFiles {
		t.Run("allowed_"+file, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/open-vscode?file="+file, nil)
			req.RemoteAddr = "127.0.0.1:54321"
			rec := httptest.NewRecorder()

			srv.handleWorkspaceOpenVSCode(rec, req, validProject)

			if rec.Code == http.StatusBadRequest {
				t.Errorf("Request with allowed file %q should not return 400 Bad Request", file)
			}
		})
	}

	expectedPaths := []string{
		filepath.Join(validProject, "GOAL.md"),
		filepath.Join(validProject, ".sgai", "PROJECT_MANAGEMENT.md"),
	}
	if !slices.Equal(mock.calls, expectedPaths) {
		t.Errorf("editor.open called with %v; want %v", mock.calls, expectedPaths)
	}

	for _, file := range disallowedFiles {
		t.Run("disallowed_"+file, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/open-vscode?file="+file, nil)
			req.RemoteAddr = "127.0.0.1:54321"
			rec := httptest.NewRecorder()

			srv.handleWorkspaceOpenVSCode(rec, req, validProject)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Request with disallowed file %q should return 400 Bad Request, got %d", file, rec.Code)
			}
		})
	}
}

func TestHandleWorkspaceOpenVSCodeEditorError(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	mock := &mockEditorOpener{err: os.ErrPermission}
	srv := NewServer(rootDir)
	srv.editorAvailable = true
	srv.editor = mock

	req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/open-vscode", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleWorkspaceOpenVSCode(rec, req, validProject)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("editor error should return 500, got %d", rec.Code)
	}
}

func TestNewServerWithConfigEditorFallback(t *testing.T) {
	t.Run("fallbackToDefaultWhenEnvEditorUnavailable", func(t *testing.T) {
		rootDir := t.TempDir()
		fakeBinDir := t.TempDir()
		fakeCode := filepath.Join(fakeBinDir, "code")
		if err := os.WriteFile(fakeCode, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("failed to create fake code binary: %v", err)
		}
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("EDITOR", "/nonexistent/editor/binary")
		t.Setenv("VISUAL", "")
		srv := NewServerWithConfig(rootDir, "")
		if srv.editorName != defaultEditorPreset {
			t.Errorf("editorName = %q; want %q (default preset fallback)", srv.editorName, defaultEditorPreset)
		}
	})

	t.Run("usesEnvEditorWhenAvailable", func(t *testing.T) {
		rootDir := t.TempDir()
		fakeEditor := filepath.Join(t.TempDir(), "fakeeditor")
		if err := os.WriteFile(fakeEditor, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("failed to create fake editor: %v", err)
		}
		t.Setenv("EDITOR", fakeEditor)
		t.Setenv("VISUAL", "")
		srv := NewServerWithConfig(rootDir, "")
		if srv.editorName == defaultEditorPreset {
			t.Errorf("editorName = %q; should use env EDITOR, not default preset", srv.editorName)
		}
		if !srv.editorAvailable {
			t.Error("editorAvailable = false; want true for available env editor")
		}
	})

	t.Run("usesConfigEditorOverEnv", func(t *testing.T) {
		rootDir := t.TempDir()
		fakeBinDir := t.TempDir()
		fakeCode := filepath.Join(fakeBinDir, "code")
		if err := os.WriteFile(fakeCode, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("failed to create fake code binary: %v", err)
		}
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("EDITOR", "/nonexistent/editor/binary")
		t.Setenv("VISUAL", "")
		srv := NewServerWithConfig(rootDir, "code")
		if srv.editorName != "code" {
			t.Errorf("editorName = %q; want %q", srv.editorName, "code")
		}
		if !srv.editorAvailable {
			t.Error("editorAvailable = false; want true for explicitly configured editor")
		}
	})
}

func TestHandleWorkspaceInit(t *testing.T) {
	t.Run("postCreatesWorkspace", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "new-project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		srv := NewServer(rootDir)

		req := httptest.NewRequest(http.MethodPost, "/workspaces/new-project/init", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceInit(rec, req, projectDir)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("POST init expected 303 redirect, got %d", rec.Code)
		}

		sgaiDir := filepath.Join(projectDir, ".sgai")
		info, err := os.Stat(sgaiDir)
		if err != nil {
			t.Fatalf(".sgai directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Fatal(".sgai is not a directory")
		}

		goalPath := filepath.Join(projectDir, "GOAL.md")
		content, err := os.ReadFile(goalPath)
		if err != nil {
			t.Fatalf("GOAL.md was not created: %v", err)
		}
		if len(content) == 0 {
			t.Fatal("GOAL.md is empty")
		}

		assertSkeletonUnpacked(t, projectDir)
	})

	t.Run("nonPostMethodNotAllowed", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "new-project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		srv := NewServer(rootDir)

		methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/workspaces/new-project/init", nil)
			rec := httptest.NewRecorder()

			srv.handleWorkspaceInit(rec, req, projectDir)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s request expected 405, got %d", method, rec.Code)
			}
		}
	})

	t.Run("initSetsHasWorkspaceTrue", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "uninit-project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		if hassgaiDirectory(projectDir) {
			t.Fatal("project should not have .sgai before init")
		}

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodPost, "/workspaces/uninit-project/init", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceInit(rec, req, projectDir)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("POST init expected 303 redirect, got %d", rec.Code)
		}

		if !hassgaiDirectory(projectDir) {
			t.Fatal("project should have .sgai after init")
		}
	})
}

func TestIsStaleWorkingState(t *testing.T) {
	cases := []struct {
		name    string
		running bool
		status  string
		want    bool
	}{
		{"runningWorking", true, state.StatusWorking, false},
		{"runningAgentDone", true, state.StatusAgentDone, false},
		{"runningComplete", true, state.StatusComplete, false},
		{"runningWaitingForHuman", true, state.StatusWaitingForHuman, false},
		{"stoppedWorking", false, state.StatusWorking, true},
		{"stoppedAgentDone", false, state.StatusAgentDone, true},
		{"stoppedComplete", false, state.StatusComplete, false},
		{"stoppedWaitingForHuman", false, state.StatusWaitingForHuman, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wf := state.Workflow{Status: tc.status}
			got := isStaleWorkingState(tc.running, wf)
			if got != tc.want {
				t.Errorf("isStaleWorkingState(running=%v, status=%q) = %v; want %v", tc.running, tc.status, got, tc.want)
			}
		})
	}
}

func TestHandleWorkspaceResetState(t *testing.T) {
	t.Run("postDeletesStateFile", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		sp := statePath(projectDir)
		if err := state.Save(sp, state.Workflow{Status: state.StatusWorking}); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/reset-state", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceResetState(rec, req, projectDir)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("POST reset-state expected 303 redirect, got %d", rec.Code)
		}

		loc := rec.Header().Get("Location")
		if !strings.Contains(loc, "/progress") {
			t.Errorf("expected redirect to progress page, got %q", loc)
		}

		if _, err := os.Stat(sp); !os.IsNotExist(err) {
			t.Error("state.json should be deleted after reset")
		}
	})

	t.Run("postMissingFileSucceeds", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/reset-state", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceResetState(rec, req, projectDir)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("POST reset-state with no state file expected 303 redirect, got %d", rec.Code)
		}
	})

	t.Run("nonPostMethodNotAllowed", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		srv := NewServer(rootDir)
		methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/workspaces/test-project/reset-state", nil)
			rec := httptest.NewRecorder()

			srv.handleWorkspaceResetState(rec, req, projectDir)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s request expected 405, got %d", method, rec.Code)
			}
		}
	})

	t.Run("redirectsToProgress", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/reset-state", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceResetState(rec, req, projectDir)

		loc := rec.Header().Get("Location")
		if !strings.HasSuffix(loc, "/progress") {
			t.Errorf("expected redirect location ending with /progress, got %q", loc)
		}
	})
}

func TestHasAnyNeedsInput(t *testing.T) {
	t.Run("emptySlice", func(t *testing.T) {
		got := hasAnyNeedsInput([]workspaceInfo{})
		if got {
			t.Error("hasAnyNeedsInput([]) = true; want false")
		}
	})

	t.Run("nilSlice", func(t *testing.T) {
		got := hasAnyNeedsInput(nil)
		if got {
			t.Error("hasAnyNeedsInput(nil) = true; want false")
		}
	})

	t.Run("noWorkspaceNeedsInput", func(t *testing.T) {
		workspaces := []workspaceInfo{
			{Directory: "/a", NeedsInput: false},
			{Directory: "/b", NeedsInput: false},
			{Directory: "/c", NeedsInput: false},
		}
		got := hasAnyNeedsInput(workspaces)
		if got {
			t.Error("hasAnyNeedsInput(none need input) = true; want false")
		}
	})

	t.Run("oneWorkspaceNeedsInput", func(t *testing.T) {
		workspaces := []workspaceInfo{
			{Directory: "/a", NeedsInput: false},
			{Directory: "/b", NeedsInput: true},
			{Directory: "/c", NeedsInput: false},
		}
		got := hasAnyNeedsInput(workspaces)
		if !got {
			t.Error("hasAnyNeedsInput(one needs input) = false; want true")
		}
	})

	t.Run("multipleWorkspacesNeedInput", func(t *testing.T) {
		workspaces := []workspaceInfo{
			{Directory: "/a", NeedsInput: true},
			{Directory: "/b", NeedsInput: true},
		}
		got := hasAnyNeedsInput(workspaces)
		if !got {
			t.Error("hasAnyNeedsInput(multiple need input) = false; want true")
		}
	})
}

func TestCollectInProgressWorkspaces(t *testing.T) {
	t.Run("emptyGroups", func(t *testing.T) {
		got := collectInProgressWorkspaces([]workspaceGroup{})
		if len(got) != 0 {
			t.Errorf("collectInProgressWorkspaces([]) = %d items; want 0", len(got))
		}
	})

	t.Run("noInProgress", func(t *testing.T) {
		groups := []workspaceGroup{
			{Root: workspaceInfo{Directory: "/a", InProgress: false}},
			{Root: workspaceInfo{Directory: "/b", InProgress: false}},
		}
		got := collectInProgressWorkspaces(groups)
		if len(got) != 0 {
			t.Errorf("collectInProgressWorkspaces(none in progress) = %d items; want 0", len(got))
		}
	})

	t.Run("rootInProgress", func(t *testing.T) {
		groups := []workspaceGroup{
			{Root: workspaceInfo{Directory: "/a", InProgress: true}},
			{Root: workspaceInfo{Directory: "/b", InProgress: false}},
		}
		got := collectInProgressWorkspaces(groups)
		if len(got) != 1 {
			t.Errorf("collectInProgressWorkspaces(one root in progress) = %d items; want 1", len(got))
		}
		if got[0].Directory != "/a" {
			t.Errorf("got directory %q; want %q", got[0].Directory, "/a")
		}
	})

	t.Run("forkInProgress", func(t *testing.T) {
		groups := []workspaceGroup{
			{
				Root: workspaceInfo{Directory: "/a", InProgress: false},
				Forks: []workspaceInfo{
					{Directory: "/a/fork1", InProgress: true},
					{Directory: "/a/fork2", InProgress: false},
				},
			},
		}
		got := collectInProgressWorkspaces(groups)
		if len(got) != 1 {
			t.Errorf("collectInProgressWorkspaces(one fork in progress) = %d items; want 1", len(got))
		}
		if got[0].Directory != "/a/fork1" {
			t.Errorf("got directory %q; want %q", got[0].Directory, "/a/fork1")
		}
	})

	t.Run("mixedInProgress", func(t *testing.T) {
		groups := []workspaceGroup{
			{
				Root: workspaceInfo{Directory: "/a", InProgress: true},
				Forks: []workspaceInfo{
					{Directory: "/a/fork1", InProgress: true},
					{Directory: "/a/fork2", InProgress: false},
				},
			},
			{Root: workspaceInfo{Directory: "/b", InProgress: false}},
		}
		got := collectInProgressWorkspaces(groups)
		if len(got) != 2 {
			t.Errorf("collectInProgressWorkspaces(mixed) = %d items; want 2", len(got))
		}
	})
}

func TestBuildWorkspacePageData(t *testing.T) {
	t.Run("inProgressWorkspacesLenZeroWhenEmpty", func(t *testing.T) {
		data := buildWorkspacePageData([]workspaceGroup{}, "/path", "tab", "session", "")
		if len(data.InProgressWorkspaces) != 0 {
			t.Errorf("len(InProgressWorkspaces) = %d; want 0", len(data.InProgressWorkspaces))
		}
	})

	t.Run("hasNeedsInputWorkspaceCorrectlySet", func(t *testing.T) {
		groups := []workspaceGroup{
			{Root: workspaceInfo{Directory: "/a", InProgress: true, NeedsInput: false}},
		}
		data := buildWorkspacePageData(groups, "/a", "tab", "", "")
		if data.HasNeedsInputWorkspace {
			t.Error("HasNeedsInputWorkspace = true; want false (no workspace needs input)")
		}

		groupsWithNeedsInput := []workspaceGroup{
			{Root: workspaceInfo{Directory: "/a", InProgress: true, NeedsInput: true}},
		}
		dataWithNeedsInput := buildWorkspacePageData(groupsWithNeedsInput, "/a", "tab", "", "")
		if !dataWithNeedsInput.HasNeedsInputWorkspace {
			t.Error("HasNeedsInputWorkspace = false; want true (workspace needs input)")
		}
	})
}

func assertSkeletonUnpacked(t *testing.T, projectDir string) {
	t.Helper()
	skeletonFiles := []string{
		filepath.Join(projectDir, ".sgai", "agent", "coordinator.md"),
		filepath.Join(projectDir, ".sgai", "opencode.jsonc"),
	}
	for _, path := range skeletonFiles {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("skeleton file not found after init: %s", path)
		}
	}
}

func TestUnpackSkeleton(t *testing.T) {
	dir := t.TempDir()
	if err := unpackSkeleton(dir); err != nil {
		t.Fatalf("unpackSkeleton failed: %v", err)
	}
	assertSkeletonUnpacked(t, dir)
}

func TestAddGitExclude(t *testing.T) {
	t.Run("addsExcludeEntry", func(t *testing.T) {
		dir := t.TempDir()
		gitInfoDir := filepath.Join(dir, ".git", "info")
		if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
			t.Fatalf("failed to create .git/info: %v", err)
		}
		if err := addGitExclude(dir); err != nil {
			t.Fatalf("addGitExclude failed: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(gitInfoDir, "exclude"))
		if err != nil {
			t.Fatalf("failed to read exclude file: %v", err)
		}
		if !strings.Contains(string(content), "/.sgai") {
			t.Error("exclude file does not contain /.sgai")
		}
	})

	t.Run("skipsWhenAlreadyPresent", func(t *testing.T) {
		dir := t.TempDir()
		gitInfoDir := filepath.Join(dir, ".git", "info")
		if err := os.MkdirAll(gitInfoDir, 0755); err != nil {
			t.Fatalf("failed to create .git/info: %v", err)
		}
		excludePath := filepath.Join(gitInfoDir, "exclude")
		if err := os.WriteFile(excludePath, []byte("/.sgai\n"), 0644); err != nil {
			t.Fatalf("failed to write exclude: %v", err)
		}
		if err := addGitExclude(dir); err != nil {
			t.Fatalf("addGitExclude failed: %v", err)
		}
		content, err := os.ReadFile(excludePath)
		if err != nil {
			t.Fatalf("failed to read exclude file: %v", err)
		}
		if strings.Count(string(content), "/.sgai") != 1 {
			t.Error("/.sgai should appear exactly once when already present")
		}
	})

	t.Run("noGitDirectory", func(t *testing.T) {
		dir := t.TempDir()
		if err := addGitExclude(dir); err != nil {
			t.Fatalf("addGitExclude should not fail without .git: %v", err)
		}
	})
}

func TestWriteGoalExample(t *testing.T) {
	dir := t.TempDir()
	if err := writeGoalExample(dir); err != nil {
		t.Fatalf("writeGoalExample failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "GOAL.md"))
	if err != nil {
		t.Fatalf("GOAL.md was not created: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("GOAL.md is empty")
	}
}
