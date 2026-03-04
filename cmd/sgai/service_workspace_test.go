package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForkWorkspaceServiceRollbackOnPostAddFailures(t *testing.T) {
	t.Run("unpackSkeletonFailure", func(t *testing.T) {
		logPath := installFakeJJForForkWorkspaceTests(t)
		t.Setenv("FAKE_JJ_PRECREATE_SGAI_FILE", "1")

		srv, workspacePath := newForkWorkspaceTestServer(t)
		_, errFork := srv.forkWorkspaceService(workspacePath, "# Goal\n\n- [ ] keep behavior\n")
		if errFork == nil {
			t.Fatal("forkWorkspaceService() expected error, got nil")
		}
		if !strings.Contains(errFork.Error(), "failed to unpack skeleton") {
			t.Fatalf("forkWorkspaceService() error = %q, want unpack skeleton failure", errFork)
		}

		forkPath := readForkPathFromFakeJJLog(t, logPath)
		assertForkWasRolledBack(t, logPath, forkPath)
	})

	t.Run("goalFileWriteFailure", func(t *testing.T) {
		logPath := installFakeJJForForkWorkspaceTests(t)
		t.Setenv("FAKE_JJ_PRECREATE_GOAL_DIR", "1")

		srv, workspacePath := newForkWorkspaceTestServer(t)
		_, errFork := srv.forkWorkspaceService(workspacePath, "# Goal\n\n- [ ] keep behavior\n")
		if errFork == nil {
			t.Fatal("forkWorkspaceService() expected error, got nil")
		}
		if !strings.Contains(errFork.Error(), "failed to create GOAL.md") {
			t.Fatalf("forkWorkspaceService() error = %q, want GOAL.md failure", errFork)
		}

		forkPath := readForkPathFromFakeJJLog(t, logPath)
		assertForkWasRolledBack(t, logPath, forkPath)
	})

	t.Run("addGitExcludeFailure", func(t *testing.T) {
		logPath := installFakeJJForForkWorkspaceTests(t)
		t.Setenv("FAKE_JJ_PRECREATE_GIT_FILE", "1")

		srv, workspacePath := newForkWorkspaceTestServer(t)
		_, errFork := srv.forkWorkspaceService(workspacePath, "# Goal\n\n- [ ] keep behavior\n")
		if errFork == nil {
			t.Fatal("forkWorkspaceService() expected error, got nil")
		}
		if !strings.Contains(errFork.Error(), "failed to add git exclude") {
			t.Fatalf("forkWorkspaceService() error = %q, want add git exclude failure", errFork)
		}

		forkPath := readForkPathFromFakeJJLog(t, logPath)
		assertForkWasRolledBack(t, logPath, forkPath)
	})
}

func newForkWorkspaceTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	rootDir := t.TempDir()
	workspacePath := filepath.Join(rootDir, "root-workspace")
	if errMkdir := os.MkdirAll(workspacePath, 0o755); errMkdir != nil {
		t.Fatalf("failed to create workspace directory: %v", errMkdir)
	}
	srv := NewServer(rootDir)
	srv.pinnedConfigDir = t.TempDir()
	return srv, workspacePath
}

func installFakeJJForForkWorkspaceTests(t *testing.T) string {
	t.Helper()
	fakeBinDir := t.TempDir()
	fakeJJ := filepath.Join(fakeBinDir, "jj")
	logPath := filepath.Join(t.TempDir(), "fake-jj.log")

	script := `#!/bin/sh
set -eu

if [ "${FAKE_JJ_LOG:-}" != "" ]; then
	printf "%s %s %s\n" "${1:-}" "${2:-}" "${3:-}" >> "${FAKE_JJ_LOG}"
fi

if [ "${1:-}" = "workspace" ] && [ "${2:-}" = "add" ]; then
	fork_path="${3:-}"
	mkdir -p "$fork_path"
	if [ "${FAKE_JJ_PRECREATE_SGAI_FILE:-0}" = "1" ]; then
		touch "$fork_path/.sgai"
	fi
	if [ "${FAKE_JJ_PRECREATE_GOAL_DIR:-0}" = "1" ]; then
		mkdir -p "$fork_path/GOAL.md"
	fi
	if [ "${FAKE_JJ_PRECREATE_GIT_FILE:-0}" = "1" ]; then
		touch "$fork_path/.git"
	fi
	exit 0
fi

if [ "${1:-}" = "workspace" ] && [ "${2:-}" = "forget" ]; then
	if [ "${FAKE_JJ_FORGET_FAIL:-0}" = "1" ]; then
		echo "forget failed" >&2
		exit 1
	fi
	exit 0
fi

exit 0
`

	if errWrite := os.WriteFile(fakeJJ, []byte(script), 0o755); errWrite != nil {
		t.Fatalf("failed to create fake jj: %v", errWrite)
	}

	t.Setenv("FAKE_JJ_LOG", logPath)
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func readForkPathFromFakeJJLog(t *testing.T, logPath string) string {
	t.Helper()
	content, errRead := os.ReadFile(logPath)
	if errRead != nil {
		t.Fatalf("failed to read fake jj log: %v", errRead)
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(content)), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 3)
		if len(parts) < 3 {
			continue
		}
		if parts[0] == "workspace" && parts[1] == "add" {
			return parts[2]
		}
	}
	t.Fatalf("workspace add call not found in fake jj log:\n%s", string(content))
	return ""
}

func assertForkWasRolledBack(t *testing.T, logPath, forkPath string) {
	t.Helper()
	if _, errStat := os.Stat(forkPath); !os.IsNotExist(errStat) {
		t.Fatalf("fork path should be removed during rollback, stat err=%v", errStat)
	}

	content, errRead := os.ReadFile(logPath)
	if errRead != nil {
		t.Fatalf("failed to read fake jj log: %v", errRead)
	}
	forkName := filepath.Base(forkPath)
	wantForgetCall := "workspace forget " + forkName
	if !strings.Contains(string(content), wantForgetCall) {
		t.Fatalf("expected rollback to call %q, log:\n%s", wantForgetCall, string(content))
	}
}
