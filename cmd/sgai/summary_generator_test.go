package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestCleanSummaryOutput(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "A simple summary.", "A simple summary."},
		{"withQuotes", `"A quoted summary."`, "A quoted summary."},
		{"withWhitespace", "  some text  \n", "some text"},
		{"withBackticks", "`summary text`", "summary text"},
		{"withSingleQuotes", "'summary text'", "summary text"},
		{"empty", "", ""},
		{"mixedQuotes", `"'summary'"`, "summary"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanSummaryOutput(tc.in)
			if got != tc.want {
				t.Errorf("cleanSummaryOutput(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSaveSummaryIfNotManual(t *testing.T) {
	t.Run("savesWhenNotManual", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-project")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		if _, errSave := state.NewCoordinatorWith(statePath(workspace), state.Workflow{}); errSave != nil {
			t.Fatal(errSave)
		}

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		srv := NewServer(rootDir)
		srv.shutdownCtx = ctx
		gen := newSummaryGenerator(ctx, srv)
		gen.saveSummaryIfNotManual(workspace, "A test summary")

		loadedCoord, errLoad := state.NewCoordinator(statePath(workspace))
		if errLoad != nil {
			t.Fatal(errLoad)
		}
		loaded := loadedCoord.State()
		if loaded.Summary != "A test summary" {
			t.Errorf("summary = %q; want %q", loaded.Summary, "A test summary")
		}
	})

	t.Run("skipsWhenManual", func(t *testing.T) {
		rootDir := t.TempDir()
		workspace := filepath.Join(rootDir, "test-project")
		if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
			t.Fatal(errMkdir)
		}
		createsgaiDir(t, workspace)

		if _, errSave := state.NewCoordinatorWith(statePath(workspace), state.Workflow{
			Summary:       "User written summary",
			SummaryManual: true,
		}); errSave != nil {
			t.Fatal(errSave)
		}

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		srv := NewServer(rootDir)
		srv.shutdownCtx = ctx
		gen := newSummaryGenerator(ctx, srv)
		gen.saveSummaryIfNotManual(workspace, "Auto generated summary")

		loadedCoord, errLoad := state.NewCoordinator(statePath(workspace))
		if errLoad != nil {
			t.Fatal(errLoad)
		}
		loaded := loadedCoord.State()
		if loaded.Summary != "User written summary" {
			t.Errorf("summary = %q; want %q (should not be overwritten)", loaded.Summary, "User written summary")
		}
	})
}

func TestSummaryGeneratorDebounce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rootDir := t.TempDir()
	srv := NewServer(rootDir)
	srv.shutdownCtx = ctx

	gen := newSummaryGenerator(ctx, srv)
	t.Cleanup(gen.stop)

	gen.trigger("/workspace/a")
	gen.trigger("/workspace/a")
	gen.trigger("/workspace/a")

	time.Sleep(100 * time.Millisecond)

	gen.mu.Lock()
	d, ok := gen.debounceMap["/workspace/a"]
	gen.mu.Unlock()

	if !ok {
		t.Fatal("expected debouncer entry for /workspace/a")
	}

	d.mu.Lock()
	hasTimer := d.timer != nil
	d.mu.Unlock()

	if !hasTimer {
		t.Fatal("debounce timer should still be pending (2s debounce, only 100ms elapsed)")
	}
}

func TestSummaryGeneratorStopCancelsTimers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rootDir := t.TempDir()
	srv := NewServer(rootDir)
	srv.shutdownCtx = ctx

	gen := newSummaryGenerator(ctx, srv)

	gen.trigger("/workspace/a")
	gen.trigger("/workspace/b")

	gen.stop()

	gen.mu.Lock()
	for _, d := range gen.debounceMap {
		d.mu.Lock()
		if d.timer != nil {
			if !d.timer.Stop() {
				t.Log("timer was already fired or stopped")
			}
		}
		d.mu.Unlock()
	}
	gen.mu.Unlock()
}

func TestReadGoalBody(t *testing.T) {
	t.Run("withFrontmatter", func(t *testing.T) {
		rootDir := t.TempDir()
		goalContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n\nBuild a cool project.\n"
		if errWrite := os.WriteFile(filepath.Join(rootDir, "GOAL.md"), []byte(goalContent), 0644); errWrite != nil {
			t.Fatal(errWrite)
		}

		body := readGoalBody(rootDir)
		if body != "Build a cool project." {
			t.Errorf("readGoalBody() = %q; want %q", body, "Build a cool project.")
		}
	})

	t.Run("noFile", func(t *testing.T) {
		body := readGoalBody(t.TempDir())
		if body != "" {
			t.Errorf("readGoalBody() = %q; want empty", body)
		}
	})
}

