package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkflow_UnmarshalJSON_NewFormat(t *testing.T) {
	input := `{
		"status": "working",
		"agentSequence": [
			{"agent": "coordinator", "startTime": "2025-12-21T18:26:00Z", "isCurrent": false},
			{"agent": "backend-go-developer", "startTime": "2025-12-21T18:27:00Z", "isCurrent": true}
		]
	}`

	var workflow Workflow
	if err := json.Unmarshal([]byte(input), &workflow); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if workflow.Status != "working" {
		t.Errorf("Status = %q; want %q", workflow.Status, "working")
	}

	want := []AgentSequenceEntry{
		{Agent: "coordinator", StartTime: "2025-12-21T18:26:00Z", IsCurrent: false},
		{Agent: "backend-go-developer", StartTime: "2025-12-21T18:27:00Z", IsCurrent: true},
	}

	if len(workflow.AgentSequence) != len(want) {
		t.Fatalf("len(AgentSequence) = %d; want %d", len(workflow.AgentSequence), len(want))
	}

	for i, got := range workflow.AgentSequence {
		if got != want[i] {
			t.Errorf("AgentSequence[%d] = %+v; want %+v", i, got, want[i])
		}
	}
}

func TestWorkflow_UnmarshalJSON_EmptySequence(t *testing.T) {
	input := `{
		"status": "working"
	}`

	var workflow Workflow
	if err := json.Unmarshal([]byte(input), &workflow); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if workflow.Status != "working" {
		t.Errorf("Status = %q; want %q", workflow.Status, "working")
	}

	if workflow.AgentSequence != nil {
		t.Errorf("AgentSequence = %+v; want nil", workflow.AgentSequence)
	}
}

func TestWorkflow_RoundTrip(t *testing.T) {
	original := Workflow{
		Status: "working",
		Task:   "test task",
		AgentSequence: []AgentSequenceEntry{
			{Agent: "coordinator", StartTime: "2025-12-21T18:26:00Z", IsCurrent: false},
			{Agent: "backend-go-developer", StartTime: "2025-12-21T18:27:00Z", IsCurrent: true},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Workflow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Status != original.Status {
		t.Errorf("Status = %q; want %q", decoded.Status, original.Status)
	}

	if decoded.Task != original.Task {
		t.Errorf("Task = %q; want %q", decoded.Task, original.Task)
	}

	if len(decoded.AgentSequence) != len(original.AgentSequence) {
		t.Fatalf("len(AgentSequence) = %d; want %d", len(decoded.AgentSequence), len(original.AgentSequence))
	}

	for i, got := range decoded.AgentSequence {
		if got != original.AgentSequence[i] {
			t.Errorf("AgentSequence[%d] = %+v; want %+v", i, got, original.AgentSequence[i])
		}
	}
}

func TestSave_CreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, ".sgai", "state.json")

	workflow := Workflow{
		Status: "working",
		Task:   "test task",
	}

	if err := Save(nestedPath, workflow); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load(nestedPath)
	if err != nil {
		t.Fatalf("Load() after Save() failed: %v", err)
	}

	if loaded.Status != workflow.Status {
		t.Errorf("Status = %q; want %q", loaded.Status, workflow.Status)
	}

	if loaded.Task != workflow.Task {
		t.Errorf("Task = %q; want %q", loaded.Task, workflow.Task)
	}
}

