package main

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdhocStatusService(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.adhocStatusService(wsDir)
	assert.False(t, result.Running)
	assert.Empty(t, result.Output)
	assert.Equal(t, "adhoc status", result.Message)
}

func TestAdhocStartServiceEmptyPrompt(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.adhocStartService(wsDir, "", "claude-opus-4")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "required")
}

func TestAdhocStartServiceEmptyModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.adhocStartService(wsDir, "do something", "")
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "required")
}

func TestAdhocStopService(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.adhocStopService(wsDir)
	assert.False(t, result.Running)
	assert.Equal(t, "ad-hoc stopped", result.Message)
}

func TestAdhocStartServiceAlreadyRunningReturnsExisting(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-adhoc-running")
	st := server.getAdhocState(wsDir)
	st.mu.Lock()
	st.running = true
	st.output.WriteString("test output")
	st.mu.Unlock()
	result := server.adhocStartService(wsDir, "prompt", "model")
	assert.Nil(t, result.Error)
	assert.True(t, result.Running)
	assert.Contains(t, result.Output, "test output")
}

func TestGetAdhocStateCreation(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "adhoc-create")
	st1 := srv.getAdhocState(wsDir)
	st2 := srv.getAdhocState(wsDir)
	assert.Same(t, st1, st2)
}

func TestAdhocStopNotRunning(t *testing.T) {
	st := &adhocPromptState{}
	st.stop()
	assert.False(t, st.running)
}

func TestAdhocStopAlreadyStopped(t *testing.T) {
	st := &adhocPromptState{}
	st.mu.Lock()
	st.running = false
	st.mu.Unlock()
	st.stop()
	assert.False(t, st.running)
}

func TestLockedWriterStripsANSI(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := &lockedWriter{mu: &mu, buf: &buf}
	_, err := w.Write([]byte("\x1b[31mred text\x1b[0m"))
	assert.NoError(t, err)
	assert.Equal(t, "red text", buf.String())
}

func TestLockedWriterPlainTextPassthrough(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := &lockedWriter{mu: &mu, buf: &buf}
	n, err := w.Write([]byte("plain text"))
	assert.NoError(t, err)
	assert.Equal(t, 10, n)
	assert.Equal(t, "plain text", buf.String())
}

func TestAnsiEscapePatternMatches(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"colorCode", "\x1b[31mhello\x1b[0m", "hello"},
		{"boldCode", "\x1b[1mbold\x1b[22m", "bold"},
		{"noAnsi", "plain text", "plain text"},
		{"mixed", "before\x1b[32mgreen\x1b[0mafter", "beforegreenafter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansiEscapePattern.ReplaceAllString(tt.input, "")
			assert.Equal(t, tt.want, got)
		})
	}
}
