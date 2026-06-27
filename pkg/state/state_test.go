package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHumanPending(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"waitingForHuman", StatusWaitingForHuman, true},
		{"working", StatusWorking, false},
		{"agentDone", StatusAgentDone, false},
		{"complete", StatusComplete, false},
		{"empty", "", false},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHumanPending(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeedsHumanInput(t *testing.T) {
	tests := []struct {
		name     string
		workflow Workflow
		expected bool
	}{
		{
			name: "waitingWithQuestion",
			workflow: Workflow{
				Status:              StatusWaitingForHuman,
				MultiChoiceQuestion: &MultiChoiceQuestion{Questions: []QuestionItem{{Question: "test"}}},
			},
			expected: true,
		},
		{
			name: "waitingWithMessage",
			workflow: Workflow{
				Status:       StatusWaitingForHuman,
				HumanMessage: "Please respond",
			},
			expected: true,
		},
		{
			name: "waitingWithBoth",
			workflow: Workflow{
				Status:              StatusWaitingForHuman,
				MultiChoiceQuestion: &MultiChoiceQuestion{Questions: []QuestionItem{{Question: "test"}}},
				HumanMessage:        "Please respond",
			},
			expected: true,
		},
		{
			name: "waitingWithoutQuestionOrMessage",
			workflow: Workflow{
				Status: StatusWaitingForHuman,
			},
			expected: false,
		},
		{
			name: "workingWithQuestion",
			workflow: Workflow{
				Status:              StatusWorking,
				MultiChoiceQuestion: &MultiChoiceQuestion{Questions: []QuestionItem{{Question: "test"}}},
			},
			expected: false,
		},
		{
			name: "workingWithMessage",
			workflow: Workflow{
				Status:       StatusWorking,
				HumanMessage: "Please respond",
			},
			expected: false,
		},
		{
			name: "agentDoneWithQuestion",
			workflow: Workflow{
				Status:              StatusAgentDone,
				MultiChoiceQuestion: &MultiChoiceQuestion{Questions: []QuestionItem{{Question: "test"}}},
			},
			expected: false,
		},
		{
			name: "completeWithQuestion",
			workflow: Workflow{
				Status:              StatusComplete,
				MultiChoiceQuestion: &MultiChoiceQuestion{Questions: []QuestionItem{{Question: "test"}}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.workflow.NeedsHumanInput()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAskAndWaitKeepsQuestionInMemoryOnly(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := NewCoordinatorWith(statePath, Workflow{Status: StatusWorking})
	require.NoError(t, errCoord)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)
	go func() {
		_, errAsk := coord.AskAndWait(ctx, &MultiChoiceQuestion{
			Questions: []QuestionItem{{Question: "Pick one", Choices: []string{"A", "B"}}},
		}, "Pick one")
		errCh <- errAsk
	}()

	require.Eventually(t, func() bool {
		return coord.State().NeedsHumanInput()
	}, time.Second, 10*time.Millisecond)

	persisted, errPersisted := NewCoordinator(statePath)
	require.NoError(t, errPersisted)
	assert.False(t, persisted.State().NeedsHumanInput())
	assert.Empty(t, persisted.State().HumanMessage)
	assert.Nil(t, persisted.State().MultiChoiceQuestion)

	cancel()
	select {
	case errAsk := <-errCh:
		require.ErrorIs(t, errAsk, context.Canceled)
	case <-time.After(time.Second):
		require.Fail(t, "AskAndWait did not return after cancellation")
	}
	assert.False(t, coord.State().NeedsHumanInput())
}

func TestTokenUsageAdd(t *testing.T) {
	tests := []struct {
		name     string
		t1       TokenUsage
		t2       TokenUsage
		expected TokenUsage
	}{
		{
			name: "addTwoUsages",
			t1: TokenUsage{
				Input:     50,
				Output:    30,
				Reasoning: 20,
			},
			t2: TokenUsage{
				Input:     100,
				Output:    60,
				Reasoning: 40,
			},
			expected: TokenUsage{
				Input:     150,
				Output:    90,
				Reasoning: 60,
			},
		},
		{
			name: "addZeroUsage",
			t1: TokenUsage{
				Input:     50,
				Output:    30,
				Reasoning: 20,
			},
			t2: TokenUsage{},
			expected: TokenUsage{
				Input:     50,
				Output:    30,
				Reasoning: 20,
			},
		},
		{
			name:     "addTwoZeroUsages",
			t1:       TokenUsage{},
			t2:       TokenUsage{},
			expected: TokenUsage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.t1.Add(tt.t2)
			assert.Equal(t, tt.expected, tt.t1)
		})
	}
}

func TestWorkflowToolsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		workflow Workflow
		expected bool
	}{
		{
			name:     "defaultFalse",
			workflow: Workflow{},
			expected: false,
		},
		{
			name:     "interactiveTrue",
			workflow: Workflow{InteractionMode: ModeInteractive},
			expected: true,
		},
		{
			name:     "selfDriveFalse",
			workflow: Workflow{InteractionMode: ModeSelfDrive},
			expected: false,
		},
		{
			name:     "continuousFalse",
			workflow: Workflow{InteractionMode: ModeContinuous},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.workflow.ToolsAllowed()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCoordinator(t *testing.T) {
	t.Run("loadExistingState", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		wf := Workflow{
			Status:   StatusWorking,
			Task:     "test task",
			Progress: []ProgressEntry{{Description: "test progress"}},
		}
		require.NoError(t, save(statePath, wf))

		coord, err := NewCoordinator(statePath)
		require.NoError(t, err)
		require.NotNil(t, coord)

		state := coord.State()
		assert.Equal(t, StatusWorking, state.Status)
		assert.Equal(t, "test task", state.Task)
		assert.Len(t, state.Progress, 1)
	})

	t.Run("loadNonexistentState", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		_, err := NewCoordinator(statePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading coordinator state")
	})
}

func TestNewCoordinatorEmpty(t *testing.T) {
	t.Run("createEmptyCoordinator", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		coord := NewCoordinatorEmpty(statePath)
		require.NotNil(t, coord)

		state := coord.State()
		assert.Equal(t, StatusWorking, state.Status)
		assert.Empty(t, state.Task)
		assert.Empty(t, state.Progress)
	})
}

func TestNewCoordinatorWith(t *testing.T) {
	t.Run("createWithInitialWorkflow", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		wf := Workflow{
			Status:   StatusWorking,
			Task:     "initial task",
			Progress: []ProgressEntry{{Description: "initial progress"}},
		}

		coord, err := NewCoordinatorWith(statePath, wf)
		require.NoError(t, err)
		require.NotNil(t, coord)

		state := coord.State()
		assert.Equal(t, StatusWorking, state.Status)
		assert.Equal(t, "initial task", state.Task)
		assert.Len(t, state.Progress, 1)

		loaded, err := load(statePath)
		require.NoError(t, err)
		assert.Equal(t, wf.Status, loaded.Status)
		assert.Equal(t, wf.Task, loaded.Task)
	})
}

func TestWorkflowJSONOmitsMessages(t *testing.T) {
	data, err := json.Marshal(Workflow{Status: StatusWorking})
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"messages"`)
	assert.NotContains(t, string(data), `"currentModel"`)
	assert.NotContains(t, string(data), `"modelStatuses"`)

	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	coord, err := NewCoordinatorWith(statePath, Workflow{Status: StatusWorking})
	require.NoError(t, err)
	require.NotNil(t, coord)

	content, err := os.ReadFile(statePath)
	require.NoError(t, err)
	assert.NotContains(t, string(content), `"messages"`)
	assert.NotContains(t, string(content), `"currentModel"`)
	assert.NotContains(t, string(content), `"modelStatuses"`)
}

func TestCoordinatorState(t *testing.T) {
	t.Run("returnStateSnapshot", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		wf := Workflow{
			Status: StatusWorking,
			Task:   "test task",
		}

		coord, err := NewCoordinatorWith(statePath, wf)
		require.NoError(t, err)

		state := coord.State()
		assert.Equal(t, StatusWorking, state.Status)
		assert.Equal(t, "test task", state.Task)
	})
}

func TestCoordinatorUpdateState(t *testing.T) {
	t.Run("updateAndPersist", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		wf := Workflow{
			Status: StatusWorking,
			Task:   "initial task",
		}

		coord, err := NewCoordinatorWith(statePath, wf)
		require.NoError(t, err)

		err = coord.UpdateState(func(wf *Workflow) {
			wf.Task = "updated task"
			wf.Progress = append(wf.Progress, ProgressEntry{Description: "new progress"})
		})
		require.NoError(t, err)

		state := coord.State()
		assert.Equal(t, "updated task", state.Task)
		assert.Len(t, state.Progress, 1)

		loaded, err := load(statePath)
		require.NoError(t, err)
		assert.Equal(t, "updated task", loaded.Task)
		assert.Len(t, loaded.Progress, 1)
	})

	t.Run("onUpdateCallback", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")

		wf := Workflow{Status: StatusWorking}
		coord, err := NewCoordinatorWith(statePath, wf)
		require.NoError(t, err)

		callbackCalled := false
		coord.OnUpdate(func() {
			callbackCalled = true
		})

		err = coord.UpdateState(func(wf *Workflow) {
			wf.Task = "new task"
		})
		require.NoError(t, err)
		assert.True(t, callbackCalled)
	})
}
