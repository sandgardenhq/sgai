package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

func installFakeJJWithWorkspaceList(t *testing.T, workspaceCount int, extraHandlers string) {
	t.Helper()
	fakeBinDir := t.TempDir()
	fakeJJ := filepath.Join(fakeBinDir, "jj")
	var workspaceOutput strings.Builder
	for i := range workspaceCount {
		if i == 0 {
			workspaceOutput.WriteString("default: /path/to/root\\n")
		} else {
			fmt.Fprintf(&workspaceOutput, "fork-%d: /path/to/fork-%d\\n", i, i)
		}
	}
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	script.WriteString("if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"root\" ]; then\n")
	script.WriteString("  if [ -n \"$JJ_FAKE_ROOT\" ]; then\n")
	script.WriteString("    printf \"%s\" \"$JJ_FAKE_ROOT\"\n")
	script.WriteString("  else\n")
	script.WriteString("    pwd\n")
	script.WriteString("  fi\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"list\" ]; then\n")
	script.WriteString("  printf \"" + workspaceOutput.String() + "\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	if extraHandlers != "" {
		script.WriteString(extraHandlers)
	}
	script.WriteString("exit 0\n")
	if err := os.WriteFile(fakeJJ, []byte(script.String()), 0755); err != nil {
		t.Fatalf("failed to create fake jj: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
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

func TestHandleNewWorkspacePostRedirectsToCompose(t *testing.T) {
	composerSessionsMu.Lock()
	composerSessions = make(map[string]*composerSession)
	composerSessionsMu.Unlock()

	rootDir := t.TempDir()
	srv := NewServer(rootDir)

	form := url.Values{}
	form.Set("name", "new-workspace")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	srv.handleNewWorkspacePost(resp, req)

	result := resp.Result()
	if result.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d; want %d", result.StatusCode, http.StatusSeeOther)
	}

	location := result.Header.Get("Location")
	if location != "/compose?workspace=new-workspace" {
		t.Fatalf("location = %q; want %q", location, "/compose?workspace=new-workspace")
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

func TestNormalizeForkName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercases",
			input: "MyFork",
			want:  "myfork",
		},
		{
			name:  "replacesSpaces",
			input: "My Fork",
			want:  "my-fork",
		},
		{
			name:  "trimsAndCollapsesSpaces",
			input: "  My   Fork  ",
			want:  "my-fork",
		},
		{
			name:  "preservesDashesAndUnderscores",
			input: "My-Fork_Name",
			want:  "my-fork-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeForkName(tt.input); got != tt.want {
				t.Errorf("normalizeForkName(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandleNewForkGet(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 2, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/root-workspace/fork/new", nil)
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleNewForkGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET new fork expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Fork Name") {
		t.Errorf("GET new fork response missing fork name label")
	}
}

func TestHandleNewForkPost(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 2, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("name=My_Fork")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/root-workspace/fork/new", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleNewForkPost(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST new fork expected 303, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("POST new fork expected Location header")
	}
	if !strings.Contains(location, "/workspaces/my-fork/spec") {
		t.Errorf("POST new fork redirect = %q; want fork spec", location)
	}
	goalPath := filepath.Join(rootDir, "my-fork", "GOAL.md")
	if _, err := os.Stat(goalPath); err != nil {
		t.Fatalf("expected GOAL.md in fork: %v", err)
	}
}

func TestHandleNewForkPostRejectsInvalidName(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 2, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("name=bad!name")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/root-workspace/fork/new", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleNewForkPost(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST new fork invalid expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "workspace name can only contain lowercase letters") {
		t.Errorf("POST new fork invalid expected validation error")
	}
}

func TestHandleNewForkGetSingleWorkspace(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 1, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/root-workspace/fork/new", nil)
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleNewForkGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET new fork on single workspace expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNewForkPostSingleWorkspace(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 1, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("name=my-fork")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/root-workspace/fork/new", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleNewForkPost(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST new fork on single workspace expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/workspaces/my-fork/spec") {
		t.Errorf("POST new fork redirect = %q; want fork spec page", location)
	}
}

func TestHandleWorkspaceStartRejectsRoot(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 2, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("auto=false")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/root-workspace/start", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	srv.handleWorkspaceStart(rec, req, rootPath)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST start root expected 400, got %d", rec.Code)
	}
}

func TestHandleWorkspaceStartRejectsSingleRootWorkspace(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 1, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "solo-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("auto=false")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/solo-workspace/start", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	srv.handleWorkspaceStart(rec, req, rootPath)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST start root workspace expected 400, got %d", rec.Code)
	}
}

