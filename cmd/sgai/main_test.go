package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYAMLFrontmatterAgentsAndModel(t *testing.T) {
	content := []byte(`---
agents:
  - go
  - react
model: "openai/gpt-5.5 (xhigh)"
interactive: true
completionGateScript: make test
retrospective: true
---
# Goal
`)

	metadata, errParse := parseYAMLFrontmatter(content)

	require.NoError(t, errParse)
	assert.Equal(t, []string{"go", "react"}, metadata.Agents)
	assert.Equal(t, "openai/gpt-5.5 (xhigh)", metadata.Model)
	assert.Equal(t, "make test", metadata.CompletionGateScript)
	assert.Equal(t, "true", metadata.Retrospective)
}

func TestBuildAgentArgsCoordinatorModelVariant(t *testing.T) {
	args := buildAgentArgs("coordinator", "openai/gpt-5.5 (xhigh)", "")

	assert.Equal(t, []string{
		"run",
		"--format=json",
		"--agent",
		"coordinator",
		"--model",
		"openai/gpt-5.5",
		"--variant",
		"xhigh",
		"--title",
		"coordinator [openai/gpt-5.5 (xhigh)]",
		"--thinking",
	}, args)
}

func TestBuildAgentMessageListsConfiguredAgentsForCoordinatorDelegation(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "go.md"), []byte("---\ndescription: Go backend work\n---\n"), 0o644))

	msg := buildAgentMessage(agentRunConfig{dir: dir, agent: "coordinator"}, state.Workflow{VisitCounts: map[string]int{"coordinator": 1, "go": 0}}, GoalMetadata{Agents: []string{"go"}})

	assert.Contains(t, msg, "## Available OpenCode Subagents for Delegation")
	assert.Contains(t, msg, "go: Go backend work")
	assert.NotContains(t, delegationSection(msg), "coordinator")
	assert.Contains(t, msg, "coordinator: 1 visits")
	assert.Contains(t, msg, "go: 0 visits")
}

func delegationSection(msg string) string {
	start := strings.Index(msg, "## Available OpenCode Subagents for Delegation")
	if start < 0 {
		return ""
	}
	section := msg[start:]
	end := strings.Index(section, "## Visit Counts")
	if end < 0 {
		return section
	}
	return section[:end]
}

func TestHandleCompleteStatusDoesNotRouteRetrospectiveAgent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	coord, errCoord := state.NewCoordinatorWith(filepath.Join(dir, ".sgai", "state.json"), state.Workflow{})
	require.NoError(t, errCoord)

	result := handleCompleteStatus(t.Context(), agentRunConfig{dir: dir, agent: "coordinator", coord: coord, paddedsgai: "sgai"}, state.Workflow{Status: state.StatusComplete, VisitCounts: map[string]int{}}, state.Workflow{}, GoalMetadata{Retrospective: "true", Agents: []string{"go", "retrospective"}})

	assert.Equal(t, state.StatusComplete, result.Status)
	assert.Nil(t, result.Navigate)
}

func TestBuildAllAgentsExcludesSTPAAnalyst(t *testing.T) {
	agents := buildAllAgents([]string{"go", "stpa-analyst", "react"})

	assert.Equal(t, []string{"coordinator", "go", "react"}, agents)
}

func TestParseYAMLFrontmatterExcludesSTPAAnalyst(t *testing.T) {
	content := []byte(`---
agents:
  - go
  - stpa-analyst
  - react
---
# Goal
`)

	metadata, errParse := parseYAMLFrontmatter(content)

	require.NoError(t, errParse)
	assert.Equal(t, []string{"go", "react"}, metadata.Agents)
}

func TestModelFromGoalReadsTopLevelModel(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\nmodel: openai/gpt-5.5 (xhigh)\n---\n# Goal"), 0o644))

	assert.Equal(t, "openai/gpt-5.5 (xhigh)", modelFromGoal(dir))
}

func TestParseModelAndVariant(t *testing.T) {
	model, variant := parseModelAndVariant("openai/gpt-5.5 (xhigh)")
	assert.Equal(t, "openai/gpt-5.5", model)
	assert.Equal(t, "xhigh", variant)

	model, variant = parseModelAndVariant("openai/gpt-5.5")
	assert.Equal(t, "openai/gpt-5.5", model)
	assert.Empty(t, variant)
}

func TestCountPendingTodosIgnoresCoordinatorRuntimeTodos(t *testing.T) {
	wf := state.Workflow{Todos: []state.TodoItem{{Content: "pending", Status: "pending"}}}
	assert.Equal(t, 0, countPendingTodos(wf, "coordinator"))
	assert.Equal(t, 1, countPendingTodos(wf, "go"))
}

func TestSafetyAnalysisComposerTextUsesSkillsNotWorkflowAgents(t *testing.T) {
	content := buildGOALContent(composerState{Description: "Goal", SafetyAnalysis: true, Agents: []composerAgentConf{{Name: "go", Selected: true}}, Model: "openai/gpt-5.5 (xhigh)"})

	assert.Contains(t, content, "stpa-overview")
	assert.NotContains(t, strings.Split(content, "---")[1], "stpa-analyst")
}
