package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type createWorkspaceResult struct {
	Name string
	Dir  string
}

func (s *Server) createWorkspaceService(name string) (createWorkspaceResult, error) {
	if errMsg := validateWorkspaceName(name); errMsg != "" {
		return createWorkspaceResult{}, fmt.Errorf("%s", errMsg)
	}

	workspacePath := filepath.Join(s.rootDir, name)
	if _, errStat := os.Stat(workspacePath); errStat == nil {
		return createWorkspaceResult{}, fmt.Errorf("a directory with this name already exists")
	} else if !os.IsNotExist(errStat) {
		return createWorkspaceResult{}, fmt.Errorf("failed to check workspace path")
	}

	if errMkdir := os.MkdirAll(workspacePath, 0755); errMkdir != nil {
		return createWorkspaceResult{}, fmt.Errorf("failed to create workspace directory")
	}

	if errInit := initializeWorkspace(workspacePath); errInit != nil {
		return createWorkspaceResult{}, fmt.Errorf("failed to initialize workspace")
	}

	s.invalidateWorkspaceScanCache()

	s.mu.Lock()
	s.pinnedDirs[workspacePath] = true
	s.mu.Unlock()
	_ = s.savePinnedProjects()

	return createWorkspaceResult{Name: name, Dir: workspacePath}, nil
}

type forkWorkspaceResult struct {
	Name      string
	Dir       string
	Parent    string
	CreatedAt string
}

func (s *Server) forkWorkspaceService(workspacePath, name string) (forkWorkspaceResult, error) {
	if s.classifyWorkspaceCached(workspacePath) == workspaceFork {
		return forkWorkspaceResult{}, fmt.Errorf("forks cannot create new forks")
	}

	normalized := normalizeForkName(name)
	if errMsg := validateWorkspaceName(normalized); errMsg != "" {
		return forkWorkspaceResult{}, fmt.Errorf("%s", errMsg)
	}

	parentDir := filepath.Dir(workspacePath)
	forkPath := filepath.Join(parentDir, normalized)
	if _, errStat := os.Stat(forkPath); errStat == nil {
		return forkWorkspaceResult{}, fmt.Errorf("a directory with this name already exists")
	} else if !os.IsNotExist(errStat) {
		return forkWorkspaceResult{}, fmt.Errorf("failed to check workspace path")
	}

	cmd := exec.Command("jj", "workspace", "add", forkPath)
	cmd.Dir = workspacePath
	output, errFork := cmd.CombinedOutput()
	if errFork != nil {
		return forkWorkspaceResult{}, fmt.Errorf("failed to fork workspace: %v: %s", errFork, output)
	}

	if errSkel := unpackSkeleton(forkPath); errSkel != nil {
		return forkWorkspaceResult{}, fmt.Errorf("failed to unpack skeleton: %w", errSkel)
	}
	if errExclude := addGitExclude(forkPath); errExclude != nil {
		return forkWorkspaceResult{}, fmt.Errorf("failed to add git exclude: %w", errExclude)
	}
	if errGoal := writeGoalExample(forkPath); errGoal != nil {
		return forkWorkspaceResult{}, fmt.Errorf("failed to create GOAL.md: %w", errGoal)
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)

	s.mu.Lock()
	s.pinnedDirs[forkPath] = true
	s.mu.Unlock()
	_ = s.savePinnedProjects()

	s.notifyStateChange()

	return forkWorkspaceResult{
		Name:      normalized,
		Dir:       forkPath,
		Parent:    filepath.Base(workspacePath),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

type forkWithGoalResult struct {
	Name   string
	Dir    string
	Parent string
}

func (s *Server) forkWithGoalService(workspacePath, goalContent string) (forkWithGoalResult, error) {
	if s.classifyWorkspaceCached(workspacePath) == workspaceFork {
		return forkWithGoalResult{}, fmt.Errorf("forks cannot create new forks")
	}

	name := generateRandomForkName()
	if errMsg := validateWorkspaceName(name); errMsg != "" {
		return forkWithGoalResult{}, fmt.Errorf("%s", errMsg)
	}

	parentDir := filepath.Dir(workspacePath)
	forkPath := filepath.Join(parentDir, name)
	if _, errStat := os.Stat(forkPath); errStat == nil {
		return forkWithGoalResult{}, fmt.Errorf("a directory with this name already exists")
	} else if !os.IsNotExist(errStat) {
		return forkWithGoalResult{}, fmt.Errorf("failed to check workspace path")
	}

	cmd := exec.Command("jj", "workspace", "add", forkPath)
	cmd.Dir = workspacePath
	output, errFork := cmd.CombinedOutput()
	if errFork != nil {
		return forkWithGoalResult{}, fmt.Errorf("failed to fork workspace: %v: %s", errFork, output)
	}

	if errSkel := unpackSkeleton(forkPath); errSkel != nil {
		return forkWithGoalResult{}, fmt.Errorf("failed to unpack skeleton: %w", errSkel)
	}
	if errExclude := addGitExclude(forkPath); errExclude != nil {
		return forkWithGoalResult{}, fmt.Errorf("failed to add git exclude: %w", errExclude)
	}

	goalPath := filepath.Join(forkPath, "GOAL.md")
	if errWrite := os.WriteFile(goalPath, []byte(goalContent), 0644); errWrite != nil {
		return forkWithGoalResult{}, fmt.Errorf("failed to write GOAL.md: %w", errWrite)
	}

	description := goalDescriptionFromContent(goalContent)
	coord := s.workspaceCoordinator(forkPath)
	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Summary = description
	}); errUpdate != nil {
		return forkWithGoalResult{}, fmt.Errorf("failed to save fork state: %w", errUpdate)
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)

	s.mu.Lock()
	s.pinnedDirs[forkPath] = true
	s.mu.Unlock()
	_ = s.savePinnedProjects()

	s.notifyStateChange()

	return forkWithGoalResult{
		Name:   name,
		Dir:    forkPath,
		Parent: filepath.Base(workspacePath),
	}, nil
}

