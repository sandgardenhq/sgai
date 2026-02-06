package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type composerState struct {
	Description    string              `json:"description"`
	Interactive    string              `json:"interactive"`
	CompletionGate string              `json:"completionGate"`
	Agents         []composerAgentConf `json:"agents"`
	Flow           string              `json:"flow"`
	Tasks          string              `json:"tasks"`
}

type composerAgentConf struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
	Model    string `json:"model"`
}

type composerSession struct {
	mu    sync.Mutex
	state composerState
}

type composerAgentInfo struct {
	Name        string
	Description string
	Selected    bool
	Model       string
}

type composerPageData struct {
	Directory   string
	DirName     string
	State       composerState
	Agents      []composerAgentInfo
	Models      []string
	Preview     template.HTML
	FlowError   string
	SaveSuccess bool
	SaveError   string
}

type composerDiffData struct {
	HasChanges bool
	OldPreview string
	NewPreview string
	Changes    []string
}

var (
	composerSessionsMu sync.Mutex
	composerSessions   = make(map[string]*composerSession)
)

func getComposerSession(workspacePath string) *composerSession {
	composerSessionsMu.Lock()
	defer composerSessionsMu.Unlock()

	if existing, ok := composerSessions[workspacePath]; ok {
		return existing
	}

	cs := &composerSession{}
	cs.state = loadComposerStateFromDisk(workspacePath)
	composerSessions[workspacePath] = cs
	return cs
}

func loadComposerStateFromDisk(dir string) composerState {
	goalPath := filepath.Join(dir, "GOAL.md")
	content, err := os.ReadFile(goalPath)
	if err != nil {
		return defaultComposerState()
	}

	metadata, err := parseYAMLFrontmatter(content)
	if err != nil {
		return defaultComposerState()
	}

	bodyContent := extractBodyContent(content)

	state := composerState{
		Description:    extractDescriptionFromBody(bodyContent),
		Interactive:    metadata.Interactive,
		CompletionGate: metadata.CompletionGateScript,
		Flow:           metadata.Flow,
		Tasks:          extractTasksFromBody(bodyContent),
	}

	for agentName, modelVal := range metadata.Models {
		model := ""
		if s, ok := modelVal.(string); ok {
			model = s
		}
		state.Agents = append(state.Agents, composerAgentConf{
			Name:     agentName,
			Selected: true,
			Model:    model,
		})
	}

	slices.SortFunc(state.Agents, func(a, b composerAgentConf) int {
		return strings.Compare(a.Name, b.Name)
	})

	return state
}

func defaultComposerState() composerState {
	return composerState{
		Interactive: "yes",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: ""},
		},
	}
}

func extractBodyContent(content []byte) string {
	body := extractBody(content)
	return string(body)
}

func extractDescriptionFromBody(body string) string {
	lines := strings.Split(body, "\n")
	var descLines []string
	inTasks := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Tasks") || strings.HasPrefix(trimmed, "## Task") {
			inTasks = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && inTasks {
			inTasks = false
		}
		if !inTasks {
			descLines = append(descLines, line)
		}
	}

	desc := strings.TrimSpace(strings.Join(descLines, "\n"))
	desc = strings.TrimPrefix(desc, "# ")
	if idx := strings.Index(desc, "\n"); idx > 0 {
		firstLine := strings.TrimSpace(desc[:idx])
		rest := strings.TrimSpace(desc[idx:])
		if rest != "" {
			desc = firstLine + "\n\n" + rest
		} else {
			desc = firstLine
		}
	}
	return desc
}

func extractTasksFromBody(body string) string {
	lines := strings.Split(body, "\n")
	var taskLines []string
	inTasks := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Tasks") || strings.HasPrefix(trimmed, "## Task") {
			inTasks = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && inTasks {
			break
		}
		if inTasks {
			taskLines = append(taskLines, line)
		}
	}

	return strings.TrimSpace(strings.Join(taskLines, "\n"))
}

