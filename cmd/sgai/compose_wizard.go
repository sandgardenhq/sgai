package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

type workflowTemplate struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Agents      []composerAgentConf
	Flow        string
	Interactive string
}

const defaultAgentModel = "anthropic/claude-opus-4-6"

var workflowTemplates = []workflowTemplate{
	{
		ID:          "backend",
		Name:        "Backend Development",
		Description: "Go developer with code reviewer and safety analysis",
		Icon:        "⚙️",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "backend-go-developer", Selected: true, Model: defaultAgentModel},
			{Name: "go-readability-reviewer", Selected: true, Model: defaultAgentModel},
			{Name: "stpa-analyst", Selected: true, Model: defaultAgentModel},
		},
		Flow: `"backend-go-developer" -> "go-readability-reviewer"
"backend-go-developer" -> "stpa-analyst"
"go-readability-reviewer" -> "stpa-analyst"`,
		Interactive: "yes",
	},
	{
		ID:          "frontend",
		Name:        "Frontend — HTMX",
		Description: "HTMX/PicoCSS developer with UI reviewer",
		Icon:        "🎨",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "htmx-picocss-frontend-developer", Selected: true, Model: defaultAgentModel},
			{Name: "htmx-picocss-frontend-reviewer", Selected: true, Model: defaultAgentModel},
		},
		Flow:        `"htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"`,
		Interactive: "yes",
	},
	{
		ID:          "fullstack",
		Name:        "Full Stack",
		Description: "Backend + Frontend developers with reviewers and safety analysis",
		Icon:        "🚀",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "backend-go-developer", Selected: true, Model: defaultAgentModel},
			{Name: "go-readability-reviewer", Selected: true, Model: defaultAgentModel},
			{Name: "htmx-picocss-frontend-developer", Selected: true, Model: defaultAgentModel},
			{Name: "htmx-picocss-frontend-reviewer", Selected: true, Model: defaultAgentModel},
			{Name: "stpa-analyst", Selected: true, Model: defaultAgentModel},
		},
		Flow: `"backend-go-developer" -> "go-readability-reviewer"
"backend-go-developer" -> "stpa-analyst"
"go-readability-reviewer" -> "stpa-analyst"
"htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
"htmx-picocss-frontend-reviewer" -> "stpa-analyst"`,
		Interactive: "yes",
	},
	{
		ID:          "research",
		Name:        "Research & Analysis",
		Description: "General-purpose agent with critical evaluation council",
		Icon:        "🔬",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "general-purpose", Selected: true, Model: defaultAgentModel},
			{Name: "project-critic-council", Selected: true, Model: defaultAgentModel},
		},
		Flow:        `"general-purpose" -> "project-critic-council"`,
		Interactive: "yes",
	},
	{
		ID:          "custom",
		Name:        "Custom",
		Description: "Start with a blank slate — pick your own agents",
		Icon:        "🛠️",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
		},
		Flow:        "",
		Interactive: "yes",
	},
	{
		ID:          "react",
		Name:        "Frontend — React",
		Description: "React developer with code reviewer",
		Icon:        "⚛️",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "react-developer", Selected: true, Model: defaultAgentModel},
			{Name: "react-reviewer", Selected: true, Model: defaultAgentModel},
		},
		Flow:        `"react-developer" -> "react-reviewer"`,
		Interactive: "yes",
	},
	{
		ID:          "shell",
		Name:        "Shell Scripting",
		Description: "Shell script developer with reviewer",
		Icon:        "🐚",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "shell-script-coder", Selected: true, Model: defaultAgentModel},
			{Name: "shell-script-reviewer", Selected: true, Model: defaultAgentModel},
		},
		Flow:        `"shell-script-coder" -> "shell-script-reviewer"`,
		Interactive: "yes",
	},
	{
		ID:          "website",
		Name:        "Marketing Website",
		Description: "Webmaster with frontend reviewer",
		Icon:        "🌐",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "webmaster", Selected: true, Model: defaultAgentModel},
			{Name: "htmx-picocss-frontend-reviewer", Selected: true, Model: defaultAgentModel},
		},
		Flow:        `"webmaster" -> "htmx-picocss-frontend-reviewer"`,
		Interactive: "yes",
	},
	{
		ID:          "c4docs",
		Name:        "C4 Architecture Docs",
		Description: "C4 model documentation chain: code → component → container → context",
		Icon:        "📐",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "c4-code", Selected: true, Model: defaultAgentModel},
			{Name: "c4-component", Selected: true, Model: defaultAgentModel},
			{Name: "c4-container", Selected: true, Model: defaultAgentModel},
			{Name: "c4-context", Selected: true, Model: defaultAgentModel},
		},
		Flow: `"c4-code" -> "c4-component"
"c4-component" -> "c4-container"
"c4-container" -> "c4-context"`,
		Interactive: "yes",
	},
	{
		ID:          "claudesdk",
		Name:        "Claude SDK App",
		Description: "Build with Claude Agent SDK, verified for TS and Python",
		Icon:        "🤖",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "general-purpose", Selected: true, Model: defaultAgentModel},
			{Name: "agent-sdk-verifier-ts", Selected: true, Model: defaultAgentModel},
			{Name: "agent-sdk-verifier-py", Selected: true, Model: defaultAgentModel},
		},
		Flow: `"general-purpose" -> "agent-sdk-verifier-ts"
"general-purpose" -> "agent-sdk-verifier-py"`,
		Interactive: "yes",
	},
	{
		ID:          "openaisdk",
		Name:        "OpenAI SDK App",
		Description: "Build with OpenAI Agents SDK, verified for TS and Python",
		Icon:        "🧠",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
			{Name: "general-purpose", Selected: true, Model: defaultAgentModel},
			{Name: "openai-sdk-verifier-ts", Selected: true, Model: defaultAgentModel},
			{Name: "openai-sdk-verifier-py", Selected: true, Model: defaultAgentModel},
		},
		Flow: `"general-purpose" -> "openai-sdk-verifier-ts"
"general-purpose" -> "openai-sdk-verifier-py"`,
		Interactive: "yes",
	},
}

