package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestBuildMCPServerUsesCoordinatorCurrentAgentAsFallback(t *testing.T) {
	t.Run("noHeaderFallsBackToCoordinatorCurrentAgent", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, ".sgai", "state.json")

		coord := state.NewCoordinatorEmpty(statePath)
		if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
			wf.Status = state.StatusWorking
			wf.CurrentAgent = "backend-go-developer"
		}); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		server := buildMCPServer(tmpDir, httptest.NewRequest(http.MethodPost, "/mcp", nil), coord, nil)
		if server == nil {
			t.Fatal("expected non-nil server")
		}

		result, err := updateWorkflowState(context.Background(), coord, resolveCallerAgent("coordinator", coord), updateWorkflowStateArgs{
			Status:      "working",
			Task:        "doing backend work",
			AddProgress: "wrote some code",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == "" {
			t.Fatal("expected non-empty result")
		}

		s := coord.State()
		if len(s.Progress) != 1 {
			t.Fatalf("expected 1 progress entry, got %d", len(s.Progress))
		}
		if s.Progress[0].Agent != "backend-go-developer" {
			t.Errorf("expected progress agent %q (from coordinator state), got %q", "backend-go-developer", s.Progress[0].Agent)
		}
	})

	t.Run("headerTakesPrecedenceOverCoordinatorState", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, ".sgai", "state.json")

		coord := state.NewCoordinatorEmpty(statePath)
		if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
			wf.Status = state.StatusWorking
			wf.CurrentAgent = "backend-go-developer"
		}); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("X-Sgai-Agent-Identity", "go-readability-reviewer")

		server := buildMCPServer(tmpDir, req, coord, nil)
		if server == nil {
			t.Fatal("expected non-nil server")
		}

		agentName := parseAgentIdentityHeader(req)

		result, err := updateWorkflowState(context.Background(), coord, agentName, updateWorkflowStateArgs{
			Status:      "working",
			AddProgress: "reviewed code",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == "" {
			t.Fatal("expected non-empty result")
		}

		s := coord.State()
		if len(s.Progress) != 1 {
			t.Fatalf("expected 1 progress entry, got %d", len(s.Progress))
		}
		if s.Progress[0].Agent != "go-readability-reviewer" {
			t.Errorf("expected progress agent %q (from header), got %q", "go-readability-reviewer", s.Progress[0].Agent)
		}
	})

	t.Run("coordinatorAgentStaysCoordinatorWhenCurrentAgentIsCoordinator", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, ".sgai", "state.json")

		coord := state.NewCoordinatorEmpty(statePath)
		if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
			wf.Status = state.StatusWorking
			wf.CurrentAgent = "coordinator"
		}); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		agentName := resolveCallerAgent("coordinator", coord)
		if agentName != "coordinator" {
			t.Errorf("expected %q when current agent is coordinator, got %q", "coordinator", agentName)
		}
	})
}

func TestMCPHTTPServerAgentIdentityFallback(t *testing.T) {
	t.Run("mcpRequestWithoutHeaderUsesCoordinatorCurrentAgent", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, ".sgai", "state.json")

		coord := state.NewCoordinatorEmpty(statePath)
		if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
			wf.Status = state.StatusWorking
			wf.CurrentAgent = "react-developer"
		}); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		mcpHandler := buildMCPHTTPHandler(tmpDir, coord, nil)

		body, errMarshal := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "sgai_update_workflow_state",
				"arguments": map[string]any{
					"status":      "working",
					"task":        "building UI",
					"addProgress": "implemented component",
				},
			},
		})
		if errMarshal != nil {
			t.Fatal(errMarshal)
		}

		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mcpHandler.ServeHTTP(w, req)

		s := coord.State()
		if len(s.Progress) > 0 && s.Progress[len(s.Progress)-1].Agent != "react-developer" {
			t.Errorf("expected progress agent %q (from coordinator state), got %q",
				"react-developer", s.Progress[len(s.Progress)-1].Agent)
		}
	})
}
