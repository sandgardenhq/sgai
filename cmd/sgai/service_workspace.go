package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

var (
	errForkOfFork           = errors.New("forks cannot create new forks")
	errGoalContentEmpty     = errors.New("GOAL.md must have content describing the goal")
	errDirectoryExists      = errors.New("a directory with this name already exists")
	errWorkspaceNameInvalid = errors.New("workspace name is invalid")
	errMessageNotFound      = errors.New("message not found")
)

func generateRandomForkName() string {
	adjectives := [20]string{
		"bold", "swift", "calm", "keen", "warm",
		"cool", "soft", "firm", "pure", "wild",
		"deep", "tall", "fair", "vast", "slim",
		"rich", "free", "rare", "true", "wise",
	}
	colors := [15]string{
		"red", "blue", "green", "gold", "gray",
		"teal", "pink", "jade", "ruby", "sage",
		"plum", "mint", "onyx", "navy", "lime",
	}
	const suffixChars = "0123456789aeiou"

	adjective := adjectives[rand.IntN(len(adjectives))]
	color := colors[rand.IntN(len(colors))]
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = suffixChars[rand.IntN(len(suffixChars))]
	}
	return adjective + "-" + color + "-" + string(suffix)
}

type createWorkspaceResult struct {
	Name string
	Dir  string
}

func (s *Server) createWorkspaceService(name string) (createWorkspaceResult, error) {
	if errMsg := validateWorkspaceName(name); errMsg != "" {
		return createWorkspaceResult{}, fmt.Errorf("%w: %s", errWorkspaceNameInvalid, errMsg)
	}

	workspacePath := filepath.Join(s.rootDir, name)
	if _, errStat := os.Stat(workspacePath); errStat == nil {
		return createWorkspaceResult{}, errDirectoryExists
	} else if !os.IsNotExist(errStat) {
		return createWorkspaceResult{}, fmt.Errorf("failed to check workspace path: %w", errStat)
	}

	if errMkdir := os.MkdirAll(workspacePath, 0755); errMkdir != nil {
		return createWorkspaceResult{}, fmt.Errorf("failed to create workspace directory: %w", errMkdir)
	}

	if errInit := initializeWorkspace(workspacePath); errInit != nil {
		return createWorkspaceResult{}, fmt.Errorf("failed to initialize workspace: %w", errInit)
	}

	s.invalidateWorkspaceScanCache()

	s.mu.Lock()
	s.pinnedDirs[resolveSymlinks(workspacePath)] = true
	s.mu.Unlock()
	if errSave := s.savePinnedProjects(); errSave != nil {
		log.Println("failed to persist pins:", errSave)
	}

	return createWorkspaceResult{Name: name, Dir: workspacePath}, nil
}

type forkWorkspaceResult struct {
	Name      string
	Dir       string
	Parent    string
	CreatedAt string
}

