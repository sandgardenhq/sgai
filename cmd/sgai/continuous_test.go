package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasHumanPartnerMessage(t *testing.T) {
	tests := []struct {
		name          string
		messages      []state.Message
		expectedFound bool
		expectedID    int
	}{
		{
			name:          "emptyMessages",
			messages:      []state.Message{},
			expectedFound: false,
		},
		{
			name: "humanPartnerMessage",
			messages: []state.Message{
				{ID: 1, FromAgent: "Human Partner", Read: false},
			},
			expectedFound: true,
			expectedID:    1,
		},
		{
			name: "humanPartnerMessageAlreadyRead",
			messages: []state.Message{
				{ID: 1, FromAgent: "Human Partner", Read: true},
			},
			expectedFound: false,
		},
		{
			name: "otherAgentMessage",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", Read: false},
			},
			expectedFound: false,
		},
		{
			name: "mixedMessages",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", Read: false},
				{ID: 2, FromAgent: "Human Partner", Read: false},
				{ID: 3, FromAgent: "agent2", Read: false},
			},
			expectedFound: true,
			expectedID:    2,
		},
		{
			name: "multipleHumanPartnerMessages",
			messages: []state.Message{
				{ID: 1, FromAgent: "Human Partner", Read: true},
				{ID: 2, FromAgent: "Human Partner", Read: false},
			},
			expectedFound: true,
			expectedID:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, msg := hasHumanPartnerMessage(tt.messages)
			assert.Equal(t, tt.expectedFound, found)
			if tt.expectedFound {
				assert.NotNil(t, msg)
				assert.Equal(t, tt.expectedID, msg.ID)
			} else {
				assert.Nil(t, msg)
			}
		})
	}
}

func TestReadContinuousModePrompt(t *testing.T) {
	tests := []struct {
		name        string
		goalContent string
		expected    string
	}{
		{
			name: "withContinuousModePrompt",
			goalContent: `---
continuousModePrompt: "Check for new issues every hour"
---
# Test Goal`,
			expected: "Check for new issues every hour",
		},
		{
			name: "withoutContinuousModePrompt",
			goalContent: `---
---
# Test Goal`,
			expected: "",
		},
		{
			name:        "noGoalFile",
			goalContent: "",
			expected:    "",
		},
		{
			name: "emptyContinuousModePrompt",
			goalContent: `---
continuousModePrompt: ""
---
# Test Goal`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := t.TempDir()

			if tt.goalContent != "" {
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(tt.goalContent), 0644))
			}

			result := readContinuousModePrompt(workspacePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadContinuousModeAutoCron(t *testing.T) {
	tests := []struct {
		name           string
		goalContent    string
		expectedDur    time.Duration
		expectedPrompt string
	}{
		{
			name: "withAutoCron",
			goalContent: `---
continuousModeAuto: "1h"
continuousModeCron: "Check for updates"
---
# Test Goal`,
			expectedDur:    time.Hour,
			expectedPrompt: "Check for updates",
		},
		{
			name: "withAutoCronNoPrompt",
			goalContent: `---
continuousModeAuto: "30m"
---
# Test Goal`,
			expectedDur:    30 * time.Minute,
			expectedPrompt: "",
		},
		{
			name: "withoutAutoCron",
			goalContent: `---
---
# Test Goal`,
			expectedDur:    0,
			expectedPrompt: "",
		},
		{
			name:           "noGoalFile",
			goalContent:    "",
			expectedDur:    0,
			expectedPrompt: "",
		},
		{
			name: "invalidDuration",
			goalContent: `---
continuousModeAuto: "invalid"
---
# Test Goal`,
			expectedDur:    0,
			expectedPrompt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspacePath := t.TempDir()

			if tt.goalContent != "" {
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(tt.goalContent), 0644))
			}

			dur, prompt := readContinuousModeAutoCron(workspacePath)
			assert.Equal(t, tt.expectedDur, dur)
			assert.Equal(t, tt.expectedPrompt, prompt)
		})
	}
}

func TestUpdateContinuousModeState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:   state.StatusWorking,
		Progress: []state.ProgressEntry{},
	})
	require.NoError(t, err)

	updateContinuousModeState(coord, "running tests", "test-agent", "started test execution")

	snapshot := coord.State()
	assert.Equal(t, "running tests", snapshot.Task)
	assert.Equal(t, "test-agent", snapshot.CurrentAgent)
	assert.Len(t, snapshot.Progress, 1)
	assert.Equal(t, "test-agent", snapshot.Progress[0].Agent)
	assert.Equal(t, "started test execution", snapshot.Progress[0].Description)
}

func TestUpdateContinuousModeProgress(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusWorking,
		Progress: []state.ProgressEntry{
			{Agent: "initial", Description: "first entry"},
		},
	})
	require.NoError(t, err)

	updateContinuousModeProgress(coord, "completed phase 2")

	snapshot := coord.State()
	assert.Len(t, snapshot.Progress, 2)
	assert.Equal(t, "continuous-mode", snapshot.Progress[1].Agent)
	assert.Equal(t, "completed phase 2", snapshot.Progress[1].Description)
}

