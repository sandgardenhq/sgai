package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		maxLen int
		want   string
	}{
		{"shortMessage", "hello", 10, "hello"},
		{"exactLength", "hello", 5, "hello"},
		{"truncated", "hello world this is a long message", 10, "hello worl..."},
		{"emptyMessage", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateMessage(tt.msg, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateMessage(%q, %d) = %q, want %q", tt.msg, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestWorkspaceDirName(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"simplePath", "/home/user/workspace", "workspace"},
		{"nestedPath", "/a/b/c/d", "d"},
		{"singleDir", "workspace", "workspace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := workspaceDirName(tt.dir)
			if got != tt.want {
				t.Errorf("workspaceDirName(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestWatcherSnapshotDetection(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "test-workspace")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatal(err)
	}

	wfState := state.Workflow{
		Status: state.StatusWorking,
		Progress: []state.ProgressEntry{
			{Timestamp: "2026-01-01T00:00:00Z", Agent: "test-agent", Description: "step 1"},
		},
	}
	if err := state.Save(statePath(wsDir), wfState); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmpDir, "sessions.json")
	db := newSessionDB(dbPath)
	if err := db.load(); err != nil {
		t.Fatal(err)
	}

	watcher := &slackNotificationWatcher{
		sessions:  db,
		snapshots: make(map[string]slackWatcherSnapshot),
	}

	key := "C1:ts1"
	sess := slackSession{
		WorkspaceDir: wsDir,
		ChannelID:    "C1",
	}

	watcher.checkWorkspace(key, sess)

	snapshot, ok := watcher.snapshots[key]
	if !ok {
		t.Fatal("snapshot should be created on first check")
	}
	if snapshot.status != state.StatusWorking {
		t.Errorf("status = %q, want %q", snapshot.status, state.StatusWorking)
	}
	if snapshot.progressLen != 1 {
		t.Errorf("progressLen = %d, want 1", snapshot.progressLen)
	}
}

func TestWatcherMissingStateFile(t *testing.T) {
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "no-state-workspace")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	watcher := &slackNotificationWatcher{
		snapshots: make(map[string]slackWatcherSnapshot),
	}

	key := "C1:ts1"
	sess := slackSession{
		WorkspaceDir: wsDir,
		ChannelID:    "C1",
	}

	watcher.snapshots[key] = slackWatcherSnapshot{status: "old"}

	watcher.checkWorkspace(key, sess)

	_, ok := watcher.snapshots[key]
	if ok {
		t.Error("snapshot should be removed when state file is missing")
	}
}

func TestStatePath(t *testing.T) {
	got := statePath("/workspace/test")
	want := filepath.Join("/workspace/test", ".sgai", "state.json")
	if got != want {
		t.Errorf("statePath = %q, want %q", got, want)
	}
}
