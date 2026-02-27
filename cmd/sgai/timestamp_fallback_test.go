package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestTimestampFallbackLogic(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, ".sgai", "state.json")

	if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
		t.Fatal(err)
	}

	testTime := "2025-12-14T18:00:00Z"

	testState := state.Workflow{
		Messages: []state.Message{
			{
				ID:        1,
				FromAgent: "coordinator",
				ToAgent:   "backend-go-developer",
				Body:      "New message with CreatedAt",
				CreatedAt: testTime,
			},
			{
				ID:        2,
				FromAgent: "coordinator",
				ToAgent:   "backend-go-developer",
				Body:      "Old read message with only ReadAt",
				ReadAt:    testTime,
			},
			{
				ID:        3,
				FromAgent: "coordinator",
				ToAgent:   "backend-go-developer",
				Body:      "Old unread message with no timestamps",
			},
		},
	}

	data, err := json.MarshalIndent(testState, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	loadedCoord, err := state.NewCoordinator(stateFile)
	if err != nil {
		t.Fatalf("NewCoordinator failed: %v", err)
	}
	wfState := loadedCoord.State()

	if len(wfState.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(wfState.Messages))
	}

	for i, msg := range wfState.Messages {
		createdAt := msg.CreatedAt
		if createdAt == "" {
			createdAt = msg.ReadAt
		}
		if createdAt == "" {
			createdAt = "1970-01-01T00:00:00Z"
		}

		parsedTime, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			t.Errorf("Message %d: failed to parse timestamp %q: %v", i+1, createdAt, err)
			continue
		}

		switch i {
		case 0:
			if parsedTime.Year() != 2025 {
				t.Errorf("Message 1 (CreatedAt): expected year 2025, got %d", parsedTime.Year())
			}
		case 1:
			if parsedTime.Year() != 2025 {
				t.Errorf("Message 2 (ReadAt fallback): expected year 2025, got %d", parsedTime.Year())
			}
		case 2:
			if parsedTime.Year() != 1970 {
				t.Errorf("Message 3 (no timestamps): expected year 1970, got %d", parsedTime.Year())
			}
		}
	}
}
