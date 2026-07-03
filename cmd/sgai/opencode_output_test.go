package main

import (
	"bytes"
	"io"
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

func TestAnsiStrippingWriterStripsReferenceSequences(t *testing.T) {
	var buf bytes.Buffer
	writer := &ansiStrippingWriter{w: &buf}
	input := "\x1b[31mred\x1b[0m \x1b]8;;https://example.com\x07link\x1b]8;;\x07 \x1b[?25lhidden\x1b[?25h"

	n, errWrite := writer.Write([]byte(input))

	require.NoError(t, errWrite)
	assert.Equal(t, len(input), n)
	assert.Equal(t, "red link hidden", buf.String())
}

func TestAnsiStrippingWriterStripsSequencesSplitAcrossWrites(t *testing.T) {
	var buf bytes.Buffer
	writer := &ansiStrippingWriter{w: &buf}

	chunks := []string{
		"start \x1b[",
		"31mred\x1b]8;;https://example.com",
		"\x07link\x1b]8;;",
		"\x07 \x1b[?25",
		"lend\x1b[0",
		"m",
	}

	for _, chunk := range chunks {
		n, errWrite := writer.Write([]byte(chunk))
		require.NoError(t, errWrite)
		assert.Equal(t, len(chunk), n)
	}

	assert.Equal(t, "start redlink end", buf.String())
}

func TestAnsiStrippingWriterStripsNonCSISequencesSplitAcrossWrites(t *testing.T) {
	var buf bytes.Buffer
	writer := &ansiStrippingWriter{w: &buf}

	chunks := []string{
		"start \x1b",
		"(",
		"0box\x1b(",
		"B ascii \x1b#",
		"8done",
	}

	for _, chunk := range chunks {
		n, errWrite := writer.Write([]byte(chunk))
		require.NoError(t, errWrite)
		assert.Equal(t, len(chunk), n)
	}

	assert.Equal(t, "start box ascii done", buf.String())
}

func TestAnsiStrippingWriterShortWriteReturnsError(t *testing.T) {
	writer := &ansiStrippingWriter{w: shortWriter{n: 2}}

	n, errWrite := writer.Write([]byte("plain"))

	assert.Equal(t, 0, n)
	assert.ErrorIs(t, errWrite, io.ErrShortWrite)
}

func TestPrefixWriterPreservesLogicalLineAcrossChunkedWrites(t *testing.T) {
	var buf bytes.Buffer
	writer := &prefixWriter{prefix: " agent ", w: &buf}

	_, errFirst := writer.Write([]byte("split"))
	_, errSecond := writer.Write([]byte(" logical"))
	_, errThird := writer.Write([]byte(" line\nnext line\n"))

	require.NoError(t, errFirst)
	require.NoError(t, errSecond)
	require.NoError(t, errThird)
	assert.Regexp(t, `^\[[^\]]+\] agent split logical line\n\[[^\]]+\] agent next line\n$`, buf.String())
}

func TestPrefixWriterFlushWritesTrailingLogicalLine(t *testing.T) {
	var buf bytes.Buffer
	writer := &prefixWriter{prefix: " agent ", w: &buf}

	_, errWrite := writer.Write([]byte("unterminated line"))
	errFlush := writer.Flush()

	require.NoError(t, errWrite)
	require.NoError(t, errFlush)
	assert.Regexp(t, `^\[[^\]]+\] agent unterminated line\n$`, buf.String())
}

func TestSessionIDCaptureWriterCapturesAnsiWrappedPluginLineAfterStripping(t *testing.T) {
	var detected bytes.Buffer
	var passthrough bytes.Buffer
	capture := &sessionIDCaptureWriter{detectedWriter: &detected, passthrough: &passthrough}
	writer := &ansiStrippingWriter{w: capture}

	_, errWrite := writer.Write([]byte("\x1b[32m{\"sessionID\":\"ansi-session\"}\x1b[0m\nvisible\x1b[31m text\x1b[0m\n"))

	require.NoError(t, errWrite)
	assert.Equal(t, "ansi-session", capture.sessionID)
	assert.Equal(t, "Detected sessionID: ansi-session\n", detected.String())
	assert.Equal(t, "visible text\n", passthrough.String())
}

type shortWriter struct {
	n int
}

func (w shortWriter) Write(data []byte) (int, error) {
	if w.n > len(data) {
		return len(data), nil
	}
	return w.n, nil
}
