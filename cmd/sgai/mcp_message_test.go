package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func setupCoordinatorWithMessages(t *testing.T, messages []state.Message) *state.Coordinator {
	t.Helper()
	tmpDir := setupStateDir(t, state.Workflow{
		Status:       state.StatusWorking,
		CurrentAgent: "coordinator",
		Messages:     messages,
	})
	statePath := filepath.Join(tmpDir, ".sgai", "state.json")
	coord, err := state.NewCoordinator(statePath)
	if err != nil {
		t.Fatalf("failed to create coordinator: %v", err)
	}
	return coord
}

func TestCheckInboxUsesCa11erAgent(t *testing.T) {
	t.Run("marksMessagesForCallerAgentAsRead", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "backend-go-developer", Body: "do the work", Read: false},
			{ID: 2, FromAgent: "coordinator", ToAgent: "react-developer", Body: "build the UI", Read: false},
		})

		result, err := checkInbox(coord, "backend-go-developer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "do the work") {
			t.Errorf("expected message body in result, got: %s", result)
		}

		s := coord.State()
		for _, msg := range s.Messages {
			if msg.ToAgent == "backend-go-developer" && !msg.Read {
				t.Errorf("expected message to backend-go-developer to be marked read")
			}
			if msg.ToAgent == "react-developer" && msg.Read {
				t.Errorf("expected message to react-developer to remain unread")
			}
		}
	})

	t.Run("doesNotMarkMessagesForOtherAgents", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "react-developer", Body: "build UI", Read: false},
		})

		result, err := checkInbox(coord, "backend-go-developer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "You have no messages.") {
			t.Errorf("expected no messages for backend-go-developer, got: %s", result)
		}

		s := coord.State()
		if s.Messages[0].Read {
			t.Errorf("expected message to react-developer to remain unread")
		}
	})

	t.Run("setsReadByToCallerAgent", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "go-readability-reviewer", Body: "review code", Read: false},
		})

		_, err := checkInbox(coord, "go-readability-reviewer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if s.Messages[0].ReadBy != "go-readability-reviewer" {
			t.Errorf("expected ReadBy=go-readability-reviewer, got %q", s.Messages[0].ReadBy)
		}
	})
}

func TestCheckOutboxUsesCa11erAgent(t *testing.T) {
	t.Run("showsMessagesFromCallerAgent", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, []state.Message{
			{ID: 1, FromAgent: "backend-go-developer", ToAgent: "go-readability-reviewer", Body: "please review", Read: false},
			{ID: 2, FromAgent: "react-developer", ToAgent: "go-readability-reviewer", Body: "also review this", Read: false},
		})

		result, err := checkOutbox(coord, "backend-go-developer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "please review") {
			t.Errorf("expected backend message in outbox, got: %s", result)
		}
		if strings.Contains(result, "also review this") {
			t.Errorf("expected react-developer message NOT in backend outbox, got: %s", result)
		}
	})

	t.Run("returnsNoMessagesWhenNoneSent", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "backend-go-developer", Body: "do work", Read: false},
		})

		result, err := checkOutbox(coord, "backend-go-developer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "You have not sent any messages.") {
			t.Errorf("expected no outbox messages, got: %s", result)
		}
	})
}

func TestSendMessageUsesCallerAgent(t *testing.T) {
	t.Run("setsFromAgentToCallerAgent", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, nil)
		dagAgents := []string{"coordinator", "backend-go-developer", "go-readability-reviewer"}

		_, err := sendMessage(coord, dagAgents, "backend-go-developer", "go-readability-reviewer", "please review my code")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		s := coord.State()
		if len(s.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(s.Messages))
		}
		if s.Messages[0].FromAgent != "backend-go-developer" {
			t.Errorf("expected FromAgent=backend-go-developer, got %q", s.Messages[0].FromAgent)
		}
	})

	t.Run("addsSelfDriveReminderForNonCoordinatorCaller", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, nil)
		dagAgents := []string{"coordinator", "backend-go-developer", "go-readability-reviewer"}

		result, err := sendMessage(coord, dagAgents, "backend-go-developer", "go-readability-reviewer", "review please")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "IMPORTANT: To receive a response") {
			t.Errorf("expected self-drive reminder for non-coordinator, got: %s", result)
		}
	})

	t.Run("noSelfDriveReminderForCoordinator", func(t *testing.T) {
		coord := setupCoordinatorWithMessages(t, nil)
		dagAgents := []string{"coordinator", "backend-go-developer", "go-readability-reviewer"}

		result, err := sendMessage(coord, dagAgents, "coordinator", "backend-go-developer", "do work")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "IMPORTANT: To receive a response") {
			t.Errorf("expected no self-drive reminder for coordinator, got: %s", result)
		}
	})
}
