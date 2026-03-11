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
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{"alreadyHasCoordinator", []string{"coordinator", "builder", "reviewer"}, []string{"coordinator", "builder", "reviewer"}},
		{"noCoordinator", []string{"builder", "reviewer"}, []string{"coordinator", "builder", "reviewer"}},
		{"empty", []string{}, []string{"coordinator"}},
		{"onlyCoordinator", []string{"coordinator"}, []string{"coordinator"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildAllAgents(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestComputeLongestNameLen(t *testing.T) {
	cases := []struct {
		name   string
		agents []string
		want   int
	}{
		{"empty", []string{}, len("sgai")},
		{"shortNames", []string{"a", "bb"}, len("sgai")},
		{"longName", []string{"very-long-agent-name"}, len("very-long-agent-name")},
		{"mixedLengths", []string{"ab", "coordinator", "z"}, len("coordinator")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeLongestNameLen(tc.agents)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveCurrentAgent(t *testing.T) {
	t.Run("emptyDefaultsToCoordinator", func(t *testing.T) {
		r := &workflowRunner{wfState: state.Workflow{CurrentAgent: ""}}
		assert.Equal(t, "coordinator", r.resolveCurrentAgent())
	})

	t.Run("returnsCurrentAgent", func(t *testing.T) {
		r := &workflowRunner{wfState: state.Workflow{CurrentAgent: "builder"}}
		assert.Equal(t, "builder", r.resolveCurrentAgent())
	})
}

func buildTestDag(edges map[string][]string, entryNodes []string) *dag {
	d := &dag{
		Nodes:      make(map[string]*dagNode),
		EntryNodes: entryNodes,
	}
	for from, toList := range edges {
		node := d.ensureNode(from)
		for _, to := range toList {
			toNode := d.ensureNode(to)
			node.Successors = append(node.Successors, to)
			toNode.Predecessors = append(toNode.Predecessors, from)
		}
	}
	return d
}

func TestResolveNextAgent(t *testing.T) {
	t.Run("redirectsToPendingMessages", func(t *testing.T) {
		r := &workflowRunner{
			paddedsgai: "test",
			flowDag:    buildTestDag(map[string][]string{"coordinator": {"reviewer"}}, []string{"coordinator"}),
			wfState: state.Workflow{
				Messages: []state.Message{
					{ID: 1, FromAgent: "coordinator", ToAgent: "reviewer", Body: "review please", Read: false},
				},
			},
		}
		got := r.resolveNextAgent("coordinator")
		assert.Equal(t, "reviewer", got)
	})

	t.Run("terminalNodeReturnsCoordinator", func(t *testing.T) {
		r := &workflowRunner{
			paddedsgai: "test",
			flowDag:    buildTestDag(map[string][]string{"coordinator": {"reviewer"}}, []string{"coordinator"}),
			wfState:    state.Workflow{Messages: []state.Message{}},
		}
		got := r.resolveNextAgent("reviewer")
		assert.Equal(t, "coordinator", got)
	})

	t.Run("coordinatorGoesToFirstEntry", func(t *testing.T) {
		r := &workflowRunner{
			paddedsgai: "test",
			flowDag:    buildTestDag(map[string][]string{"coordinator": {"builder"}}, []string{"builder"}),
			wfState:    state.Workflow{Messages: []state.Message{}},
		}
		got := r.resolveNextAgent("coordinator")
		assert.Equal(t, "builder", got)
	})
}

func TestPrepareAgent(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:      state.StatusWorking,
		VisitCounts: map[string]int{},
	})
	require.NoError(t, errCoord)

	r := &workflowRunner{
		dir:           dir,
		paddedsgai:    "test",
		coord:         coord,
		previousAgent: "",
		wfState: state.Workflow{
			Status:      state.StatusWorking,
			VisitCounts: map[string]int{},
		},
	}

	r.prepareAgent("coordinator")
	assert.Equal(t, "coordinator", r.previousAgent)
	assert.Equal(t, "coordinator", r.wfState.CurrentAgent)
	assert.Equal(t, 1, r.wfState.VisitCounts["coordinator"])

	r.prepareAgent("builder")
	assert.Equal(t, "builder", r.previousAgent)
	assert.Equal(t, "builder", r.wfState.CurrentAgent)
	assert.Equal(t, 1, r.wfState.VisitCounts["builder"])
	assert.Empty(t, r.wfState.Todos)
}

func TestHandleTrigger(t *testing.T) {
	t.Run("ignoresNonSteeringTrigger", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
		require.NoError(t, errCoord)

		r := &workflowRunner{coord: coord}
		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0o644))

		r.handleTrigger(triggerGoal, goalPath)
	})

	t.Run("steeringNoMessages", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{Messages: []state.Message{}})
		require.NoError(t, errCoord)

		r := &workflowRunner{coord: coord}
		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0o644))

		r.handleTrigger(triggerSteering, goalPath)
	})

	t.Run("steeringWithHumanMessage", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Messages: []state.Message{
				{
					ID:        1,
					FromAgent: "Human Partner",
					ToAgent:   "coordinator",
					Body:      "Add logging",
					Read:      false,
				},
			},
		})
		require.NoError(t, errCoord)

		r := &workflowRunner{coord: coord}
		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("# Goal\n\nOriginal content"), 0o644))

		r.handleTrigger(triggerSteering, goalPath)

		goalContent, errRead := os.ReadFile(goalPath)
		require.NoError(t, errRead)
		assert.Contains(t, string(goalContent), "Add logging")
	})
}

