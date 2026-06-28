package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartSessionService(t *testing.T) {
	t.Skip("Integration test - requires full workflow execution")
	tests := []struct {
		name        string
		auto        bool
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, startSessionServiceResult)
	}{
		{
			name: "startSessionInBrainstormingMode",
			auto: false,
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, initializeWorkspace(workspacePath))
				goalContent := "---\nagents:\n  - agent1\n  - agent2\nmodel: openai/gpt-5.5\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, result startSessionServiceResult) {
				assert.Equal(t, "running", result.Status)
				assert.True(t, result.Running)
			},
		},
		{
			name: "startSessionInSelfDriveMode",
			auto: true,
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, initializeWorkspace(workspacePath))
				goalContent := "---\nagents:\n  - agent1\n  - agent2\nmodel: openai/gpt-5.5\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, result startSessionServiceResult) {
				assert.Equal(t, "running", result.Status)
				assert.True(t, result.Running)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server.shutdownCtx = ctx

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result, err := server.startSessionService(workspacePath, tt.auto)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}

			server.stopSessionService(workspacePath)
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestStartInteractionModeUsesRuntimeModes(t *testing.T) {
	tests := []struct {
		name             string
		auto             bool
		continuousPrompt string
		expected         string
	}{
		{name: "newInteractiveSessionStartsInteractive", expected: state.ModeInteractive},
		{name: "autoStartsSelfDrive", auto: true, expected: state.ModeSelfDrive},
		{name: "continuousStartsContinuous", continuousPrompt: "run forever", expected: state.ModeContinuous},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startInteractionMode(tt.auto, tt.continuousPrompt)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStopSessionService(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string, *Server)
		validate  func(*testing.T, stopSessionResult)
	}{
		{
			name: "stopRunningSession",
			setupFunc: func(t *testing.T, workspacePath string, _ *Server) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			validate: func(t *testing.T, result stopSessionResult) {
				assert.Equal(t, "stopped", result.Status)
				assert.False(t, result.Running)
				assert.Contains(t, result.Message, "session")
			},
		},
		{
			name: "stopAlreadyStoppedSession",
			setupFunc: func(t *testing.T, workspacePath string, _ *Server) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			validate: func(t *testing.T, result stopSessionResult) {
				assert.Equal(t, "stopped", result.Status)
				assert.False(t, result.Running)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath, server)

			result := server.stopSessionService(workspacePath)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestRespondServiceDeliversToAskUserQuestion(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-delivery")
	coord, errCoord := state.NewCoordinatorWith(statePath(wsDir), state.Workflow{
		InteractionMode: state.ModeInteractive,
	})
	require.NoError(t, errCoord)

	server.mu.Lock()
	server.sessions[wsDir] = &session{coord: coord, running: true}
	server.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		result, errAsk := askUserQuestion(ctx, coord, askUserQuestionArgs{
			Questions: []questionItem{{Question: "Which database?", Choices: []string{"PostgreSQL", "SQLite"}}},
		})
		if errAsk != nil {
			errCh <- errAsk
			return
		}
		resultCh <- result
	}()

	require.Eventually(t, func() bool {
		return coord.State().NeedsHumanInput()
	}, time.Second, 10*time.Millisecond)

	questionID := generateQuestionID(coord.State())
	result, errRespond := server.respondService(wsDir, questionID, "Use PostgreSQL", nil)
	require.NoError(t, errRespond)
	assert.True(t, result.Success)

	select {
	case errAsk := <-errCh:
		require.NoError(t, errAsk)
	case answer := <-resultCh:
		assert.Contains(t, answer, "Human response: Use PostgreSQL")
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
}

func TestStartSessionServiceValidation(t *testing.T) {
	t.Skip("Integration test - requires full workflow execution with MCP server and agent coordination")
	tests := []struct {
		name        string
		workspace   string
		auto        bool
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
	}{
		{
			name:      "startSessionOnStandaloneWorkspace",
			workspace: "standalone-workspace",
			auto:      false,
			setupFunc: func(t *testing.T, rootDir string) {
				workspacePath := filepath.Join(rootDir, "standalone-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server.shutdownCtx = ctx

			tt.setupFunc(t, rootDir)

			workspacePath := filepath.Join(rootDir, tt.workspace)
			result, err := server.startSessionService(workspacePath, tt.auto)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "running", result.Status)
			assert.True(t, result.Running)

			server.stopSessionService(workspacePath)
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestStopSessionServiceIdempotency(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	result1 := server.stopSessionService(workspacePath)
	assert.Equal(t, "stopped", result1.Status)
	assert.False(t, result1.Running)
	assert.Contains(t, result1.Message, "session already stopped")

	result2 := server.stopSessionService(workspacePath)
	assert.Equal(t, "stopped", result2.Status)
	assert.False(t, result2.Running)
	assert.Contains(t, result2.Message, "session already stopped")
}

func TestStartSessionServiceRootWorkspace(t *testing.T) {
	t.Skip("Integration test - requires real jj repository with multiple workspaces")
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server.shutdownCtx = ctx

	rootPath := filepath.Join(rootDir, "root-workspace")
	require.NoError(t, os.MkdirAll(rootPath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".sgai"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".jj", "repo"), 0755))

	_, err := server.startSessionService(rootPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "root workspace cannot start agentic work")
}

func TestStartSessionServiceStandaloneWorkspace(t *testing.T) {
	t.Skip("Integration test - requires MCP server and agent coordination")
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server.shutdownCtx = ctx

	workspacePath := filepath.Join(rootDir, "standalone-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, initializeWorkspace(workspacePath))

	goalContent := "---\nagents:\n  - agent1\n  - agent2\nmodel: openai/gpt-5.5\n---\n# Test Goal"
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))

	result, err := server.startSessionService(workspacePath, false)
	require.NoError(t, err)
	assert.Equal(t, "running", result.Status)
	assert.True(t, result.Running)

	server.stopSessionService(workspacePath)
	time.Sleep(100 * time.Millisecond)
}

func TestRespondViaCoordinatorServiceNoQuestion(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	coord := state.NewCoordinatorEmpty(statePath(workspacePath))
	req := apiRespondRequest{
		QuestionID: "test-question-1",
		Answer:     "Test answer",
	}

	_, err := server.respondViaCoordinatorService(coord, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending question")
}

func TestRespondViaCoordinatorServiceQuestionExpired(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	coord := state.NewCoordinatorEmpty(statePath(workspacePath))
	require.NoError(t, coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWaitingForHuman
		wf.HumanMessage = "What should I do?"
	}))

	req := apiRespondRequest{
		QuestionID: "wrong-question-id",
		Answer:     "Test answer",
	}

	_, err := server.respondViaCoordinatorService(coord, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "question expired")
}

func TestRespondViaCoordinatorServiceEmptyResponse(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	coord := state.NewCoordinatorEmpty(statePath(workspacePath))
	require.NoError(t, coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWaitingForHuman
		wf.HumanMessage = "What should I do?"
	}))

	currentID := generateQuestionID(coord.State())
	req := apiRespondRequest{
		QuestionID: currentID,
		Answer:     "",
	}

	_, err := server.respondViaCoordinatorService(coord, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response cannot be empty")
}

func TestRespondViaCoordinatorServiceWorkGateApproval(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	coord, errCoord := state.NewCoordinatorWith(statePath(workspacePath), state.Workflow{
		InteractionMode: state.ModeInteractive,
	})
	require.NoError(t, errCoord)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		result, errAsk := askUserWorkGate(ctx, coord, "summary")
		if errAsk != nil {
			errCh <- errAsk
			return
		}
		resultCh <- result
	}()

	require.Eventually(t, func() bool {
		return coord.State().NeedsHumanInput()
	}, time.Second, 10*time.Millisecond)

	currentID := generateQuestionID(coord.State())
	req := apiRespondRequest{
		QuestionID:      currentID,
		SelectedChoices: []string{workGateApprovalText},
	}

	result, err := server.respondViaCoordinatorService(coord, req)
	require.NoError(t, err)
	assert.True(t, result.Success)
	select {
	case errAsk := <-errCh:
		require.NoError(t, errAsk)
	case answer := <-resultCh:
		assert.Contains(t, answer, workGateApprovalText)
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}

	wfState := coord.State()
	assert.Equal(t, state.ModeSelfDrive, wfState.InteractionMode)
}

func TestHandleRespondViaCoordinatorNoQuestion(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-noq")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	body := `{"answer":"test","questionId":"q-1"}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-noq/respond", body)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRespondViaCoordinatorEmptyResponse(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-empty")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Pick an option",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		},
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	qid := generateQuestionID(coord.State())
	body := `{"answer":"","questionId":"` + qid + `"}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-empty/respond", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRespondViaCoordinatorWorkGateApproval(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-gate")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		InteractionMode: state.ModeInteractive,
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		result, errAsk := askUserWorkGate(ctx, coord, "summary")
		if errAsk != nil {
			errCh <- errAsk
			return
		}
		resultCh <- result
	}()

	require.Eventually(t, func() bool {
		return coord.State().NeedsHumanInput()
	}, time.Second, 10*time.Millisecond)

	qid := generateQuestionID(coord.State())
	body := `{"answer":"","questionId":"` + qid + `","selectedChoices":["` + workGateApprovalText + `"]}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-gate/respond", body)
	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case errAsk := <-errCh:
		require.NoError(t, errAsk)
	case answer := <-resultCh:
		assert.Contains(t, answer, workGateApprovalText)
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}

	updatedState := coord.State()
	assert.Equal(t, state.ModeSelfDrive, updatedState.InteractionMode)

	blockedResult, errQuestion := askUserQuestion(ctx, coord, askUserQuestionArgs{
		Questions: []questionItem{{Question: "How should I proceed?", Choices: []string{"Ask again"}}},
	})
	require.NoError(t, errQuestion)
	assert.Contains(t, blockedResult, "Questions are not allowed")
	assert.False(t, coord.State().NeedsHumanInput())
}

func TestStopSessionServiceRunningSession(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: true}
	server.mu.Unlock()
	result := server.stopSessionService(wsDir)
	assert.Equal(t, "session stopped", result.Message)
	assert.False(t, result.Running)
}

func TestStopSessionServiceNotRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	result := server.stopSessionService(wsDir)
	assert.False(t, result.Running)
}

func TestRespondServiceInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	_, err := server.respondService(wsDir, "test response", "", nil)
	assert.Error(t, err)
}
