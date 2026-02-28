package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type composeStateResult struct {
	Workspace      string
	State          composerState
	Wizard         apiWizardState
	TechStackItems []apiTechStackItem
	FlowError      string
}

func (s *Server) composeStateService(workspacePath string) composeStateResult {
	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	wizard := syncWizardState(cs.wizard, currentState)
	cs.mu.Unlock()

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	return composeStateResult{
		Workspace:      filepath.Base(workspacePath),
		State:          currentState,
		Wizard:         apiWizardState(wizard),
		TechStackItems: buildAPITechStackItems(wizard.TechStack),
		FlowError:      flowErr,
	}
}

type composeSaveResult struct {
	Saved     bool
	Workspace string
}

func (s *Server) composeSaveService(workspacePath, ifMatch string) (composeSaveResult, error) {
	goalPath := filepath.Join(workspacePath, "GOAL.md")

	if ifMatch != "" {
		currentContent, errRead := os.ReadFile(goalPath)
		if errRead != nil && !os.IsNotExist(errRead) {
			return composeSaveResult{}, fmt.Errorf("failed to read current GOAL.md")
		}
		currentEtag := computeEtag(currentContent)
		if ifMatch != currentEtag {
			return composeSaveResult{}, fmt.Errorf("GOAL.md has been modified by another session")
		}
	}

	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	goalContent := buildGOALContent(currentState)

	if errWrite := os.WriteFile(goalPath, []byte(goalContent), 0644); errWrite != nil {
		return composeSaveResult{}, fmt.Errorf("failed to save GOAL.md: %w", errWrite)
	}

	s.notifyStateChange()

	return composeSaveResult{Saved: true, Workspace: filepath.Base(workspacePath)}, nil
}

type composeTemplatesResult struct {
	Templates []apiComposeTemplateEntry
}

func (s *Server) composeTemplatesService() composeTemplatesResult {
	entries := make([]apiComposeTemplateEntry, len(workflowTemplates))
	for i, tmpl := range workflowTemplates {
		entries[i] = apiComposeTemplateEntry(tmpl)
	}
	return composeTemplatesResult{Templates: entries}
}

type composePreviewResult struct {
	Content   string
	FlowError string
	Etag      string
}

func (s *Server) composePreviewService(workspacePath string) (composePreviewResult, error) {
	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	preview := buildGOALContent(currentState)

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	existingContent, errRead := os.ReadFile(goalPath)
	if errRead != nil && !os.IsNotExist(errRead) {
		return composePreviewResult{}, fmt.Errorf("failed to read current GOAL.md")
	}
	etag := computeEtag(existingContent)

	return composePreviewResult{Content: preview, FlowError: flowErr, Etag: etag}, nil
}

type composeDraftResult struct {
	Saved bool
}

func (s *Server) composeDraftService(workspacePath string, state composerState, wizard wizardState) composeDraftResult {
	cs := s.getComposerSession(workspacePath)
	cs.mu.Lock()
	cs.state = state
	cs.wizard = wizard
	cs.mu.Unlock()

	s.notifyStateChange()

	return composeDraftResult{Saved: true}
}
