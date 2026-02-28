package main

import (
	"fmt"
	"os/exec"
)

type workspaceStateResult struct {
	Workspace apiWorkspaceFullState
	Found     bool
}

func (s *Server) getWorkspaceStateService(workspaceName string) (workspaceStateResult, error) {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return workspaceStateResult{}, errScan
	}

	for _, grp := range groups {
		if grp.Root.DirName == workspaceName {
			ws := s.buildWorkspaceFullState(grp.Root, groups)
			return workspaceStateResult{Workspace: ws, Found: true}, nil
		}
		for _, fork := range grp.Forks {
			if fork.DirName == workspaceName {
				ws := s.buildWorkspaceFullState(fork, groups)
				return workspaceStateResult{Workspace: ws, Found: true}, nil
			}
		}
	}

	return workspaceStateResult{Found: false}, nil
}

func (s *Server) getWorkflowSVGService(workspacePath string) string {
	wfState := s.workspaceCoordinator(workspacePath).State()
	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "Unknown"
	}
	return s.getWorkflowSVGCached(workspacePath, currentAgent)
}

type workspaceDiffResult struct {
	Diff string
}

func (s *Server) workspaceDiffService(workspacePath string) workspaceDiffResult {
	if !hasJJRepo(workspacePath) {
		return workspaceDiffResult{}
	}
	return workspaceDiffResult{Diff: collectJJFullDiff(workspacePath)}
}

type updateDescriptionResult struct {
	Updated     bool
	Description string
}

func (s *Server) updateDescriptionService(workspacePath, description string) (updateDescriptionResult, error) {
	cmd := exec.Command("jj", "desc", "-m", description)
	cmd.Dir = workspacePath
	if output, errCmd := cmd.CombinedOutput(); errCmd != nil {
		return updateDescriptionResult{}, fmt.Errorf("failed to update description: %s", output)
	}

	s.notifyStateChange()

	return updateDescriptionResult{Updated: true, Description: description}, nil
}