func (s *Server) handleCompose(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	agents := loadAvailableAgents(workspacePath)
	models := loadAvailableModels()

	agentInfos := buildAgentInfoList(agents, currentState.Agents)

	preview := generateGOALPreview(currentState)

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	data := composerPageData{
		Directory: workspacePath,
		DirName:   filepath.Base(workspacePath),
		State:     currentState,
		Agents:    agentInfos,
		Models:    models,
		Preview:   template.HTML(preview),
		FlowError: flowErr,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("compose.html"), data)
}

func (s *Server) handleComposeUpdatePanel(w http.ResponseWriter, r *http.Request, workspacePath, panel string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()

	switch panel {
	case "description":
		cs.state.Description = r.FormValue("description")
	case "frontmatter":
		cs.state.Interactive = r.FormValue("interactive")
		cs.state.CompletionGate = r.FormValue("completionGate")
	case "agents":
		cs.state.Agents = parseAgentFormValues(r, cs.state.Agents, loadAvailableAgents(workspacePath))
	case "flow":
		cs.state.Flow = r.FormValue("flow")
	case "tasks":
		cs.state.Tasks = r.FormValue("tasks")
	}

	currentState := cs.state
	cs.mu.Unlock()

	preview := generateGOALPreview(currentState)

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Hx-Trigger", "previewUpdated")

	data := struct {
		Preview   template.HTML
		FlowError string
	}{
		Preview:   template.HTML(preview),
		FlowError: flowErr,
	}
	executeTemplate(w, templates.Lookup("compose_preview_partial.html"), data)
}