func TestHandleForkMergeRejectsDirtyFork(t *testing.T) {
	rec := runForkMergeDirty(t, "confirm_delete=true")

	if rec.Code != http.StatusOK {
		t.Fatalf("POST merge dirty expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "fork has uncommitted changes") {
		t.Fatalf("POST merge dirty missing error message")
	}
}

func TestHandleForkMergeAcceptsConfirm(t *testing.T) {
	rec := runForkMergeDirty(t, "confirm=true")

	if rec.Code != http.StatusOK {
		t.Fatalf("POST merge confirm expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "fork has uncommitted changes") {
		t.Fatalf("POST merge confirm missing error message")
	}
}

func runForkMergeDirty(t *testing.T, confirmParam string) *httptest.ResponseRecorder {
	t.Helper()

	diffHandler := "if [ \"$1\" = \"diff\" ]; then\n  printf \"M file.go\\n\"\n  exit 0\nfi\n"
	installFakeJJWithWorkspaceList(t, 2, diffHandler)

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)
	t.Setenv("JJ_FAKE_ROOT", rootPath)

	forkPath := filepath.Join(rootDir, "fork-workspace")
	if err := os.MkdirAll(forkPath, 0755); err != nil {
		t.Fatalf("failed to create fork workspace: %v", err)
	}
	createsgaiDir(t, forkPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("fork_dir=" + url.QueryEscape(forkPath) + "&" + confirmParam)
	req := httptest.NewRequest(http.MethodPost, "/workspaces/root-workspace/merge", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "root-workspace")
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()

	srv.handleForkMerge(rec, req)

	return rec
}

func TestRenderNewForkWithError(t *testing.T) {
	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}

	srv := NewServer(rootDir)
	rec := httptest.NewRecorder()

	srv.renderNewForkWithError(rec, rootPath, "bad fork name")

	if rec.Code != http.StatusOK {
		t.Fatalf("render new fork expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bad fork name") {
		t.Fatalf("render new fork missing error message")
	}
}

func TestCountForkCommitsAhead(t *testing.T) {
	fakeBinDir := t.TempDir()
	fakeJJ := filepath.Join(fakeBinDir, "jj")
	if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nif [ \"$1\" = \"log\" ]; then\n  printf \"id1\\nid2\\n\"\n  exit 0\nfi\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to create fake jj: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	rootDir := t.TempDir()
	forkDir := t.TempDir()

	if got := countForkCommitsAhead(rootDir, forkDir); got != 2 {
		t.Errorf("countForkCommitsAhead() = %d; want 2", got)
	}
}

func TestHandleForkMergeRequiresPost(t *testing.T) {
	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/root-workspace/merge", nil)
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleForkMerge(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET merge expected 405, got %d", rec.Code)
	}
}

func TestHandleForkMergeRequiresConfirmation(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 2, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)
	t.Setenv("JJ_FAKE_ROOT", rootPath)

	forkPath := filepath.Join(rootDir, "fork-workspace")
	if err := os.MkdirAll(forkPath, 0755); err != nil {
		t.Fatalf("failed to create fork workspace: %v", err)
	}
	createsgaiDir(t, forkPath)

	srv := NewServer(rootDir)
	body := strings.NewReader("fork_dir=" + url.QueryEscape(forkPath))
	req := httptest.NewRequest(http.MethodPost, "/workspaces/root-workspace/merge", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "root-workspace")
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()

	srv.handleForkMerge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST merge missing confirmation expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "confirmation required") {
		t.Fatalf("POST merge missing confirmation error message")
	}
}

func TestRenderRootWorkspaceContent(t *testing.T) {
	logHandler := "if [ \"$1\" = \"log\" ]; then\n  printf \"id1\\n\"\n  exit 0\nfi\n"
	installFakeJJWithWorkspaceList(t, 2, logHandler)

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)
	t.Setenv("JJ_FAKE_ROOT", rootPath)

	forkPath := filepath.Join(rootDir, "fork-workspace")
	if err := os.MkdirAll(forkPath, 0755); err != nil {
		t.Fatalf("failed to create fork workspace: %v", err)
	}
	createsgaiDir(t, forkPath)

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/root-workspace/forks", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	result := srv.renderRootWorkspaceContent(rootPath, "", req, "")

	body := string(result.Content)
	if !strings.Contains(body, "Forks") {
		t.Fatalf("root workspace content missing forks tab")
	}
	if !strings.Contains(body, "fork-workspace") {
		t.Fatalf("root workspace content missing fork name")
	}
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

func TestHandleWorkspaceSteer(t *testing.T) {
	t.Run("postInsertsMessageBeforeOldestUnread", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		initialState := state.Workflow{
			Status: state.StatusWorking,
			Messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg1", Read: true},
				{ID: 2, FromAgent: "agent2", ToAgent: "agent3", Body: "msg2", Read: false},
				{ID: 3, FromAgent: "agent3", ToAgent: "agent1", Body: "msg3", Read: false},
			},
		}
		if err := state.Save(statePath(projectDir), initialState); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		form := strings.NewReader("message=test%20steering")
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/steer", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		srv.handleWorkspaceSteer(rec, req, projectDir)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("POST steer expected 303 redirect, got %d", rec.Code)
		}

		wfState, err := state.Load(statePath(projectDir))
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		if len(wfState.Messages) != 4 {
			t.Fatalf("expected 4 messages, got %d", len(wfState.Messages))
		}

		newMsg := wfState.Messages[1]
		if newMsg.FromAgent != "Human Partner" {
			t.Errorf("expected FromAgent 'Human Partner', got %q", newMsg.FromAgent)
		}
		if newMsg.ToAgent != "coordinator" {
			t.Errorf("expected ToAgent 'coordinator', got %q", newMsg.ToAgent)
		}
		if !strings.Contains(newMsg.Body, "Re-steering instruction:") {
			t.Errorf("expected body to contain 'Re-steering instruction:', got %q", newMsg.Body)
		}
		if !strings.Contains(newMsg.Body, "test steering") {
			t.Errorf("expected body to contain 'test steering', got %q", newMsg.Body)
		}
	})

	t.Run("emptyMessageRedirectsWithoutChange", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		initialState := state.Workflow{
			Status:   state.StatusWorking,
			Messages: []state.Message{{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg1", Read: false}},
		}
		if err := state.Save(statePath(projectDir), initialState); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		form := strings.NewReader("message=")
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/steer", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		srv.handleWorkspaceSteer(rec, req, projectDir)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("expected 303 redirect, got %d", rec.Code)
		}

		wfState, err := state.Load(statePath(projectDir))
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		if len(wfState.Messages) != 1 {
			t.Errorf("expected messages unchanged (1), got %d", len(wfState.Messages))
		}
	})

	t.Run("nonPostMethodNotAllowed", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		srv := NewServer(rootDir)
		methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/workspaces/test-project/steer", nil)
			rec := httptest.NewRecorder()

			srv.handleWorkspaceSteer(rec, req, projectDir)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s request expected 405, got %d", method, rec.Code)
			}
		}
	})

	t.Run("insertsAtTopWhenAllMessagesRead", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		initialState := state.Workflow{
			Status: state.StatusWorking,
			Messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg1", Read: true},
				{ID: 2, FromAgent: "agent2", ToAgent: "agent3", Body: "msg2", Read: true},
			},
		}
		if err := state.Save(statePath(projectDir), initialState); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		form := strings.NewReader("message=new%20steering")
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/steer", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		srv.handleWorkspaceSteer(rec, req, projectDir)

		wfState, err := state.Load(statePath(projectDir))
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		if len(wfState.Messages) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(wfState.Messages))
		}

		firstMsg := wfState.Messages[0]
		if firstMsg.FromAgent != "Human Partner" {
			t.Errorf("expected new message at top, got from %q", firstMsg.FromAgent)
		}
	})
}