func TestMarkMessageAsRead(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusWorking,
		Messages: []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "run the tests", Read: false},
			{ID: 2, FromAgent: "agent1", ToAgent: "agent2", Body: "other msg", Read: false},
		},
	})
	require.NoError(t, err)

	markMessageAsRead(coord, 1)

	snapshot := coord.State()
	assert.True(t, snapshot.Messages[0].Read)
	assert.Equal(t, "continuous-mode", snapshot.Messages[0].ReadBy)
	assert.NotEmpty(t, snapshot.Messages[0].ReadAt)
	assert.False(t, snapshot.Messages[1].Read)
}

func TestMarkMessageAsReadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusWorking,
		Messages: []state.Message{
			{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "msg", Read: false},
		},
	})
	require.NoError(t, err)

	markMessageAsRead(coord, 999)

	snapshot := coord.State()
	assert.False(t, snapshot.Messages[0].Read)
}

func TestResetWorkflowForNextCycle(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusComplete,
		InteractionMode: state.ModeSelfDrive,
		CurrentAgent:    "backend-developer",
	})
	require.NoError(t, err)

	resetWorkflowForNextCycle(coord)

	snapshot := coord.State()
	assert.Equal(t, state.StatusWorking, snapshot.Status)
	assert.Equal(t, state.ModeContinuous, snapshot.InteractionMode)
	assert.Equal(t, "coordinator", snapshot.CurrentAgent)
}

func TestPrependSteeringMessage(t *testing.T) {
	tests := []struct {
		name         string
		existingGoal string
		message      string
		expected     string
		skipCreate   bool
	}{
		{
			name:         "noFrontmatter",
			existingGoal: "# My Goal\n\nSome content",
			message:      "Steering message",
			expected:     "Steering message\n\n# My Goal\n\nSome content",
		},
		{
			name: "withFrontmatter",
			existingGoal: `---
flow: |
  "a" -> "b"
---
# My Goal

Some content`,
			message: "Steering message",
			expected: `---
flow: |
  "a" -> "b"
---

Steering message

# My Goal

Some content`,
		},
		{
			name: "emptyGoal",
			existingGoal: `---
---
`,
			message:  "Steering message",
			expected: "---\n---\n\nSteering message\n\n",
		},
		{
			name:         "emptyContent",
			existingGoal: "",
			message:      "Steering message",
			expected:     "Steering message\n\n",
			skipCreate:   true,
		},
		{
			name: "unclosedFrontmatter",
			existingGoal: `---
flow: |
  "a" -> "b"
# My Goal`,
			message:  "Steering message",
			expected: "Steering message\n\n---\nflow: |\n  \"a\" -> \"b\"\n# My Goal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			goalPath := filepath.Join(tmpDir, "GOAL.md")
			if tt.existingGoal != "" && !tt.skipCreate {
				require.NoError(t, os.WriteFile(goalPath, []byte(tt.existingGoal), 0644))
			}

			err := prependSteeringMessage(goalPath, tt.message)
			if tt.skipCreate {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			content, err := os.ReadFile(goalPath)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(content))
		})
	}
}

func TestPrependSteeringMessageNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	goalPath := filepath.Join(tmpDir, "GOAL.md")

	err := prependSteeringMessage(goalPath, "test message")
	assert.Error(t, err)
}

func TestWatchForTriggerCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("# Goal"), 0644))

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
	require.NoError(t, errCoord)

	result := watchForTrigger(ctx, dir, coord, "checksum123", 0, "")
	assert.Equal(t, triggerNone, result)
}

func TestWatchForTriggerGoalChanged(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal version 2"), 0644))

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
	require.NoError(t, errCoord)

	result := watchForTrigger(ctx, dir, coord, "stale-checksum", 0, "")
	assert.Equal(t, triggerGoal, result)
}

func TestWatchForTriggerSteeringMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0644))

	checksum, errChecksum := computeGoalChecksum(goalPath)
	require.NoError(t, errChecksum)

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Messages: []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "please fix", Read: false},
		},
	})
	require.NoError(t, errCoord)

	result := watchForTrigger(ctx, dir, coord, checksum, 0, "")
	assert.Equal(t, triggerSteering, result)
}

func TestWatchForTriggerAutoTimer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0644))

	checksum, errChecksum := computeGoalChecksum(goalPath)
	require.NoError(t, errChecksum)

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
	require.NoError(t, errCoord)

	result := watchForTrigger(ctx, dir, coord, checksum, 1*time.Millisecond, "")
	assert.Equal(t, triggerAuto, result)
}

func TestWatchForTriggerCronWithAutoFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0644))

	checksum, errChecksum := computeGoalChecksum(goalPath)
	require.NoError(t, errChecksum)

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
	require.NoError(t, errCoord)

	result := watchForTrigger(ctx, dir, coord, checksum, 1*time.Millisecond, "* * * * *")
	assert.Equal(t, triggerAuto, result)
}

func TestWatchForTriggerInvalidCronExpression(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0644))

	checksum, errChecksum := computeGoalChecksum(goalPath)
	require.NoError(t, errChecksum)

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
	require.NoError(t, errCoord)

	result := watchForTrigger(ctx, dir, coord, checksum, 1*time.Millisecond, "invalid cron")
	assert.Equal(t, triggerAuto, result)
}

func TestTriggerKindConstants(t *testing.T) {
	assert.Equal(t, triggerKind(""), triggerNone)
	assert.Equal(t, triggerKind("goal-changed"), triggerGoal)
	assert.Equal(t, triggerKind("steering-message"), triggerSteering)
	assert.Equal(t, triggerKind("auto-timer"), triggerAuto)
	assert.Equal(t, triggerKind("cron-schedule"), triggerCron)
}