func TestIsHumanPending(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"waitingForHuman", StatusWaitingForHuman, true},
		{"working", StatusWorking, false},
		{"agentDone", StatusAgentDone, false},
		{"complete", StatusComplete, false},
		{"empty", "", false},
		{"whitespace", " ", false},
		{"tab", "\t", false},
		{"similarWithPlural", "waiting-for-humans", false},
		{"uppercaseVariant", "WAITING-FOR-HUMAN", false},
		{"trailingSpace", "waiting-for-human ", false},
		{"leadingSpace", " waiting-for-human", false},
		{"arbitraryString", "some-random-status", false},
		{"removedHumanCommunication", "human-communication", false},
		{"removedAutoSession", "auto-session", false},
		{"partialMatch", "waiting", false},
		{"partialMatchDash", "waiting-for", false},
		{"newline", "waiting-for-human\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHumanPending(tt.status); got != tt.want {
				t.Errorf("IsHumanPending(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestNeedsHumanInput(t *testing.T) {
	t.Run("waitingWithMessage", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman, HumanMessage: "please respond"}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true")
		}
	})

	t.Run("waitingWithMultiChoice", func(t *testing.T) {
		w := Workflow{
			Status: StatusWaitingForHuman,
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "pick one", Choices: []string{"a", "b"}}},
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true")
		}
	})

	t.Run("waitingWithBoth", func(t *testing.T) {
		w := Workflow{
			Status:       StatusWaitingForHuman,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "pick one", Choices: []string{"a", "b"}}},
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true")
		}
	})

	t.Run("waitingWithoutContent", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false")
		}
	})

	t.Run("workingWithMessage", func(t *testing.T) {
		w := Workflow{Status: StatusWorking, HumanMessage: "please respond"}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false")
		}
	})

	t.Run("emptyWorkflow", func(t *testing.T) {
		w := Workflow{}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false")
		}
	})

	t.Run("workingWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       StatusWorking,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is working")
		}
	})

	t.Run("agentDoneWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       StatusAgentDone,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is agent-done")
		}
	})

	t.Run("completeWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       StatusComplete,
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is complete")
		}
	})

	t.Run("emptyStatusWithBothFieldsSet", func(t *testing.T) {
		w := Workflow{
			Status:       "",
			HumanMessage: "choose",
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false when status is empty")
		}
	})

	t.Run("waitingWithMultiChoiceOnly", func(t *testing.T) {
		w := Workflow{
			Status: StatusWaitingForHuman,
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions: []QuestionItem{{Question: "q", Choices: []string{"a"}}},
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true with only MultiChoiceQuestion")
		}
	})

	t.Run("waitingWithMessageOnly", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman, HumanMessage: "question"}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true with only HumanMessage")
		}
	})

	t.Run("waitingWithEmptyMessageAndNilQuestion", func(t *testing.T) {
		w := Workflow{Status: StatusWaitingForHuman, HumanMessage: ""}
		if w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = false with empty message and nil question")
		}
	})

	t.Run("waitingWithWorkGateQuestion", func(t *testing.T) {
		w := Workflow{
			Status: StatusWaitingForHuman,
			MultiChoiceQuestion: &MultiChoiceQuestion{
				Questions:  []QuestionItem{{Question: "q", Choices: []string{"yes", "no"}}},
				IsWorkGate: true,
			},
		}
		if !w.NeedsHumanInput() {
			t.Error("expected NeedsHumanInput() = true for work gate question")
		}
	})
}

func TestSave_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sgaiDir := filepath.Join(tmpDir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	statePath := filepath.Join(sgaiDir, "state.json")

	workflow := Workflow{Status: "complete"}

	if err := Save(statePath, workflow); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := Load(statePath)
	if err != nil {
		t.Fatalf("Load() after Save() failed: %v", err)
	}

	if loaded.Status != "complete" {
		t.Errorf("Status = %q; want %q", loaded.Status, "complete")
	}
}

func TestProgressEntry_UnmarshalJSON_NewFormat(t *testing.T) {
	input := `{"timestamp":"2026-01-01T10:43:36-08:00","agent":"coordinator","description":"Started assessing GOAL.md"}`

	var entry ProgressEntry
	if err := json.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if entry.Timestamp != "2026-01-01T10:43:36-08:00" {
		t.Errorf("Timestamp = %q; want %q", entry.Timestamp, "2026-01-01T10:43:36-08:00")
	}

	if entry.Agent != "coordinator" {
		t.Errorf("Agent = %q; want %q", entry.Agent, "coordinator")
	}

	if entry.Description != "Started assessing GOAL.md" {
		t.Errorf("Description = %q; want %q", entry.Description, "Started assessing GOAL.md")
	}
}
