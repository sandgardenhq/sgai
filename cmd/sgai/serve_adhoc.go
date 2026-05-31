package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b[^[\]].?`)

type adhocPromptState struct {
	mu            sync.Mutex
	running       bool
	output        bytes.Buffer
	rawOutput     bytes.Buffer
	cmd           *exec.Cmd
	selectedModel string
	promptText    string
}

func (s *Server) getAdhocState(workspacePath string) *adhocPromptState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.adhocStates[workspacePath]
	if st == nil {
		st = &adhocPromptState{}
		s.adhocStates[workspacePath] = st
	}
	return st
}

type lockedWriter struct {
	mu      *sync.Mutex
	buf     *bytes.Buffer
	raw     bool
	pending []byte
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	stripped := ansiEscapePattern.ReplaceAll(p, nil)
	if w.raw {
		w.buf.Write(stripped)
		return len(p), nil
	}
	formatted, pending := formatAdhocDisplayOutput(append(w.pending, stripped...))
	w.pending = pending
	w.buf.Write(formatted)
	return len(p), nil
}

func formatAdhocDisplayOutput(data []byte) ([]byte, []byte) {
	if len(data) == 0 {
		return nil, nil
	}
	var out bytes.Buffer
	lines := strings.SplitAfter(string(data), "\n")
	for index, line := range lines {
		if line == "" {
			continue
		}
		if index == len(lines)-1 && !strings.HasSuffix(line, "\n") && strings.HasPrefix(strings.TrimSpace(line), "{") {
			return out.Bytes(), []byte(line)
		}
		trimmed := strings.TrimSuffix(line, "\n")
		if formatted, ok := formatAdhocJSONLine(trimmed); ok {
			out.Write(formatted)
			continue
		}
		out.WriteString(line)
	}
	return out.Bytes(), nil
}

func formatAdhocJSONLine(line string) ([]byte, bool) {
	if strings.TrimSpace(line) == "" {
		return []byte(line), true
	}
	var event streamEvent
	if errJSON := json.Unmarshal([]byte(line), &event); errJSON != nil || event.Type == "" {
		return nil, false
	}
	var buf bytes.Buffer
	writer := &jsonPrettyWriter{w: &buf}
	if _, errWrite := writer.Write([]byte(line + "\n")); errWrite != nil {
		return nil, false
	}
	writer.Flush()
	return buf.Bytes(), true
}

func (st *adhocPromptState) stop() {
	st.mu.Lock()
	if !st.running {
		st.mu.Unlock()
		return
	}
	cmd := st.cmd
	st.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		pgid := -cmd.Process.Pid
		_ = syscall.Kill(pgid, syscall.SIGTERM)

		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(gracefulShutdownTimeout):
			_ = syscall.Kill(pgid, syscall.SIGKILL)
			<-done
		}
	}

	st.mu.Lock()
	st.running = false
	st.cmd = nil
	st.output.WriteString("\n[stopped by user]\n")
	st.mu.Unlock()
}
