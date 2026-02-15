package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"
)

type slackSession struct {
	WorkspaceDir        string `json:"workspaceDir"`
	SessionID           string `json:"sessionId"`
	ChannelID           string `json:"channelId"`
	ThreadTS            string `json:"threadTs"`
	CreatedAt           string `json:"createdAt"`
	EventUpdatesEnabled bool   `json:"eventUpdatesEnabled"`
}

type sessionDB struct {
	mu       sync.Mutex
	filePath string
	sessions map[string]slackSession
}

func defaultSessionDBPath() string {
	return filepath.Join(xdg.ConfigHome, "sgai", "slack-sessions.json")
}

func newSessionDB(filePath string) *sessionDB {
	return &sessionDB{
		filePath: filePath,
		sessions: make(map[string]slackSession),
	}
}

func sessionKey(channelID, threadTS string) string {
	return channelID + ":" + threadTS
}

func (db *sessionDB) load() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.loadLocked()
}

func (db *sessionDB) loadLocked() error {
	data, err := os.ReadFile(db.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			db.sessions = make(map[string]slackSession)
			return nil
		}
		return fmt.Errorf("reading session database %s: %w", db.filePath, err)
	}

	var sessions map[string]slackSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("parsing session database %s: %w", db.filePath, err)
	}

	if sessions == nil {
		sessions = make(map[string]slackSession)
	}
	db.sessions = sessions
	return nil
}

func (db *sessionDB) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(db.filePath), 0755); err != nil {
		return fmt.Errorf("creating session database directory: %w", err)
	}

	data, err := json.MarshalIndent(db.sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding session database: %w", err)
	}

	return os.WriteFile(db.filePath, data, 0644)
}

func (db *sessionDB) get(channelID, threadTS string) (slackSession, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	sess, ok := db.sessions[sessionKey(channelID, threadTS)]
	return sess, ok
}

func (db *sessionDB) put(channelID, threadTS string, sess slackSession) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.sessions[sessionKey(channelID, threadTS)] = sess
	return db.saveLocked()
}

func (db *sessionDB) delete(channelID, threadTS string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	delete(db.sessions, sessionKey(channelID, threadTS))
	return db.saveLocked()
}

func (db *sessionDB) allConnected() map[string]slackSession {
	db.mu.Lock()
	defer db.mu.Unlock()
	result := make(map[string]slackSession, len(db.sessions))
	for k, v := range db.sessions {
		if v.WorkspaceDir != "" {
			result[k] = v
		}
	}
	return result
}

func (db *sessionDB) updateEventUpdates(channelID, threadTS string, enabled bool) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	key := sessionKey(channelID, threadTS)
	sess, ok := db.sessions[key]
	if !ok {
		return fmt.Errorf("session not found: %s", key)
	}
	sess.EventUpdatesEnabled = enabled
	db.sessions[key] = sess
	return db.saveLocked()
}

func (db *sessionDB) updateSessionID(channelID, threadTS, sessionID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	key := sessionKey(channelID, threadTS)
	sess, ok := db.sessions[key]
	if !ok {
		return fmt.Errorf("session not found: %s", key)
	}
	sess.SessionID = sessionID
	db.sessions[key] = sess
	return db.saveLocked()
}
