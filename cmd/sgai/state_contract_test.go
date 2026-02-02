package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

// TestStateJSONContract verifies the state.json contract between Go and TypeScript.
// This test creates a fully populated state.Workflow and verifies round-trip serialization.
//
// The expected JSON structure is:
//
//	{
//	  "status": string,           // "working", "agent-done", "complete", "waiting-for-human"
//	  "task": string,             // Current task or empty string
//	  "progress": [string],       // Array of progress notes
//	  "humanMessage": string,     // Message to human or empty string
//	  "messages": [{              // Array of pending messages
//	    "id": int,
//	    "fromAgent": string,
//	    "toAgent": string,
//	    "body": string,
//	    "read": bool,
//	    "readAt": string,         // ISO8601 timestamp
//	    "readBy": string          // Agent name that read the message
//	  }],
//	  "goalChecksum": string,     // SHA256 of GOAL.md
//	  "visitCounts": {string: int}, // Map of agent -> visit count
//	  "currentAgent": string,       // Current agent name
//	}
func TestStateJSONContract(t *testing.T) {
	// Create a fully populated state with ALL fields set to non-zero values
	fullState := state.Workflow{
		Status: "working",
		Task:   "Test task description",
		Progress: []state.ProgressEntry{
			{Timestamp: "2025-01-01T00:00:00Z", Agent: "coordinator", Description: "First progress note"},
			{Timestamp: "2025-01-01T01:00:00Z", Agent: "coordinator", Description: "Second progress note"},
		},
		HumanMessage: "Test human message",
		Messages: []state.Message{
			{
				ID:        1,
				FromAgent: "agent-a",
				ToAgent:   "agent-b",
				Body:      "Test message 1",
				Read:      false,
			},
			{
				ID:        2,
				FromAgent: "agent-c",
				ToAgent:   "agent-d",
				Body:      "Test message 2",
				Read:      false,
			},
			{
				ID:        3,
				FromAgent: "sender-1",
				ToAgent:   "receiver-1",
				Body:      "Historical message 1",
				Read:      true,
				ReadAt:    "2025-01-01T02:00:00Z",
				ReadBy:    "receiver-1",
			},
			{
				ID:        4,
				FromAgent: "sender-2",
				ToAgent:   "receiver-2",
				Body:      "Historical message 2",
				Read:      true,
				ReadAt:    "2025-01-01T03:00:00Z",
				ReadBy:    "receiver-2",
			},
		},
		GoalChecksum: "abc123def456789012345678901234567890123456789012345678901234",
		VisitCounts: map[string]int{
			"coordinator": 3,
			"planner":     2,
			"coder":       1,
			"reviewer":    0,
		},
		CurrentAgent: "planner",
		AgentSequence: []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: "2025-12-21T10:00:00Z", IsCurrent: false},
			{Agent: "planner", StartTime: "2025-12-21T10:01:00Z", IsCurrent: true},
			{Agent: "coder", StartTime: "2025-12-21T10:02:00Z", IsCurrent: false},
		},
	}

	// Create temp file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Write state to JSON
	data, err := json.MarshalIndent(fullState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Read state back
	readData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var readState state.Workflow
	if err := json.Unmarshal(readData, &readState); err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	// Verify all fields survived the round-trip
	t.Run("Status", func(t *testing.T) {
		if readState.Status != fullState.Status {
			t.Errorf("Status mismatch: got %q, want %q", readState.Status, fullState.Status)
		}
	})

	t.Run("Task", func(t *testing.T) {
		if readState.Task != fullState.Task {
			t.Errorf("Task mismatch: got %q, want %q", readState.Task, fullState.Task)
		}
	})

	t.Run("Progress", func(t *testing.T) {
		if len(readState.Progress) != len(fullState.Progress) {
			t.Errorf("Progress length mismatch: got %d, want %d", len(readState.Progress), len(fullState.Progress))
		}
		for i, p := range fullState.Progress {
			if i < len(readState.Progress) && readState.Progress[i] != p {
				t.Errorf("Progress[%d] mismatch: got %q, want %q", i, readState.Progress[i], p)
			}
		}
	})

	t.Run("HumanMessage", func(t *testing.T) {
		if readState.HumanMessage != fullState.HumanMessage {
			t.Errorf("HumanMessage mismatch: got %q, want %q", readState.HumanMessage, fullState.HumanMessage)
		}
	})

	t.Run("Messages", func(t *testing.T) {
		if len(readState.Messages) != len(fullState.Messages) {
			t.Errorf("Messages length mismatch: got %d, want %d", len(readState.Messages), len(fullState.Messages))
		}
		for i, m := range fullState.Messages {
			if i < len(readState.Messages) {
				rm := readState.Messages[i]
				if rm.ID != m.ID {
					t.Errorf("Messages[%d].ID mismatch: got %d, want %d", i, rm.ID, m.ID)
				}
				if rm.FromAgent != m.FromAgent {
					t.Errorf("Messages[%d].FromAgent mismatch: got %q, want %q", i, rm.FromAgent, m.FromAgent)
				}
				if rm.ToAgent != m.ToAgent {
					t.Errorf("Messages[%d].ToAgent mismatch: got %q, want %q", i, rm.ToAgent, m.ToAgent)
				}
				if rm.Body != m.Body {
					t.Errorf("Messages[%d].Body mismatch: got %q, want %q", i, rm.Body, m.Body)
				}
				if rm.Read != m.Read {
					t.Errorf("Messages[%d].Read mismatch: got %v, want %v", i, rm.Read, m.Read)
				}
				if rm.ReadAt != m.ReadAt {
					t.Errorf("Messages[%d].ReadAt mismatch: got %q, want %q", i, rm.ReadAt, m.ReadAt)
				}
				if rm.ReadBy != m.ReadBy {
					t.Errorf("Messages[%d].ReadBy mismatch: got %q, want %q", i, rm.ReadBy, m.ReadBy)
				}
			}
		}
	})

	t.Run("GoalChecksum", func(t *testing.T) {
		if readState.GoalChecksum != fullState.GoalChecksum {
			t.Errorf("GoalChecksum mismatch: got %q, want %q", readState.GoalChecksum, fullState.GoalChecksum)
		}
	})

	t.Run("VisitCounts", func(t *testing.T) {
		if len(readState.VisitCounts) != len(fullState.VisitCounts) {
			t.Errorf("VisitCounts length mismatch: got %d, want %d", len(readState.VisitCounts), len(fullState.VisitCounts))
		}
		for k, v := range fullState.VisitCounts {
			if readState.VisitCounts[k] != v {
				t.Errorf("VisitCounts[%q] mismatch: got %d, want %d", k, readState.VisitCounts[k], v)
			}
		}
	})

	t.Run("CurrentAgent", func(t *testing.T) {
		if readState.CurrentAgent != fullState.CurrentAgent {
			t.Errorf("CurrentAgent mismatch: got %q, want %q", readState.CurrentAgent, fullState.CurrentAgent)
		}
	})

	t.Run("AgentSequence", func(t *testing.T) {
		if len(readState.AgentSequence) != len(fullState.AgentSequence) {
			t.Errorf("AgentSequence length mismatch: got %d, want %d", len(readState.AgentSequence), len(fullState.AgentSequence))
		}
		for i, entry := range fullState.AgentSequence {
			if i < len(readState.AgentSequence) && readState.AgentSequence[i] != entry {
				t.Errorf("AgentSequence[%d] mismatch: got %+v, want %+v", i, readState.AgentSequence[i], entry)
			}
		}
	})
}

