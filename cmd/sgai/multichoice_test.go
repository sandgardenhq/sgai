package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestAskUserQuestion(t *testing.T) {
	t.Run("singleQuestionSetsStateCorrectly", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
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

		result, err := askUserQuestion(workDir, args)
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		if result == "" {
			t.Error("expected non-empty result")
		}

		loadedState := loadState(t, workDir)

		if loadedState.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q, got %q", state.StatusWaitingForHuman, loadedState.Status)
		}

		if loadedState.MultiChoiceQuestion == nil {
			t.Fatal("expected MultiChoiceQuestion to be set")
		}

		if len(loadedState.MultiChoiceQuestion.Questions) != 1 {
			t.Fatalf("expected 1 question, got %d", len(loadedState.MultiChoiceQuestion.Questions))
		}

		q := loadedState.MultiChoiceQuestion.Questions[0]
		if q.Question != args.Questions[0].Question {
			t.Errorf("question mismatch: got %q, want %q", q.Question, args.Questions[0].Question)
		}

		if len(q.Choices) != len(args.Questions[0].Choices) {
			t.Errorf("choices length mismatch: got %d, want %d", len(q.Choices), len(args.Questions[0].Choices))
		}

		if q.MultiSelect != args.Questions[0].MultiSelect {
			t.Errorf("multiSelect mismatch: got %v, want %v", q.MultiSelect, args.Questions[0].MultiSelect)
		}

		if loadedState.HumanMessage != args.Questions[0].Question {
			t.Errorf("humanMessage should equal first question: got %q, want %q", loadedState.HumanMessage, args.Questions[0].Question)
		}
	})

	t.Run("multipleQuestionsSetsStateCorrectly", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

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

		result, err := askUserQuestion(workDir, args)
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		if !strings.Contains(result, "2 question(s)") {
			t.Errorf("expected result to mention 2 questions, got: %s", result)
		}

		loadedState := loadState(t, workDir)

		if len(loadedState.MultiChoiceQuestion.Questions) != 2 {
			t.Fatalf("expected 2 questions, got %d", len(loadedState.MultiChoiceQuestion.Questions))
		}

		if loadedState.MultiChoiceQuestion.Questions[0].Question != "Which database?" {
			t.Errorf("first question mismatch")
		}

		if loadedState.MultiChoiceQuestion.Questions[1].MultiSelect != true {
			t.Errorf("second question multiSelect should be true")
		}
	})

	t.Run("emptyQuestionsReturnsError", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{},
		}

		result, _ := askUserQuestion(workDir, args)

		if !strings.Contains(result, "Error: At least one question is required") {
			t.Errorf("expected error message, got: %q", result)
		}
		if !strings.Contains(result, `"questions"`) {
			t.Errorf("expected error message to include JSON example, got: %q", result)
		}
	})

	t.Run("questionWithEmptyChoicesReturnsError", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{
					Question:    "Question with no choices",
					Choices:     []string{},
					MultiSelect: false,
				},
			},
		}

		result, _ := askUserQuestion(workDir, args)

		if result != "Error: Question 1 has no choices" {
			t.Errorf("expected error message, got: %q", result)
		}
	})
}

func TestAskUserQuestionStress(t *testing.T) {
	t.Run("batchOfTenQuestions", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		questions := make([]questionItem, 10)
		for i := range questions {
			questions[i] = questionItem{
				Question:    fmt.Sprintf("Question %d?", i+1),
				Choices:     []string{"yes", "no", "maybe"},
				MultiSelect: i%2 == 0,
			}
		}

		result, err := askUserQuestion(workDir, askUserQuestionArgs{Questions: questions})
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		if !strings.Contains(result, "10 question(s)") {
			t.Errorf("expected result to mention 10 questions, got: %s", result)
		}

		loadedState := loadState(t, workDir)

		if loadedState.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q, got %q", state.StatusWaitingForHuman, loadedState.Status)
		}

		if len(loadedState.MultiChoiceQuestion.Questions) != 10 {
			t.Fatalf("expected 10 questions, got %d", len(loadedState.MultiChoiceQuestion.Questions))
		}
	})

	t.Run("preservesExistingProgress", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
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

		_, err := askUserQuestion(workDir, args)
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if len(loadedState.Progress) != 2 {
			t.Errorf("expected 2 progress entries preserved, got %d", len(loadedState.Progress))
		}
	})

	t.Run("setsStatusToWaitingForHuman", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusAgentDone, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{Question: "Pick?", Choices: []string{"X", "Y"}},
			},
		}

		_, err := askUserQuestion(workDir, args)
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if loadedState.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q, got %q", state.StatusWaitingForHuman, loadedState.Status)
		}
	})

	t.Run("nilQuestionsReturnsError", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{Questions: nil}

		result, _ := askUserQuestion(workDir, args)

		if !strings.Contains(result, "Error: At least one question is required") {
			t.Errorf("expected error message, got: %q", result)
		}
		if !strings.Contains(result, `"questions"`) {
			t.Errorf("expected error message to include JSON example, got: %q", result)
		}
	})

	t.Run("questionWithNilChoicesReturnsError", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{Question: "No choices?", Choices: nil},
			},
		}

		result, _ := askUserQuestion(workDir, args)

		if result != "Error: Question 1 has no choices" {
			t.Errorf("expected error message, got: %q", result)
		}
	})
}

