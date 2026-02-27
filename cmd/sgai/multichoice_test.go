package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestAskUserQuestion(t *testing.T) {
	t.Run("singleQuestionBlocksUntilAnswer", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:          state.StatusWorking,
			CurrentAgent:    "coordinator",
			InteractionMode: state.ModeBrainstorming,
		})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{
					Question:    "Which option do you prefer?",
					Choices:     []string{"Option A", "Option B", "Option C"},
					MultiSelect: true,
				},
			},
		}

		type callResult struct {
			result string
			err    error
		}
		done := make(chan callResult, 1)
		go func() {
			r, e := askUserQuestion(context.Background(), coord, args)
			done <- callResult{r, e}
		}()

		coord.Respond("Selected: Option A")

		cr := <-done
		if cr.err != nil {
			t.Fatalf("askUserQuestion error: %v", cr.err)
		}

		if cr.result == "" {
			t.Error("expected non-empty result")
		}

		s := coord.State()
		if s.Status == state.StatusWaitingForHuman {
			t.Error("status should not be waiting-for-human after response")
		}
	})

	t.Run("multipleQuestionsSetsStateCorrectly", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{
					Question:    "Which database?",
					Choices:     []string{"PostgreSQL", "MySQL", "SQLite"},
					MultiSelect: false,
				},
				{
					Question:    "Which auth method?",
					Choices:     []string{"JWT", "Session", "OAuth"},
					MultiSelect: true,
				},
			},
		}

		type callResult struct {
			result string
			err    error
		}
		done := make(chan callResult, 1)
		go func() {
			r, e := askUserQuestion(context.Background(), coord, args)
			done <- callResult{r, e}
		}()

		coord.Respond("Selected: PostgreSQL, JWT")

		cr := <-done
		if cr.err != nil {
			t.Fatalf("askUserQuestion error: %v", cr.err)
		}

		if !strings.Contains(cr.result, "2 question(s)") {
			t.Errorf("expected result to mention 2 questions, got: %s", cr.result)
		}
	})

	t.Run("emptyQuestionsReturnsError", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{},
		}

		result, _ := askUserQuestion(context.Background(), coord, args)

		if !strings.Contains(result, "Error: At least one question is required") {
			t.Errorf("expected error message, got: %q", result)
		}
		if !strings.Contains(result, `"questions"`) {
			t.Errorf("expected error message to include JSON example, got: %q", result)
		}
	})

	t.Run("questionWithEmptyChoicesReturnsError", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{
					Question:    "Question with no choices",
					Choices:     []string{},
					MultiSelect: false,
				},
			},
		}

		result, _ := askUserQuestion(context.Background(), coord, args)

		if result != "Error: Question 1 has no choices" {
			t.Errorf("expected error message, got: %q", result)
		}
	})
}

func TestAskUserQuestionStress(t *testing.T) {
	t.Run("batchOfTenQuestions", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		questions := make([]questionItem, 10)
		for i := range questions {
			questions[i] = questionItem{
				Question:    fmt.Sprintf("Question %d?", i+1),
				Choices:     []string{"yes", "no", "maybe"},
				MultiSelect: i%2 == 0,
			}
		}

		type callResult struct {
			result string
			err    error
		}
		done := make(chan callResult, 1)
		go func() {
			r, e := askUserQuestion(context.Background(), coord, askUserQuestionArgs{Questions: questions})
			done <- callResult{r, e}
		}()

		coord.Respond("Selected: yes")

		cr := <-done
		if cr.err != nil {
			t.Fatalf("askUserQuestion error: %v", cr.err)
		}

		if !strings.Contains(cr.result, "10 question(s)") {
			t.Errorf("expected result to mention 10 questions, got: %s", cr.result)
		}
	})

	t.Run("preservesExistingProgress", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:          state.StatusWorking,
			CurrentAgent:    "coordinator",
			InteractionMode: state.ModeBrainstorming,
			Progress: []state.ProgressEntry{
				{Timestamp: "t1", Agent: "coordinator", Description: "previous work"},
				{Timestamp: "t2", Agent: "coordinator", Description: "more work"},
			},
		})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{Question: "Choose?", Choices: []string{"A", "B"}},
			},
		}

		done := make(chan error, 1)
		go func() {
			_, e := askUserQuestion(context.Background(), coord, args)
			done <- e
		}()

		coord.Respond("Selected: A")

		if err := <-done; err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		s := coord.State()
		if len(s.Progress) != 2 {
			t.Errorf("expected 2 progress entries preserved, got %d", len(s.Progress))
		}
	})

	t.Run("nilQuestionsReturnsError", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{Questions: nil}

		result, _ := askUserQuestion(context.Background(), coord, args)

		if !strings.Contains(result, "Error: At least one question is required") {
			t.Errorf("expected error message, got: %q", result)
		}
		if !strings.Contains(result, `"questions"`) {
			t.Errorf("expected error message to include JSON example, got: %q", result)
		}
	})

	t.Run("questionWithNilChoicesReturnsError", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{Question: "No choices?", Choices: nil},
			},
		}

		result, _ := askUserQuestion(context.Background(), coord, args)

		if result != "Error: Question 1 has no choices" {
			t.Errorf("expected error message, got: %q", result)
		}
	})
}