type wizardState struct {
	CurrentStep    int
	FromTemplate   string
	Description    string
	TechStack      []string
	SafetyAnalysis bool
	Interactive    string
	CompletionGate string
}

func defaultWizardState() wizardState {
	return wizardState{
		CurrentStep:    1,
		SafetyAnalysis: false,
		Interactive:    "yes",
	}
}

type wizardPageData struct {
	Directory      string
	DirName        string
	Step           int
	TotalSteps     int
	Wizard         wizardState
	State          composerState
	Templates      []workflowTemplate
	Agents         []composerAgentInfo
	Models         []string
	Preview        template.HTML
	FlowError      string
	TechStackItems []techStackItem
}

type techStackItem struct {
	ID       string
	Name     string
	Selected bool
}

var defaultTechStackItems = []techStackItem{
	{ID: "go", Name: "Go"},
	{ID: "htmx", Name: "HTMX"},
	{ID: "react", Name: "React"},
	{ID: "python", Name: "Python"},
	{ID: "typescript", Name: "TypeScript"},
	{ID: "shell", Name: "Shell/Bash"},
	{ID: "general-purpose", Name: "General Purpose Development"},
	{ID: "claudesdk", Name: "Claude SDK"},
	{ID: "openaisdk", Name: "OpenAI SDK"},
}

func (s *Server) handleComposeLanding(w http.ResponseWriter, _ *http.Request, workspacePath string) {
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

	data := wizardPageData{
		Directory:      workspacePath,
		DirName:        filepath.Base(workspacePath),
		Step:           0,
		TotalSteps:     4,
		Templates:      workflowTemplates,
		State:          currentState,
		Agents:         agentInfos,
		Models:         models,
		Preview:        template.HTML(preview),
		FlowError:      flowErr,
		TechStackItems: defaultTechStackItems,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("compose_landing.html"), data)
}

