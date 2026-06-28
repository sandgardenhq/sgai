package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type startSessionServiceResult struct {
	Name           string
	Status         string
	Running        bool
	Message        string
	AlreadyRunning bool
}

func (s *Server) startSessionService(workspacePath string, auto bool) (startSessionServiceResult, error) {
	if s.classifyWorkspaceCached(workspacePath) == workspaceRoot {
		return startSessionServiceResult{}, fmt.Errorf("root workspace cannot start agentic work")
	}

	coord := s.workspaceCoordinator(workspacePath)
	continuousPrompt := readContinuousModePrompt(workspacePath)

	interactionMode := startInteractionMode(auto, continuousPrompt)

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.InteractionMode = interactionMode
	}); errUpdate != nil {
		return startSessionServiceResult{}, fmt.Errorf("failed to save workflow state: %w", errUpdate)
	}

	result := s.startSession(workspacePath)

	name := filepath.Base(workspacePath)

	if result.alreadyRunning {
		return startSessionServiceResult{
			Name:           name,
			Status:         "running",
			Running:        true,
			Message:        "session already running",
			AlreadyRunning: true,
		}, nil
	}

	if result.startError != nil {
		return startSessionServiceResult{}, result.startError
	}

	s.notifyStateChange()

	return startSessionServiceResult{
		Name:    name,
		Status:  "running",
		Running: true,
		Message: "session started",
	}, nil
}

func startInteractionMode(auto bool, continuousPrompt string) string {
	switch {
	case continuousPrompt != "":
		return state.ModeContinuous
	case auto:
		return state.ModeSelfDrive
	default:
		return state.ModeInteractive
	}
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

	wsName := filepath.Base(workspacePath)
	coord := s.sessionCoordinator(workspacePath)
	if coord != nil {
		log.Println("respond-service:", wsName, "delivering via session coordinator")
		return s.respondViaCoordinatorService(coord, req)
	}

	log.Println("respond-service:", wsName, "rejected, no session coordinator available for delivery")
	return respondResult{}, fmt.Errorf("no active session coordinator")
}

func (s *Server) respondViaCoordinatorService(coord *state.Coordinator, req apiRespondRequest) (respondResult, error) {
	wfState := coord.State()

	if !wfState.NeedsHumanInput() {
		log.Println("respond-service: coordinator path rejected, no pending question, status:", wfState.Status)
		return respondResult{}, fmt.Errorf("no pending question")
	}

	currentID := generateQuestionID(wfState)
	if req.QuestionID != currentID {
		log.Println("respond-service: coordinator path rejected, question expired, got:", req.QuestionID, "want:", currentID)
		return respondResult{}, fmt.Errorf("question expired")
	}

	responseText := buildAPIResponseText(req)
	if responseText == "" {
		return respondResult{}, fmt.Errorf("response cannot be empty")
	}

	if !coord.Respond(responseText) {
		return respondResult{}, fmt.Errorf("no active question receiver")
	}

	s.notifyStateChange()

	return respondResult{Success: true, Message: "response submitted"}, nil
}
