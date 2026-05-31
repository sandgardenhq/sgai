package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

type workspaceDiffResult struct {
	Diff string
}

func (s *Server) workspaceDiffService(workspacePath string) workspaceDiffResult {
	if !hasJJRepo(workspacePath) {
		return workspaceDiffResult{}
	}
	return workspaceDiffResult{Diff: collectJJFullDiff(workspacePath)}
}

func (s *Server) getAgentDelegationSVGService(workspacePath string) string {
	goalPath := filepath.Join(workspacePath, "GOAL.md")
	if _, errStat := os.Stat(goalPath); errStat != nil {
		return ""
	}
	metadata, errParse := parseYAMLFrontmatterFromFile(goalPath)
	if errParse != nil {
		return ""
	}
	agents := delegatableAgents(metadata.Agents)
	if len(agents) == 0 {
		return ""
	}
	return buildAgentDelegationSVG(agents)
}

func buildAgentDelegationSVG(agents []string) string {
	var buf bytes.Buffer
	errEscape := xml.EscapeText(&buf, []byte(strings.Join(agents, " → ")))
	if errEscape != nil {
		return ""
	}
	return `<svg xmlns="http://www.w3.org/2000/svg"><text>` + buf.String() + `</text></svg>`
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