func (s *Server) forkWorkspaceService(workspacePath, goalContent string) (forkWorkspaceResult, error) {
	if s.classifyWorkspaceCached(workspacePath) == workspaceFork {
		return forkWorkspaceResult{}, errForkOfFork
	}

	if goalContentBodyIsEmpty(goalContent) {
		return forkWorkspaceResult{}, errGoalContentEmpty
	}

	name := generateRandomForkName()
	parentDir := filepath.Dir(workspacePath)
	forkPath := filepath.Join(parentDir, name)
	if _, errStat := os.Stat(forkPath); errStat == nil {
		return forkWorkspaceResult{}, errDirectoryExists
	} else if !os.IsNotExist(errStat) {
		return forkWorkspaceResult{}, fmt.Errorf("failed to check workspace path: %w", errStat)
	}

	cmd := exec.Command("jj", "workspace", "add", forkPath)
	cmd.Dir = workspacePath
	output, errFork := cmd.CombinedOutput()
	if errFork != nil {
		return forkWorkspaceResult{}, fmt.Errorf("failed to fork workspace: %w: %s", errFork, output)
	}

	if errSkel := unpackSkeleton(forkPath); errSkel != nil {
		return forkWorkspaceResult{}, failForkWorkspaceSetup(workspacePath, forkPath, "failed to unpack skeleton", errSkel)
	}
	if errExclude := addGitExclude(forkPath); errExclude != nil {
		return forkWorkspaceResult{}, failForkWorkspaceSetup(workspacePath, forkPath, "failed to add git exclude", errExclude)
	}
	if errGoal := writeGoalContent(forkPath, goalContent); errGoal != nil {
		return forkWorkspaceResult{}, failForkWorkspaceSetup(workspacePath, forkPath, "failed to create GOAL.md", errGoal)
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(workspacePath)

	s.mu.Lock()
	s.pinnedDirs[resolveSymlinks(forkPath)] = true
	s.mu.Unlock()
	if errSave := s.savePinnedProjects(); errSave != nil {
		log.Println("failed to persist pins:", errSave)
	}

	s.notifyStateChange()

	return forkWorkspaceResult{
		Name:      name,
		Dir:       forkPath,
		Parent:    filepath.Base(workspacePath),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func failForkWorkspaceSetup(workspacePath, forkPath, message string, errCause error) error {
	errSetup := fmt.Errorf("%s: %w", message, errCause)
	errRollback := rollbackForkWorkspaceCreation(workspacePath, forkPath)
	if errRollback != nil {
		return errors.Join(errSetup, fmt.Errorf("failed to rollback fork workspace creation: %w", errRollback))
	}
	return errSetup
}

func rollbackForkWorkspaceCreation(workspacePath, forkPath string) error {
	forkName := filepath.Base(forkPath)
	forgetCmd := exec.Command("jj", "workspace", "forget", forkName)
	forgetCmd.Dir = workspacePath
	output, errForget := forgetCmd.CombinedOutput()

	var errForgetWrapped error
	if errForget != nil {
		errForgetWrapped = fmt.Errorf("failed to forget fork workspace during rollback: %w: %s", errForget, output)
	}

	errRemove := os.RemoveAll(forkPath)
	if errRemove != nil {
		errRemove = fmt.Errorf("failed to remove fork workspace during rollback: %w", errRemove)
	}

	if errForgetWrapped != nil || errRemove != nil {
		return errors.Join(errForgetWrapped, errRemove)
	}
	return nil
}

func goalContentBodyIsEmpty(goalContent string) bool {
	body := stripFrontmatter(goalContent)
	return strings.TrimSpace(body) == ""
}

func writeGoalContent(dir, content string) error {
	goalPath := filepath.Join(dir, "GOAL.md")
	return os.WriteFile(goalPath, []byte(content), 0644)
}

type deleteForkResult struct {
	Deleted bool
	Message string
}

func (s *Server) deleteForkByPathService(forkDir string) (deleteForkResult, error) {
	rootPath := getRootWorkspacePath(forkDir)
	if rootPath == "" {
		return deleteForkResult{}, fmt.Errorf("could not determine root workspace for fork")
	}
	return s.deleteForkService(rootPath, forkDir, true)
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
	delete(s.pinnedDirs, resolveSymlinks(workspacePath))
	delete(s.sessions, workspacePath)
	delete(s.everStartedDirs, workspacePath)
	s.mu.Unlock()
	if errSave := s.savePinnedProjects(); errSave != nil {
		log.Println("failed to persist pins:", errSave)
	}

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

	if !slices.ContainsFunc(wfState.Messages, func(msg state.Message) bool {
		return msg.ID == messageID
	}) {
		return deleteMessageResult{}, errMessageNotFound
	}

	if errUpdate := coord.UpdateState(func(wf *state.Workflow) {
		i := slices.IndexFunc(wf.Messages, func(msg state.Message) bool {
			return msg.ID == messageID
		})
		if i >= 0 {
			wf.Messages = slices.Delete(wf.Messages, i, i+1)
		}
	}); errUpdate != nil {
		return deleteMessageResult{}, fmt.Errorf("failed to save workspace state: %w", errUpdate)
	}

	s.notifyStateChange()

	return deleteMessageResult{Deleted: true, ID: messageID, Message: "message deleted successfully"}, nil
}
