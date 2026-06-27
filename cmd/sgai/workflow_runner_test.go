package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareCoordinatorPreservesTodos(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	coord, errCoord := state.NewCoordinatorWith(filepath.Join(dir, ".sgai", "state.json"), state.Workflow{})
	require.NoError(t, errCoord)

	r := &workflowRunner{dir: dir, paddedsgai: "test", coord: coord, wfState: state.Workflow{Todos: []state.TodoItem{{Content: "existing", Status: "pending"}}}}

	r.prepareCoordinator()

	assert.Equal(t, []state.TodoItem{{Content: "existing", Status: "pending"}}, r.wfState.Todos)
}

func TestBuildWorkflowRunnerResumesExistingStateAfterGoalEdit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\nmodel: openai/gpt-5.5\n---\n# Changed Goal"), 0o644))
	_, errCoord := state.NewCoordinatorWith(filepath.Join(dir, ".sgai", "state.json"), state.Workflow{
		Status: state.StatusWorking,
		Task:   "existing task",
	})
	require.NoError(t, errCoord)

	runner, cleanup, ok := buildWorkflowRunner(dir, "", nil, nil)
	t.Cleanup(cleanup)

	require.True(t, ok)
	assert.Equal(t, "existing task", runner.wfState.Task)
}

func TestParseYAMLFrontmatterFromFileMissingGoal(t *testing.T) {
	metadata, errParse := parseYAMLFrontmatterFromFile(filepath.Join(t.TempDir(), "GOAL.md"))
	require.NoError(t, errParse)
	assert.Empty(t, metadata.Agents)
	assert.Empty(t, metadata.Model)
}
