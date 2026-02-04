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

func (s *Server) loadAdhocModels() []string {
	s.adhocModelsMu.Lock()
	defer s.adhocModelsMu.Unlock()
	if s.cachedModels != nil {
		return s.cachedModels
	}
	validModels, err := fetchValidModels()
	if err != nil {
		log.Println("fetching models for ad-hoc prompt:", err)
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

	cmd := exec.Command("opencode", "run", "-m", model)
	cmd.Dir = workspacePath
	cmd.Stdin = strings.NewReader(prompt)
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

func resolveCoordinatorDefaultModel(dir string) string {
	modelSpec := lookupModelForAgent(dir, "coordinator")
	if modelSpec == "" {
		return ""
	}
	model, _ := parseModelAndVariant(modelSpec)
	return model
}

func (s *Server) renderTreesRunTabToBuffer(buf *bytes.Buffer, dir string) {
	adhocModels := s.loadAdhocModels()
	st := s.getAdhocState(dir)
	st.mu.Lock()
	adhocRunning := st.running
	adhocSelectedModel := st.selectedModel
	adhocPromptText := st.promptText
	adhocOutput := st.output.String()
	st.mu.Unlock()

	if adhocSelectedModel == "" {
		adhocSelectedModel = resolveCoordinatorDefaultModel(dir)
	}

	data := struct {
		Directory          string
		DirName            string
		AdhocModels        []string
		AdhocRunning       bool
		AdhocSelectedModel string
		AdhocPromptText    string
		AdhocOutput        string
	}{
		Directory:          dir,
		DirName:            filepath.Base(dir),
		AdhocModels:        adhocModels,
		AdhocRunning:       adhocRunning,
		AdhocSelectedModel: adhocSelectedModel,
		AdhocPromptText:    adhocPromptText,
		AdhocOutput:        adhocOutput,
	}

	if err := templates.Lookup("trees_run_content.html").Execute(buf, data); err != nil {
		log.Println("template execution failed:", err)
	}
}
