package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionDBCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.json")
	db := newSessionDB(dbPath)

	if err := db.load(); err != nil {
		t.Fatalf("load empty database: %v", err)
	}

	sess := slackSession{
		WorkspaceDir: "/workspace/test",
		SessionID:    "sess-123",
		ChannelID:    "C123",
		CreatedAt:    "2026-01-01T00:00:00Z",
	}
	if err := db.put("C123", "1234567890.123456", sess); err != nil {
		t.Fatalf("put session: %v", err)
	}

	got, ok := db.get("C123", "1234567890.123456")
	if !ok {
		t.Fatal("session not found after put")
	}
	if got.WorkspaceDir != "/workspace/test" {
		t.Errorf("workspaceDir = %q, want %q", got.WorkspaceDir, "/workspace/test")
	}
	if got.SessionID != "sess-123" {
		t.Errorf("sessionID = %q, want %q", got.SessionID, "sess-123")
	}

	_, ok = db.get("C999", "0000000000.000000")
	if ok {
		t.Error("non-existent session should not be found")
	}

	if err := db.delete("C123", "1234567890.123456"); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	_, ok = db.get("C123", "1234567890.123456")
	if ok {
		t.Error("session should not exist after delete")
	}
}

func TestSessionDBPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.json")

	db1 := newSessionDB(dbPath)
	if err := db1.load(); err != nil {
		t.Fatalf("load db1: %v", err)
	}

	sess := slackSession{
		WorkspaceDir: "/workspace/alpha",
		SessionID:    "sess-abc",
		ChannelID:    "C100",
		CreatedAt:    "2026-01-01T00:00:00Z",
	}
	if err := db1.put("C100", "1111111111.000000", sess); err != nil {
		t.Fatalf("put session: %v", err)
	}

	db2 := newSessionDB(dbPath)
	if err := db2.load(); err != nil {
		t.Fatalf("load db2: %v", err)
	}

	got, ok := db2.get("C100", "1111111111.000000")
	if !ok {
		t.Fatal("session not found in reloaded database")
	}
	if got.WorkspaceDir != "/workspace/alpha" {
		t.Errorf("workspaceDir = %q, want %q", got.WorkspaceDir, "/workspace/alpha")
	}
}

func TestSessionDBLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nonexistent", "sessions.json")
	db := newSessionDB(dbPath)

	if err := db.load(); err != nil {
		t.Fatalf("loading non-existent database should succeed: %v", err)
	}

	_, ok := db.get("C123", "1234567890.123456")
	if ok {
		t.Error("non-existent database should have no sessions")
	}
}

func TestSessionDBAllConnected(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.json")
	db := newSessionDB(dbPath)
	if err := db.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := db.put("C1", "ts1", slackSession{
		WorkspaceDir: "/workspace/alpha",
		ChannelID:    "C1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.put("C2", "ts2", slackSession{
		WorkspaceDir: "/workspace/beta",
		ChannelID:    "C2",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.put("C3", "ts3", slackSession{
		WorkspaceDir: "",
		ChannelID:    "C3",
	}); err != nil {
		t.Fatal(err)
	}

	connected := db.allConnected()
	if len(connected) != 2 {
		t.Errorf("expected 2 connected sessions, got %d", len(connected))
	}
}

func TestSessionDBUpdateEventUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.json")
	db := newSessionDB(dbPath)
	if err := db.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := db.put("C1", "ts1", slackSession{
		WorkspaceDir:        "/workspace/test",
		ChannelID:           "C1",
		EventUpdatesEnabled: false,
	}); err != nil {
		t.Fatal(err)
	}

	if err := db.updateEventUpdates("C1", "ts1", true); err != nil {
		t.Fatalf("updateEventUpdates: %v", err)
	}

	got, ok := db.get("C1", "ts1")
	if !ok {
		t.Fatal("session not found")
	}
	if !got.EventUpdatesEnabled {
		t.Error("event updates should be enabled")
	}

	if err := db.updateEventUpdates("C999", "ts999", true); err == nil {
		t.Error("updating non-existent session should return error")
	}
}

func TestSessionDBUpdateSessionID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.json")
	db := newSessionDB(dbPath)
	if err := db.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := db.put("C1", "ts1", slackSession{
		WorkspaceDir: "/workspace/test",
		ChannelID:    "C1",
		SessionID:    "old-session",
	}); err != nil {
		t.Fatal(err)
	}

	if err := db.updateSessionID("C1", "ts1", "new-session"); err != nil {
		t.Fatalf("updateSessionID: %v", err)
	}

	got, ok := db.get("C1", "ts1")
	if !ok {
		t.Fatal("session not found")
	}
	if got.SessionID != "new-session" {
		t.Errorf("sessionID = %q, want %q", got.SessionID, "new-session")
	}
}

func TestSessionDBCorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sessions.json")

	if err := os.WriteFile(dbPath, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	db := newSessionDB(dbPath)
	if err := db.load(); err == nil {
		t.Error("loading corrupt database should return error")
	}
}

func TestSessionKey(t *testing.T) {
	key := sessionKey("C123", "1234567890.123456")
	if key != "C123:1234567890.123456" {
		t.Errorf("sessionKey = %q, want %q", key, "C123:1234567890.123456")
	}
}
