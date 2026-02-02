package main

import (
	"bytes"
	"log"
	"maps"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
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

func (s *Server) isAdhocPromptEnabled(dir string) bool {
	if s.enableAdhocPrompt {
		return true
	}
	config, err := loadProjectConfig(dir)
	if err != nil {
		log.Printf("loading project config for ad-hoc prompt check: %v", err)
		return false
	}
	if config == nil {
		return false
	}
	return config.EnableAdhocPrompt
}

func (s *Server) loadAdhocModels() []string {
	s.adhocModelsMu.Lock()
	defer s.adhocModelsMu.Unlock()
	if s.cachedModels != nil {
		return s.cachedModels
	}
	validModels, err := fetchValidModels()
	if err != nil {
		log.Printf("fetching models for ad-hoc prompt: %v", err)
		return nil
	}
	s.cachedModels = slices.Sorted(maps.Keys(validModels))
	return s.cachedModels
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

func (s *Server) handleAdhocSaveState(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if !s.isAdhocPromptEnabled(workspacePath) {
		http.Error(w, "ad-hoc prompt not enabled", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	if model := r.FormValue("model"); model != "" {
		st.selectedModel = model
	}
	if prompt, ok := r.Form["prompt"]; ok && len(prompt) > 0 {
		st.promptText = strings.TrimSpace(prompt[0])
	}
	st.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdhocSubmit(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if !s.isAdhocPromptEnabled(workspacePath) {
		http.Error(w, "ad-hoc prompt not enabled", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	prompt := strings.TrimSpace(r.FormValue("prompt"))
	model := strings.TrimSpace(r.FormValue("model"))
	if prompt == "" || model == "" {
		http.Error(w, "prompt and model are required", http.StatusBadRequest)
		return
	}

	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	if st.running {
		st.mu.Unlock()
		s.renderAdhocOutput(w, workspacePath)
		return
	}

	st.running = true
	st.output.Reset()
	st.selectedModel = model
	st.promptText = prompt

	cmd := exec.Command("opencode", "run", "-m", model, prompt)
	cmd.Dir = workspacePath
	writer := &lockedWriter{mu: &st.mu, buf: &st.output}
	cmd.Stdout = writer
	cmd.Stderr = writer

	if errStart := cmd.Start(); errStart != nil {
		st.running = false
		st.mu.Unlock()
		http.Error(w, "failed to start command", http.StatusInternalServerError)
		return
	}

	st.cmd = cmd
	st.mu.Unlock()

	go func() {
		errWait := cmd.Wait()
		st.mu.Lock()
		if errWait != nil {
			st.output.WriteString("\n[command exited with error: " + errWait.Error() + "]\n")
		}
		st.running = false
		st.cmd = nil
		st.mu.Unlock()
	}()

	s.renderAdhocOutput(w, workspacePath)
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

func (s *Server) handleAdhocOutput(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	if !s.isAdhocPromptEnabled(workspacePath) {
		http.Error(w, "ad-hoc prompt not enabled", http.StatusForbidden)
		return
	}
	s.renderAdhocOutput(w, workspacePath)
}

func (s *Server) renderAdhocOutput(w http.ResponseWriter, workspacePath string) {
	st := s.getAdhocState(workspacePath)
	st.mu.Lock()
	running := st.running
	outputText := st.output.String()
	st.mu.Unlock()

	dirName := filepath.Base(workspacePath)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("adhoc_output.html"), struct {
		DirName string
		Running bool
		Output  string
	}{dirName, running, outputText})
}
