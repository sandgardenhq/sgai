package main

import (
	"context"
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

func setupCoordinator(t *testing.T, initial state.Workflow) *state.Coordinator {
	t.Helper()
	workDir := setupStateDir(t, initial)
	statePath := filepath.Join(workDir, ".sgai", "state.json")
	coord, err := state.NewCoordinator(statePath)
	if err != nil {
		t.Fatalf("failed to create coordinator: %v", err)
	}
	return coord
}

func loadState(t *testing.T, workDir string) state.Workflow {
	t.Helper()
	stPath := filepath.Join(workDir, ".sgai", "state.json")
	coord, err := state.NewCoordinator(stPath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	return coord.State()
}

func TestUpdateWorkflowStatePreservesWaitingForHumanStatus(t *testing.T) {
	t.Run("preservedOnWorking", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status:      "working",
			Task:        "doing something",
			AddProgress: "progress note",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
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
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status:      "agent-done",
			AddProgress: "agent finished",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q preserved, got %q", state.StatusWaitingForHuman, s.Status)
		}

		if !strings.Contains(result, "Waiting for human response") {
			t.Errorf("expected preservation message, got: %s", result)
		}
	})

	t.Run("preservedOnComplete", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status:      "complete",
			AddProgress: "claiming completion",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q preserved, got %q", state.StatusWaitingForHuman, s.Status)
		}

		if !strings.Contains(result, "Waiting for human response") {
			t.Errorf("expected preservation message, got: %s", result)
		}
	})

	t.Run("preservedOnEmptyStatus", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status:      "",
			Task:        "new task",
			AddProgress: "still working",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
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
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			CurrentAgent: "coordinator",
			Progress: []state.ProgressEntry{
				{Timestamp: "t1", Agent: "coordinator", Description: "existing note"},
			},
		})

		_, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status:      "working",
			Task:        "new task during wait",
			AddProgress: "additional progress",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
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
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status:      "agent-done",
			AddProgress: "finished after human responded",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Status != state.StatusAgentDone {
			t.Errorf("expected status %q after human responded, got %q", state.StatusAgentDone, s.Status)
		}
	})

	t.Run("workingToAgentDoneTransitionsNormally", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "backend-go-developer",
		})

		result, err := updateWorkflowState(context.Background(), coord, "backend-go-developer", updateWorkflowStateArgs{
			Status:      "agent-done",
			AddProgress: "done with work",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
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
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status: "working",
			Task:   "updated task",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Status != state.StatusWorking {
			t.Errorf("expected status %q, got %q", state.StatusWorking, s.Status)
		}
	})

	t.Run("invalidStatusRejectedRegardless", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status: "invalid-status",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Error: Invalid status") {
			t.Errorf("expected invalid status error, got: %s", result)
		}

		s := coord.State()
		if s.Status != state.StatusWorking {
			t.Errorf("status should not change on invalid input, got %q", s.Status)
		}
	})

	t.Run("invalidStatusRejectedWhenNotPreserved", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusAgentDone,
			CurrentAgent: "coordinator",
		})

		result, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
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
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status: "agent-done",
			Task:   "should be cleared",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Task != "" {
			t.Errorf("expected task cleared on agent-done, got %q", s.Task)
		}
	})

	t.Run("taskClearedOnComplete", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(context.Background(), coord, "coordinator", updateWorkflowStateArgs{
			Status: "complete",
			Task:   "should be cleared",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Task != "" {
			t.Errorf("expected task cleared on complete, got %q", s.Task)
		}
	})
}

func TestUpdateWorkflowStateProgressUsesCallerAgent(t *testing.T) {
	t.Run("progressEntryUsesCallerAgentNotStateCurrentAgent", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(context.Background(), coord, "backend-go-developer", updateWorkflowStateArgs{
			Status:      "working",
			Task:        "doing backend work",
			AddProgress: "wrote the handler",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if len(s.Progress) != 1 {
			t.Fatalf("expected 1 progress entry, got %d", len(s.Progress))
		}
		if s.Progress[0].Agent != "backend-go-developer" {
			t.Errorf("expected progress agent %q, got %q", "backend-go-developer", s.Progress[0].Agent)
		}
		if s.Progress[0].Description != "wrote the handler" {
			t.Errorf("expected progress description %q, got %q", "wrote the handler", s.Progress[0].Description)
		}
	})

	t.Run("multipleAgentsGetCorrectAgentInProgress", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		})

		_, err := updateWorkflowState(context.Background(), coord, "react-developer", updateWorkflowStateArgs{
			Status:      "working",
			AddProgress: "built the component",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = updateWorkflowState(context.Background(), coord, "go-readability-reviewer", updateWorkflowStateArgs{
			Status:      "working",
			AddProgress: "reviewed the code",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if len(s.Progress) != 2 {
			t.Fatalf("expected 2 progress entries, got %d", len(s.Progress))
		}
		if s.Progress[0].Agent != "react-developer" {
			t.Errorf("expected first progress agent %q, got %q", "react-developer", s.Progress[0].Agent)
		}
		if s.Progress[1].Agent != "go-readability-reviewer" {
			t.Errorf("expected second progress agent %q, got %q", "go-readability-reviewer", s.Progress[1].Agent)
		}
	})
}
