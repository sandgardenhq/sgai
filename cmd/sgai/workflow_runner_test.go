package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAllAgents(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "addsCoordinator", in: []string{"go", "react"}, want: []string{"coordinator", "go", "react"}},
		{name: "keepsCoordinator", in: []string{"coordinator", "go"}, want: []string{"coordinator", "go"}},
		{name: "empty", in: nil, want: []string{"coordinator"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildAllAgents(tt.in))
		})
	}
}

func TestResolveNextAgentAlwaysReturnsCoordinator(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{Navigate: &state.NavigationRequest{To: "go", Reason: "implement"}})
	require.NoError(t, errCoord)

	r := &workflowRunner{paddedsgai: "test", coord: coord, wfState: coord.State()}
	assert.Equal(t, "coordinator", r.resolveNextAgent("coordinator"))
	assert.Nil(t, coord.State().Navigate)

	r.wfState = coord.State()
	assert.Equal(t, "coordinator", r.resolveNextAgent("go"))
}

func TestPrepareAgentTracksVisits(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	coord, errCoord := state.NewCoordinatorWith(filepath.Join(dir, ".sgai", "state.json"), state.Workflow{})
	require.NoError(t, errCoord)

	r := &workflowRunner{dir: dir, paddedsgai: "test", coord: coord, wfState: state.Workflow{VisitCounts: map[string]int{}, Todos: []state.TodoItem{{Content: "stale", Status: "pending"}}}}

	r.prepareAgent("coordinator")
	r.prepareAgent("go")

	assert.Equal(t, "go", r.wfState.CurrentAgent)
	assert.Equal(t, 1, r.wfState.VisitCounts["coordinator"])
	assert.Equal(t, 1, r.wfState.VisitCounts["go"])
	assert.Empty(t, r.wfState.Todos)
}

func TestCanResumeWorkflowRequiresMatchingGoalChecksum(t *testing.T) {
	tests := []struct {
		name        string
		wfState     state.Workflow
		newChecksum string
		want        bool
	}{
		{
			name:        "matching checksum resumes active workflow",
			wfState:     state.Workflow{Status: state.StatusWorking, GoalChecksum: "abc"},
			newChecksum: "abc",
			want:        true,
		},
		{
			name:        "changed checksum starts fresh workflow",
			wfState:     state.Workflow{Status: state.StatusWorking, GoalChecksum: "abc"},
			newChecksum: "def",
			want:        false,
		},
		{
			name:        "missing stored checksum cannot resume",
			wfState:     state.Workflow{Status: state.StatusWorking},
			newChecksum: "abc",
			want:        false,
		},
		{
			name:        "empty state cannot resume",
			wfState:     state.Workflow{},
			newChecksum: "abc",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canResumeWorkflow(tt.wfState, tt.newChecksum))
		})
	}
}

func TestStopSessionPreservesOriginalGoalChecksum(t *testing.T) {
	server, rootDir := setupTestServer(t)
	workspacePath := setupTestWorkspace(t, rootDir, "changed-goal")
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("original goal"), 0o644))
	originalChecksum, errChecksum := computeGoalChecksum(goalPath)
	require.NoError(t, errChecksum)

	coord, errCoord := state.NewCoordinatorWith(statePath(workspacePath), state.Workflow{
		Status:       state.StatusWorking,
		GoalChecksum: originalChecksum,
	})
	require.NoError(t, errCoord)
	server.sessions[workspacePath] = &session{coord: coord, running: true}

	require.NoError(t, os.WriteFile(goalPath, []byte("changed goal"), 0o644))
	changedChecksum, errChangedChecksum := computeGoalChecksum(goalPath)
	require.NoError(t, errChangedChecksum)
	require.NotEqual(t, originalChecksum, changedChecksum)

	server.stopSession(workspacePath)

	freshCoord, errFresh := state.NewCoordinator(statePath(workspacePath))
	require.NoError(t, errFresh)
	wfState := freshCoord.State()
	assert.Equal(t, originalChecksum, wfState.GoalChecksum)
	assert.False(t, canResumeWorkflow(wfState, changedChecksum))
}

func TestParseYAMLFrontmatterFromFileMissingGoal(t *testing.T) {
	metadata, errParse := parseYAMLFrontmatterFromFile(filepath.Join(t.TempDir(), "GOAL.md"))
	require.NoError(t, errParse)
	assert.Empty(t, metadata.Agents)
	assert.Empty(t, metadata.Model)
}
