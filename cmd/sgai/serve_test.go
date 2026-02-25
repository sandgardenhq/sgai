package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func installFakeJJWithWorkspaceList(t *testing.T, workspaceCount int) {
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
	script.WriteString("if [ \"$1\" = \"workspace\" ] && [ \"$2\" = \"list\" ]; then\n")
	script.WriteString("  printf \"" + workspaceOutput.String() + "\"\n")
	script.WriteString("  exit 0\n")
	script.WriteString("fi\n")
	script.WriteString("exit 0\n")
	if err := os.WriteFile(fakeJJ, []byte(script.String()), 0755); err != nil {
		t.Fatalf("failed to create fake jj: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
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

func TestDashboardBaseURL(t *testing.T) {
	cases := []struct {
		name       string
		listenAddr string
		want       string
	}{
		{
			name:       "loopbackV4",
			listenAddr: "127.0.0.1:8181",
			want:       "http://127.0.0.1:8181",
		},
		{
			name:       "wildcardV4",
			listenAddr: "0.0.0.0:8181",
			want:       "http://127.0.0.1:8181",
		},
		{
			name:       "wildcardV6",
			listenAddr: "[::]:8181",
			want:       "http://[::1]:8181",
		},
		{
			name:       "emptyHost",
			listenAddr: ":8181",
			want:       "http://127.0.0.1:8181",
		},
		{
			name:       "hostname",
			listenAddr: "example.test:8080",
			want:       "http://example.test:8080",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dashboardBaseURL(tc.listenAddr)
			if got != tc.want {
				t.Errorf("dashboardBaseURL(%q) = %q; want %q", tc.listenAddr, got, tc.want)
			}
		})
	}
}

func TestHandleAPIOpenInOpenCodeNoRunningSessionFromLocalhost(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/test-project/open-opencode", nil)
	req.SetPathValue("name", "test-project")
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleAPIOpenInOpenCode(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("request with no running session should return 409 Conflict, got %d", rec.Code)
	}
}

func TestHandleAPIOpenInOpenCodeForbiddenForRemote(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/test-project/open-opencode", nil)
	req.SetPathValue("name", "test-project")
	req.RemoteAddr = "192.168.1.100:54321"
	rec := httptest.NewRecorder()

	srv.handleAPIOpenInOpenCode(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("remote request should return 403 Forbidden, got %d", rec.Code)
	}
}

func TestHandleAPIOpenInOpenCodeFactoryNotRunning(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/test-project/open-opencode", nil)
	req.SetPathValue("name", "test-project")
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleAPIOpenInOpenCode(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("request with factory not running should return 409 Conflict, got %d", rec.Code)
	}
}

func TestHandleAPIOpenInOpenCodeFactoryNotRunningSessionExists(t *testing.T) {
	rootDir := t.TempDir()
	validProject := filepath.Join(rootDir, "test-project")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	createsgaiDir(t, validProject)

	srv := NewServer(rootDir)
	srv.sessions[validProject] = &session{running: false}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/test-project/open-opencode", nil)
	req.SetPathValue("name", "test-project")
	req.RemoteAddr = "127.0.0.1:54321"
	rec := httptest.NewRecorder()

	srv.handleAPIOpenInOpenCode(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("request with session stopped should return 409 Conflict, got %d", rec.Code)
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

func TestCountForkCommitsAhead(t *testing.T) {
	fakeBinDir := t.TempDir()
	fakeJJ := filepath.Join(fakeBinDir, "jj")
	if err := os.WriteFile(fakeJJ, []byte("#!/bin/sh\nif [ \"$1\" = \"log\" ]; then\n  printf \"id1\\nid2\\n\"\n  exit 0\nfi\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to create fake jj: %v", err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	forkDir := t.TempDir()

	if got := countForkCommitsAhead("main", forkDir); got != 2 {
		t.Errorf("countForkCommitsAhead() = %d; want 2", got)
	}
}

func TestClassifyWorkspace(t *testing.T) {
	t.Run("noJJRepo", func(t *testing.T) {
		dir := t.TempDir()
		if got := classifyWorkspace(dir); got != workspaceStandalone {
			t.Errorf("classifyWorkspace() = %q; want %q for directory without .jj/repo", got, workspaceStandalone)
		}
	})

	t.Run("repoIsFile", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".jj"), 0755); err != nil {
			t.Fatalf("failed to create .jj dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".jj", "repo"), []byte("/some/path"), 0644); err != nil {
			t.Fatalf("failed to create repo file: %v", err)
		}
		if got := classifyWorkspace(dir); got != workspaceFork {
			t.Errorf("classifyWorkspace() = %q; want %q for .jj/repo as file", got, workspaceFork)
		}
	})

	t.Run("singleWorkspace", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 1)

		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".jj", "repo"), 0755); err != nil {
			t.Fatalf("failed to create jj repo: %v", err)
		}
		if got := classifyWorkspace(dir); got != workspaceStandalone {
			t.Errorf("classifyWorkspace() = %q; want %q for single workspace", got, workspaceStandalone)
		}
	})

	t.Run("multipleWorkspaces", func(t *testing.T) {
		installFakeJJWithWorkspaceList(t, 2)

		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".jj", "repo"), 0755); err != nil {
			t.Fatalf("failed to create jj repo: %v", err)
		}
		if got := classifyWorkspace(dir); got != workspaceRoot {
			t.Errorf("classifyWorkspace() = %q; want %q for multiple workspaces", got, workspaceRoot)
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
		if err := os.MkdirAll(filepath.Join(dir, ".jj", "repo"), 0755); err != nil {
			t.Fatalf("failed to create jj repo: %v", err)
		}
		if got := classifyWorkspace(dir); got != workspaceStandalone {
			t.Errorf("classifyWorkspace() = %q; want %q when jj command fails", got, workspaceStandalone)
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
		dirA := t.TempDir()
		dirB := t.TempDir()
		data, _ := json.Marshal([]string{dirA, dirB})
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
		if !srv.pinnedDirs[dirA] {
			t.Errorf("expected %s to be pinned", dirA)
		}
		if !srv.pinnedDirs[dirB] {
			t.Errorf("expected %s to be pinned", dirB)
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

	t.Run("prunesNonexistentDirectories", func(t *testing.T) {
		configDir := t.TempDir()
		realDir := t.TempDir()
		data, _ := json.Marshal([]string{realDir, "/nonexistent/path/that/does/not/exist"})
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
		if len(srv.pinnedDirs) != 1 {
			t.Fatalf("pinnedDirs should have 1 entry; got %d", len(srv.pinnedDirs))
		}
		if !srv.pinnedDirs[realDir] {
			t.Errorf("expected %s to be pinned", realDir)
		}
		diskData, err := os.ReadFile(filepath.Join(configDir, "pinned.json"))
		if err != nil {
			t.Fatalf("failed to read pinned.json: %v", err)
		}
		var diskDirs []string
		if err := json.Unmarshal(diskData, &diskDirs); err != nil {
			t.Fatalf("failed to parse pinned.json: %v", err)
		}
		if len(diskDirs) != 1 {
			t.Fatalf("pinned.json on disk should have 1 entry; got %d", len(diskDirs))
		}
		if diskDirs[0] != realDir {
			t.Errorf("pinned.json on disk should contain %s; got %s", realDir, diskDirs[0])
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
		projectDir := t.TempDir()
		srv := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv.togglePin(projectDir); err != nil {
			t.Fatalf("togglePin() unexpected error: %v", err)
		}
		srv2 := &Server{
			pinnedDirs:      make(map[string]bool),
			pinnedConfigDir: configDir,
		}
		if err := srv2.loadPinnedProjects(); err != nil {
			t.Fatalf("loadPinnedProjects() unexpected error: %v", err)
		}
		if !srv2.isPinned(projectDir) {
			t.Error("pinned project should persist across server instances")
		}
	})
}

func writeStateFile(t *testing.T, dir, status string) {
	t.Helper()
	wf := state.Workflow{Status: status}
	if err := state.Save(statePath(dir), wf); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}
}

func TestClearEverStartedOnCompletion(t *testing.T) {
	t.Run("completedWorkspace", func(t *testing.T) {
		dir := t.TempDir()
		writeStateFile(t, dir, state.StatusComplete)
		srv := &Server{
			everStartedDirs: map[string]bool{dir: true},
		}
		srv.clearEverStartedOnCompletion(dir)
		if srv.everStartedDirs[dir] {
			t.Error("everStartedDirs should be cleared for completed workspace")
		}
	})

	t.Run("workingWorkspace", func(t *testing.T) {
		dir := t.TempDir()
		writeStateFile(t, dir, state.StatusWorking)
		srv := &Server{
			everStartedDirs: map[string]bool{dir: true},
		}
		srv.clearEverStartedOnCompletion(dir)
		if !srv.everStartedDirs[dir] {
			t.Error("everStartedDirs should persist for working workspace")
		}
	})

	t.Run("agentDoneWorkspace", func(t *testing.T) {
		dir := t.TempDir()
		writeStateFile(t, dir, state.StatusAgentDone)
		srv := &Server{
			everStartedDirs: map[string]bool{dir: true},
		}
		srv.clearEverStartedOnCompletion(dir)
		if !srv.everStartedDirs[dir] {
			t.Error("everStartedDirs should persist for agent-done workspace")
		}
	})

	t.Run("missingStateFile", func(t *testing.T) {
		dir := t.TempDir()
		srv := &Server{
			everStartedDirs: map[string]bool{dir: true},
		}
		srv.clearEverStartedOnCompletion(dir)
		if !srv.everStartedDirs[dir] {
			t.Error("everStartedDirs should persist when state file is missing")
		}
	})
}
