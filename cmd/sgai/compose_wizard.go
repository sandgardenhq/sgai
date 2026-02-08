package main

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

func syncWizardState(wizard wizardState, st composerState) wizardState {
	if len(wizard.TechStack) == 0 {
		wizard.TechStack = techStackFromAgents(st.Agents)
	}
	if !wizard.SafetyAnalysis {
		wizard.SafetyAnalysis = agentSelected(st.Agents, "stpa-analyst")
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
