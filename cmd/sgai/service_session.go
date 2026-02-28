package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type startSessionResult2 struct {
	Name           string
	Status         string
	Running        bool
	Message        string
	AlreadyRunning bool
}

func (s *Server) startSessionService(workspacePath string, auto bool) (startSessionResult2, error) {
	if s.classifyWorkspaceCached(workspacePath) == workspaceRoot {
		return startSessionResult2{}, fmt.Errorf("root workspace cannot start agentic work")
	}

	coord := s.workspaceCoordinator(workspacePath)
	continuousPrompt := readContinuousModePrompt(workspacePath)

	var interactionMode string
	switch {
	case continuousPrompt != "":
		interactionMode = state.ModeContinuous
	case auto:
		interactionMode = state.ModeSelfDrive
	default:
		interactionMode = state.ModeBrainstorming
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.InteractionMode = interactionMode
	}); errUpdate != nil {
		return startSessionResult2{}, fmt.Errorf("failed to save workflow state: %w", errUpdate)
	}

	result := s.startSession(workspacePath)

	name := filepath.Base(workspacePath)

	if result.alreadyRunning {
		return startSessionResult2{
			Name:           name,
			Status:         "running",
			Running:        true,
			Message:        "session already running",
			AlreadyRunning: true,
		}, nil
	}

	if result.startError != nil {
		return startSessionResult2{}, result.startError
	}

	s.notifyStateChange()

	return startSessionResult2{
		Name:    name,
		Status:  "running",
		Running: true,
		Message: "session started",
	}, nil
}

type stopSessionResult struct {
	Name    string
	Status  string
	Running bool
	Message string
}

func (s *Server) stopSessionService(workspacePath string) stopSessionResult {
	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()

	var alreadyStopped bool
	if sess == nil {
		alreadyStopped = true
	} else {
		sess.mu.Lock()
		alreadyStopped = !sess.running
		sess.mu.Unlock()
	}

	s.stopSession(workspacePath)

	message := "session stopped"
	if alreadyStopped {
		message = "session already stopped"
	}

	s.notifyStateChange()

	return stopSessionResult{
		Name:    filepath.Base(workspacePath),
		Status:  "stopped",
		Running: false,
		Message: message,
	}
}

type respondResult struct {
	Success bool
	Message string
}

func (s *Server) respondService(workspacePath, questionID, answer string, selectedChoices []string) (respondResult, error) {
	req := apiRespondRequest{
		QuestionID:      questionID,
		Answer:          answer,
		SelectedChoices: selectedChoices,
	}

	coord := s.sessionCoordinator(workspacePath)
	if coord != nil {
		return s.respondViaCoordinatorService(coord, req)
	}

	return s.respondLegacyService(workspacePath, req)
}

func (s *Server) respondViaCoordinatorService(coord *state.Coordinator, req apiRespondRequest) (respondResult, error) {
	wfState := coord.State()

	if !wfState.NeedsHumanInput() {
		return respondResult{}, fmt.Errorf("no pending question")
	}

	currentID := generateQuestionID(wfState)
	if req.QuestionID != currentID {
		return respondResult{}, fmt.Errorf("question expired")
	}

	responseText := buildAPIResponseText(req, wfState.MultiChoiceQuestion)
	if responseText == "" {
		return respondResult{}, fmt.Errorf("response cannot be empty")
	}

	if wfState.MultiChoiceQuestion != nil && wfState.MultiChoiceQuestion.IsWorkGate {
		approvedViaSelection := slices.Contains(req.SelectedChoices, workGateApprovalText)
		if approvedViaSelection {
			if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
				if wf.InteractionMode == state.ModeBrainstorming {
					wf.InteractionMode = state.ModeBuilding
				}
			}); errUpdate != nil {
				return respondResult{}, fmt.Errorf("failed to save work gate approval: %w", errUpdate)
			}
		}
	}

	coord.Respond(responseText)
	s.notifyStateChange()

	return respondResult{Success: true, Message: "response submitted"}, nil
}

func (s *Server) respondLegacyService(workspacePath string, req apiRespondRequest) (respondResult, error) {
	coord := s.workspaceCoordinator(workspacePath)
	wfState := coord.State()

	if !wfState.NeedsHumanInput() {
		return respondResult{}, fmt.Errorf("no pending question in legacy path")
	}

	currentID := generateQuestionID(wfState)
	if req.QuestionID != currentID {
		return respondResult{}, fmt.Errorf("question expired")
	}

	responseText := buildAPIResponseText(req, wfState.MultiChoiceQuestion)
	if responseText == "" {
		return respondResult{}, fmt.Errorf("response cannot be empty")
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Status = state.StatusWorking
		wf.HumanMessage = ""
		wf.MultiChoiceQuestion = nil
		wf.Task = ""
	}); errUpdate != nil {
		return respondResult{}, fmt.Errorf("failed to save state: %w", errUpdate)
	}

	s.notifyStateChange()

	return respondResult{Success: true, Message: "response submitted"}, nil
}

type steerResult struct {
	Success bool
	Message string
}

func (s *Server) steerService(workspacePath, message string) (steerResult, error) {
	if strings.TrimSpace(message) == "" {
		return steerResult{}, fmt.Errorf("message cannot be empty")
	}

	coord := s.workspaceCoordinator(workspacePath)
	steerBody := "Re-steering instruction: " + strings.TrimSpace(message)
	steerCreatedAt := time.Now().UTC().Format(time.RFC3339)

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		newMsg := state.Message{
			ID:        nextMessageID(wf.Messages),
			FromAgent: "Human Partner",
			ToAgent:   "coordinator",
			Body:      steerBody,
			CreatedAt: steerCreatedAt,
		}
		insertIdx := findSteerInsertPosition(wf.Messages)
		wf.Messages = slices.Insert(wf.Messages, insertIdx, newMsg)
	}); errUpdate != nil {
		return steerResult{}, fmt.Errorf("failed to save state: %w", errUpdate)
	}

	s.notifyStateChange()

	return steerResult{Success: true, Message: "steering instruction added"}, nil
}
