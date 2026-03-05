package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashTodos(t *testing.T) {
	tests := []struct {
		name         string
		projectTodos []state.TodoItem
		agentTodos   []state.TodoItem
	}{
		{
			name:         "emptyTodos",
			projectTodos: []state.TodoItem{},
			agentTodos:   []state.TodoItem{},
		},
		{
			name: "projectTodosOnly",
			projectTodos: []state.TodoItem{
				{Content: "Task 1", Status: "pending", Priority: "high"},
			},
			agentTodos: []state.TodoItem{},
		},
		{
			name:         "agentTodosOnly",
			projectTodos: []state.TodoItem{},
			agentTodos: []state.TodoItem{
				{Content: "Task 2", Status: "completed", Priority: "low"},
			},
		},
		{
			name: "bothTodos",
			projectTodos: []state.TodoItem{
				{Content: "Task 1", Status: "pending", Priority: "high"},
			},
			agentTodos: []state.TodoItem{
				{Content: "Task 2", Status: "completed", Priority: "low"},
			},
		},
		{
			name: "multipleTodos",
			projectTodos: []state.TodoItem{
				{Content: "Task 1", Status: "pending", Priority: "high"},
				{Content: "Task 2", Status: "in_progress", Priority: "medium"},
			},
			agentTodos: []state.TodoItem{
				{Content: "Task 3", Status: "completed", Priority: "low"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashTodos(tt.projectTodos, tt.agentTodos)
			assert.Len(t, result, 16)
			assert.NotEmpty(t, result)
		})
	}
}

func TestHashTodosConsistency(t *testing.T) {
	todos := []state.TodoItem{
		{Content: "Task 1", Status: "pending", Priority: "high"},
	}

	hash1 := hashTodos(todos, []state.TodoItem{})
	hash2 := hashTodos(todos, []state.TodoItem{})

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func TestHashTodosDifferent(t *testing.T) {
	todos1 := []state.TodoItem{
		{Content: "Task 1", Status: "pending", Priority: "high"},
	}
	todos2 := []state.TodoItem{
		{Content: "Task 2", Status: "pending", Priority: "high"},
	}

	hash1 := hashTodos(todos1, []state.TodoItem{})
	hash2 := hashTodos(todos2, []state.TodoItem{})

	assert.NotEqual(t, hash1, hash2, "different input should produce different hash")
}

func TestHashMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []state.Message
	}{
		{
			name:     "emptyMessages",
			messages: []state.Message{},
		},
		{
			name: "singleMessage",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello"},
			},
		},
		{
			name: "multipleMessages",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello"},
				{ID: 2, FromAgent: "agent2", ToAgent: "agent1", Body: "World"},
			},
		},
		{
			name: "messageWithReadStatus",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello", Read: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashMessages(tt.messages)
			assert.Len(t, result, 16)
			assert.NotEmpty(t, result)
		})
	}
}