type deleteForkResult struct {
	Deleted bool
	Message string
}

func (s *Server) deleteForkService(workspacePath, forkDir string, confirm bool) (deleteForkResult, error) {
	if s.classifyWorkspaceCached(workspacePath) != workspaceRoot {
		return deleteForkResult{}, fmt.Errorf("workspace is not a root")
	}

	if !confirm {
		return deleteForkResult{}, fmt.Errorf("confirmation required to delete fork")
	}

	validatedForkDir, errValidate := s.validateDirectory(forkDir)
	if errValidate != nil {
		return deleteForkResult{}, fmt.Errorf("invalid fork directory")
	}

	if s.classifyWorkspaceCached(validatedForkDir) != workspaceFork {
		return deleteForkResult{}, fmt.Errorf("fork workspace not found")
	}

	if getRootWorkspacePath(validatedForkDir) != workspacePath {
		return deleteForkResult{}, fmt.Errorf("fork does not belong to root")
	}

	forkName := filepath.Base(validatedForkDir)
	s.stopSession(validatedForkDir)

	forgetCmd := exec.Command("jj", "workspace", "forget", forkName)
	forgetCmd.Dir = workspacePath
	if _, errForget := forgetCmd.CombinedOutput(); errForget != nil {
		return deleteForkResult{}, fmt.Errorf("failed to forget fork workspace")
	}

	if errRemove := os.RemoveAll(validatedForkDir); errRemove != nil {
		return deleteForkResult{}, fmt.Errorf("failed to remove fork directory")
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)
	s.classifyCache.delete(validatedForkDir)
	s.notifyStateChange()

	return deleteForkResult{Deleted: true, Message: "fork deleted successfully"}, nil
}

type renameWorkspaceResult struct {
	Name    string
	OldName string
	Dir     string
}

func (s *Server) renameWorkspaceService(workspacePath, newName string) (renameWorkspaceResult, error) {
	if s.classifyWorkspaceCached(workspacePath) != workspaceFork {
		return renameWorkspaceResult{}, fmt.Errorf("only forks can be renamed")
	}

	normalized := normalizeForkName(newName)
	if errMsg := validateWorkspaceName(normalized); errMsg != "" {
		return renameWorkspaceResult{}, fmt.Errorf("%s", errMsg)
	}

	s.mu.Lock()
	sess := s.sessions[workspacePath]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running := sess.running
		sess.mu.Unlock()
		if running {
			return renameWorkspaceResult{}, fmt.Errorf("cannot rename: session is running")
		}
	}

	rootDir := getRootWorkspacePath(workspacePath)

	oldName := filepath.Base(workspacePath)
	parentDir := filepath.Dir(workspacePath)
	newPath := filepath.Join(parentDir, normalized)
	if _, errStat := os.Stat(newPath); errStat == nil {
		return renameWorkspaceResult{}, fmt.Errorf("a directory with this name already exists")
	} else if !os.IsNotExist(errStat) {
		return renameWorkspaceResult{}, fmt.Errorf("failed to check target path")
	}

	if errRename := os.Rename(workspacePath, newPath); errRename != nil {
		return renameWorkspaceResult{}, fmt.Errorf("failed to rename directory: %v", errRename)
	}

	cmd := exec.Command("jj", "workspace", "rename", normalized)
	cmd.Dir = newPath
	if output, errJJ := cmd.CombinedOutput(); errJJ != nil {
		return renameWorkspaceResult{}, fmt.Errorf("jj workspace rename failed: %v: %s", errJJ, output)
	}

	s.mu.Lock()
	if existing, ok := s.sessions[workspacePath]; ok {
		delete(s.sessions, workspacePath)
		s.sessions[newPath] = existing
	}
	if s.everStartedDirs[workspacePath] {
		delete(s.everStartedDirs, workspacePath)
		s.everStartedDirs[newPath] = true
	}
	pinReKeyed := s.pinnedDirs[workspacePath]
	if pinReKeyed {
		delete(s.pinnedDirs, workspacePath)
		s.pinnedDirs[newPath] = true
	}
	if existing, ok := s.adhocStates[workspacePath]; ok {
		delete(s.adhocStates, workspacePath)
		s.adhocStates[newPath] = existing
	}
	s.mu.Unlock()

	s.composerSessionsMu.Lock()
	if existing, ok := s.composerSessions[workspacePath]; ok {
		delete(s.composerSessions, workspacePath)
		s.composerSessions[newPath] = existing
	}
	s.composerSessionsMu.Unlock()

	s.classifyCache.delete(workspacePath)
	s.classifyCache.delete(newPath)
	if rootDir != "" {
		s.classifyCache.delete(rootDir)
		s.bookmarkCache.delete(rootDir)
	}

	if pinReKeyed {
		if errSave := s.savePinnedProjects(); errSave != nil {
			return renameWorkspaceResult{}, fmt.Errorf("persisting re-keyed pins: %w", errSave)
		}
	}
	s.invalidateWorkspaceScanCache()
	s.notifyStateChange()

	return renameWorkspaceResult{Name: normalized, OldName: oldName, Dir: newPath}, nil
}

