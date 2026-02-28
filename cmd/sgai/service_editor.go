package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type openEditorResult struct {
	Opened  bool
	Editor  string
	Message string
}

func (s *Server) openEditorService(workspacePath string) (openEditorResult, error) {
	if !s.editorAvailable {
		return openEditorResult{}, fmt.Errorf("no editor available")
	}

	if errOpen := s.editor.open(workspacePath); errOpen != nil {
		return openEditorResult{}, fmt.Errorf("failed to open editor: %v", errOpen)
	}

	return openEditorResult{Opened: true, Editor: s.editorName, Message: "opened in editor"}, nil
}

func (s *Server) openEditorGoalService(workspacePath string) (openEditorResult, error) {
	return s.openEditorFileService(workspacePath, "GOAL.md")
}

func (s *Server) openEditorProjectManagementService(workspacePath string) (openEditorResult, error) {
	return s.openEditorFileService(workspacePath, filepath.Join(".sgai", "PROJECT_MANAGEMENT.md"))
}

func (s *Server) openEditorFileService(workspacePath, relPath string) (openEditorResult, error) {
	if !s.editorAvailable {
		return openEditorResult{}, fmt.Errorf("no editor available")
	}

	targetPath := filepath.Join(workspacePath, relPath)
	if _, errStat := os.Stat(targetPath); errStat != nil {
		return openEditorResult{}, fmt.Errorf("file not found")
	}

	if errOpen := s.editor.open(targetPath); errOpen != nil {
		return openEditorResult{}, fmt.Errorf("failed to open editor: %v", errOpen)
	}

	return openEditorResult{Opened: true, Editor: s.editorName, Message: "opened in editor"}, nil
}

type openOpencodeResult struct {
	Opened  bool
	Message string
}

func (s *Server) openOpencodeService(workspacePath string) (openOpencodeResult, error) {
	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()

	if sess == nil {
		return openOpencodeResult{}, fmt.Errorf("factory is not running")
	}

	sess.mu.Lock()
	running := sess.running
	sess.mu.Unlock()

	if !running {
		return openOpencodeResult{}, fmt.Errorf("factory is not running")
	}

	wfState := s.workspaceCoordinator(workspacePath).State()
	currentAgent := wfState.CurrentAgent
	sessionID := wfState.SessionID

	models := modelsForAgentFromGoal(workspacePath, currentAgent)
	var model string
	if len(models) > 0 {
		model, _ = parseModelAndVariant(models[0])
	}

	interactive := "yes"
	if wfState.InteractionMode == state.ModeSelfDrive {
		interactive = "auto"
	}

	execPath, errExec := os.Executable()
	if errExec != nil {
		return openOpencodeResult{}, fmt.Errorf("failed to resolve executable path")
	}

	opencodeCmd := fmt.Sprintf("opencode --session %q --agent %q", sessionID, currentAgent)
	if model != "" {
		opencodeCmd += fmt.Sprintf(" --model %q", model)
	}
	scriptContent := fmt.Sprintf("#!/bin/bash\ntrap 'rm -f \"$0\"' EXIT\ncd %q\nexport OPENCODE_CONFIG_DIR=.sgai\nexport SGAI_MCP_EXECUTABLE=%q\nexport SGAI_MCP_INTERACTIVE=%q\n%s\n",
		workspacePath, execPath, interactive, opencodeCmd)

	scriptPath, errScript := writeOpenCodeScript(scriptContent)
	if errScript != nil {
		return openOpencodeResult{}, fmt.Errorf("failed to prepare opencode script")
	}

	if errOpen := openInTerminal(scriptPath); errOpen != nil {
		_ = os.Remove(scriptPath)
		return openOpencodeResult{}, fmt.Errorf("failed to open terminal")
	}

	return openOpencodeResult{Opened: true, Message: "opened in opencode"}, nil
}