func TestHashMessagesConsistency(t *testing.T) {
	messages := []state.Message{
		{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello"},
	}

	hash1 := hashMessages(messages)
	hash2 := hashMessages(messages)

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
}

func TestHashMessagesDifferent(t *testing.T) {
	messages1 := []state.Message{
		{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello"},
	}
	messages2 := []state.Message{
		{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "World"},
	}

	hash1 := hashMessages(messages1)
	hash2 := hashMessages(messages2)

	assert.NotEqual(t, hash1, hash2, "different input should produce different hash")
}

func TestHashGoalFile(t *testing.T) {
	tests := []struct {
		name     string
		fileInfo os.FileInfo
	}{
		{
			name:     "nilFileInfo",
			fileInfo: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashGoalFile(tt.fileInfo)
			if tt.fileInfo == nil {
				assert.Empty(t, result)
			} else {
				assert.Len(t, result, 16)
			}
		})
	}
}

func TestHashGoalFileWithFileInfo(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "goal_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	_, _ = tmpFile.WriteString("test content")
	_ = tmpFile.Close()

	fileInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	result := hashGoalFile(fileInfo)
	assert.Len(t, result, 16)
	assert.NotEmpty(t, result)
}

func TestHashGoalFileConsistency(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "goal_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	_, _ = tmpFile.WriteString("test content")
	_ = tmpFile.Close()

	fileInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	hash1 := hashGoalFile(fileInfo)
	hash2 := hashGoalFile(fileInfo)

	assert.Equal(t, hash1, hash2, "same file info should produce same hash")
}

func TestHashGoalFileDifferentAfterModification(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "goal_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	_, _ = tmpFile.WriteString("test content")
	_ = tmpFile.Close()

	fileInfo1, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	hash1 := hashGoalFile(fileInfo1)

	time.Sleep(10 * time.Millisecond)

	_ = os.WriteFile(tmpFile.Name(), []byte("modified content"), 0644)

	fileInfo2, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	hash2 := hashGoalFile(fileInfo2)

	assert.NotEqual(t, hash1, hash2, "modified file should produce different hash")
}

func TestBuildStateSnapshot(t *testing.T) {
	modTime := time.Now()
	wfState := state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "test question",
		Progress: []state.ProgressEntry{
			{Agent: "coordinator", Description: "started"},
		},
		ProjectTodos: []state.TodoItem{
			{Content: "task1", Status: "pending"},
		},
		Messages: []state.Message{
			{ID: 1, FromAgent: "dev", ToAgent: "coord", Body: "done"},
		},
	}

	snapshot := buildStateSnapshot(modTime, wfState, nil)
	assert.Equal(t, modTime, snapshot.modTime)
	assert.Equal(t, state.StatusWaitingForHuman, snapshot.status)
	assert.True(t, snapshot.needsInput)
	assert.Equal(t, 1, snapshot.progressLen)
	assert.NotEmpty(t, snapshot.todosHash)
	assert.NotEmpty(t, snapshot.messagesHash)
	assert.Empty(t, snapshot.goalHash)
}

func TestBuildStateSnapshotWithGoalInfo(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "goal_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })
	_, _ = tmpFile.WriteString("# Goal")
	_ = tmpFile.Close()

	goalInfo, err := os.Stat(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	modTime := time.Now()
	snapshot := buildStateSnapshot(modTime, state.Workflow{}, goalInfo)
	assert.NotEmpty(t, snapshot.goalHash)
	assert.False(t, snapshot.goalModTime.IsZero())
}

func TestEmitStateChangeEvents(t *testing.T) {
	server, _ := setupTestServer(t)

	t.Run("noChange", func(_ *testing.T) {
		prev := workspaceStateSnapshot{
			status:      state.StatusWorking,
			needsInput:  false,
			progressLen: 5,
			todosHash:   "abc",
		}
		server.emitStateChangeEvents("/test", prev, prev)
	})

	t.Run("statusChange", func(_ *testing.T) {
		prev := workspaceStateSnapshot{status: state.StatusWorking}
		current := workspaceStateSnapshot{status: state.StatusComplete}
		server.emitStateChangeEvents("/test", prev, current)
	})

	t.Run("needsInputChange", func(_ *testing.T) {
		prev := workspaceStateSnapshot{needsInput: false}
		current := workspaceStateSnapshot{needsInput: true}
		server.emitStateChangeEvents("/test", prev, current)
	})

	t.Run("progressChange", func(_ *testing.T) {
		prev := workspaceStateSnapshot{progressLen: 5}
		current := workspaceStateSnapshot{progressLen: 6}
		server.emitStateChangeEvents("/test", prev, current)
	})
}

func TestStateWatcherEmitChangeEventsStatusChange(t *testing.T) {
	srv, _ := setupTestServer(t)
	prev := workspaceStateSnapshot{status: state.StatusWorking, todosHash: "abc", messagesHash: "def", goalHash: "ghi"}
	changed := workspaceStateSnapshot{status: state.StatusComplete, todosHash: "abc", messagesHash: "def", goalHash: "ghi"}
	srv.emitStateChangeEvents("dir", prev, changed)
}

func TestStateWatcherEmitNoChangeEvents(t *testing.T) {
	srv, _ := setupTestServer(t)
	s := workspaceStateSnapshot{status: state.StatusWorking, progressLen: 3, todosHash: "abc", messagesHash: "def", goalHash: "ghi"}
	srv.emitStateChangeEvents("dir", s, s)
}

func TestCheckWorkspaceStateSecondVisitNoChange(t *testing.T) {
	srv, _ := setupTestServer(t)
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	sp := filepath.Join(sgaiDir, "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{Status: state.StatusComplete})
	require.NoError(t, errCoord)

	snapshots := make(map[string]workspaceStateSnapshot)
	active := make(map[string]bool)

	srv.checkWorkspaceState(dir, snapshots, active)
	assert.Contains(t, snapshots, dir)

	srv.checkWorkspaceState(dir, snapshots, active)
	assert.Contains(t, snapshots, dir)
}

func TestCheckWorkspaceStateWithMapChanges(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "cwsc-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusComplete,
		Task:   "all done",
	})
	require.NoError(t, errCoord)

	snapshots := make(map[string]workspaceStateSnapshot)
	activeWorkspaces := make(map[string]bool)

	srv.checkWorkspaceState(wsDir, snapshots, activeWorkspaces)
	assert.True(t, activeWorkspaces[wsDir])
	assert.Contains(t, snapshots, wsDir)
	assert.Equal(t, string(state.StatusComplete), snapshots[wsDir].status)
}

