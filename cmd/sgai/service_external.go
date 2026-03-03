package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var (
	errPathNotAbsolute = errors.New("path must be absolute")
	errNotADirectory   = errors.New("path is not a directory")
	errUnderRootDir    = errors.New("path is within the root directory; use a local workspace instead")
	errAlreadyAttached = errors.New("directory is already attached as an external workspace")
	errNotAttached     = errors.New("directory is not attached as an external workspace")
)

func (s *Server) externalFilePath() string {
	return filepath.Join(s.externalConfigDir, "external.json")
}

func (s *Server) loadExternalDirs() error {
	data, errRead := os.ReadFile(s.externalFilePath())
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return nil
		}
		return fmt.Errorf("reading external dirs: %w", errRead)
	}
	var dirs []string
	if errJSON := json.Unmarshal(data, &dirs); errJSON != nil {
		return fmt.Errorf("parsing external dirs: %w", errJSON)
	}
	existing := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		if _, errStat := os.Stat(d); errStat == nil {
			existing[resolveSymlinks(d)] = true
		}
	}
	s.mu.Lock()
	s.externalDirs = existing
	s.mu.Unlock()
	return nil
}

func (s *Server) saveExternalDirs() error {
	s.mu.Lock()
	dirs := slices.Collect(maps.Keys(s.externalDirs))
	s.mu.Unlock()
	if dirs == nil {
		dirs = []string{}
	}
	slices.Sort(dirs)
	if errDir := os.MkdirAll(s.externalConfigDir, 0o755); errDir != nil {
		return fmt.Errorf("creating external config directory: %w", errDir)
	}
	data, errJSON := json.Marshal(dirs)
	if errJSON != nil {
		return fmt.Errorf("encoding external dirs: %w", errJSON)
	}
	if errWrite := os.WriteFile(s.externalFilePath(), data, 0o644); errWrite != nil {
		return fmt.Errorf("writing external dirs: %w", errWrite)
	}
	return nil
}

type attachExternalResult struct {
	Name    string
	Dir     string
	HasGoal bool
}

func (s *Server) attachExternalWorkspaceService(path string) (attachExternalResult, error) {
	if !filepath.IsAbs(path) {
		return attachExternalResult{}, errPathNotAbsolute
	}

	info, errStat := os.Stat(path)
	if errStat != nil {
		if os.IsNotExist(errStat) {
			return attachExternalResult{}, fmt.Errorf("directory does not exist: %s", path)
		}
		return attachExternalResult{}, fmt.Errorf("checking path: %w", errStat)
	}
	if !info.IsDir() {
		return attachExternalResult{}, errNotADirectory
	}

	canonical := resolveSymlinks(path)

	absRoot, errAbs := filepath.Abs(s.rootDir)
	if errAbs != nil {
		absRoot = s.rootDir
	}
	rootResolved := resolveSymlinks(absRoot)

	if canonical == rootResolved || strings.HasPrefix(canonical+string(filepath.Separator), rootResolved+string(filepath.Separator)) {
		return attachExternalResult{}, errUnderRootDir
	}

	s.mu.Lock()
	if s.externalDirs[canonical] {
		s.mu.Unlock()
		return attachExternalResult{}, errAlreadyAttached
	}
	s.mu.Unlock()

	hasGoal := false
	if _, errGoal := os.Stat(filepath.Join(path, "GOAL.md")); errGoal == nil {
		hasGoal = true
	}

	sgaiDir := filepath.Join(path, ".sgai")
	if _, errSGAI := os.Stat(sgaiDir); os.IsNotExist(errSGAI) {
		if errInit := initializeWorkspace(path); errInit != nil {
			return attachExternalResult{}, fmt.Errorf("initializing workspace: %w", errInit)
		}
	}

	s.mu.Lock()
	if s.externalDirs[canonical] {
		s.mu.Unlock()
		return attachExternalResult{}, errAlreadyAttached
	}
	s.externalDirs[canonical] = true
	s.mu.Unlock()
	if errSave := s.saveExternalDirs(); errSave != nil {
		return attachExternalResult{}, fmt.Errorf("saving external dirs: %w", errSave)
	}

	s.invalidateWorkspaceScanCache()
	s.notifyStateChange()

	return attachExternalResult{
		Name:    filepath.Base(path),
		Dir:     path,
		HasGoal: hasGoal,
	}, nil
}

type detachExternalResult struct {
	Detached bool
	Message  string
}

func (s *Server) detachExternalWorkspaceService(path string) (detachExternalResult, error) {
	canonical := resolveSymlinks(path)

	s.mu.Lock()
	attached := s.externalDirs[canonical]
	if !attached {
		s.mu.Unlock()
		return detachExternalResult{}, errNotAttached
	}
	delete(s.externalDirs, canonical)
	s.mu.Unlock()

	if errSave := s.saveExternalDirs(); errSave != nil {
		return detachExternalResult{}, fmt.Errorf("saving external dirs: %w", errSave)
	}

	s.invalidateWorkspaceScanCache()
	s.notifyStateChange()

	return detachExternalResult{Detached: true, Message: "external workspace detached"}, nil
}

func (s *Server) isExternalWorkspace(workspacePath string) bool {
	canonical := resolveSymlinks(workspacePath)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.externalDirs[canonical]
}

type deleteExternalForkResult struct {
	Deleted bool
	Message string
}

func (s *Server) deleteExternalForkService(forkDir string) (deleteExternalForkResult, error) {
	rootDir := getRootWorkspacePath(forkDir)
	if rootDir == "" {
		return deleteExternalForkResult{}, fmt.Errorf("could not determine root workspace for fork")
	}

	forkName := filepath.Base(forkDir)
	s.stopSession(forkDir)

	forgetCmd := exec.Command("jj", "workspace", "forget", forkName)
	forgetCmd.Dir = rootDir
	if _, errForget := forgetCmd.CombinedOutput(); errForget != nil {
		return deleteExternalForkResult{}, fmt.Errorf("failed to forget fork workspace: %w", errForget)
	}

	canonical := resolveSymlinks(forkDir)

	if errRemove := os.RemoveAll(forkDir); errRemove != nil {
		return deleteExternalForkResult{}, fmt.Errorf("failed to remove fork directory: %w", errRemove)
	}
	s.mu.Lock()
	delete(s.externalDirs, canonical)
	s.mu.Unlock()

	if errSave := s.saveExternalDirs(); errSave != nil {
		return deleteExternalForkResult{}, fmt.Errorf("saving external dirs: %w", errSave)
	}

	s.invalidateWorkspaceScanCache()
	s.classifyCache.delete(rootDir)
	s.classifyCache.delete(forkDir)
	s.notifyStateChange()

	return deleteExternalForkResult{Deleted: true, Message: "external fork deleted successfully"}, nil
}

type directoryEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

func browseDirectoriesService(path string) ([]directoryEntry, error) {
	if path == "" {
		home, errHome := os.UserHomeDir()
		if errHome != nil {
			path = "/"
		} else {
			path = home
		}
	}

	entries, errRead := os.ReadDir(path)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return nil, fmt.Errorf("directory does not exist: %s", path)
		}
		return nil, fmt.Errorf("reading directory: %w", errRead)
	}

	var result []directoryEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		result = append(result, directoryEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(path, entry.Name()),
			IsDir: true,
		})
	}

	return result, nil
}
