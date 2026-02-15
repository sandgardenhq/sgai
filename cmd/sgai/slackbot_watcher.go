package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/slack-go/slack"
)

type slackNotificationWatcher struct {
	sessions   *sessionDB
	slackAPI   *slack.Client
	pollTicker *time.Ticker
	snapshots  map[string]slackWatcherSnapshot
}

type slackWatcherSnapshot struct {
	modTime     time.Time
	status      string
	needsInput  bool
	progressLen int
}

func newSlackNotificationWatcher(sessions *sessionDB, api *slack.Client) *slackNotificationWatcher {
	return &slackNotificationWatcher{
		sessions:  sessions,
		slackAPI:  api,
		snapshots: make(map[string]slackWatcherSnapshot),
	}
}

func (w *slackNotificationWatcher) run(ctx context.Context) {
	w.pollTicker = time.NewTicker(2500 * time.Millisecond)
	defer w.pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.pollTicker.C:
			w.poll()
		}
	}
}

func (w *slackNotificationWatcher) poll() {
	connected := w.sessions.allConnected()
	activeKeys := make(map[string]bool)

	for key, sess := range connected {
		activeKeys[key] = true
		w.checkWorkspace(key, sess)
	}

	for key := range w.snapshots {
		if !activeKeys[key] {
			delete(w.snapshots, key)
		}
	}
}

func (w *slackNotificationWatcher) checkWorkspace(key string, sess slackSession) {
	stPath := statePath(sess.WorkspaceDir)
	info, errStat := os.Stat(stPath)
	if errStat != nil {
		delete(w.snapshots, key)
		return
	}

	prev, hasPrev := w.snapshots[key]
	if hasPrev && info.ModTime().Equal(prev.modTime) {
		return
	}

	wfState, errLoad := state.Load(stPath)
	if errLoad != nil {
		return
	}

	current := slackWatcherSnapshot{
		modTime:     info.ModTime(),
		status:      wfState.Status,
		needsInput:  wfState.NeedsHumanInput(),
		progressLen: len(wfState.Progress),
	}
	w.snapshots[key] = current

	if !hasPrev {
		return
	}

	w.detectAndNotify(sess, prev, current, wfState)
}

func (w *slackNotificationWatcher) detectAndNotify(sess slackSession, prev, current slackWatcherSnapshot, wfState state.Workflow) {
	wsName := workspaceDirName(sess.WorkspaceDir)

	if !prev.needsInput && current.needsInput {
		var msg strings.Builder
		msg.WriteString(fmt.Sprintf(":raised_hand: *%s* needs your input", wsName))
		if wfState.HumanMessage != "" {
			msg.WriteString(fmt.Sprintf("\n> %s", truncateMessage(wfState.HumanMessage, 200)))
		}
		if wfState.MultiChoiceQuestion != nil {
			for _, q := range wfState.MultiChoiceQuestion.Questions {
				msg.WriteString(fmt.Sprintf("\nChoices: %s", strings.Join(q.Choices, " | ")))
			}
		}
		w.sendMessage(sess, msg.String())
	}

	if prev.status != state.StatusComplete && current.status == state.StatusComplete {
		w.sendMessage(sess, fmt.Sprintf(":white_check_mark: *%s* has completed", wsName))
	}

	if prev.status == "" && current.status == state.StatusWorking {
		w.sendMessage(sess, fmt.Sprintf(":rocket: *%s* has started", wsName))
	}

	if sess.EventUpdatesEnabled && current.progressLen > prev.progressLen {
		newEntries := wfState.Progress[prev.progressLen:]
		for _, entry := range newEntries {
			w.sendMessage(sess, fmt.Sprintf("`[%s]` %s", entry.Agent, entry.Description))
		}
	}
}

func (w *slackNotificationWatcher) sendMessage(sess slackSession, text string) {
	if sess.ChannelID == "" {
		return
	}
	opts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(sess.ThreadTS),
	}
	_, _, errPost := w.slackAPI.PostMessage(sess.ChannelID, opts...)
	if errPost != nil {
		log.Println("slack notification watcher: posting message:", errPost)
	}
}

func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "..."
}

func workspaceDirName(dir string) string {
	return filepath.Base(dir)
}
