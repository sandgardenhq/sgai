package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type composerState struct {
	Description    string              `json:"description"`
	CompletionGate string              `json:"completionGate"`
	SafetyAnalysis bool                `json:"safetyAnalysis"`
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

func (srv *Server) getComposerSession(workspacePath string) *composerSession {
	srv.composerSessionsMu.Lock()
	defer srv.composerSessionsMu.Unlock()

	if existing, ok := srv.composerSessions[workspacePath]; ok {
		return existing
	}

	cs := &composerSession{}
	cs.state = loadComposerStateFromDisk(workspacePath)
	cs.wizard = defaultWizardState()
	srv.composerSessions[workspacePath] = cs
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

	bodyContent := string(extractBody(content))
	legacySafetyAnalysis := strings.Contains(metadata.Flow, "stpa-analyst")

	st := composerState{
		Description:    extractDescriptionFromBody(bodyContent),
		CompletionGate: metadata.CompletionGateScript,
		SafetyAnalysis: bodyHasSafetyAnalysis(bodyContent) || legacySafetyAnalysis,
		Flow:           activeComposerFlow(metadata.Flow),
		Tasks:          extractTasksFromBody(bodyContent),
	}

	for agentName, modelVal := range metadata.Models {
		if agentName == "stpa-analyst" {
			st.SafetyAnalysis = true
			continue
		}
		model := ""
		if s, ok := modelVal.(string); ok {
			model = s
		}
		st.Agents = append(st.Agents, composerAgentConf{
			Name:     agentName,
			Selected: true,
			Model:    model,
		})
	}

	slices.SortFunc(st.Agents, func(a, b composerAgentConf) int {
		return strings.Compare(a.Name, b.Name)
	})

	return st
}

func bodyHasSafetyAnalysis(body string) bool {
	return strings.Contains(body, "## Safety Analysis") || strings.Contains(body, "stpa-overview")
}

func defaultComposerState() composerState {
	return composerState{
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true, Model: defaultAgentModel},
		},
	}
}

func extractDescriptionFromBody(body string) string {
	lines := strings.Split(body, "\n")
	var descLines []string
	inTasks := false
	inSafetyAnalysis := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Safety Analysis") {
			inSafetyAnalysis = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && inSafetyAnalysis {
			inSafetyAnalysis = false
		}
		if strings.HasPrefix(trimmed, "## Tasks") || strings.HasPrefix(trimmed, "## Task") {
			inTasks = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && inTasks {
			inTasks = false
		}
		if !inTasks && !inSafetyAnalysis {
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

func buildGOALContent(st composerState) string {
	var buf bytes.Buffer

	buf.WriteString("---\n")

	flow := activeComposerFlow(st.Flow)
	if flow != "" {
		buf.WriteString("flow: |\n")
		for line := range strings.SplitSeq(flow, "\n") {
			buf.WriteString("  ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	agents := activeComposerAgents(st.Agents)
	hasSelectedAgents := false
	for _, a := range agents {
		if a.Selected {
			hasSelectedAgents = true
			break
		}
	}

	if hasSelectedAgents {
		buf.WriteString("models:\n")
		for _, a := range agents {
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

	if st.CompletionGate != "" {
		buf.WriteString("completionGateScript: ")
		buf.WriteString(st.CompletionGate)
		buf.WriteString("\n")
	}

	buf.WriteString("---\n\n")

	if st.Description != "" {
		buf.WriteString(st.Description)
		buf.WriteString("\n\n")
	}

	if st.SafetyAnalysis {
		buf.WriteString("## Safety Analysis\n\n")
		buf.WriteString("- STPA analysis is a skill workflow, not a routable workflow agent; do not add `stpa-analyst` to GOAL `flow` or `models`.\n")
		buf.WriteString("- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.\n")
		buf.WriteString("- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.\n\n")
	}

	if st.Tasks != "" {
		buf.WriteString("## Tasks\n\n")
		buf.WriteString(st.Tasks)
		buf.WriteString("\n")
	}

	return buf.String()
}

func activeComposerFlow(flow string) string {
	var lines []string
	for line := range strings.SplitSeq(flow, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "stpa-analyst") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func activeComposerAgents(agents []composerAgentConf) []composerAgentConf {
	active := make([]composerAgentConf, 0, len(agents))
	for _, agent := range agents {
		if agent.Name == "stpa-analyst" {
			continue
		}
		active = append(active, agent)
	}
	return active
}
