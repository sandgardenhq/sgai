package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestMessageCreatedAtTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	sgaiDir := filepath.Join(tmpDir, ".sgai")
	if err := os.MkdirAll(sgaiDir, 0755); err != nil {
		t.Fatalf("Failed to create .sgai directory: %v", err)
	}

	statePath := filepath.Join(sgaiDir, "state.json")

	initialState := state.Workflow{
		Status:       "working",
		CurrentAgent: "coordinator",
		Messages:     []state.Message{},
		VisitCounts: map[string]int{
			"coordinator":          1,
			"backend-go-developer": 0,
		},
	}

	if err := state.Save(statePath, initialState); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	beforeSend := time.Now().UTC()

	result, err := sendMessage(tmpDir, "backend-go-developer", "Test message body")
	if err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	if result == "" {
		t.Fatal("sendMessage returned empty result")
	}

	loadedState, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("Failed to load state after sendMessage: %v", err)
	}

	if len(loadedState.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(loadedState.Messages))
	}

	msg := loadedState.Messages[0]

	if msg.CreatedAt == "" {
		t.Fatal("Message CreatedAt field is empty - this is the bug!")
	}

	parsedTime, err := time.Parse(time.RFC3339, msg.CreatedAt)
	if err != nil {
		t.Fatalf("Failed to parse CreatedAt timestamp '%s': %v", msg.CreatedAt, err)
	}

	timeDiff := parsedTime.Sub(beforeSend)
	if timeDiff < -time.Second || timeDiff > time.Second {
		t.Errorf("CreatedAt timestamp %v differs from expected time by %v (should be within 1 second)",
			parsedTime, timeDiff)
	}

	if msg.FromAgent != "coordinator" {
		t.Errorf("Expected FromAgent=coordinator, got %s", msg.FromAgent)
	}

	if msg.ToAgent != "backend-go-developer" {
		t.Errorf("Expected ToAgent=backend-go-developer, got %s", msg.ToAgent)
	}

	if msg.Body != "Test message body" {
		t.Errorf("Expected Body='Test message body', got %s", msg.Body)
	}

	if msg.Read {
		t.Error("Expected Read=false, got true")
	}
}
