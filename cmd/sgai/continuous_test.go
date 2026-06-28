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

func TestResetWorkflowForNextCycle(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusComplete,
		InteractionMode: state.ModeSelfDrive,
	})
	require.NoError(t, err)

	resetWorkflowForNextCycle(coord)

	snapshot := coord.State()
	assert.Equal(t, state.StatusWorking, snapshot.Status)
	assert.Equal(t, state.ModeContinuous, snapshot.InteractionMode)
}

func TestWatchForTriggerCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("# Goal"), 0644))

	result := watchForTrigger(ctx, dir, "checksum123", 0, "")
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

	result := watchForTrigger(ctx, dir, "stale-checksum", 0, "")
	assert.Equal(t, triggerGoal, result)
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

	result := watchForTrigger(ctx, dir, checksum, 1*time.Millisecond, "")
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

	result := watchForTrigger(ctx, dir, checksum, 1*time.Millisecond, "* * * * *")
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

	result := watchForTrigger(ctx, dir, checksum, 1*time.Millisecond, "invalid cron")
	assert.Equal(t, triggerAuto, result)
}

func TestTriggerKindConstants(t *testing.T) {
	assert.Equal(t, triggerKind(""), triggerNone)
	assert.Equal(t, triggerKind("goal-changed"), triggerGoal)
	assert.Equal(t, triggerKind("auto-timer"), triggerAuto)
	assert.Equal(t, triggerKind("cron-schedule"), triggerCron)
}