func TestBuildStateSnapshotWithGoalHash(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("# Goal"), 0o644))
	sp := filepath.Join(sgaiDir, "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	srv, _ := setupTestServer(t)
	snapshots := make(map[string]workspaceStateSnapshot)
	activeWorkspaces := make(map[string]bool)
	srv.checkWorkspaceState(dir, snapshots, activeWorkspaces)

	snap, exists := snapshots[dir]
	assert.True(t, exists)
	assert.NotEmpty(t, snap.goalHash)
}

func TestHashGoalFileNilInfo(t *testing.T) {
	hash := hashGoalFile(nil)
	assert.Empty(t, hash)
}

func TestHashGoalFileValidInfo(t *testing.T) {
	dir := t.TempDir()
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0o644))

	info, errStat := os.Stat(goalPath)
	require.NoError(t, errStat)

	hash := hashGoalFile(info)
	assert.NotEmpty(t, hash)
}

func TestPollWorkspaceStatesWithMultipleWorkspaces(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir1 := setupTestWorkspace(t, rootDir, "poll-ws1")
	wsDir2 := setupTestWorkspace(t, rootDir, "poll-ws2")
	sp1 := filepath.Join(wsDir1, ".sgai", "state.json")
	sp2 := filepath.Join(wsDir2, ".sgai", "state.json")
	_, errCoord1 := state.NewCoordinatorWith(sp1, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord1)
	_, errCoord2 := state.NewCoordinatorWith(sp2, state.Workflow{
		Status: state.StatusWorking,
		Task:   "building",
	})
	require.NoError(t, errCoord2)

	snapshots := make(map[string]workspaceStateSnapshot)
	srv.pollWorkspaceStates(snapshots)
	assert.NotEmpty(t, snapshots)
}

func TestPollWorkspaceStatesCleanupRemoved(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "poll-cleanup")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	snapshots := make(map[string]workspaceStateSnapshot)
	snapshots["/nonexistent/removed-ws"] = workspaceStateSnapshot{status: "old"}
	srv.pollWorkspaceStates(snapshots)
	_, exists := snapshots["/nonexistent/removed-ws"]
	assert.False(t, exists)
}
