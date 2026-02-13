package main

import (
	"bytes"
	"os/exec"
	"regexp"
	"sync"
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
