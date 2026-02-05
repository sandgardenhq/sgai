package main

import (
	"bytes"
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
	mu     sync.Mutex
	state  composerState
	wizard wizardState
}

type composerAgentInfo struct {
	Name        string
	Description string
	Selected    bool
	Model       string
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
	cs.wizard = defaultWizardState()
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
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
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

	w.Header().Set("Hx-Redirect", "/workspaces/"+filepath.Base(workspacePath)+"/progress")
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

func generateGOALPreview(state composerState) string {
	content := buildGOALContent(state)
	frontmatter, body := splitFrontmatterAndBody(content)

	var result bytes.Buffer
	if frontmatter != "" {
		result.WriteString(`<pre class="yaml-frontmatter"><code>`)
		result.WriteString(template.HTMLEscapeString(frontmatter))
		result.WriteString(`</code></pre>`)
	}

	if body != "" {
		rendered, errRender := renderMarkdown([]byte(body))
		if errRender != nil {
			result.WriteString(template.HTMLEscapeString(body))
		} else {
			result.WriteString(rendered)
		}
	}

	return result.String()
}

func splitFrontmatterAndBody(content string) (frontmatter, body string) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}

	rest := content[4:]
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx < 0 {
		endIdx = strings.Index(rest, "\n---")
		if endIdx < 0 {
			return "", content
		}
	}

	frontmatter = "---\n" + rest[:endIdx] + "\n---"
	body = strings.TrimPrefix(rest[endIdx+4:], "\n")
	return frontmatter, body
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
		s.handleComposeLanding(w, r, workspacePath)
	case strings.HasPrefix(path, "template/"):
		templateID := strings.TrimPrefix(path, "template/")
		s.handleComposeTemplate(w, r, workspacePath, templateID)
	case strings.HasPrefix(path, "wizard/step/"):
		stepStr := strings.TrimPrefix(path, "wizard/step/")
		step := parseWizardStep(stepStr)
		if r.Method == http.MethodPost {
			s.handleComposeWizardUpdate(w, r, workspacePath, step)
		} else {
			s.handleComposeWizardStep(w, r, workspacePath, step)
		}
	case path == "wizard/finish":
		s.handleComposeWizardFinish(w, r, workspacePath)
	case path == "templates":
		s.handleComposeTemplatesJSON(w, r, workspacePath)
	case path == "preview":
		s.handleComposePreview(w, r, workspacePath)
	case path == "save":
		s.handleComposeSave(w, r, workspacePath)
	case path == "reset":
		s.handleComposeReset(w, r, workspacePath)
	default:
		http.NotFound(w, r)
	}
}

func parseWizardStep(s string) int {
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		return int(s[0] - '0')
	}
	return 1
}
