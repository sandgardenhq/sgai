package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func setupStateDir(t *testing.T, initial state.Workflow) string {
	t.Helper()
	tmpDir := t.TempDir()
	sgaiDir := filepath.Join(tmpDir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sgaiDir, "state.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func loadState(t *testing.T, workDir string) state.Workflow {
	t.Helper()
	statePath := filepath.Join(workDir, ".sgai", "state.json")
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	return s
}

func TestUpdateWorkflowStatePreservesWaitingForHumanStatus(t *testing.T) {
	t.Run("preservedOnWorking", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "working",
			Task:        "doing something",
			AddProgress: "progress note",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q preserved, got %q", state.StatusWaitingForHuman, s.Status)
		}

		if !strings.Contains(result, "Waiting for human response") {
			t.Errorf("expected preservation message, got: %s", result)
		}

		if s.Task != "doing something" {
			t.Errorf("expected task to be updated, got %q", s.Task)
		}

		if len(s.Progress) != 1 || s.Progress[0].Description != "progress note" {
			t.Errorf("expected progress note to be added, got %v", s.Progress)
		}
	})

	t.Run("preservedOnAgentDone", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "agent-done",
			AddProgress: "agent finished",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q preserved, got %q", state.StatusWaitingForHuman, s.Status)
		}

		if !strings.Contains(result, "Waiting for human response") {
			t.Errorf("expected preservation message, got: %s", result)
		}
	})

	t.Run("preservedOnComplete", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "complete",
			AddProgress: "claiming completion",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q preserved, got %q", state.StatusWaitingForHuman, s.Status)
		}

		if !strings.Contains(result, "Waiting for human response") {
			t.Errorf("expected preservation message, got: %s", result)
		}
	})

	t.Run("preservedOnEmptyStatus", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "",
			Task:        "new task",
			AddProgress: "still working",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q preserved, got %q", state.StatusWaitingForHuman, s.Status)
		}

		if s.Task != "new task" {
			t.Errorf("expected task updated to %q, got %q", "new task", s.Task)
		}

		if !strings.Contains(result, "Waiting for human response") {
			t.Errorf("expected preservation message, got: %s", result)
		}
	})

	t.Run("taskAndProgressUpdatedDuringPreservation", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
			Progress: []state.ProgressEntry{
				{Timestamp: "t1", Agent: "coordinator", Description: "existing note"},
			},
		})

		_, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "working",
			Task:        "new task during wait",
			AddProgress: "additional progress",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Task != "new task during wait" {
			t.Errorf("expected task updated, got %q", s.Task)
		}

		if len(s.Progress) != 2 {
			t.Fatalf("expected 2 progress entries, got %d", len(s.Progress))
		}
		if s.Progress[0].Description != "existing note" {
			t.Errorf("expected first progress preserved, got %q", s.Progress[0].Description)
		}
		if s.Progress[1].Description != "additional progress" {
			t.Errorf("expected second progress added, got %q", s.Progress[1].Description)
		}
	})

	t.Run("workingAfterHumanRespondedTransitionsNormally", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "agent-done",
			AddProgress: "finished after human responded",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusAgentDone {
			t.Errorf("expected status %q after human responded, got %q", state.StatusAgentDone, s.Status)
		}
	})

	t.Run("workingToAgentDoneTransitionsNormally", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "backend-go-developer",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status:      "agent-done",
			AddProgress: "done with work",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusAgentDone {
			t.Errorf("expected status %q, got %q", state.StatusAgentDone, s.Status)
		}

		if strings.Contains(result, "Waiting for human response") {
			t.Errorf("should not have preservation message for normal transition, got: %s", result)
		}

		if !strings.Contains(result, "State updated successfully") {
			t.Errorf("expected success message, got: %s", result)
		}
	})

	t.Run("workingToWorkingTransitionsNormally", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status: "working",
			Task:   "updated task",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusWorking {
			t.Errorf("expected status %q, got %q", state.StatusWorking, s.Status)
		}
	})

	t.Run("invalidStatusRejectedRegardless", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status: "invalid-status",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Error: Invalid status") {
			t.Errorf("expected invalid status error, got: %s", result)
		}

		s := loadState(t, workDir)
		if s.Status != state.StatusWorking {
			t.Errorf("status should not change on invalid input, got %q", s.Status)
		}
	})

	t.Run("invalidStatusRejectedWhenNotPreserved", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusAgentDone,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status: "bogus",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Error: Invalid status") {
			t.Errorf("expected invalid status error, got: %s", result)
		}
	})

	t.Run("taskClearedOnAgentDone", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status: "agent-done",
			Task:   "should be cleared",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Task != "" {
			t.Errorf("expected task cleared on agent-done, got %q", s.Task)
		}
	})

	t.Run("taskClearedOnComplete", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(workDir, updateWorkflowStateArgs{
			Status: "complete",
			Task:   "should be cleared",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := loadState(t, workDir)
		if s.Task != "" {
			t.Errorf("expected task cleared on complete, got %q", s.Task)
		}
	})
}