func TestHandleAPIUpdateSummary(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)
	if _, errSave := state.NewCoordinatorWith(statePath(workspace), state.Workflow{Summary: "old summary"}); errSave != nil {
		t.Fatal(errSave)
	}

	srv := NewServer(rootDir)
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	body := `{"summary": "My custom summary"}`
	req := httptest.NewRequest("PUT", "/api/v1/workspaces/test-project/summary", strings.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp apiUpdateSummaryResponse
	if errDecode := json.NewDecoder(rr.Body).Decode(&resp); errDecode != nil {
		t.Fatal(errDecode)
	}
	if !resp.Updated {
		t.Fatal("expected Updated=true")
	}
	if resp.Summary != "My custom summary" {
		t.Errorf("summary = %q; want %q", resp.Summary, "My custom summary")
	}

	loadedCoord, errLoad := state.NewCoordinator(statePath(workspace))
	if errLoad != nil {
		t.Fatal(errLoad)
	}
	loaded := loadedCoord.State()
	if loaded.Summary != "My custom summary" {
		t.Errorf("persisted summary = %q; want %q", loaded.Summary, "My custom summary")
	}
	if !loaded.SummaryManual {
		t.Fatal("SummaryManual should be true after manual update")
	}
}

func TestHandleAPIUpdateSummaryPreventsAutoOverwrite(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if _, errSave := state.NewCoordinatorWith(statePath(workspace), state.Workflow{}); errSave != nil {
		t.Fatal(errSave)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	srv := NewServer(rootDir)
	srv.shutdownCtx = ctx
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	body := `{"summary": "Manual summary"}`
	req := httptest.NewRequest("PUT", "/api/v1/workspaces/test-project/summary", strings.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", rr.Code, http.StatusOK)
	}

	gen := newSummaryGenerator(ctx, srv)
	gen.saveSummaryIfNotManual(workspace, "This should not overwrite")

	loadedCoord, errLoad := state.NewCoordinator(statePath(workspace))
	if errLoad != nil {
		t.Fatal(errLoad)
	}
	loaded := loadedCoord.State()
	if loaded.Summary != "Manual summary" {
		t.Errorf("summary = %q; want %q (should remain manual)", loaded.Summary, "Manual summary")
	}
	if !loaded.SummaryManual {
		t.Fatal("SummaryManual should remain true")
	}
}

func TestBuildWorkspaceFullStateIncludesSummary(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if _, errSave := state.NewCoordinatorWith(statePath(workspace), state.Workflow{
		Summary:       "Test project summary",
		SummaryManual: true,
	}); errSave != nil {
		t.Fatal(errSave)
	}

	srv := NewServer(rootDir)
	ws := workspaceInfo{
		Directory: workspace,
		DirName:   "test-project",
	}
	detail := srv.buildWorkspaceFullState(ws, nil)
	if detail.Summary != "Test project summary" {
		t.Errorf("detail.Summary = %q; want %q", detail.Summary, "Test project summary")
	}
	if !detail.SummaryManual {
		t.Fatal("detail.SummaryManual should be true")
	}
}

func TestBuildWorkspaceFullStateIncludesSummaryInState(t *testing.T) {
	rootDir := t.TempDir()
	workspace := filepath.Join(rootDir, "test-project")
	if errMkdir := os.MkdirAll(workspace, 0755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	createsgaiDir(t, workspace)

	if _, errSave := state.NewCoordinatorWith(statePath(workspace), state.Workflow{
		Summary: "Sidebar summary",
	}); errSave != nil {
		t.Fatal(errSave)
	}

	srv := NewServer(rootDir)
	ws := workspaceInfo{
		Directory:    workspace,
		DirName:      "test-project",
		HasWorkspace: true,
	}
	detail := srv.buildWorkspaceFullState(ws, nil)
	if detail.Summary != "Sidebar summary" {
		t.Errorf("detail.Summary = %q; want %q", detail.Summary, "Sidebar summary")
	}
}
