package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestMessageCreatedAtTimestamp(t *testing.T) {
	tmpDir := setupStateDir(t, state.Workflow{
		Status:       "working",
		CurrentAgent: "coordinator",
		Messages:     []state.Message{},
	})
	statePath := filepath.Join(tmpDir, ".sgai", "state.json")
	coord, err := state.NewCoordinator(statePath)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	dagAgents := []string{"coordinator", "backend-go-developer"}

	beforeSend := time.Now().UTC()

	result, err := sendMessage(coord, dagAgents, "coordinator", "backend-go-developer", "Test message body")
	if err != nil {
		t.Fatalf("sendMessage failed: %v", err)
	}

	if result == "" {
		t.Fatal("sendMessage returned empty result")
	}

	s := coord.State()

	if len(s.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(s.Messages))
	}

	msg := s.Messages[0]

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
