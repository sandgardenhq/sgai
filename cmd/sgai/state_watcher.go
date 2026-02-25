package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
)

type workspaceStateSnapshot struct {
	modTime             time.Time
	status              string
	needsInput          bool
	progressLen         int
	todosHash           string
	messagesHash        string
	goalModTime         time.Time
	goalHash            string
	summaryGenTriggered bool
}

func (s *Server) startStateWatcher() {
	go s.stateWatcherLoop(s.shutdownCtx)
}

func (s *Server) stateWatcherLoop(ctx context.Context) {
	snapshots := make(map[string]workspaceStateSnapshot)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollWorkspaceStates(snapshots)
		}
	}
}

func (s *Server) pollWorkspaceStates(snapshots map[string]workspaceStateSnapshot) {
	groups, errScan := s.scanWorkspaceGroups()
	if errScan != nil {
		return
	}

	activeWorkspaces := make(map[string]bool)

	for _, grp := range groups {
		s.checkWorkspaceState(grp.Root.Directory, snapshots, activeWorkspaces)
		for _, fork := range grp.Forks {
			s.checkWorkspaceState(fork.Directory, snapshots, activeWorkspaces)
		}
	}

	for dir := range snapshots {
		if !activeWorkspaces[dir] {
			delete(snapshots, dir)
		}
	}
}

func (s *Server) checkWorkspaceState(dir string, snapshots map[string]workspaceStateSnapshot, activeWorkspaces map[string]bool) {
	activeWorkspaces[dir] = true
	stPath := statePath(dir)
	goalPath := filepath.Join(dir, "GOAL.md")

	info, errStat := os.Stat(stPath)
	if errStat != nil {
		delete(snapshots, dir)
		return
	}

	goalInfo, errGoalStat := os.Stat(goalPath)
	if errGoalStat != nil {
		goalInfo = nil
	}

	prev, hasPrev := snapshots[dir]
	if hasPrev && info.ModTime().Equal(prev.modTime) {
		goalChanged := false
		if goalInfo != nil {
			goalChanged = !goalInfo.ModTime().Equal(prev.goalModTime)
		}
		if !goalChanged {
			return
		}
	}

	wfState, errLoad := state.Load(stPath)
	if errLoad != nil {
		return
	}

	current := buildStateSnapshot(info.ModTime(), wfState, goalInfo)

	if !hasPrev {
		if s.summaryGen != nil && wfState.Summary == "" && !wfState.SummaryManual {
			current.summaryGenTriggered = true
			s.summaryGen.trigger(dir)
		}
		snapshots[dir] = current
		return
	}

	s.emitStateChangeEvents(dir, prev, current)

	if s.summaryGen != nil && wfState.Summary == "" && !wfState.SummaryManual && !prev.summaryGenTriggered {
		current.summaryGenTriggered = true
		s.summaryGen.trigger(dir)
	} else if prev.summaryGenTriggered {
		current.summaryGenTriggered = true
	}

	snapshots[dir] = current
}

func buildStateSnapshot(modTime time.Time, wfState state.Workflow, goalInfo os.FileInfo) workspaceStateSnapshot {
	snapshot := workspaceStateSnapshot{
		modTime:      modTime,
		status:       wfState.Status,
		needsInput:   wfState.NeedsHumanInput(),
		progressLen:  len(wfState.Progress),
		todosHash:    hashTodos(wfState.ProjectTodos, wfState.Todos),
		messagesHash: hashMessages(wfState.Messages),
	}
	if goalInfo != nil {
		snapshot.goalModTime = goalInfo.ModTime()
		snapshot.goalHash = hashGoalFile(goalInfo)
	}
	return snapshot
}

func (s *Server) emitStateChangeEvents(workspacePath string, prev, current workspaceStateSnapshot) {
	changed := prev.status != current.status ||
		prev.needsInput != current.needsInput ||
		current.progressLen > prev.progressLen ||
		prev.todosHash != current.todosHash ||
		prev.messagesHash != current.messagesHash ||
		prev.goalHash != current.goalHash

	if prev.goalHash != current.goalHash && s.summaryGen != nil {
		s.summaryGen.trigger(workspacePath)
	}

	if changed {
		s.notifyStateChange()
	}
}

func hashTodos(projectTodos, agentTodos []state.TodoItem) string {
	h := sha256.New()
	data, errMarshal := json.Marshal(struct {
		Project []state.TodoItem `json:"p"`
		Agent   []state.TodoItem `json:"a"`
	}{projectTodos, agentTodos})
	if errMarshal != nil {
		return ""
	}
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func hashMessages(messages []state.Message) string {
	h := sha256.New()
	data, errMarshal := json.Marshal(messages)
	if errMarshal != nil {
		return ""
	}
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func hashGoalFile(goalInfo os.FileInfo) string {
	if goalInfo == nil {
		return ""
	}
	h := sha256.New()
	_, _ = fmt.Fprint(h, goalInfo.ModTime().String())
	_, _ = fmt.Fprintf(h, "%d", goalInfo.Size())
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