func TestHandleWorkspaceMessageAction(t *testing.T) {
	t.Run("deleteRemovesMessage", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		initialState := state.Workflow{
			Status: state.StatusWorking,
			Messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg1", Read: false},
				{ID: 2, FromAgent: "agent2", ToAgent: "agent3", Body: "msg2", Read: false},
				{ID: 3, FromAgent: "agent3", ToAgent: "agent1", Body: "msg3", Read: false},
			},
		}
		if err := state.Save(statePath(projectDir), initialState); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodDelete, "/workspaces/test-project/messages/2", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceMessageAction(rec, req, projectDir, "2")

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("DELETE message expected 303 redirect, got %d", rec.Code)
		}

		wfState, err := state.Load(statePath(projectDir))
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		if len(wfState.Messages) != 2 {
			t.Fatalf("expected 2 messages after delete, got %d", len(wfState.Messages))
		}

		for _, msg := range wfState.Messages {
			if msg.ID == 2 {
				t.Error("message ID 2 should have been deleted")
			}
		}
	})

	t.Run("deleteNonexistentMessageSucceeds", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		initialState := state.Workflow{
			Status:   state.StatusWorking,
			Messages: []state.Message{{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg1", Read: false}},
		}
		if err := state.Save(statePath(projectDir), initialState); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodDelete, "/workspaces/test-project/messages/999", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceMessageAction(rec, req, projectDir, "999")

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("DELETE nonexistent message expected 303 redirect, got %d", rec.Code)
		}

		wfState, err := state.Load(statePath(projectDir))
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		if len(wfState.Messages) != 1 {
			t.Errorf("expected messages unchanged (1), got %d", len(wfState.Messages))
		}
	})

	t.Run("invalidMessageIDReturnsBadRequest", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodDelete, "/workspaces/test-project/messages/notanumber", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceMessageAction(rec, req, projectDir, "notanumber")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("invalid message ID expected 400, got %d", rec.Code)
		}
	})

	t.Run("nonDeleteMethodNotAllowed", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		srv := NewServer(rootDir)
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/workspaces/test-project/messages/1", nil)
			rec := httptest.NewRecorder()

			srv.handleWorkspaceMessageAction(rec, req, projectDir, "1")

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s request expected 405, got %d", method, rec.Code)
			}
		}
	})

	t.Run("redirectsToMessagesTab", func(t *testing.T) {
		rootDir := t.TempDir()
		projectDir := filepath.Join(rootDir, "test-project")
		createsgaiDir(t, projectDir)

		initialState := state.Workflow{
			Status:   state.StatusWorking,
			Messages: []state.Message{{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg1", Read: false}},
		}
		if err := state.Save(statePath(projectDir), initialState); err != nil {
			t.Fatalf("failed to create state file: %v", err)
		}

		srv := NewServer(rootDir)
		req := httptest.NewRequest(http.MethodDelete, "/workspaces/test-project/messages/1", nil)
		rec := httptest.NewRecorder()

		srv.handleWorkspaceMessageAction(rec, req, projectDir, "1")

		loc := rec.Header().Get("Location")
		if !strings.HasSuffix(loc, "/messages") {
			t.Errorf("expected redirect location ending with /messages, got %q", loc)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("findOldestUnreadMessageIndex", func(t *testing.T) {
		tests := []struct {
			name     string
			messages []state.Message
			want     int
		}{
			{
				name:     "emptyMessages",
				messages: []state.Message{},
				want:     0,
			},
			{
				name: "allRead",
				messages: []state.Message{
					{ID: 1, Read: true},
					{ID: 2, Read: true},
				},
				want: 0,
			},
			{
				name: "firstUnread",
				messages: []state.Message{
					{ID: 1, Read: false},
					{ID: 2, Read: true},
				},
				want: 0,
			},
			{
				name: "middleUnread",
				messages: []state.Message{
					{ID: 1, Read: true},
					{ID: 2, Read: false},
					{ID: 3, Read: false},
				},
				want: 1,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := findOldestUnreadMessageIndex(tt.messages)
				if got != tt.want {
					t.Errorf("findOldestUnreadMessageIndex() = %d; want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("generateNewMessageID", func(t *testing.T) {
		tests := []struct {
			name     string
			messages []state.Message
			want     int
		}{
			{
				name:     "emptyMessages",
				messages: []state.Message{},
				want:     1,
			},
			{
				name: "singleMessage",
				messages: []state.Message{
					{ID: 5},
				},
				want: 6,
			},
			{
				name: "multipleMessages",
				messages: []state.Message{
					{ID: 1},
					{ID: 10},
					{ID: 5},
				},
				want: 11,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := generateNewMessageID(tt.messages)
				if got != tt.want {
					t.Errorf("generateNewMessageID() = %d; want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("insertMessageAt", func(t *testing.T) {
		newMsg := state.Message{ID: 99, Body: "new"}
		tests := []struct {
			name     string
			messages []state.Message
			index    int
			wantLen  int
			wantPos  int
		}{
			{
				name:     "emptySlice",
				messages: []state.Message{},
				index:    0,
				wantLen:  1,
				wantPos:  0,
			},
			{
				name:     "insertAtBeginning",
				messages: []state.Message{{ID: 1}, {ID: 2}},
				index:    0,
				wantLen:  3,
				wantPos:  0,
			},
			{
				name:     "insertInMiddle",
				messages: []state.Message{{ID: 1}, {ID: 2}, {ID: 3}},
				index:    1,
				wantLen:  4,
				wantPos:  1,
			},
			{
				name:     "insertAtEnd",
				messages: []state.Message{{ID: 1}, {ID: 2}},
				index:    5,
				wantLen:  3,
				wantPos:  2,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := insertMessageAt(tt.messages, tt.index, newMsg)
				if len(got) != tt.wantLen {
					t.Errorf("insertMessageAt() len = %d; want %d", len(got), tt.wantLen)
				}
				if got[tt.wantPos].ID != 99 {
					t.Errorf("new message not at expected position %d", tt.wantPos)
				}
			})
		}
	})

	t.Run("removeMessageByID", func(t *testing.T) {
		tests := []struct {
			name     string
			messages []state.Message
			id       int
			wantLen  int
		}{
			{
				name:     "emptySlice",
				messages: []state.Message{},
				id:       1,
				wantLen:  0,
			},
			{
				name:     "removeExisting",
				messages: []state.Message{{ID: 1}, {ID: 2}, {ID: 3}},
				id:       2,
				wantLen:  2,
			},
			{
				name:     "removeNonexistent",
				messages: []state.Message{{ID: 1}, {ID: 2}},
				id:       99,
				wantLen:  2,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := removeMessageByID(tt.messages, tt.id)
				if len(got) != tt.wantLen {
					t.Errorf("removeMessageByID() len = %d; want %d", len(got), tt.wantLen)
				}
				for _, msg := range got {
					if msg.ID == tt.id && tt.name == "removeExisting" {
						t.Errorf("message ID %d should have been removed", tt.id)
					}
				}
			})
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

func TestIsRootWorkspace(t *testing.T) {
	t.Run("noJJWorkspace", func(t *testing.T) {
		fakeBinDir := t.TempDir()
		fakeJJ := filepath.Join(fakeBinDir, "jj")
		if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatalf("failed to create fake jj: %v", err)
		}
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		dir := t.TempDir()
		if isRootWorkspace(dir) {
			t.Error("isRootWorkspace() = true; want false for non-jj directory")
		}
	})

	t.Run("rootWorkspace", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 1, "")

		dir := t.TempDir()
		if !isRootWorkspace(dir) {
			t.Error("isRootWorkspace() = false; want true for root workspace")
		}
	})

	t.Run("rootWithMultipleForks", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 2, "")

		dir := t.TempDir()
		if !isRootWorkspace(dir) {
			t.Error("isRootWorkspace() = false; want true for root with multiple forks")
		}
	})

	t.Run("forkWorkspace", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 2, "")

		dir := t.TempDir()
		t.Setenv("JJ_FAKE_ROOT", "/some/other/root")
		if isRootWorkspace(dir) {
			t.Error("isRootWorkspace() = true; want false for fork workspace")
		}
	})

	t.Run("jjCommandFails", func(t *testing.T) {
		fakeBinDir := t.TempDir()
		fakeJJ := filepath.Join(fakeBinDir, "jj")
		if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatalf("failed to create fake jj: %v", err)
		}
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		dir := t.TempDir()
		if isRootWorkspace(dir) {
			t.Error("isRootWorkspace() = true; want false when jj command fails")
		}
	})
}

func TestIsForkWorkspace(t *testing.T) {
	t.Run("noJJWorkspace", func(t *testing.T) {
		fakeBinDir := t.TempDir()
		fakeJJ := filepath.Join(fakeBinDir, "jj")
		if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatalf("failed to create fake jj: %v", err)
		}
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		dir := t.TempDir()
		if isForkWorkspace(dir) {
			t.Error("isForkWorkspace() = true; want false for non-jj directory")
		}
	})

	t.Run("rootWorkspace", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 1, "")

		dir := t.TempDir()
		if isForkWorkspace(dir) {
			t.Error("isForkWorkspace() = true; want false for root workspace")
		}
	})

	t.Run("forkWorkspace", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 2, "")

		dir := t.TempDir()
		t.Setenv("JJ_FAKE_ROOT", "/some/other/root")
		if !isForkWorkspace(dir) {
			t.Error("isForkWorkspace() = false; want true for fork workspace")
		}
	})
}

func TestGetRootWorkspacePath(t *testing.T) {
	t.Run("noJJWorkspace", func(t *testing.T) {
		fakeBinDir := t.TempDir()
		fakeJJ := filepath.Join(fakeBinDir, "jj")
		if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatalf("failed to create fake jj: %v", err)
		}
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		dir := t.TempDir()
		if got := getRootWorkspacePath(dir); got != "" {
			t.Errorf("getRootWorkspacePath() = %q; want empty string", got)
		}
	})

	t.Run("returnsRoot", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 1, "")

		dir := t.TempDir()
		t.Setenv("JJ_FAKE_ROOT", "/expected/root")
		got := getRootWorkspacePath(dir)
		if got != "/expected/root" {
			t.Errorf("getRootWorkspacePath() = %q; want %q", got, "/expected/root")
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

func setupForkFixture(t *testing.T, rootDir, forkName string) string {
	t.Helper()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	forkPath := filepath.Join(rootDir, forkName)
	if err := os.MkdirAll(forkPath, 0755); err != nil {
		t.Fatalf("failed to create fork workspace: %v", err)
	}
	createsgaiDir(t, forkPath)

	installFakeJJWithWorkspaceList(t, 2, "")
	t.Setenv("JJ_FAKE_ROOT", rootPath)

	return forkPath
}

func TestHandleRenameForkGet(t *testing.T) {
	rootDir := t.TempDir()
	setupForkFixture(t, rootDir, "my-fork")

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/my-fork/rename", nil)
	req.SetPathValue("name", "my-fork")
	rec := httptest.NewRecorder()

	srv.handleRenameForkGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET rename fork expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Rename Fork") {
		t.Error("response missing 'Rename Fork' title")
	}
	if !strings.Contains(body, "my-fork") {
		t.Error("response missing current fork name")
	}
}

func TestHandleRenameForkGetRejectsRoot(t *testing.T) {
	installFakeJJWithWorkspaceList(t, 1, "")

	rootDir := t.TempDir()
	rootPath := filepath.Join(rootDir, "root-workspace")
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		t.Fatalf("failed to create root workspace: %v", err)
	}
	createsgaiDir(t, rootPath)

	srv := NewServer(rootDir)
	req := httptest.NewRequest(http.MethodGet, "/workspaces/root-workspace/rename", nil)
	req.SetPathValue("name", "root-workspace")
	rec := httptest.NewRecorder()

	srv.handleRenameForkGet(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET rename fork for root expected 400, got %d", rec.Code)
	}
}