func (s *Server) handleComposeTemplate(w http.ResponseWriter, r *http.Request, workspacePath, templateID string) {
	var selectedTemplate *workflowTemplate
	for i := range workflowTemplates {
		if workflowTemplates[i].ID == templateID {
			selectedTemplate = &workflowTemplates[i]
			break
		}
	}

	if selectedTemplate == nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	agents := make([]composerAgentConf, len(selectedTemplate.Agents))
	copy(agents, selectedTemplate.Agents)
	cs.state.Agents = agents
	cs.state.Flow = selectedTemplate.Flow
	cs.state.Interactive = selectedTemplate.Interactive
	applyTemplateWizardState(cs, selectedTemplate)
	cs.mu.Unlock()

	redirectURL := "/compose/wizard/step/1?workspace=" + filepath.Base(workspacePath) + "&template=" + templateID
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *Server) handleComposeWizardStep(w http.ResponseWriter, r *http.Request, workspacePath string, step int) {
	if step < 1 || step > 4 {
		http.Redirect(w, r, "/compose?workspace="+filepath.Base(workspacePath), http.StatusSeeOther)
		return
	}

	cs := getComposerSession(workspacePath)
	cs.mu.Lock()
	currentState := cs.state
	wizard := syncWizardState(cs.wizard, currentState)
	cs.mu.Unlock()

	wizard.CurrentStep = step

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

	techStack := make([]techStackItem, len(defaultTechStackItems))
	copy(techStack, defaultTechStackItems)
	selectedTech := make(map[string]bool)
	for _, ts := range wizard.TechStack {
		selectedTech[ts] = true
	}
	for i := range techStack {
		techStack[i].Selected = selectedTech[techStack[i].ID]
	}

	data := wizardPageData{
		Directory:      workspacePath,
		DirName:        filepath.Base(workspacePath),
		Step:           step,
		TotalSteps:     4,
		Wizard:         wizard,
		State:          currentState,
		Templates:      workflowTemplates,
		Agents:         agentInfos,
		Models:         models,
		Preview:        template.HTML(preview),
		FlowError:      flowErr,
		TechStackItems: techStack,
	}

	templateName := fmt.Sprintf("compose_wizard_step%d.html", step)
	tmpl := templates.Lookup(templateName)
	if tmpl == nil {
		tmpl = templates.Lookup("compose_wizard_base.html")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, tmpl, data)
}

