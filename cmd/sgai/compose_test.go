package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultComposerStateUsesModelWithoutSelectableCoordinator(t *testing.T) {
	state := defaultComposerState()

	assert.Equal(t, defaultCoordinatorModel, state.Model)
	assert.Empty(t, state.Agents)
}

func TestLoadComposerStateFromDiskReadsAgentsAndModel(t *testing.T) {
	dir := t.TempDir()
	goalContent := `---
agents:
  - go
  - react
model: openai/gpt-5.5 (xhigh)
completionGateScript: make test
retrospective: true
---
# My Project

Description here.

## Tasks

- Task 1
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte(goalContent), 0o644))

	state := loadComposerStateFromDisk(dir)

	assert.Equal(t, "My Project\n\nDescription here.", state.Description)
	assert.Equal(t, "make test", state.CompletionGate)
	assert.True(t, state.Retrospective)
	assert.Equal(t, "openai/gpt-5.5 (xhigh)", state.Model)
	assert.Equal(t, "- Task 1", state.Tasks)
	require.Len(t, state.Agents, 2)
	assert.Equal(t, "go", state.Agents[0].Name)
	assert.Equal(t, "react", state.Agents[1].Name)
}

func TestBuildGOALContentWritesAgentsAndModel(t *testing.T) {
	content := buildGOALContent(composerState{
		Description:    "My Project\n\nDescription here.",
		CompletionGate: "make test",
		Retrospective:  true,
		Model:          "openai/gpt-5.5 (xhigh)",
		Agents: []composerAgentConf{
			{Name: "coordinator", Selected: true},
			{Name: "go", Selected: true},
			{Name: "react", Selected: false},
		},
		Tasks: "- Task 1",
	})

	assert.Contains(t, content, "agents:\n  - \"go\"")
	assert.Contains(t, content, "model: \"openai/gpt-5.5 (xhigh)\"")
	assert.Contains(t, content, "completionGateScript: make test")
	assert.Contains(t, content, "retrospective: true")
	assert.NotContains(t, content, "coordinator")
	assert.NotContains(t, content, "react")
	assert.NotContains(t, content, "flow:")
}

func TestBuildGOALContentOmitsEmptyAgentsList(t *testing.T) {
	content := buildGOALContent(composerState{Description: "Goal", Model: "openai/gpt-5.5 (xhigh)", Agents: []composerAgentConf{{Name: "go", Selected: false}}})

	assert.NotContains(t, content, "agents:")
	assert.Contains(t, content, "model: \"openai/gpt-5.5 (xhigh)\"")
}

func TestActiveComposerAgentsSkipsSTPAAnalyst(t *testing.T) {
	agents := activeComposerAgents([]composerAgentConf{{Name: "go", Selected: true}, {Name: "stpa-analyst", Selected: true}, {Name: "coordinator", Selected: true}})

	require.Len(t, agents, 1)
	assert.Equal(t, "go", agents[0].Name)
}

func TestWorkflowTemplatesDoNotExposeCoordinatorAsSelectableAgent(t *testing.T) {
	for _, tmpl := range workflowTemplates() {
		t.Run(tmpl.ID, func(t *testing.T) {
			for _, agent := range tmpl.Agents {
				assert.NotEqual(t, "coordinator", agent.Name)
			}
		})
	}
}

func TestDefaultTechStackItemsReturnsFreshSlice(t *testing.T) {
	first := defaultTechStackItems()
	second := defaultTechStackItems()
	require.NotEmpty(t, first)
	require.NotEmpty(t, second)

	first[0].Name = "changed"

	assert.Equal(t, "Go", second[0].Name)
}