func TestHandleRenameForkPost(t *testing.T) {
	rootDir := t.TempDir()
	setupForkFixture(t, rootDir, "old-fork")

	srv := NewServer(rootDir)
	body := strings.NewReader("name=new-fork")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/old-fork/rename", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "old-fork")
	rec := httptest.NewRecorder()

	srv.handleRenameForkPost(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST rename fork expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/workspaces/new-fork/progress") {
		t.Errorf("POST rename fork redirect = %q; want /workspaces/new-fork/progress", location)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "new-fork")); err != nil {
		t.Errorf("new-fork directory should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "old-fork")); !os.IsNotExist(err) {
		t.Errorf("old-fork directory should not exist after rename")
	}
}

func TestHandleRenameForkPostRejectsInvalidName(t *testing.T) {
	rootDir := t.TempDir()
	setupForkFixture(t, rootDir, "my-fork")

	srv := NewServer(rootDir)
	body := strings.NewReader("name=bad!name")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/my-fork/rename", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "my-fork")
	rec := httptest.NewRecorder()

	srv.handleRenameForkPost(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST rename fork invalid expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "workspace name can only contain lowercase letters") {
		t.Error("POST rename fork invalid expected validation error")
	}
}

func TestHandleRenameForkPostRejectsExistingTarget(t *testing.T) {
	rootDir := t.TempDir()
	setupForkFixture(t, rootDir, "my-fork")
	if err := os.MkdirAll(filepath.Join(rootDir, "taken-name"), 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	srv := NewServer(rootDir)
	body := strings.NewReader("name=taken-name")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/my-fork/rename", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "my-fork")
	rec := httptest.NewRecorder()

	srv.handleRenameForkPost(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST rename fork existing target expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "a directory with this name already exists") {
		t.Error("expected 'already exists' error message")
	}
}

func TestHandleRenameForkPostRejectsRunningSession(t *testing.T) {
	rootDir := t.TempDir()
	forkPath := setupForkFixture(t, rootDir, "my-fork")

	srv := NewServer(rootDir)
	srv.mu.Lock()
	srv.sessions[forkPath] = &session{running: true}
	srv.mu.Unlock()

	body := strings.NewReader("name=new-name")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/my-fork/rename", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "my-fork")
	rec := httptest.NewRecorder()

	srv.handleRenameForkPost(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST rename fork running session expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cannot rename: session is running") {
		t.Error("expected 'session is running' error message")
	}
}

func TestHandleRenameForkPostRekeysSession(t *testing.T) {
	fakeBinDir := t.TempDir()
	fakeJJ := filepath.Join(fakeBinDir, "jj")
	if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to create fake jj: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	rootDir := t.TempDir()
	forkPath := setupForkFixture(t, rootDir, "old-fork")

	srv := NewServer(rootDir)
	stoppedSession := &session{running: false}
	srv.mu.Lock()
	srv.sessions[forkPath] = stoppedSession
	srv.everStartedDirs[forkPath] = true
	srv.mu.Unlock()

	body := strings.NewReader("name=new-fork")
	req := httptest.NewRequest(http.MethodPost, "/workspaces/old-fork/rename", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("name", "old-fork")
	rec := httptest.NewRecorder()

	srv.handleRenameForkPost(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST rename fork rekey expected 303, got %d: %s", rec.Code, rec.Body.String())
	}

	newPath := filepath.Join(rootDir, "new-fork")
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.sessions[forkPath] != nil {
		t.Error("old fork path should no longer have a session")
	}
	if srv.sessions[newPath] != stoppedSession {
		t.Error("new fork path should have the moved session")
	}
	if srv.everStartedDirs[forkPath] {
		t.Error("old fork path should no longer be in everStartedDirs")
	}
	if !srv.everStartedDirs[newPath] {
		t.Error("new fork path should be in everStartedDirs")
	}
}

func TestPinnedFilePath(t *testing.T) {
	srv := &Server{pinnedConfigDir: "/tmp/test-sgai"}
	want := "/tmp/test-sgai/pinned.json"
	got := srv.pinnedFilePath()
	if got != want {
		t.Errorf("pinnedFilePath() = %q; want %q", got, want)
	}
}

func TestLoadPinnedProjects(t *testing.T) {
	t.Run("missingFile", func(t *testing.T) {
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: t.TempDir(),
		}
		if err := srv.loadPinnedProjects(); err != nil {
			t.Fatalf("loadPinnedProjects() unexpected error: %v", err)
		}
		if len(srv.pinnedDirs) != 0 {
			t.Errorf("pinnedDirs should be empty; got %d entries", len(srv.pinnedDirs))
		}
	})

	t.Run("emptyArray", func(t *testing.T) {
		configDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(configDir, "pinned.json"), []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.loadPinnedProjects(); err != nil {
			t.Fatalf("loadPinnedProjects() unexpected error: %v", err)
		}
		if len(srv.pinnedDirs) != 0 {
			t.Errorf("pinnedDirs should be empty; got %d entries", len(srv.pinnedDirs))
		}
	})

	t.Run("withPaths", func(t *testing.T) {
		configDir := t.TempDir()
		data, _ := json.Marshal([]string{"/path/to/a", "/path/to/b"})
		if err := os.WriteFile(filepath.Join(configDir, "pinned.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.loadPinnedProjects(); err != nil {
			t.Fatalf("loadPinnedProjects() unexpected error: %v", err)
		}
		if len(srv.pinnedDirs) != 2 {
			t.Fatalf("pinnedDirs should have 2 entries; got %d", len(srv.pinnedDirs))
		}
		if !srv.pinnedDirs["/path/to/a"] {
			t.Error("expected /path/to/a to be pinned")
		}
		if !srv.pinnedDirs["/path/to/b"] {
			t.Error("expected /path/to/b to be pinned")
		}
	})

	t.Run("invalidJSON", func(t *testing.T) {
		configDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(configDir, "pinned.json"), []byte("{invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.loadPinnedProjects(); err == nil {
			t.Error("loadPinnedProjects() should return error for invalid JSON")
		}
	})
}

func TestSavePinnedProjects(t *testing.T) {
	t.Run("createsDirectoryAndFile", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "sgai")
		srv := &Server{
			pinnedDirs:      map[string]bool{"/path/to/a": true, "/path/to/b": true},
			pinnedConfigDir: configDir,
		}
		if err := srv.savePinnedProjects(); err != nil {
			t.Fatalf("savePinnedProjects() unexpected error: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(configDir, "pinned.json"))
		if err != nil {
			t.Fatalf("failed to read pinned.json: %v", err)
		}
		var dirs []string
		if err := json.Unmarshal(data, &dirs); err != nil {
			t.Fatalf("failed to parse pinned.json: %v", err)
		}
		if len(dirs) != 2 {
			t.Fatalf("expected 2 paths; got %d", len(dirs))
		}
		if dirs[0] != "/path/to/a" || dirs[1] != "/path/to/b" {
			t.Errorf("unexpected paths: %v", dirs)
		}
	})

	t.Run("emptyPinnedDirs", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "sgai")
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.savePinnedProjects(); err != nil {
			t.Fatalf("savePinnedProjects() unexpected error: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(configDir, "pinned.json"))
		if err != nil {
			t.Fatalf("failed to read pinned.json: %v", err)
		}
		var dirs []string
		if err := json.Unmarshal(data, &dirs); err != nil {
			t.Fatalf("failed to parse pinned.json: %v", err)
		}
		if len(dirs) != 0 {
			t.Errorf("expected empty array; got %v", dirs)
		}
	})
}

func TestIsPinned(t *testing.T) {
	srv := &Server{
		mu:         sync.Mutex{},
		pinnedDirs: map[string]bool{"/pinned/project": true},
	}
	if !srv.isPinned("/pinned/project") {
		t.Error("isPinned(/pinned/project) = false; want true")
	}
	if srv.isPinned("/not/pinned") {
		t.Error("isPinned(/not/pinned) = true; want false")
	}
}

func TestTogglePin(t *testing.T) {
	t.Run("pinAndUnpin", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "sgai")
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.togglePin("/path/to/project"); err != nil {
			t.Fatalf("togglePin() unexpected error: %v", err)
		}
		if !srv.isPinned("/path/to/project") {
			t.Error("project should be pinned after first toggle")
		}
		if err := srv.togglePin("/path/to/project"); err != nil {
			t.Fatalf("togglePin() unexpected error: %v", err)
		}
		if srv.isPinned("/path/to/project") {
			t.Error("project should not be pinned after second toggle")
		}
	})

	t.Run("persistsToDisk", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "sgai")
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.togglePin("/path/to/project"); err != nil {
			t.Fatalf("togglePin() unexpected error: %v", err)
		}
		srv2 := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv2.loadPinnedProjects(); err != nil {
			t.Fatalf("loadPinnedProjects() unexpected error: %v", err)
		}
		if !srv2.isPinned("/path/to/project") {
			t.Error("pinned project should persist across server instances")
		}
	})
}