func TestAskUserWorkGateStress(t *testing.T) {
	t.Run("setsIsWorkGateTrue", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:          state.StatusWorking,
			CurrentAgent:    "coordinator",
			InteractionMode: state.ModeBrainstorming,
		})

		_, err := askUserWorkGate(workDir, "test summary")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if !loadedState.MultiChoiceQuestion.IsWorkGate {
			t.Error("expected IsWorkGate to be true")
		}
	})

	t.Run("setsStatusToWaitingForHuman", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		_, err := askUserWorkGate(workDir, "test summary")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if loadedState.Status != state.StatusWaitingForHuman {
			t.Errorf("expected status %q, got %q", state.StatusWaitingForHuman, loadedState.Status)
		}
	})

	t.Run("preservesExistingMessages", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{
			Status:          state.StatusWorking,
			CurrentAgent:    "coordinator",
			InteractionMode: state.ModeBrainstorming,
			Messages: []state.Message{
				{ID: 1, FromAgent: "coordinator", ToAgent: "developer", Body: "do stuff"},
				{ID: 2, FromAgent: "developer", ToAgent: "coordinator", Body: "done"},
			},
		})

		_, err := askUserWorkGate(workDir, "test summary")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if len(loadedState.Messages) != 2 {
			t.Errorf("expected 2 messages preserved, got %d", len(loadedState.Messages))
		}
	})

	t.Run("setsHumanMessage", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		_, err := askUserWorkGate(workDir, "test summary")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if loadedState.HumanMessage == "" {
			t.Error("expected HumanMessage to be set")
		}
	})

	t.Run("hasTwoChoices", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		_, err := askUserWorkGate(workDir, "test summary")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		loadedState := loadState(t, workDir)

		if len(loadedState.MultiChoiceQuestion.Questions) != 1 {
			t.Fatalf("expected 1 question, got %d", len(loadedState.MultiChoiceQuestion.Questions))
		}

		if len(loadedState.MultiChoiceQuestion.Questions[0].Choices) != 2 {
			t.Errorf("expected 2 choices, got %d", len(loadedState.MultiChoiceQuestion.Questions[0].Choices))
		}
	})

	t.Run("emptySummaryReturnsError", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		result, err := askUserWorkGate(workDir, "")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		if !strings.Contains(result, "Error:") {
			t.Errorf("expected error message for empty summary, got: %q", result)
		}
	})

	t.Run("whitespaceSummaryReturnsError", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		result, err := askUserWorkGate(workDir, "   \t\n  ")
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		if !strings.Contains(result, "Error:") {
			t.Errorf("expected error message for whitespace-only summary, got: %q", result)
		}
	})

	t.Run("summaryAppearsInQuestionText", func(t *testing.T) {
		workDir := setupStateDir(t, state.Workflow{Status: state.StatusWorking, InteractionMode: state.ModeBrainstorming})

		summary := "## What Will Be Built\n- Feature X\n\n## Key Decisions\n- Use approach A"

		_, err := askUserWorkGate(workDir, summary)
		if err != nil {
			t.Fatalf("askUserWorkGate error: %v", err)
		}

		loadedState := loadState(t, workDir)

		questionText := loadedState.MultiChoiceQuestion.Questions[0].Question
		if !strings.Contains(questionText, summary) {
			t.Errorf("expected question to contain summary, got: %q", questionText)
		}

		if !strings.Contains(questionText, "Is the definition complete?") {
			t.Errorf("expected question to contain approval prompt, got: %q", questionText)
		}

		if !strings.Contains(loadedState.HumanMessage, summary) {
			t.Errorf("expected HumanMessage to contain summary, got: %q", loadedState.HumanMessage)
		}
	})
}

func TestMultiChoiceQuestionStateRoundTrip(t *testing.T) {
	workDir := setupStateDir(t, state.Workflow{
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

	readState := loadState(t, workDir)

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
	workDir := setupStateDir(t, state.Workflow{
		Status:       state.StatusWorking,
		CurrentAgent: "coordinator",
	})

	loadedState := loadState(t, workDir)

	if loadedState.MultiChoiceQuestion != nil {
		t.Error("multiChoiceQuestion should be nil when not set")
	}
}