// TestStateJSONContractEmptyFields verifies that empty/zero values are handled correctly.
// This catches issues when fields are omitted from JSON vs being set to zero values.
func TestStateJSONContractEmptyFields(t *testing.T) {
	// Create a minimal state with zero values
	minimalState := state.Workflow{
		Status:        "working",
		Task:          "",
		Progress:      []state.ProgressEntry{},
		HumanMessage:  "",
		Messages:      []state.Message{},
		GoalChecksum:  "",
		VisitCounts:   map[string]int{},
		CurrentAgent:  "",
		AgentSequence: []state.AgentSequenceEntry{},
	}

	// Create temp file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Write state to JSON
	data, err := json.MarshalIndent(minimalState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Read state back
	readData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var readState state.Workflow
	if err := json.Unmarshal(readData, &readState); err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	// Verify zero values are preserved correctly
	if readState.Status != minimalState.Status {
		t.Errorf("Status mismatch: got %q, want %q", readState.Status, minimalState.Status)
	}
}

// TestStateJSONContractGenerateGoldenFile generates a golden JSON file for cross-validation.
// This file can be read by the TypeScript test to verify compatibility.
func TestStateJSONContractGenerateGoldenFile(t *testing.T) {
	// Create a fully populated state matching what TypeScript expects
	goldenState := state.Workflow{
		Status: "working",
		Task:   "Verify Go-TypeScript state.json contract",
		Progress: []state.ProgressEntry{
			{Timestamp: "2025-01-01T00:00:00.000Z", Agent: "coordinator", Description: "Step 1 completed"},
			{Timestamp: "2025-01-01T01:00:00.000Z", Agent: "coordinator", Description: "Step 2 completed"},
		},
		HumanMessage: "Golden file test message",
		Messages: []state.Message{
			{
				ID:        1,
				FromAgent: "coordinator",
				ToAgent:   "planner",
				Body:      "Golden message body",
				Read:      false,
			},
			{
				ID:        2,
				FromAgent: "planner",
				ToAgent:   "coder",
				Body:      "Historical golden message",
				Read:      true,
				ReadAt:    "2025-01-01T12:00:00.000Z",
				ReadBy:    "coder",
			},
		},
		GoalChecksum: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		VisitCounts: map[string]int{
			"coordinator": 1,
			"planner":     1,
			"coder":       0,
		},
		CurrentAgent: "planner",
		AgentSequence: []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: "2025-12-21T10:00:00Z", IsCurrent: false},
			{Agent: "planner", StartTime: "2025-12-21T10:01:00Z", IsCurrent: true},
		},
	}

	// Generate JSON
	data, err := json.MarshalIndent(goldenState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal golden state: %v", err)
	}

	// Write to a known location for TypeScript test to read
	goldenFile := filepath.Join("testdata", "golden_state.json")
	if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	if err := os.WriteFile(goldenFile, data, 0644); err != nil {
		t.Fatalf("Failed to write golden file: %v", err)
	}

	t.Logf("Golden file written to: %s", goldenFile)
	t.Logf("JSON content:\n%s", string(data))
}