func TestCollectInProgressWorkspacesWithPinned(t *testing.T) {
	t.Run("pinnedWorkspaceIncludedInProgress", func(t *testing.T) {
		groups := []workspaceGroup{
			{Root: workspaceInfo{Directory: "/a", InProgress: false, Pinned: false}},
			{Root: workspaceInfo{Directory: "/b", InProgress: true, Pinned: true}},
		}
		got := collectInProgressWorkspaces(groups)
		if len(got) != 1 {
			t.Fatalf("expected 1 in-progress workspace; got %d", len(got))
		}
		if got[0].Directory != "/b" {
			t.Errorf("expected /b; got %q", got[0].Directory)
		}
		if !got[0].Pinned {
			t.Error("expected workspace to have Pinned=true")
		}
	})

	t.Run("pinnedForkIncludedInProgress", func(t *testing.T) {
		groups := []workspaceGroup{
			{
				Root: workspaceInfo{Directory: "/a", InProgress: false},
				Forks: []workspaceInfo{
					{Directory: "/a/fork1", InProgress: true, Pinned: true},
				},
			},
		}
		got := collectInProgressWorkspaces(groups)
		if len(got) != 1 {
			t.Fatalf("expected 1 in-progress workspace; got %d", len(got))
		}
		if !got[0].Pinned {
			t.Error("expected fork to have Pinned=true")
		}
	})
}

