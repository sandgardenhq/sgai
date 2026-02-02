package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type workspaceInfo struct {
	Directory    string
	DirName      string
	IsRoot       bool
	Running      bool
	NeedsInput   bool
	InProgress   bool
	HasWorkspace bool
}

type workspaceGroup struct {
	Root  workspaceInfo
	Forks []workspaceInfo
}

func hasJJDirectory(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".jj"))
	return err == nil && info.IsDir()
}

func hassgaiDirectory(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".sgai"))
	return err == nil && info.IsDir()
}

func isRootWorkspace(dir string) bool {
	repoPath := filepath.Join(dir, ".jj", "repo")
	info, err := os.Stat(repoPath)
	return err == nil && info.IsDir()
}

func isForkWorkspace(dir string) bool {
	repoPath := filepath.Join(dir, ".jj", "repo")
	info, err := os.Stat(repoPath)
	return err == nil && !info.IsDir()
}

func getRootWorkspacePath(forkDir string) string {
	repoPath := filepath.Join(forkDir, ".jj", "repo")
	content, err := os.ReadFile(repoPath)
	if err != nil {
		return ""
	}
	rootPath := strings.TrimSpace(string(content))
	if rootPath == "" {
		return ""
	}
	jjDir := filepath.Dir(rootPath)
	return filepath.Dir(jjDir)
}

func (s *Server) getWorkspaceStatus(dir string) (running bool, needsInput bool) {
	s.mu.Lock()
	sess := s.sessions[dir]
	s.mu.Unlock()
	if sess != nil {
		sess.mu.Lock()
		running = sess.running
		sess.mu.Unlock()
	}

	wfState, _ := state.Load(statePath(dir))
	needsInput = wfState.NeedsHumanInput()
	return running, needsInput
}

func (s *Server) createWorkspaceInfo(dir, dirName string, isRoot, hasWorkspace bool) workspaceInfo {
	running, needsInput := s.getWorkspaceStatus(dir)
	inProgress := running || needsInput || s.wasEverStarted(dir)

	return workspaceInfo{
		Directory:    dir,
		DirName:      dirName,
		IsRoot:       isRoot,
		Running:      running,
		NeedsInput:   needsInput,
		InProgress:   inProgress,
		HasWorkspace: hasWorkspace,
	}
}

func (s *Server) wasEverStarted(dir string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.everStartedDirs[dir]
}

func (s *Server) scanWorkspaceGroups() ([]workspaceGroup, error) {
	projects, err := scanForProjects(s.rootDir)
	if err != nil {
		return nil, err
	}

	rootMap := make(map[string]*workspaceGroup)
	var standaloneGroups []workspaceGroup

	for _, proj := range projects {
		if !hasJJDirectory(proj.Directory) {
			standaloneGroups = append(standaloneGroups, workspaceGroup{
				Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
			})
			continue
		}

		if isRootWorkspace(proj.Directory) {
			if _, exists := rootMap[proj.Directory]; !exists {
				rootMap[proj.Directory] = &workspaceGroup{
					Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, true, proj.HasWorkspace),
				}
			}
			continue
		}

		if isForkWorkspace(proj.Directory) {
			rootPath := getRootWorkspacePath(proj.Directory)
			if rootPath == "" {
				standaloneGroups = append(standaloneGroups, workspaceGroup{
					Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
				})
				continue
			}

			if existing, exists := rootMap[rootPath]; exists {
				existing.Forks = append(existing.Forks, s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace))
			} else {
				rootMap[rootPath] = &workspaceGroup{
					Root:  s.createWorkspaceInfo(rootPath, filepath.Base(rootPath), true, hassgaiDirectory(rootPath)),
					Forks: []workspaceInfo{s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace)},
				}
			}
			continue
		}

		standaloneGroups = append(standaloneGroups, workspaceGroup{
			Root: s.createWorkspaceInfo(proj.Directory, proj.DirName, false, proj.HasWorkspace),
		})
	}

	var groups []workspaceGroup
	for _, grp := range rootMap {
		groups = append(groups, *grp)
	}
	groups = append(groups, standaloneGroups...)

	slices.SortFunc(groups, func(a, b workspaceGroup) int {
		return strings.Compare(strings.ToLower(a.Root.DirName), strings.ToLower(b.Root.DirName))
	})

	return groups, nil
}

func collectInProgressWorkspaces(groups []workspaceGroup) []workspaceInfo {
	var result []workspaceInfo
	for _, grp := range groups {
		if grp.Root.InProgress {
			result = append(result, grp.Root)
		}
		for _, fork := range grp.Forks {
			if fork.InProgress {
				result = append(result, fork)
			}
		}
	}
	return result
}

func hasAnyNeedsInput(workspaces []workspaceInfo) bool {
	for _, w := range workspaces {
		if w.NeedsInput {
			return true
		}
	}
	return false
}

func (s *Server) resolveWorkspaceNameToPath(workspaceName string) string {
	if workspaceName == "" {
		return ""
	}

	groups, err := s.scanWorkspaceGroups()
	if err != nil {
		return ""
	}

	for _, grp := range groups {
		if grp.Root.DirName == workspaceName {
			return grp.Root.Directory
		}
		for _, fork := range grp.Forks {
			if fork.DirName == workspaceName {
				return fork.Directory
			}
		}
	}

	return ""
}
