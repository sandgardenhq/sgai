package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircularLogBuffer(t *testing.T) {
	buf := newCircularLogBuffer()
	assert.NotNil(t, buf)
	assert.NotNil(t, buf.ring)
	assert.Equal(t, 0, buf.size)
}

func TestCircularLogBufferAdd(t *testing.T) {
	buf := newCircularLogBuffer()

	buf.add(logLine{prefix: "test", text: "line1"})
	assert.Equal(t, 1, buf.size)

	buf.add(logLine{prefix: "test", text: "line2"})
	assert.Equal(t, 2, buf.size)
}

func TestCircularLogBufferLines(t *testing.T) {
	tests := []struct {
		name     string
		addLines []logLine
		expected []logLine
	}{
		{
			name:     "emptyBuffer",
			addLines: []logLine{},
			expected: nil,
		},
		{
			name: "singleLine",
			addLines: []logLine{
				{prefix: "test", text: "line1"},
			},
			expected: []logLine{
				{prefix: "test", text: "line1"},
			},
		},
		{
			name: "multipleLines",
			addLines: []logLine{
				{prefix: "test", text: "line1"},
				{prefix: "test", text: "line2"},
				{prefix: "test", text: "line3"},
			},
			expected: []logLine{
				{prefix: "test", text: "line1"},
				{prefix: "test", text: "line2"},
				{prefix: "test", text: "line3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := newCircularLogBuffer()
			for _, line := range tt.addLines {
				buf.add(line)
			}

			lines := buf.lines()
			assert.Equal(t, len(tt.expected), len(lines))
			for i, expected := range tt.expected {
				if i < len(lines) {
					assert.Equal(t, expected.text, lines[i].text)
				}
			}
		})
	}
}

func TestNewRingWriter(t *testing.T) {
	rw := newRingWriter()
	assert.NotNil(t, rw)
	assert.NotNil(t, rw.ring)
	assert.Equal(t, 0, rw.size)
}

func TestRingWriterWrite(t *testing.T) {
	rw := newRingWriter()

	n, err := rw.Write([]byte("line1\n"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 2, rw.size)

	n, err = rw.Write([]byte("line2\n"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 4, rw.size)
}

func TestRingWriterWritePartial(t *testing.T) {
	rw := newRingWriter()

	n, err := rw.Write([]byte("partial"))
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Equal(t, 0, rw.size)
	assert.Equal(t, []byte("partial"), rw.partial)

	n, err = rw.Write([]byte(" line\n"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, 2, rw.size)
	assert.Nil(t, rw.partial)
}

func TestRingWriterDump(t *testing.T) {
	rw := newRingWriter()

	_, _ = rw.Write([]byte("line1\n"))
	_, _ = rw.Write([]byte("line2\n"))

	var buf bytes.Buffer
	rw.dump(&buf)

	output := buf.String()
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
}

func TestRingWriterDumpEmpty(t *testing.T) {
	rw := newRingWriter()

	var buf bytes.Buffer
	rw.dump(&buf)

	assert.Equal(t, "", buf.String())
}

func TestRingWriterDumpPartial(t *testing.T) {
	rw := newRingWriter()

	_, _ = rw.Write([]byte("line1\n"))
	_, _ = rw.Write([]byte("partial"))

	var buf bytes.Buffer
	rw.dump(&buf)

	output := buf.String()
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "partial")
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "singleLine",
			input:    "line1",
			expected: []string{"line1"},
		},
		{
			name:     "multipleLines",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "trailingNewline",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAgentOutputWriter(t *testing.T) {
	tests := []struct {
		name        string
		base        *bytes.Buffer
		extra       []io.Writer
		expectMulti bool
	}{
		{
			name:        "singleWriter",
			base:        &bytes.Buffer{},
			extra:       nil,
			expectMulti: false,
		},
		{
			name:        "multipleWriters",
			base:        &bytes.Buffer{},
			extra:       []io.Writer{&bytes.Buffer{}},
			expectMulti: true,
		},
		{
			name:        "nilExtraWriter",
			base:        &bytes.Buffer{},
			extra:       []io.Writer{nil},
			expectMulti: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAgentOutputWriter(tt.base, tt.extra...)
			if tt.expectMulti {
				assert.NotNil(t, result)
			} else {
				assert.Equal(t, tt.base, result)
			}
		})
	}
}

func TestPrepareLogFile(t *testing.T) {
	t.Run("newFile", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "output.log")

		f, err := prepareLogFile(logPath)
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		_, errStat := os.Stat(logPath)
		assert.NoError(t, errStat)
	})

	t.Run("rotatesExisting", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "output.log")
		require.NoError(t, os.WriteFile(logPath, []byte("old content"), 0644))

		f, err := prepareLogFile(logPath)
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		oldContent, errOld := os.ReadFile(logPath + ".old")
		require.NoError(t, errOld)
		assert.Equal(t, "old content", string(oldContent))
	})
}

func TestRotateLogFile(t *testing.T) {
	t.Run("fileNotExists", func(t *testing.T) {
		dir := t.TempDir()
		err := rotateLogFile(filepath.Join(dir, "nonexistent.log"))
		assert.NoError(t, err)
	})

	t.Run("fileExists", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "output.log")
		require.NoError(t, os.WriteFile(logPath, []byte("content"), 0644))

		err := rotateLogFile(logPath)
		assert.NoError(t, err)

		_, errStat := os.Stat(logPath)
		assert.True(t, os.IsNotExist(errStat))

		data, errRead := os.ReadFile(logPath + ".old")
		require.NoError(t, errRead)
		assert.Equal(t, "content", string(data))
	})
}

func TestSessionLogWriter(t *testing.T) {
	sess := &session{outputLog: newCircularLogBuffer()}
	srv, _ := setupTestServer(t)

	w := newSessionLogWriter(sess, "/test", srv, "test-ws")

	n, err := w.Write([]byte("hello world\n"))
	assert.NoError(t, err)
	assert.Equal(t, 12, n)

	lines := sess.outputLog.lines()
	require.GreaterOrEqual(t, len(lines), 1)
	assert.Equal(t, "hello world", lines[0].text)
}

func TestSessionLogWriterMultipleLines(t *testing.T) {
	sess := &session{outputLog: newCircularLogBuffer()}
	srv, _ := setupTestServer(t)

	w := newSessionLogWriter(sess, "/test", srv, "test-ws")

	_, _ = w.Write([]byte("line1\nline2\nline3\n"))

	lines := sess.outputLog.lines()
	require.GreaterOrEqual(t, len(lines), 3)
	assert.Equal(t, "line1", lines[0].text)
	assert.Equal(t, "line2", lines[1].text)
	assert.Equal(t, "line3", lines[2].text)
}

func TestSessionLogWriterPartialLine(t *testing.T) {
	sess := &session{outputLog: newCircularLogBuffer()}
	srv, _ := setupTestServer(t)

	w := newSessionLogWriter(sess, "/test", srv, "test-ws")

	_, _ = w.Write([]byte("part"))
	assert.Len(t, sess.outputLog.lines(), 0)

	_, _ = w.Write([]byte("ial\n"))
	lines := sess.outputLog.lines()
	require.GreaterOrEqual(t, len(lines), 1)
	assert.Equal(t, "partial", lines[0].text)
}
