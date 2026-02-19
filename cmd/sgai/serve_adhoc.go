package main

import (
	"bytes"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b[^[\]].?`)

type adhocPromptState struct {
	mu            sync.Mutex
	running       bool
	output        bytes.Buffer
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
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	stripped := ansiEscapePattern.ReplaceAll(p, nil)
	w.buf.Write(stripped)
	return len(p), nil
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
