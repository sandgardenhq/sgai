package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionIDCaptureWriterCapturesExactPluginLine(t *testing.T) {
	writer := &sessionIDCaptureWriter{}

	n, errWrite := writer.Write([]byte("starting\n{\"sessionID\":\"session-1\"}\ncontinuing\n"))

	require.NoError(t, errWrite)
	assert.Equal(t, len("starting\n{\"sessionID\":\"session-1\"}\ncontinuing\n"), n)
	assert.Equal(t, "session-1", writer.sessionID)
}

func TestSessionIDCaptureWriterCapturesChunkedPluginLine(t *testing.T) {
	writer := &sessionIDCaptureWriter{}

	_, errFirst := writer.Write([]byte("{\"session"))
	_, errSecond := writer.Write([]byte("ID\":\"session-2\"}\n"))

	require.NoError(t, errFirst)
	require.NoError(t, errSecond)
	assert.Equal(t, "session-2", writer.sessionID)
}

func TestSessionIDCaptureWriterIgnoresNestedOrExtraFields(t *testing.T) {
	writer := &sessionIDCaptureWriter{}

	_, errWrite := writer.Write([]byte("{\"type\":\"event\",\"sessionID\":\"child-session\"}\n{\"sessionID\":\"parent-session\"}\n"))

	require.NoError(t, errWrite)
	assert.Equal(t, "parent-session", writer.sessionID)
}

func TestSessionIDCaptureWriterKeepsFirstSessionForContinuation(t *testing.T) {
	writer := &sessionIDCaptureWriter{}

	_, errWrite := writer.Write([]byte("{\"sessionID\":\"parent-session\"}\n{\"sessionID\":\"child-session\"}\n"))

	require.NoError(t, errWrite)
	assert.Equal(t, "parent-session", writer.sessionID)
}

func TestSessionIDCaptureWriterPrintsDetectedSessionID(t *testing.T) {
	var buf bytes.Buffer
	writer := &sessionIDCaptureWriter{detectedWriter: &buf}

	_, errWrite := writer.Write([]byte("{\"sessionID\":\"session-3\"}\n{\"sessionID\":\"session-4\"}\n"))

	require.NoError(t, errWrite)
	assert.Equal(t, "session-3", writer.sessionID)
	assert.Equal(t, "Detected sessionID: session-3\nDetected sessionID: session-4\n", buf.String())
}

func TestSessionIDCaptureWriterHidesSessionIDControlLines(t *testing.T) {
	var detected bytes.Buffer
	var passthrough bytes.Buffer
	writer := &sessionIDCaptureWriter{detectedWriter: &detected, passthrough: &passthrough}

	_, errWrite := writer.Write([]byte("start\n{\"sessionID\":\"parent-session\"}\n\n> header\n{\"sessionID\":\"child-session\",\"agent\":\"go-reviewer\"}\n"))

	require.NoError(t, errWrite)
	assert.Equal(t, "parent-session", writer.sessionID)
	assert.Equal(t, "Detected sessionID: parent-session\nDetected sessionID for go-reviewer: child-session\n", detected.String())
	assert.Equal(t, "start\n\n> header\n", passthrough.String())
}