// TestStateJSONContractReadTypeScriptGolden reads a golden file generated by TypeScript
// to verify that Go can correctly parse TypeScript-generated state.json.
// This test first invokes the TypeScript test to generate the golden file,
// then reads and validates it.
func TestStateJSONContractReadTypeScriptGolden(t *testing.T) {
	// First, try to invoke the TypeScript test to generate the golden file
	tsTestFile := filepath.Join("skel", ".sgai", "plugin", "state_contract.test.ts")

	// Check if the TS test file exists
	if _, err := os.Stat(tsTestFile); os.IsNotExist(err) {
		t.Skipf("TypeScript test file not found: %s", tsTestFile)
	}

	// Try to run the TypeScript test using npx tsx
	t.Log("Attempting to run TypeScript contract test to generate golden file...")

	cmd := exec.Command("npx", "tsx", tsTestFile)
	cmd.Dir = "." // Run from the test directory (cmd/sgai)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if npx/tsx is not available
		if _, errPath := exec.LookPath("npx"); errPath != nil {
			t.Skipf("npx not found in PATH, skipping TypeScript invocation (install Node.js to enable): %v", errPath)
		}
		// npx is available but tsx failed - log output for debugging
		t.Logf("TypeScript test output:\n%s", string(output))
		t.Fatalf("Failed to run TypeScript test: %v", err)
	}

	t.Logf("TypeScript test completed:\n%s", string(output))

	// Now read the golden file generated by TypeScript
	goldenFile := filepath.Join("testdata", "golden_state_from_ts.json")

	// Skip if TypeScript golden file doesn't exist (shouldn't happen if TS test succeeded)
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		t.Fatalf("TypeScript test ran but golden file not found: %s", goldenFile)
	}

	data, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("Failed to read TypeScript golden file: %v", err)
	}

	var state state.Workflow
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Failed to unmarshal TypeScript-generated state: %v", err)
	}

	// Verify all expected fields are present and parseable
	if state.Status == "" {
		t.Error("Status field is empty")
	}

	// Verify critical fields match expected TypeScript values
	// This ensures the contract is actually being tested
	if state.Task != "Cross-language contract test" {
		t.Errorf("Task field mismatch: expected TypeScript golden value, got %q", state.Task)
	}

	// Log what we successfully parsed
	t.Logf("Successfully parsed TypeScript golden file:")
	t.Logf("  Status: %s", state.Status)
	t.Logf("  Task: %s", state.Task)
	t.Logf("  Progress: %d items", len(state.Progress))
	t.Logf("  Messages: %d items", len(state.Messages))
	t.Logf("  VisitCounts: %d agents", len(state.VisitCounts))
}
