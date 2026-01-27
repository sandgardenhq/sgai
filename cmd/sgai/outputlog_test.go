package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	f, err := prepareLogFile(logPath)
	if err != nil {
		t.Fatalf("prepareLogFile failed: %v", err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Logf("close failed: %v", err)
		}
	})

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("log file was not created")
	}
}

func TestRotateLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	backupPath := logPath + ".old"

	existingContent := []byte("existing log content\n")
	if err := os.WriteFile(logPath, existingContent, 0644); err != nil {
		t.Fatalf("failed to create test log file: %v", err)
	}

	if err := rotateLogFile(logPath); err != nil {
		t.Fatalf("rotateLogFile failed: %v", err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("old log file still exists after rotation")
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	if string(backupData) != string(existingContent) {
		t.Errorf("backup content mismatch: got %q, want %q", backupData, existingContent)
	}
}

func TestRotateLogFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nonexistent.log")

	if err := rotateLogFile(logPath); err != nil {
		t.Fatalf("rotateLogFile failed on non-existent file: %v", err)
	}
}

func TestPrepareLogFileWithRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	backupPath := logPath + ".old"

	existingContent := []byte("old session data\n")
	if err := os.WriteFile(logPath, existingContent, 0644); err != nil {
		t.Fatalf("failed to create existing log: %v", err)
	}

	f, err := prepareLogFile(logPath)
	if err != nil {
		t.Fatalf("prepareLogFile failed: %v", err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Logf("close failed: %v", err)
		}
	})

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("backup file was not created: %v", err)
	}

	if string(backupData) != string(existingContent) {
		t.Errorf("backup content mismatch: got %q, want %q", backupData, existingContent)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("new log file was not created: %v", err)
	}

	if info.Size() != 0 {
		t.Errorf("new log file should be empty, got size %d", info.Size())
	}
}

func TestCircularLogBuffer(t *testing.T) {
	buf := newCircularLogBuffer()

	if len(buf.lines()) != 0 {
		t.Error("new buffer should be empty")
	}

	buf.add(logLine{prefix: "[test] ", text: "line 1"})
	buf.add(logLine{prefix: "[test] ", text: "line 2"})

	lines := buf.lines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if lines[0].text != "line 1" || lines[1].text != "line 2" {
		t.Errorf("unexpected line content: %+v", lines)
	}
}

func TestCircularLogBufferOverflow(t *testing.T) {
	buf := newCircularLogBuffer()

	for range outputBufferSize + 100 {
		buf.add(logLine{prefix: "[test] ", text: "line"})
	}

	lines := buf.lines()
	if len(lines) != outputBufferSize {
		t.Errorf("buffer size should be capped at %d, got %d", outputBufferSize, len(lines))
	}
}

func TestTodoStatusSymbol(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"pending", "○"},
		{"in_progress", "◐"},
		{"completed", "●"},
		{"cancelled", "✕"},
		{"unknown", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := todoStatusSymbol(tt.status)
			if got != tt.want {
				t.Errorf("todoStatusSymbol(%q) = %q; want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsTodoTool(t *testing.T) {
	tests := []struct {
		tool string
		want bool
	}{
		{"todowrite", true},
		{"todoread", true},
		{"sgai_project_todowrite", true},
		{"sgai_project_todoread", true},
		{"read", false},
		{"write", false},
		{"sgai_other", false},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := isTodoTool(tt.tool)
			if got != tt.want {
				t.Errorf("isTodoTool(%q) = %v; want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestStripMCPTodoPrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "pure JSON array",
			input:  `[{"content":"task 1","status":"pending"}]`,
			output: `[{"content":"task 1","status":"pending"}]`,
		},
		{
			name:   "MCP format with todos plural",
			input:  "3 todos\n[{\"content\":\"task 1\"}]",
			output: "[{\"content\":\"task 1\"}]",
		},
		{
			name:   "MCP format with todo singular",
			input:  "1 todo\n[{\"content\":\"task 1\"}]",
			output: "[{\"content\":\"task 1\"}]",
		},
		{
			name:   "MCP format with zero todos",
			input:  "0 todos\n[]",
			output: "[]",
		},
		{
			name:   "no prefix match",
			input:  "some random text\n[{\"content\":\"task\"}]",
			output: "some random text\n[{\"content\":\"task\"}]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMCPTodoPrefix(tt.input)
			if got != tt.output {
				t.Errorf("stripMCPTodoPrefix(%q) = %q; want %q", tt.input, got, tt.output)
			}
		})
	}
}

func TestRingWriterBasic(t *testing.T) {
	rw := newRingWriter()
	var buf strings.Builder

	n, err := rw.Write([]byte("line 1\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 7 {
		t.Errorf("Write returned %d bytes, want 7", n)
	}

	n, err = rw.Write([]byte("line 2\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 7 {
		t.Errorf("Write returned %d bytes, want 7", n)
	}

	rw.dump(&buf)
	output := buf.String()
	expectedLines := []string{"line 1", "line 2"}
	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("dump output missing line %q: %s", line, output)
		}
	}
}

func TestRingWriterPartialLine(t *testing.T) {
	rw := newRingWriter()
	var buf strings.Builder

	if _, err := rw.Write([]byte("partial ")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := rw.Write([]byte("line")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	rw.dump(&buf)

	output := buf.String()
	if !strings.Contains(output, "partial line") {
		t.Errorf("dump should contain partial line: %s", output)
	}
}

func TestRingWriterCapacity(t *testing.T) {
	rw := newRingWriter()

	for i := range outputBufferSize + 100 {
		if _, err := rw.Write(fmt.Appendf(nil, "line %d\n", i)); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if rw.size != outputBufferSize {
		t.Errorf("size should be %d, got %d", outputBufferSize, rw.size)
	}

	var buf strings.Builder
	rw.dump(&buf)
	output := buf.String()

	if strings.Contains(output, "line 0") {
		t.Error("oldest lines should be dropped from ring buffer")
	}

	expectedLastLine := fmt.Sprintf("line %d", outputBufferSize+99)
	if !strings.Contains(output, expectedLastLine) {
		t.Errorf("dump should contain newest line %q", expectedLastLine)
	}
}

func TestRingWriterMultipleLines(t *testing.T) {
	rw := newRingWriter()
	var buf strings.Builder

	if _, err := rw.Write([]byte("line 1\nline 2\nline 3\n")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	rw.dump(&buf)

	output := buf.String()
	for _, line := range []string{"line 1", "line 2", "line 3"} {
		if !strings.Contains(output, line) {
			t.Errorf("dump output missing line %q: %s", line, output)
		}
	}
}

func TestRingWriterEmpty(t *testing.T) {
	rw := newRingWriter()
	var buf strings.Builder

	rw.dump(&buf)

	if buf.Len() != 0 {
		t.Errorf("dump should produce no output for empty buffer, got: %s", buf.String())
	}
}
