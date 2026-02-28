package main

import (
	"fmt"
	"os"
	"path/filepath"
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
