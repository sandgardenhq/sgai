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

	st := composerState{
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

func buildGOALContent(st composerState) string {
	var buf bytes.Buffer

	buf.WriteString("---\n")

	if st.Flow != "" {
		buf.WriteString("flow: |\n")
		for line := range strings.SplitSeq(st.Flow, "\n") {
			buf.WriteString("  ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	hasSelectedAgents := false
	for _, a := range st.Agents {
		if a.Selected {
			hasSelectedAgents = true
			break
		}
	}

	if hasSelectedAgents {
		buf.WriteString("models:\n")
		for _, a := range st.Agents {
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

	if st.Interactive != "" {
		buf.WriteString("interactive: ")
		buf.WriteString(st.Interactive)
		buf.WriteString("\n")
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

	if st.Tasks != "" {
		buf.WriteString("## Tasks\n\n")
		buf.WriteString(st.Tasks)
		buf.WriteString("\n")
	}

	return buf.String()
}
