package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestAskUserQuestion(t *testing.T) {
	t.Run("singleQuestionSetsStateCorrectly", func(t *testing.T) {
		tmpDir := t.TempDir()
		sgai := filepath.Join(tmpDir, ".sgai")
		if err := os.MkdirAll(sgai, 0755); err != nil {
			t.Fatal(err)
		}

		initialState := state.Workflow{
			Status:       state.StatusWorking,
			CurrentAgent: "coordinator",
		}
		statePath := filepath.Join(sgai, "state.json")
		data, _ := json.Marshal(initialState)
		if err := os.WriteFile(statePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{
					Question:    "Which option do you prefer?",
					Choices:     []string{"Option A", "Option B", "Option C"},
					MultiSelect: true,
				},
			},
		}

		result, err := askUserQuestion(tmpDir, args)
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		if result == "" {
			t.Error("expected non-empty result")
		}

		loadedState, err := state.Load(statePath)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

		if loadedState.Status != state.StatusHumanCommunication {
			t.Errorf("expected status %q, got %q", state.StatusHumanCommunication, loadedState.Status)
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
		tmpDir := t.TempDir()
		sgai := filepath.Join(tmpDir, ".sgai")
		if err := os.MkdirAll(sgai, 0755); err != nil {
			t.Fatal(err)
		}

		statePath := filepath.Join(sgai, "state.json")
		initialState := state.Workflow{Status: state.StatusWorking}
		data, _ := json.Marshal(initialState)
		if err := os.WriteFile(statePath, data, 0644); err != nil {
			t.Fatal(err)
		}

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

		result, err := askUserQuestion(tmpDir, args)
		if err != nil {
			t.Fatalf("askUserQuestion error: %v", err)
		}

		if !strings.Contains(result, "2 question(s)") {
			t.Errorf("expected result to mention 2 questions, got: %s", result)
		}

		loadedState, err := state.Load(statePath)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}

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
		tmpDir := t.TempDir()
		sgai := filepath.Join(tmpDir, ".sgai")
		if err := os.MkdirAll(sgai, 0755); err != nil {
			t.Fatal(err)
		}

		statePath := filepath.Join(sgai, "state.json")
		initialState := state.Workflow{Status: state.StatusWorking}
		data, _ := json.Marshal(initialState)
		if err := os.WriteFile(statePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		args := askUserQuestionArgs{
			Questions: []questionItem{},
		}

		result, _ := askUserQuestion(tmpDir, args)

		if result != "Error: At least one question is required" {
			t.Errorf("expected error message, got: %q", result)
		}
	})

	t.Run("questionWithEmptyChoicesReturnsError", func(t *testing.T) {
		tmpDir := t.TempDir()
		sgai := filepath.Join(tmpDir, ".sgai")
		if err := os.MkdirAll(sgai, 0755); err != nil {
			t.Fatal(err)
		}

		statePath := filepath.Join(sgai, "state.json")
		initialState := state.Workflow{Status: state.StatusWorking}
		data, _ := json.Marshal(initialState)
		if err := os.WriteFile(statePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		args := askUserQuestionArgs{
			Questions: []questionItem{
				{
					Question:    "Question with no choices",
					Choices:     []string{},
					MultiSelect: false,
				},
			},
		}

		result, _ := askUserQuestion(tmpDir, args)

		if result != "Error: Question 1 has no choices" {
			t.Errorf("expected error message, got: %q", result)
		}
	})
}

func TestParseChoiceSelection(t *testing.T) {
	choices := []string{"Option A", "Option B", "Option C", "Option D"}

	t.Run("singleSelectValid", func(t *testing.T) {
		selected, err := parseChoiceSelection("1", choices, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 1 || selected[0] != "Option A" {
			t.Errorf("expected [Option A], got %v", selected)
		}
	})

	t.Run("singleSelectMiddle", func(t *testing.T) {
		selected, err := parseChoiceSelection("3", choices, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 1 || selected[0] != "Option C" {
			t.Errorf("expected [Option C], got %v", selected)
		}
	})

	t.Run("multiSelectMultiple", func(t *testing.T) {
		selected, err := parseChoiceSelection("1,3", choices, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 2 {
			t.Errorf("expected 2 selections, got %d", len(selected))
		}
		if selected[0] != "Option A" || selected[1] != "Option C" {
			t.Errorf("expected [Option A, Option C], got %v", selected)
		}
	})

	t.Run("multiSelectWithSpaces", func(t *testing.T) {
		selected, err := parseChoiceSelection("1, 2, 4", choices, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 3 {
			t.Errorf("expected 3 selections, got %d", len(selected))
		}
	})

	t.Run("singleSelectRejectsMultiple", func(t *testing.T) {
		_, err := parseChoiceSelection("1,2", choices, false)
		if err == nil {
			t.Error("expected error for multiple selections in single-select mode")
		}
	})

	t.Run("otherOption", func(t *testing.T) {
		selected, err := parseChoiceSelection("O", choices, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 0 {
			t.Errorf("expected empty selection for Other, got %v", selected)
		}
	})

	t.Run("otherOptionLowercase", func(t *testing.T) {
		selected, err := parseChoiceSelection("o", choices, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 0 {
			t.Errorf("expected empty selection for Other, got %v", selected)
		}
	})

	t.Run("invalidNumber", func(t *testing.T) {
		_, err := parseChoiceSelection("5", choices, false)
		if err == nil {
			t.Error("expected error for out-of-range selection")
		}
	})

	t.Run("zeroNumber", func(t *testing.T) {
		_, err := parseChoiceSelection("0", choices, false)
		if err == nil {
			t.Error("expected error for zero selection")
		}
	})

	t.Run("invalidInput", func(t *testing.T) {
		_, err := parseChoiceSelection("abc", choices, false)
		if err == nil {
			t.Error("expected error for invalid input")
		}
	})
}

func TestFormatMultiChoiceResponse(t *testing.T) {
	t.Run("selectionsOnly", func(t *testing.T) {
		result := formatMultiChoiceResponse([]string{"Option A", "Option B"}, "")
		expected := "Selected: Option A, Option B"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("otherOnly", func(t *testing.T) {
		result := formatMultiChoiceResponse(nil, "Custom input here")
		expected := "Other: Custom input here"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("selectionsAndOther", func(t *testing.T) {
		result := formatMultiChoiceResponse([]string{"Option A"}, "Additional notes")
		expected := "Selected: Option A\nOther: Additional notes"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("emptySelectionsNoOther", func(t *testing.T) {
		result := formatMultiChoiceResponse([]string{}, "")
		expected := ""
		if result != expected {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestMultiChoiceQuestionStateRoundTrip(t *testing.T) {
	fullState := state.Workflow{
		Status:       state.StatusHumanCommunication,
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
	}

	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	data, err := json.MarshalIndent(fullState, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	readData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("failed to read state file: %v", err)
	}

	var readState state.Workflow
	if err := json.Unmarshal(readData, &readState); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}

	if readState.MultiChoiceQuestion == nil {
		t.Fatal("expected MultiChoiceQuestion to be preserved")
	}

	if len(readState.MultiChoiceQuestion.Questions) != len(fullState.MultiChoiceQuestion.Questions) {
		t.Errorf("questions length mismatch: got %d, want %d",
			len(readState.MultiChoiceQuestion.Questions), len(fullState.MultiChoiceQuestion.Questions))
	}

	readQ := readState.MultiChoiceQuestion.Questions[0]
	wantQ := fullState.MultiChoiceQuestion.Questions[0]

	if readQ.Question != wantQ.Question {
		t.Errorf("question mismatch: got %q, want %q", readQ.Question, wantQ.Question)
	}

	if len(readQ.Choices) != len(wantQ.Choices) {
		t.Errorf("choices length mismatch: got %d, want %d", len(readQ.Choices), len(wantQ.Choices))
	}

	if readQ.MultiSelect != wantQ.MultiSelect {
		t.Errorf("multiSelect mismatch: got %v, want %v", readQ.MultiSelect, wantQ.MultiSelect)
	}
}

func TestMultiChoiceOmitsWhenNil(t *testing.T) {
	stateWithoutMCQ := state.Workflow{
		Status:       state.StatusWorking,
		CurrentAgent: "coordinator",
	}

	data, err := json.MarshalIndent(stateWithoutMCQ, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "multiChoiceQuestion") {
		t.Errorf("multiChoiceQuestion should be omitted from JSON when nil:\n%s", jsonStr)
	}
}