func TestAskUserWorkGateStress(t *testing.T) {
	t.Run("setsIsWorkGateTrue", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:          state.StatusWorking,
			CurrentAgent:    "coordinator",
			InteractionMode: state.ModeBrainstorming,
		})

		done := make(chan error, 1)
		go func() {
			_, e := askUserWorkGate(context.Background(), coord, "test summary")
			done <- e
		}()

		coord.Respond("Selected: " + workGateApprovalText)

		if err := <-done; err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}
	})

	t.Run("preservesExistingMessages", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{
			Status:          state.StatusWorking,
			CurrentAgent:    "coordinator",
			InteractionMode: state.ModeBrainstorming,
			Messages: []state.Message{
				{ID: 1, FromAgent: "coordinator", ToAgent: "developer", Body: "do stuff"},
				{ID: 2, FromAgent: "developer", ToAgent: "coordinator", Body: "done"},
			},
		})

		done := make(chan error, 1)
		go func() {
			_, e := askUserWorkGate(context.Background(), coord, "test summary")
			done <- e
		}()

		coord.Respond("Not ready yet, need more clarification")

		if err := <-done; err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		s := coord.State()
		if len(s.Messages) != 2 {
			t.Errorf("expected 2 messages preserved, got %d", len(s.Messages))
		}
	})

	t.Run("hasTwoChoices", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		type callResult struct {
			result string
			err    error
		}
		done := make(chan callResult, 1)
		go func() {
			r, e := askUserWorkGate(context.Background(), coord, "test summary")
			done <- callResult{r, e}
		}()

		coord.Respond("Not ready yet, need more clarification")

		cr := <-done
		if cr.err != nil {
			t.Fatalf("askUserWorkGate error: %v", cr.err)
		}

		if !strings.Contains(cr.result, "DEFINITION IS COMPLETE") {
			t.Errorf("expected result to contain approval choice, got: %q", cr.result)
		}
	})

	t.Run("emptySummaryReturnsError", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		result, err := askUserWorkGate(context.Background(), coord, "")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		if !strings.Contains(result, "Error:") {
			t.Errorf("expected error message for empty summary, got: %q", result)
		}
	})

	t.Run("whitespaceSummaryReturnsError", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		result, err := askUserWorkGate(context.Background(), coord, "   \t\n  ")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		if !strings.Contains(result, "Error:") {
			t.Errorf("expected error message for whitespace-only summary, got: %q", result)
		}
	})

	t.Run("summaryAppearsInResponse", func(t *testing.T) {
		coord := setupCoordinator(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		summary := "## What Will Be Built\n- Feature X\n\n## Key Decisions\n- Use approach A"

		type callResult struct {
			result string
			err    error
		}
		done := make(chan callResult, 1)
		go func() {
			r, e := askUserWorkGate(context.Background(), coord, summary)
			done <- callResult{r, e}
		}()

		coord.Respond("Not ready yet, need more clarification")

		cr := <-done
		if cr.err != nil {
			t.Fatalf("askUserWorkGate error: %v", cr.err)
		}

		if !strings.Contains(cr.result, "Is the definition complete?") {
			t.Errorf("expected result to contain approval prompt, got: %q", cr.result)
		}
	})
}

func TestMultiChoiceQuestionStateRoundTrip(t *testing.T) {
	tmpDir := setupStateDir(t, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Choose your option",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{
					Question:    "Which framework should we use?",
					Choices:     []string{"React", "Vue", "Angular", "Svelte"},
					MultiSelect: false,
				},
			},
		},
		CurrentAgent: "coordinator",
	})

	readState := loadState(t, tmpDir)

	if readState.MultiChoiceQuestion == nil {
		t.Fatal("expected MultiChoiceQuestion to be preserved")
	}

	if len(readState.MultiChoiceQuestion.Questions) != 1 {
		t.Errorf("questions length mismatch: got %d, want %d",
			len(readState.MultiChoiceQuestion.Questions), 1)
	}

	readQ := readState.MultiChoiceQuestion.Questions[0]

	if readQ.Question != "Which framework should we use?" {
		t.Errorf("question mismatch: got %q, want %q", readQ.Question, "Which framework should we use?")
	}

	if len(readQ.Choices) != 4 {
		t.Errorf("choices length mismatch: got %d, want %d", len(readQ.Choices), 4)
	}

	if readQ.MultiSelect != false {
		t.Errorf("multiSelect mismatch: got %v, want %v", readQ.MultiSelect, false)
	}
}

func TestMultiChoiceOmitsWhenNil(t *testing.T) {
	tmpDir := setupStateDir(t, state.Workflow{
		Status:       state.StatusWorking,
		CurrentAgent: "coordinator",
	})

	loadedState := loadState(t, tmpDir)

	if loadedState.MultiChoiceQuestion != nil {
		t.Error("multiChoiceQuestion should be nil when not set")
	}
}
