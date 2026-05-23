package main

type workflowTemplate struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Agents      []composerAgentConf
	Flow        string
}

const (
	defaultCoordinatorModel = "openai/gpt-5.5 (xhigh)"
	defaultWorkerModel      = "openai/gpt-5.5 (low)"
)

func defaultModelForAgent(name string) string {
	if name == "coordinator" {
		return defaultCoordinatorModel
	}
	return defaultWorkerModel
}

var workflowTemplates = []workflowTemplate{
	{
		ID:          "backend",
		Name:        "Go Development",
		Description: "Go implementation and review wrapper",
		Icon:        "⚙️",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "go", Selected: true, Model: defaultModelForAgent("go")},
		},
		Flow: `"go"`,
	},
	{
		ID:          "frontend",
		Name:        "Frontend — HTMX",
		Description: "HTMX/PicoCSS implementation and review wrapper",
		Icon:        "🎨",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "htmx-picocss", Selected: true, Model: defaultModelForAgent("htmx-picocss")},
		},
		Flow: `"htmx-picocss"`,
	},
	{
		ID:          "fullstack",
		Name:        "Full Stack",
		Description: "Go backend and HTMX frontend wrapper agents",
		Icon:        "🚀",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "go", Selected: true, Model: defaultModelForAgent("go")},
			{Name: "htmx-picocss", Selected: true, Model: defaultModelForAgent("htmx-picocss")},
		},
		Flow: `"go"
"htmx-picocss"`,
	},
	{
		ID:          "research",
		Name:        "Research & Analysis",
		Description: "General-purpose agent with coordinator-led project critique",
		Icon:        "🔬",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "general-purpose", Selected: true, Model: defaultModelForAgent("general-purpose")},
		},
		Flow: `"general-purpose"`,
	},
	{
		ID:          "custom",
		Name:        "Custom",
		Description: "Start with a blank slate — pick your own agents",
		Icon:        "🛠️",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
		},
		Flow: "",
	},
	{
		ID:          "react",
		Name:        "Frontend — React",
		Description: "React implementation and review wrapper",
		Icon:        "⚛️",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "react", Selected: true, Model: defaultModelForAgent("react")},
		},
		Flow: `"react"`,
	},
	{
		ID:          "shell",
		Name:        "Shell Scripting",
		Description: "Shell script implementation and review wrapper",
		Icon:        "🐚",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "shell-script", Selected: true, Model: defaultModelForAgent("shell-script")},
		},
		Flow: `"shell-script"`,
	},
	{
		ID:          "website",
		Name:        "Marketing Website",
		Description: "Website implementation and review wrapper",
		Icon:        "🌐",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "webmaster", Selected: true, Model: defaultModelForAgent("webmaster")},
		},
		Flow: `"webmaster"`,
	},
	{
		ID:          "c4docs",
		Name:        "C4 Architecture Docs",
		Description: "C4 model documentation chain: code → component → container → context",
		Icon:        "📐",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "c4-code", Selected: true, Model: defaultModelForAgent("c4-code")},
			{Name: "c4-component", Selected: true, Model: defaultModelForAgent("c4-component")},
			{Name: "c4-container", Selected: true, Model: defaultModelForAgent("c4-container")},
			{Name: "c4-context", Selected: true, Model: defaultModelForAgent("c4-context")},
		},
		Flow: `"c4-code" -> "c4-component"
"c4-component" -> "c4-container"
"c4-container" -> "c4-context"`,
	},
	{
		ID:          "claudesdk",
		Name:        "Claude SDK App",
		Description: "Build with Claude Agent SDK, verified for TS and Python",
		Icon:        "🤖",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "general-purpose", Selected: true, Model: defaultModelForAgent("general-purpose")},
			{Name: "agent-sdk-verifier-ts", Selected: true, Model: defaultModelForAgent("agent-sdk-verifier-ts")},
			{Name: "agent-sdk-verifier-py", Selected: true, Model: defaultModelForAgent("agent-sdk-verifier-py")},
		},
		Flow: `"general-purpose" -> "agent-sdk-verifier-ts"
"general-purpose" -> "agent-sdk-verifier-py"`,
	},
	{
		ID:          "openaisdk",
		Name:        "OpenAI SDK App",
		Description: "Build with OpenAI Agents SDK, verified for TS and Python",
		Icon:        "🧠",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultModelForAgent("coordinator")},
			{Name: "general-purpose", Selected: true, Model: defaultModelForAgent("general-purpose")},
			{Name: "openai-sdk-verifier-ts", Selected: true, Model: defaultModelForAgent("openai-sdk-verifier-ts")},
			{Name: "openai-sdk-verifier-py", Selected: true, Model: defaultModelForAgent("openai-sdk-verifier-py")},
		},
		Flow: `"general-purpose" -> "openai-sdk-verifier-ts"
"general-purpose" -> "openai-sdk-verifier-py"`,
	},
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

func agentSelected(agents []composerAgentConf, name string) bool {
	for _, agent := range agents {
		if agent.Name == name && agent.Selected {
			return true
		}
	}
	return false
}