type getGoalResult struct {
	Content string
}

func (s *Server) getGoalService(workspacePath string) (getGoalResult, error) {
	data, errRead := os.ReadFile(filepath.Join(workspacePath, "GOAL.md"))
	if errRead != nil {
		return getGoalResult{}, fmt.Errorf("failed to read GOAL.md: %w", errRead)
	}
	return getGoalResult{Content: string(data)}, nil
}

type updateGoalResult struct {
	Updated   bool
	Workspace string
}

func (s *Server) updateGoalService(workspacePath, content string) (updateGoalResult, error) {
	if content == "" {
		return updateGoalResult{}, fmt.Errorf("content cannot be empty")
	}

	goalPath := filepath.Join(workspacePath, "GOAL.md")
	if errWrite := os.WriteFile(goalPath, []byte(content), 0644); errWrite != nil {
		return updateGoalResult{}, fmt.Errorf("failed to write GOAL.md: %w", errWrite)
	}

	prefix := workspacePath + "|"
	s.svgCache.deleteFunc(func(k string) bool {
		return strings.HasPrefix(k, prefix)
	})
	s.notifyStateChange()

	return updateGoalResult{Updated: true, Workspace: filepath.Base(workspacePath)}, nil
}

type updateSummaryResult struct {
	Updated   bool
	Summary   string
	Workspace string
}

func (s *Server) updateSummaryService(workspacePath, summary string) (updateSummaryResult, error) {
	coord := s.workspaceCoordinator(workspacePath)
	trimmed := strings.TrimSpace(summary)

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		wf.Summary = trimmed
		wf.SummaryManual = true
	}); errUpdate != nil {
		return updateSummaryResult{}, fmt.Errorf("failed to save workspace state: %w", errUpdate)
	}

	s.notifyStateChange()

	return updateSummaryResult{
		Updated:   true,
		Summary:   trimmed,
		Workspace: filepath.Base(workspacePath),
	}, nil
}

type togglePinResult struct {
	Pinned  bool
	Message string
}

func (s *Server) togglePinService(workspacePath string) (togglePinResult, error) {
	if errToggle := s.togglePin(workspacePath); errToggle != nil {
		return togglePinResult{}, fmt.Errorf("failed to toggle pin: %w", errToggle)
	}

	s.invalidateWorkspaceScanCache()
	s.notifyStateChange()

	return togglePinResult{Pinned: s.isPinned(workspacePath), Message: "pin toggled"}, nil
}

type deleteWorkspaceResult struct {
	Deleted bool
	Message string
}

func (s *Server) deleteWorkspaceService(workspacePath string) (deleteWorkspaceResult, error) {
	s.stopSession(workspacePath)

	if errRemove := os.RemoveAll(workspacePath); errRemove != nil {
		return deleteWorkspaceResult{}, fmt.Errorf("failed to remove workspace directory: %w", errRemove)
	}

	s.mu.Lock()
	delete(s.pinnedDirs, workspacePath)
	delete(s.sessions, workspacePath)
	delete(s.everStartedDirs, workspacePath)
	s.mu.Unlock()
	_ = s.savePinnedProjects()

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)
	s.notifyStateChange()

	return deleteWorkspaceResult{Deleted: true, Message: "workspace deleted successfully"}, nil
}

type deleteMessageResult struct {
	Deleted bool
	ID      int
	Message string
}

func (s *Server) deleteMessageService(workspacePath string, messageID int) (deleteMessageResult, error) {
	coord := s.workspaceCoordinator(workspacePath)
	wfState := coord.State()

	found := false
	for _, msg := range wfState.Messages {
		if msg.ID == messageID {
			found = true
			break
		}
	}

	if !found {
		return deleteMessageResult{}, fmt.Errorf("message not found")
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		for i, msg := range wf.Messages {
			if msg.ID == messageID {
				wf.Messages = slices.Delete(wf.Messages, i, i+1)
				break
			}
		}
	}); errUpdate != nil {
		return deleteMessageResult{}, fmt.Errorf("failed to save workspace state: %w", errUpdate)
	}

	s.notifyStateChange()

	return deleteMessageResult{Deleted: true, ID: messageID, Message: "message deleted successfully"}, nil
}