func TestHandleTogglePin(t *testing.T) {
	rootDir := t.TempDir()
	projectDir := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".sgai"), 0o755); err != nil {
		t.Fatal(err)
	}

	configDir := filepath.Join(t.TempDir(), "sgai")
	srv := &Server{
		sessions:         make(map[string]*session),
		everStartedDirs:  make(map[string]bool),
		pinnedDirs:       make(map[string]bool),
		pinnedConfigDir:  configDir,
		adhocStates:      make(map[string]*adhocPromptState),
		rootDir:          rootDir,
		editorAvailable:  false,
		isTerminalEditor: false,
		editorName:       "",
		editor:           &mockEditorOpener{},
	}

	t.Run("postTogglesPin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/toggle-pin", nil)
		rec := httptest.NewRecorder()
		srv.handleTogglePin(rec, req, projectDir)
		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected status %d; got %d", http.StatusSeeOther, rec.Code)
		}
		if !srv.isPinned(projectDir) {
			t.Error("project should be pinned after toggle")
		}
	})

	t.Run("secondPostUnpins", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/workspaces/test-project/toggle-pin", nil)
		rec := httptest.NewRecorder()
		srv.handleTogglePin(rec, req, projectDir)
		if srv.isPinned(projectDir) {
			t.Error("project should be unpinned after second toggle")
		}
	})

	t.Run("getNonAllowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/workspaces/test-project/toggle-pin", nil)
		rec := httptest.NewRecorder()
		srv.handleTogglePin(rec, req, projectDir)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d; got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})
}
