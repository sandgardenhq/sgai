package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestAdhocStartServiceWrapsStartError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-adhoc-start-error")

	result := server.adhocStartService(wsDir, "prompt", "model")

	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "failed to start command")
	assert.True(t, errors.Unwrap(result.Error) != nil)
}

func TestAdhocStartServiceStripsANSIFromTerminalPassthrough(t *testing.T) {
	binDir := t.TempDir()
	opencodePath := filepath.Join(binDir, "opencode")
	script := "#!/bin/sh\nprintf '\\033[31mstdout red\\033[0m'\nprintf '\\033[32mstderr green\\033[0m' >&2\n"
	require.NoError(t, os.WriteFile(opencodePath, []byte(script), 0o755))
	t.Setenv("PATH", binDir)

	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-adhoc-ansi-terminal")

	stdout, stderr := captureStdoutStderr(t, func() {
		result := server.adhocStartService(wsDir, "prompt", "model")
		require.NoError(t, result.Error)
		waitForAdhocDone(t, server.getAdhocState(wsDir))
	})

	assert.NotContains(t, stdout, "\x1b[")
	assert.NotContains(t, stderr, "\x1b[")
	assert.Contains(t, stdout, "stdout red")
	assert.Contains(t, stderr, "stderr green")

	status := server.adhocStatusService(wsDir)
	assert.NotContains(t, status.Output, "\x1b[")
	assert.Contains(t, status.Output, "stdout red")
	assert.Contains(t, status.Output, "stderr green")
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

func TestLockedWriterStripsANSISplitAcrossWrites(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := &lockedWriter{mu: &mu, buf: &buf}

	chunks := []string{
		"start \x1b[",
		"31mred\x1b]8;;https://example.com",
		"\x07link\x1b]8;;",
		"\x07 \x1b[?25",
		"lend\x1b[0",
		"m",
	}

	for _, chunk := range chunks {
		n, errWrite := w.Write([]byte(chunk))
		require.NoError(t, errWrite)
		assert.Equal(t, len(chunk), n)
	}

	assert.Equal(t, "start redlink end", buf.String())
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

func TestLockedWriterPassesThroughSessionIDLine(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := &lockedWriter{mu: &mu, buf: &buf}

	_, errWrite := w.Write([]byte(`{"sessionID":"adhoc-output-session"}` + "\n"))

	assert.NoError(t, errWrite)
	assert.Equal(t, `{"sessionID":"adhoc-output-session"}`+"\n", buf.String())
}

func TestLockedWriterPassesThroughChunkedOutput(t *testing.T) {
	var mu sync.Mutex
	var buf bytes.Buffer
	w := &lockedWriter{mu: &mu, buf: &buf}

	_, errFirst := w.Write([]byte("chunked "))
	_, errSecond := w.Write([]byte("human output\n"))

	assert.NoError(t, errFirst)
	assert.NoError(t, errSecond)
	assert.Equal(t, "chunked human output\n", buf.String())
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
			got := string(stripANSIBytes([]byte(tt.input)))
			assert.Equal(t, tt.want, got)
		})
	}
}

func captureStdoutStderr(t *testing.T, fn func()) (string, string) {
	t.Helper()
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	stdoutReader, stdoutWriter, errStdoutPipe := os.Pipe()
	require.NoError(t, errStdoutPipe)
	stderrReader, stderrWriter, errStderrPipe := os.Pipe()
	require.NoError(t, errStderrPipe)
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	stdoutDone := make(chan string, 1)
	stderrDone := make(chan string, 1)
	go readPipeToString(stdoutReader, stdoutDone)
	go readPipeToString(stderrReader, stderrDone)

	fn()

	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())
	return <-stdoutDone, <-stderrDone
}

func readPipeToString(r *os.File, done chan<- string) {
	data, errRead := io.ReadAll(r)
	if errRead != nil {
		done <- errRead.Error()
		return
	}
	done <- string(data)
}

func waitForAdhocDone(t *testing.T, st *adhocPromptState) {
	t.Helper()
	st.mu.Lock()
	done := st.done
	st.mu.Unlock()
	if done == nil {
		return
	}
	<-done
}