func (s *Server) handleComposePreview(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	preview := generateGOALPreview(currentState)

	var flowErr string
	if currentState.Flow != "" {
		if _, errParse := parseFlow(currentState.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	w.Header().Set("Content-Type", "text/html")
	data := struct {
		Preview   template.HTML
		FlowError string
	}{
		Preview:   template.HTML(preview),
		FlowError: flowErr,
	}
	executeTemplate(w, templates.Lookup("compose_preview_partial.html"), data)
}

func (s *Server) handleComposeSave(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	goalContent := buildGOALContent(currentState)
	goalPath := filepath.Join(workspacePath, "GOAL.md")

	if err := os.WriteFile(goalPath, []byte(goalContent), 0644); err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Hx-Trigger", "saveError")
		_, _ = fmt.Fprintf(w, `<div class="save-error">Failed to save: %s</div>`, template.HTMLEscapeString(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Hx-Trigger", "saveSuccess")
	_, _ = w.Write([]byte(`<div class="save-success">GOAL.md saved successfully!</div>`))
}

func (s *Server) handleComposeReset(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	newState := loadComposerStateFromDisk(workspacePath)

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	cs.state = newState
	cs.mu.Unlock()

	http.Redirect(w, r, "/compose?workspace="+filepath.Base(workspacePath), http.StatusSeeOther)
}

func (s *Server) handleComposeAssist(w http.ResponseWriter, r *http.Request, workspacePath, panel string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	prompt := buildAssistPrompt(panel, currentState, workspacePath)

	result, err := invokeLLMForAssist(prompt)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Hx-Trigger", "assistError")
		_, _ = fmt.Fprintf(w, `<div class="assist-error">AI assist failed: %s</div>`, template.HTMLEscapeString(err.Error()))
		return
	}

	cs.mu.Lock()
	applyAssistResult(panel, result, &cs.state)
	newState := cs.state
	cs.mu.Unlock()

	s.renderPanelAndPreview(w, workspacePath, panel, newState)
}

func (s *Server) handleComposeCommand(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	command := r.FormValue("command")
	if command == "" {
		http.Error(w, "command required", http.StatusBadRequest)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	cs.mu.Unlock()

	prompt := buildCommandPrompt(command, currentState, workspacePath)

	result, err := invokeLLMForCommand(prompt)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Hx-Trigger", "commandError")
		_, _ = fmt.Fprintf(w, `<div class="command-error">Command failed: %s</div>`, template.HTMLEscapeString(err.Error()))
		return
	}

	diff := computeStateDiff(currentState, result)

	w.Header().Set("Content-Type", "text/html")
	data := struct {
		Command    string
		Diff       composerDiffData
		NewState   composerState
		HasChanges bool
	}{
		Command:    command,
		Diff:       diff,
		NewState:   result,
		HasChanges: diff.HasChanges,
	}
	executeTemplate(w, templates.Lookup("compose_command_preview.html"), data)
}

func (s *Server) handleComposeApply(w http.ResponseWriter, r *http.Request, workspacePath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	var newState composerState
	stateJSON := r.FormValue("newState")
	if err := json.Unmarshal([]byte(stateJSON), &newState); err != nil {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	cs.state = newState
	cs.mu.Unlock()

	http.Redirect(w, r, "/compose?workspace="+filepath.Base(workspacePath), http.StatusSeeOther)
}

func (s *Server) handleComposeAgents(w http.ResponseWriter, _ *http.Request, workspacePath string) {
	agents := loadAvailableAgents(workspacePath)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, "encoding failed", http.StatusInternalServerError)
	}
}

func (s *Server) handleComposeModels(w http.ResponseWriter, _ *http.Request, _ string) {
	models := loadAvailableModels()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		http.Error(w, "encoding failed", http.StatusInternalServerError)
	}
}

func loadAvailableAgents(workspacePath string) []composerAgentInfo {
	agentsDir := filepath.Join(workspacePath, ".sgai", "agent")
	agentsFS := os.DirFS(agentsDir)

	var agents []composerAgentInfo
	err := fs.WalkDir(agentsFS, ".", func(path string, d fs.DirEntry, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		name := strings.TrimSuffix(path, ".md")
		content, errRead := fs.ReadFile(agentsFS, path)
		if errRead != nil {
			return nil
		}
		desc := extractFrontmatterDescription(string(content))
		agents = append(agents, composerAgentInfo{
			Name:        name,
			Description: desc,
		})
		return nil
	})
	if err != nil {
		agents = nil
	}

	slices.SortFunc(agents, func(a, b composerAgentInfo) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return agents
}

func loadAvailableModels() []string {
	cmd := exec.Command("opencode", "models")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	var models []string
	for line := range strings.SplitSeq(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			models = append(models, trimmed)
		}
	}
	return models
}

func buildAgentInfoList(available []composerAgentInfo, selected []composerAgentConf) []composerAgentInfo {
	selectedMap := make(map[string]composerAgentConf)
	for _, a := range selected {
		selectedMap[a.Name] = a
	}

	var result []composerAgentInfo
	for _, a := range available {
		info := composerAgentInfo{
			Name:        a.Name,
			Description: a.Description,
		}
		if sel, ok := selectedMap[a.Name]; ok {
			info.Selected = sel.Selected
			info.Model = sel.Model
		}
		result = append(result, info)
	}
	return result
}

func parseAgentFormValues(r *http.Request, _ []composerAgentConf, available []composerAgentInfo) []composerAgentConf {
	selectedAgents := r.Form["agents"]
	selectedSet := make(map[string]bool)
	for _, a := range selectedAgents {
		selectedSet[a] = true
	}

	var result []composerAgentConf
	for _, a := range available {
		conf := composerAgentConf{
			Name:     a.Name,
			Selected: selectedSet[a.Name],
			Model:    r.FormValue("model_" + a.Name),
		}
		result = append(result, conf)
	}
	return result
}

func generateGOALPreview(state composerState) string {
	content := buildGOALContent(state)
	rendered, err := renderMarkdown([]byte(content))
	if err != nil {
		return template.HTMLEscapeString(content)
	}
	return rendered
}

func buildGOALContent(state composerState) string {
	var buf bytes.Buffer

	buf.WriteString("---\n")

	if state.Flow != "" {
		buf.WriteString("flow: |\n")
		for line := range strings.SplitSeq(state.Flow, "\n") {
			buf.WriteString("  ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	hasSelectedAgents := false
	for _, a := range state.Agents {
		if a.Selected {
			hasSelectedAgents = true
			break
		}
	}

	if hasSelectedAgents {
		buf.WriteString("models:\n")
		for _, a := range state.Agents {
			if !a.Selected {
				continue
			}
			buf.WriteString("  \"")
			buf.WriteString(a.Name)
			buf.WriteString("\": \"")
			buf.WriteString(a.Model)
			buf.WriteString("\"\n")
		}
	}

	if state.Interactive != "" {
		buf.WriteString("interactive: ")
		buf.WriteString(state.Interactive)
		buf.WriteString("\n")
	}

	if state.CompletionGate != "" {
		buf.WriteString("completionGateScript: ")
		buf.WriteString(state.CompletionGate)
		buf.WriteString("\n")
	}

	buf.WriteString("---\n\n")

	if state.Description != "" {
		buf.WriteString(state.Description)
		buf.WriteString("\n\n")
	}

	if state.Tasks != "" {
		buf.WriteString("## Tasks\n\n")
		buf.WriteString(state.Tasks)
		buf.WriteString("\n")
	}

	return buf.String()
}

func buildAssistPrompt(panel string, state composerState, _ string) string {
	var buf bytes.Buffer

	buf.WriteString("You are helping to configure a software factory workflow. ")
	buf.WriteString("Based on the current configuration, provide suggestions for the '")
	buf.WriteString(panel)
	buf.WriteString("' section.\n\n")

	buf.WriteString("Current configuration:\n")
	buf.WriteString("Description: ")
	buf.WriteString(state.Description)
	buf.WriteString("\n")
	buf.WriteString("Interactive: ")
	buf.WriteString(state.Interactive)
	buf.WriteString("\n")
	buf.WriteString("Agents: ")
	var agentNames []string
	for _, a := range state.Agents {
		if a.Selected {
			agentNames = append(agentNames, a.Name)
		}
	}
	buf.WriteString(strings.Join(agentNames, ", "))
	buf.WriteString("\n")
	buf.WriteString("Flow: ")
	buf.WriteString(state.Flow)
	buf.WriteString("\n")
	buf.WriteString("Tasks: ")
	buf.WriteString(state.Tasks)
	buf.WriteString("\n\n")

	switch panel {
	case "description":
		buf.WriteString("Suggest an improved project description that clearly explains the goal.\n")
	case "agents":
		buf.WriteString("Suggest which agents should be selected based on the project description.\n")
	case "flow":
		buf.WriteString("Suggest a workflow DAG (in DOT format) based on the selected agents.\n")
	case "tasks":
		buf.WriteString("Suggest a task list (markdown checkbox format) based on the project description.\n")
	}

	buf.WriteString("\nRespond with JSON containing the suggested value for this panel.\n")

	return buf.String()
}

func buildCommandPrompt(command string, state composerState, _ string) string {
	var buf bytes.Buffer

	buf.WriteString("You are helping to configure a software factory workflow.\n")
	buf.WriteString("The user has requested the following change:\n\n")
	buf.WriteString(command)
	buf.WriteString("\n\n")
	buf.WriteString("Current configuration:\n")

	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	buf.Write(stateJSON)
	buf.WriteString("\n\n")

	buf.WriteString("Apply the requested change and return the complete new state as JSON.\n")
	buf.WriteString("The JSON should match this structure:\n")
	buf.WriteString(`{
  "description": "string",
  "interactive": "yes|no|auto",
  "completionGate": "string or empty",
  "agents": [{"name": "agent-name", "selected": true/false, "model": "model-name"}],
  "flow": "DOT format string",
  "tasks": "markdown task list"
}`)

	return buf.String()
}

func invokeLLMForAssist(prompt string) (string, error) {
	cmd := exec.Command("opencode", "run", "--format=text")
	cmd.Stdin = strings.NewReader(prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("opencode failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func invokeLLMForCommand(prompt string) (composerState, error) {
	cmd := exec.Command("opencode", "run", "--format=text")
	cmd.Stdin = strings.NewReader(prompt)
	output, err := cmd.Output()
	if err != nil {
		return composerState{}, fmt.Errorf("opencode failed: %w", err)
	}

	jsonStart := strings.Index(string(output), "{")
	jsonEnd := strings.LastIndex(string(output), "}")
	if jsonStart < 0 || jsonEnd < 0 || jsonEnd <= jsonStart {
		return composerState{}, fmt.Errorf("no valid JSON in response")
	}

	jsonStr := string(output)[jsonStart : jsonEnd+1]

	var result composerState
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return composerState{}, fmt.Errorf("invalid JSON response: %w", err)
	}

	return result, nil
}

func applyAssistResult(panel, result string, state *composerState) {
	switch panel {
	case "description":
		state.Description = extractJSONStringField(result, "description")
	case "flow":
		state.Flow = extractJSONStringField(result, "flow")
	case "tasks":
		state.Tasks = extractJSONStringField(result, "tasks")
	case "agents":
		var agents []composerAgentConf
		if err := json.Unmarshal([]byte(result), &agents); err == nil {
			state.Agents = agents
		}
	}
}

func extractJSONStringField(jsonStr, field string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return jsonStr
	}
	if val, ok := m[field].(string); ok {
		return val
	}
	return jsonStr
}

func computeStateDiff(old, updated composerState) composerDiffData {
	var changes []string

	if old.Description != updated.Description {
		changes = append(changes, "Description changed")
	}
	if old.Interactive != updated.Interactive {
		changes = append(changes, "Interactive mode changed")
	}
	if old.CompletionGate != updated.CompletionGate {
		changes = append(changes, "Completion gate changed")
	}
	if old.Flow != updated.Flow {
		changes = append(changes, "Workflow flow changed")
	}
	if old.Tasks != updated.Tasks {
		changes = append(changes, "Tasks changed")
	}

	oldAgentSet := make(map[string]bool)
	updatedAgentSet := make(map[string]bool)
	for _, a := range old.Agents {
		if a.Selected {
			oldAgentSet[a.Name] = true
		}
	}
	for _, a := range updated.Agents {
		if a.Selected {
			updatedAgentSet[a.Name] = true
		}
	}

	for name := range updatedAgentSet {
		if !oldAgentSet[name] {
			changes = append(changes, "Added agent: "+name)
		}
	}
	for name := range oldAgentSet {
		if !updatedAgentSet[name] {
			changes = append(changes, "Removed agent: "+name)
		}
	}

	return composerDiffData{
		HasChanges: len(changes) > 0,
		OldPreview: buildGOALContent(old),
		NewPreview: buildGOALContent(updated),
		Changes:    changes,
	}
}

func (s *Server) renderPanelAndPreview(w http.ResponseWriter, workspacePath, panel string, state composerState) {
	preview := generateGOALPreview(state)

	var flowErr string
	if state.Flow != "" {
		if _, errParse := parseFlow(state.Flow, workspacePath); errParse != nil {
			flowErr = errParse.Error()
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Hx-Trigger", "assistSuccess")

	data := struct {
		State     composerState
		Preview   template.HTML
		FlowError string
		Panel     string
	}{
		State:     state,
		Preview:   template.HTML(preview),
		FlowError: flowErr,
		Panel:     panel,
	}
	executeTemplate(w, templates.Lookup("compose_assist_result.html"), data)
}

func (s *Server) routeCompose(w http.ResponseWriter, r *http.Request) {
	workspaceParam := r.URL.Query().Get("workspace")
	if workspaceParam == "" {
		http.Redirect(w, r, "/trees", http.StatusSeeOther)
		return
	}

	workspacePath := s.resolveWorkspaceNameToPath(workspaceParam)
	if workspacePath == "" {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/compose")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" || path == "/":
		s.handleCompose(w, r, workspacePath)
	case strings.HasPrefix(path, "update/"):
		panel := strings.TrimPrefix(path, "update/")
		s.handleComposeUpdatePanel(w, r, workspacePath, panel)
	case path == "preview":
		s.handleComposePreview(w, r, workspacePath)
	case path == "save":
		s.handleComposeSave(w, r, workspacePath)
	case path == "reset":
		s.handleComposeReset(w, r, workspacePath)
	case strings.HasPrefix(path, "assist/"):
		panel := strings.TrimPrefix(path, "assist/")
		s.handleComposeAssist(w, r, workspacePath, panel)
	case path == "command":
		s.handleComposeCommand(w, r, workspacePath)
	case path == "apply":
		s.handleComposeApply(w, r, workspacePath)
	case path == "agents":
		s.handleComposeAgents(w, r, workspacePath)
	case path == "models":
		s.handleComposeModels(w, r, workspacePath)
	default:
		http.NotFound(w, r)
	}
}
