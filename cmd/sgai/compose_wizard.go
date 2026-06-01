package main

type workflowTemplate struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Agents      []composerAgentConf
}

const defaultCoordinatorModel = "openai/gpt-5.5 (xhigh)"

func workflowTemplates() []workflowTemplate {
	return []workflowTemplate{
		{
			ID:          "backend",
			Name:        "Go Development",
			Description: "Go implementation and review wrapper",
			Icon:        "⚙️",
			Agents: []composerAgentConf{
				{Name: "go", Selected: true},
			},
		},
		{
			ID:          "frontend",
			Name:        "Frontend — HTMX",
			Description: "HTMX/PicoCSS implementation and review wrapper",
			Icon:        "🎨",
			Agents: []composerAgentConf{
				{Name: "htmx-picocss", Selected: true},
			},
		},
		{
			ID:          "fullstack",
			Name:        "Full Stack",
			Description: "Go backend and HTMX frontend wrapper agents",
			Icon:        "🚀",
			Agents: []composerAgentConf{
				{Name: "go", Selected: true},
				{Name: "htmx-picocss", Selected: true},
			},
		},
		{
			ID:          "research",
			Name:        "Research & Analysis",
			Description: "General-purpose agent with coordinator-led project critique",
			Icon:        "🔬",
			Agents: []composerAgentConf{
				{Name: "general-purpose", Selected: true},
			},
		},
		{
			ID:          "custom",
			Name:        "Custom",
			Description: "Start with a blank slate — pick your own agents",
			Icon:        "🛠️",
		},
		{
			ID:          "react",
			Name:        "Frontend — React",
			Description: "React implementation and review wrapper",
			Icon:        "⚛️",
			Agents: []composerAgentConf{
				{Name: "react", Selected: true},
			},
		},
		{
			ID:          "shell",
			Name:        "Shell Scripting",
			Description: "Shell script implementation and review wrapper",
			Icon:        "🐚",
			Agents: []composerAgentConf{
				{Name: "shell-script", Selected: true},
			},
		},
		{
			ID:          "website",
			Name:        "Marketing Website",
			Description: "Website implementation and review wrapper",
			Icon:        "🌐",
			Agents: []composerAgentConf{
				{Name: "webmaster", Selected: true},
			},
		},
		{
			ID:          "c4docs",
			Name:        "C4 Architecture Docs",
			Description: "C4 model documentation chain: code → component → container → context",
			Icon:        "📐",
			Agents: []composerAgentConf{
				{Name: "c4-code", Selected: true},
				{Name: "c4-component", Selected: true},
				{Name: "c4-container", Selected: true},
				{Name: "c4-context", Selected: true},
			},
		},
		{
			ID:          "claudesdk",
			Name:        "Claude SDK App",
			Description: "Build with Claude Agent SDK, verified for TS and Python",
			Icon:        "🤖",
			Agents: []composerAgentConf{
				{Name: "general-purpose", Selected: true},
				{Name: "agent-sdk-verifier-ts", Selected: true},
				{Name: "agent-sdk-verifier-py", Selected: true},
			},
		},
		{
			ID:          "openaisdk",
			Name:        "OpenAI SDK App",
			Description: "Build with OpenAI Agents SDK, verified for TS and Python",
			Icon:        "🧠",
			Agents: []composerAgentConf{
				{Name: "general-purpose", Selected: true},
				{Name: "openai-sdk-verifier-ts", Selected: true},
				{Name: "openai-sdk-verifier-py", Selected: true},
			},
		},
	}
}

type wizardState struct {
	CurrentStep    int
	FromTemplate   string
	Description    string
	TechStack      []string
	SafetyAnalysis bool
	Retrospective  bool
	CompletionGate string
}

func defaultWizardState() wizardState {
	return wizardState{
		CurrentStep:    1,
		SafetyAnalysis: false,
	}
}

type techStackItem struct {
	ID       string
	Name     string
	Selected bool
}

func defaultTechStackItems() []techStackItem {
	items := [...]techStackItem{
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
	return items[:]
}

func syncWizardState(wizard wizardState, st composerState) wizardState {
	if len(wizard.TechStack) == 0 {
		wizard.TechStack = techStackFromAgents(st.Agents)
	}
	if !wizard.SafetyAnalysis {
		wizard.SafetyAnalysis = st.SafetyAnalysis
	}
	wizard.Retrospective = st.Retrospective
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
	if selected["go"] {
		stack = append(stack, "go")
	}
	if selected["htmx-picocss"] {
		stack = append(stack, "htmx")
	}
	if selected["react"] {
		stack = append(stack, "react")
	}
	if selected["shell-script"] {
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
