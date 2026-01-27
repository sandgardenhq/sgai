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
	nestedPath := filepath.Join(tmpDir, ".factorai", "state.json")

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

func TestSave_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	factoraiDir := filepath.Join(tmpDir, ".factorai")
	if err := os.MkdirAll(factoraiDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	statePath := filepath.Join(factoraiDir, "state.json")

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