func TestResolveRetrospectiveDirResuming(t *testing.T) {
	dir := t.TempDir()
	retroDir := filepath.Join(dir, ".sgai", "retrospectives", "2026-03-06-12-00.abcd")
	require.NoError(t, os.MkdirAll(retroDir, 0755))

	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	retroDirRel, errRel := filepath.Rel(dir, retroDir)
	require.NoError(t, errRel)
	pmContent := "---\nRetrospective Session: " + retroDirRel + "\n---\n"
	require.NoError(t, os.WriteFile(pmPath, []byte(pmContent), 0644))

	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0644))

	result := resolveRetrospectiveDir(true, dir, filepath.Join(dir, ".sgai", "retrospectives"), pmPath, stateJSONPath, goalPath)
	assert.Equal(t, retroDir, result)
}

func TestResolveRetrospectiveDirNewSession(t *testing.T) {
	dir := t.TempDir()
	retrospectivesBaseDir := filepath.Join(dir, ".sgai", "retrospectives")
	pmPath := filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md")
	stateJSONPath := filepath.Join(dir, ".sgai", "state.json")
	goalPath := filepath.Join(dir, "GOAL.md")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0o644))

	retroDir := resolveRetrospectiveDir(false, dir, retrospectivesBaseDir, pmPath, stateJSONPath, goalPath)
	assert.NotEmpty(t, retroDir)
	assert.DirExists(t, retroDir)

	goalCopy := filepath.Join(retroDir, "GOAL.md")
	assert.FileExists(t, goalCopy)

	assert.FileExists(t, pmPath)

	_, errStatState := os.Stat(stateJSONPath)
	assert.True(t, os.IsNotExist(errStatState))
}

func TestHandleWorkingLoop(t *testing.T) {
	cfg := multiModelConfig{paddedsgai: "test", agent: "builder"}
	sessionID := "session-123"

	t.Run("incrementsCounter", func(t *testing.T) {
		got := handleWorkingLoop(cfg, &sessionID, 0)
		assert.Equal(t, 1, got)
		assert.Equal(t, "session-123", sessionID)
	})

	t.Run("resetsOnMaxIterations", func(t *testing.T) {
		sid := "session-456"
		got := handleWorkingLoop(cfg, &sid, maxConsecutiveWorkingIterations-1)
		assert.Equal(t, 0, got)
		assert.Empty(t, sid)
	})
}

func TestUnlockInteractiveForRetrospective(t *testing.T) {
	t.Run("nonRetrospectiveAgent", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			InteractionMode: state.ModeBuilding,
		})
		require.NoError(t, errCoord)

		wfState := coord.State()
		unlockInteractiveForRetrospective(&wfState, "coordinator", coord, "test")
		assert.Equal(t, state.ModeBuilding, wfState.InteractionMode)
	})

	t.Run("retrospectiveAlreadyInMode", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			InteractionMode: state.ModeRetrospective,
		})
		require.NoError(t, errCoord)

		wfState := coord.State()
		unlockInteractiveForRetrospective(&wfState, "retrospective", coord, "test")
		assert.Equal(t, state.ModeRetrospective, wfState.InteractionMode)
	})

	t.Run("retrospectiveUnlocks", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			InteractionMode: state.ModeBuilding,
		})
		require.NoError(t, errCoord)

		wfState := coord.State()
		unlockInteractiveForRetrospective(&wfState, "retrospective", coord, "test")
		assert.Equal(t, state.ModeRetrospective, wfState.InteractionMode)
	})
}

func TestCopyProjectManagementToRetrospective(t *testing.T) {
	t.Run("emptyRetrospectiveDir", func(_ *testing.T) {
		copyProjectManagementToRetrospective("/tmp", "")
	})

	t.Run("noPMFile", func(t *testing.T) {
		dir := t.TempDir()
		retroDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
		copyProjectManagementToRetrospective(dir, retroDir)
		_, err := os.Stat(filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("copiesPMFile", func(t *testing.T) {
		dir := t.TempDir()
		retroDir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("# PM\nSome content"), 0o644))

		copyProjectManagementToRetrospective(dir, retroDir)

		content, err := os.ReadFile(filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "Some content")
	})
}
