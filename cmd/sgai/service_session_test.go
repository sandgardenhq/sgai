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
		validate    func(*testing.T, startSessionResult2)
	}{
		{
			name: "startSessionInBrainstormingMode",
			auto: false,
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, initializeWorkspace(workspacePath))
				goalContent := "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, result startSessionResult2) {
				assert.Equal(t, "running", result.Status)
				assert.True(t, result.Running)
			},
		},
		{
			name: "startSessionInSelfDriveMode",
			auto: true,
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, initializeWorkspace(workspacePath))
				goalContent := "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, result startSessionResult2) {
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

func TestRespondService(t *testing.T) {
	tests := []struct {
		name            string
		questionID      string
		answer          string
		selectedChoices []string
		setupFunc       func(*testing.T, string)
		wantErr         bool
		errContains     string
		validate        func(*testing.T, respondResult)
	}{
		{
			name:            "respondToQuestion",
			questionID:      "test-question-1",
			answer:          "Test answer",
			selectedChoices: []string{},
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "no pending question",
		},
		{
			name:            "respondWithEmptyAnswer",
			questionID:      "test-question-1",
			answer:          "",
			selectedChoices: []string{},
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "no pending question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result, err := server.respondService(workspacePath, tt.questionID, tt.answer, tt.selectedChoices)

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
		})
	}
}

func TestSteerService(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, steerResult)
	}{
		{
			name:    "steerWithValidMessage",
			message: "Please focus on the database implementation",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr: false,
			validate: func(t *testing.T, result steerResult) {
				assert.True(t, result.Success)
				assert.Equal(t, "steering instruction added", result.Message)
			},
		},
		{
			name:    "steerWithEmptyMessage",
			message: "",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "message cannot be empty",
		},
		{
			name:    "steerWithWhitespaceOnlyMessage",
			message: "   \t\n  ",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "message cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result, err := server.steerService(workspacePath, tt.message)

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
		})
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

func TestRespondServiceValidation(t *testing.T) {
	tests := []struct {
		name            string
		questionID      string
		answer          string
		selectedChoices []string
		setupFunc       func(*testing.T, string)
		wantErr         bool
		errContains     string
	}{
		{
			name:            "respondWithEmptyQuestionID",
			questionID:      "",
			answer:          "Test answer",
			selectedChoices: []string{},
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "no pending question",
		},
		{
			name:            "respondWithEmptyAnswerAndChoices",
			questionID:      "test-question-1",
			answer:          "",
			selectedChoices: []string{},
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "no pending question",
		},
		{
			name:            "respondWithOnlyChoices",
			questionID:      "test-question-1",
			answer:          "",
			selectedChoices: []string{"Option A", "Option B"},
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "no pending question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result, err := server.respondService(workspacePath, tt.questionID, tt.answer, tt.selectedChoices)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result.Success)
		})
	}
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

	goalContent := "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal"
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

	coord := state.NewCoordinatorEmpty(statePath(workspacePath))
	require.NoError(t, coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWaitingForHuman
		wf.InteractionMode = state.ModeBrainstorming
		wf.MultiChoiceQuestion = &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Approve this definition?"},
			},
			IsWorkGate: true,
		}
	}))

	currentID := generateQuestionID(coord.State())
	req := apiRespondRequest{
		QuestionID:      currentID,
		SelectedChoices: []string{workGateApprovalText},
	}

	result, err := server.respondViaCoordinatorService(coord, req)
	require.NoError(t, err)
	assert.True(t, result.Success)

	wfState := coord.State()
	assert.Equal(t, state.ModeBuilding, wfState.InteractionMode)
}

func TestRespondLegacyServiceNoQuestion(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	req := apiRespondRequest{
		QuestionID: "test-question-1",
		Answer:     "Test answer",
	}

	_, err := server.respondLegacyService(workspacePath, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending question")
}

func TestRespondLegacyServiceQuestionExpired(t *testing.T) {
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

	_, err := server.respondLegacyService(workspacePath, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "question expired")
}

func TestRespondLegacyServiceSuccess(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	coord := server.workspaceCoordinator(workspacePath)
	require.NoError(t, coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWaitingForHuman
		wf.HumanMessage = "What should I do?"
	}))

	currentID := generateQuestionID(coord.State())
	req := apiRespondRequest{
		QuestionID: currentID,
		Answer:     "Test answer",
	}

	result, err := server.respondLegacyService(workspacePath, req)
	require.NoError(t, err)
	assert.True(t, result.Success)

	coord = state.NewCoordinatorEmpty(statePath(workspacePath))
	wfState := coord.State()
	assert.Equal(t, state.StatusWorking, wfState.Status)
	assert.Empty(t, wfState.HumanMessage)
}

func TestHandleAPIRespondLegacyEmptyResponseBadRequest(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What?",
	})
	require.NoError(t, errCoord)
	wfState := server.workspaceCoordinator(wsDir).State()
	questionID := generateQuestionID(wfState)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/respond", `{"questionId":"`+questionID+`","answer":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIRespondLegacyExpiredQuestionConflict(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What?",
	})
	require.NoError(t, errCoord)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/respond", `{"questionId":"expired","answer":"yes"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleAPIRespondLegacySuccessful(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What should I do?",
	})
	require.NoError(t, errCoord)
	wfState := server.workspaceCoordinator(wsDir).State()
	questionID := generateQuestionID(wfState)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/respond", `{"questionId":"`+questionID+`","answer":"do this"}`)
	assert.Equal(t, http.StatusOK, w.Code)
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

func TestRespondLegacyNoQuestion(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-legacy-noq")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	body := `{"answer":"something","questionId":"fake-id"}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-legacy-noq/respond", body)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRespondLegacyPath(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-legacy")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What do?",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "What?", Choices: []string{"X", "Y"}},
			},
		},
	})
	require.NoError(t, errCoord)

	qid := generateQuestionID(state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What do?",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "What?", Choices: []string{"X", "Y"}},
			},
		},
	})
	body := `{"answer":"do X","questionId":"` + qid + `","selectedChoices":["X"]}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-legacy/respond", body)
	assert.Equal(t, http.StatusOK, w.Code)
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
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Is this ready?",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Is ready?", Choices: []string{workGateApprovalText, "Not ready"}},
			},
			IsWorkGate: true,
		},
		InteractionMode: state.ModeBrainstorming,
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	qid := generateQuestionID(coord.State())
	body := `{"answer":"","questionId":"` + qid + `","selectedChoices":["` + workGateApprovalText + `"]}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-gate/respond", body)
	assert.Equal(t, http.StatusOK, w.Code)

	updatedState := coord.State()
	assert.Equal(t, state.ModeBuilding, updatedState.InteractionMode)
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

func TestRespondServiceNoSessionFallsBackToLegacy(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What?",
	})
	require.NoError(t, errCoord)
	_, errRespond := server.respondService(wsDir, "wrong-id", "answer", nil)
	assert.Error(t, errRespond)
}

func TestSteerServiceSuccessful(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{})
	require.NoError(t, errCoord)
	result, errSteer := server.steerService(wsDir, "do something different")
	require.NoError(t, errSteer)
	assert.True(t, result.Success)
}

func TestSteerServiceEmptyMessageFails(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	_, errSteer := server.steerService(wsDir, "  ")
	assert.Error(t, errSteer)
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