func (s *Server) handleComposeWizardUpdate(w http.ResponseWriter, r *http.Request, workspacePath string, step int) {
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

	switch step {
	case 1:
		cs.state.Description = r.FormValue("description")
		cs.wizard.Description = r.FormValue("description")
	case 2:
		cs.wizard.TechStack = r.Form["techstack"]
		updateAgentsFromTechStack(cs)
	case 3:
		cs.wizard.SafetyAnalysis = r.FormValue("safetyanalysis") == "yes"
		updateSafetyAgents(cs)
	case 4:
		cs.state.Interactive = r.FormValue("interactive")
		cs.state.CompletionGate = r.FormValue("completiongate")
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
	data := struct {
		Preview   template.HTML
		FlowError string
	}{
		Preview:   template.HTML(preview),
		FlowError: flowErr,
	}
	executeTemplate(w, templates.Lookup("compose_preview_partial.html"), data)
}

type techStackAgentMapping struct {
	agents []string
}

var techStackToAgents = map[string]techStackAgentMapping{
	"go":              {agents: []string{"backend-go-developer", "go-readability-reviewer"}},
	"htmx":            {agents: []string{"htmx-picocss-frontend-developer", "htmx-picocss-frontend-reviewer"}},
	"react":           {agents: []string{"react-developer", "react-reviewer"}},
	"python":          {agents: []string{"general-purpose"}},
	"typescript":      {agents: []string{"general-purpose"}},
	"shell":           {agents: []string{"shell-script-coder", "shell-script-reviewer"}},
	"general-purpose": {agents: []string{"general-purpose"}},
	"claudesdk":       {agents: []string{"general-purpose", "agent-sdk-verifier-ts", "agent-sdk-verifier-py"}},
	"openaisdk":       {agents: []string{"general-purpose", "openai-sdk-verifier-ts", "openai-sdk-verifier-py"}},
}

func updateAgentsFromTechStack(cs *composerSession) {
	neededAgents := make(map[string]bool)
	neededAgents["coordinator"] = true

	for _, tech := range cs.wizard.TechStack {
		if mapping, ok := techStackToAgents[tech]; ok {
			for _, agent := range mapping.agents {
				neededAgents[agent] = true
			}
		}
	}

	agentIndex := make(map[string]int)
	for i, a := range cs.state.Agents {
		agentIndex[a.Name] = i
	}

	for agentName := range neededAgents {
		if _, exists := agentIndex[agentName]; !exists {
			cs.state.Agents = append(cs.state.Agents, composerAgentConf{
				Name:     agentName,
				Selected: false,
				Model:    defaultAgentModel,
			})
			agentIndex[agentName] = len(cs.state.Agents) - 1
		}
	}

	for i := range cs.state.Agents {
		if cs.state.Agents[i].Name == "coordinator" {
			cs.state.Agents[i].Selected = true
			continue
		}
		cs.state.Agents[i].Selected = neededAgents[cs.state.Agents[i].Name]
	}

	if cs.wizard.SafetyAnalysis {
		ensureAgentExists(cs, "stpa-analyst")
		setAgentSelected(cs, "stpa-analyst", true)
	}

	regenerateFlowFromAgents(cs)
}

func updateSafetyAgents(cs *composerSession) {
	ensureAgentExists(cs, "stpa-analyst")
	setAgentSelected(cs, "stpa-analyst", cs.wizard.SafetyAnalysis)
	regenerateFlowFromAgents(cs)
}

func ensureAgentExists(cs *composerSession, name string) {
	for _, a := range cs.state.Agents {
		if a.Name == name {
			return
		}
	}
	cs.state.Agents = append(cs.state.Agents, composerAgentConf{
		Name:     name,
		Selected: false,
		Model:    defaultAgentModel,
	})
}

func setAgentSelected(cs *composerSession, name string, selected bool) {
	for i := range cs.state.Agents {
		if cs.state.Agents[i].Name == name {
			cs.state.Agents[i].Selected = selected
			return
		}
	}
}

func isAgentSelected(cs *composerSession, name string) bool {
	for _, a := range cs.state.Agents {
		if a.Name == name && a.Selected {
			return true
		}
	}
	return false
}

type developerReviewerPair struct {
	developer string
	reviewer  string
}

var developerReviewerPairs = []developerReviewerPair{
	{"backend-go-developer", "go-readability-reviewer"},
	{"htmx-picocss-frontend-developer", "htmx-picocss-frontend-reviewer"},
	{"react-developer", "react-reviewer"},
	{"shell-script-coder", "shell-script-reviewer"},
	{"webmaster", "htmx-picocss-frontend-reviewer"},
}

var sdkVerifiers = []string{
	"agent-sdk-verifier-ts",
	"agent-sdk-verifier-py",
	"openai-sdk-verifier-ts",
	"openai-sdk-verifier-py",
}

func regenerateFlowFromAgents(cs *composerSession) {
	ensureReviewersSelected(cs)

	var flowLines []string

	hasStpa := isAgentSelected(cs, "stpa-analyst")

	var terminalDevelopers []string

	for _, pair := range developerReviewerPairs {
		devSelected := isAgentSelected(cs, pair.developer)
		revSelected := isAgentSelected(cs, pair.reviewer)
		if devSelected && revSelected {
			flowLines = append(flowLines, fmt.Sprintf(`"%s" -> "%s"`, pair.developer, pair.reviewer))
			if hasStpa {
				flowLines = append(flowLines, fmt.Sprintf(`"%s" -> "stpa-analyst"`, pair.reviewer))
			}
		} else if devSelected {
			terminalDevelopers = append(terminalDevelopers, pair.developer)
		}
	}

	if hasStpa {
		for _, dev := range terminalDevelopers {
			flowLines = append(flowLines, fmt.Sprintf(`"%s" -> "stpa-analyst"`, dev))
		}
	}

	hasGeneralPurpose := isAgentSelected(cs, "general-purpose")
	if hasGeneralPurpose {
		for _, verifier := range sdkVerifiers {
			if isAgentSelected(cs, verifier) {
				flowLines = append(flowLines, fmt.Sprintf(`"general-purpose" -> "%s"`, verifier))
			}
		}
		if hasStpa {
			flowLines = append(flowLines, `"general-purpose" -> "stpa-analyst"`)
		}
	}

	if isC4ChainSelected(cs) {
		flowLines = appendC4FlowLines(cs, flowLines)
	}

	for _, a := range cs.state.Agents {
		if !a.Selected || a.Name == "coordinator" {
			continue
		}
		if !isAgentInFlow(flowLines, a.Name) {
			flowLines = append(flowLines, fmt.Sprintf(`"%s"`, a.Name))
		}
	}

	cs.state.Flow = strings.Join(flowLines, "\n")
}

func applyTemplateWizardState(cs *composerSession, tmpl *workflowTemplate) {
	cs.wizard = defaultWizardState()
	cs.wizard.FromTemplate = tmpl.ID
	cs.wizard.TechStack = techStackFromAgents(tmpl.Agents)
	cs.wizard.SafetyAnalysis = agentSelected(tmpl.Agents, "stpa-analyst")
}

func syncWizardState(wizard wizardState, state composerState) wizardState {
	if len(wizard.TechStack) == 0 {
		wizard.TechStack = techStackFromAgents(state.Agents)
	}
	if !wizard.SafetyAnalysis {
		wizard.SafetyAnalysis = agentSelected(state.Agents, "stpa-analyst")
	}
	return wizard
}

func techStackFromAgents(agents []composerAgentConf) []string {
	selected := make(map[string]bool)
	for _, agent := range agents {
		if agent.Selected {
			selected[agent.Name] = true
		}
	}

	var stack []string
	if selected["general-purpose"] {
		stack = append(stack, "general-purpose")
	}
	if selected["backend-go-developer"] {
		stack = append(stack, "go")
	}
	if selected["htmx-picocss-frontend-developer"] {
		stack = append(stack, "htmx")
	}
	if selected["react-developer"] {
		stack = append(stack, "react")
	}
	if selected["shell-script-coder"] {
		stack = append(stack, "shell")
	}
	if selected["agent-sdk-verifier-ts"] || selected["agent-sdk-verifier-py"] {
		stack = append(stack, "claudesdk")
	}
	if selected["openai-sdk-verifier-ts"] || selected["openai-sdk-verifier-py"] {
		stack = append(stack, "openaisdk")
	}
	return stack
}

func agentSelected(agents []composerAgentConf, name string) bool {
	for _, agent := range agents {
		if agent.Name == name && agent.Selected {
			return true
		}
	}
	return false
}

func ensureReviewersSelected(cs *composerSession) {
	for _, pair := range developerReviewerPairs {
		if isAgentSelected(cs, pair.developer) {
			ensureAgentExists(cs, pair.reviewer)
			setAgentSelected(cs, pair.reviewer, true)
		}
	}
}

func isC4ChainSelected(cs *composerSession) bool {
	return isAgentSelected(cs, "c4-code") ||
		isAgentSelected(cs, "c4-component") ||
		isAgentSelected(cs, "c4-container") ||
		isAgentSelected(cs, "c4-context")
}

func appendC4FlowLines(cs *composerSession, flowLines []string) []string {
	c4Chain := []string{"c4-code", "c4-component", "c4-container", "c4-context"}
	var selectedC4 []string
	for _, name := range c4Chain {
		if isAgentSelected(cs, name) {
			selectedC4 = append(selectedC4, name)
		}
	}
	for i := 0; i < len(selectedC4)-1; i++ {
		flowLines = append(flowLines, fmt.Sprintf(`"%s" -> "%s"`, selectedC4[i], selectedC4[i+1]))
	}
	return flowLines
}

func isAgentInFlow(flowLines []string, agentName string) bool {
	quoted := fmt.Sprintf(`"%s"`, agentName)
	for _, line := range flowLines {
		if strings.Contains(line, quoted) {
			return true
		}
	}
	return false
}

func (s *Server) handleComposeWizardFinish(w http.ResponseWriter, _ *http.Request, workspacePath string) {
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

	data := wizardPageData{
		Directory:  workspacePath,
		DirName:    filepath.Base(workspacePath),
		Step:       5,
		TotalSteps: 4,
		State:      currentState,
		Agents:     agentInfos,
		Models:     models,
		Preview:    template.HTML(preview),
		FlowError:  flowErr,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	executeTemplate(w, templates.Lookup("compose_wizard_finish.html"), data)
}

func (s *Server) handleComposeTemplatesJSON(w http.ResponseWriter, _ *http.Request, _ string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(workflowTemplates); err != nil {
		http.Error(w, "encoding failed", http.StatusInternalServerError)
	}
}
