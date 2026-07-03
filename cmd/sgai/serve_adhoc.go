package main

import (
	"bytes"
	"os/exec"
	"sync"
)

type adhocPromptState struct {
	mu            sync.Mutex
	running       bool
	output        bytes.Buffer
	cmd           *exec.Cmd
	done          chan struct{}
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
	mu       *sync.Mutex
	buf      *bytes.Buffer
	stripper ansiStrippingWriter
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stripper.w = w.buf
	return w.stripper.Write(p)
}

func (st *adhocPromptState) stop() {
	st.mu.Lock()
	if !st.running {
		st.mu.Unlock()
		return
	}
	cmd := st.cmd
	done := st.done
	st.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		stopProcessGroup(cmd, done)
	}

	st.mu.Lock()
	st.running = false
	st.cmd = nil
	st.done = nil
	st.output.WriteString("\n[stopped by user]\n")
	st.mu.Unlock()
}
