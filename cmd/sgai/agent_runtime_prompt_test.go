package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAgentMessageIncludesCoordinatorSections(t *testing.T) {
	result := buildAgentMessage(agentRunConfig{dir: t.TempDir(), agent: "coordinator"}, state.Workflow{InteractionMode: state.ModeInteractive}, GoalMetadata{})

	assert.Contains(t, result, promptSectionPreamble)
	assert.Contains(t, result, promptSectionHumanCommDirect)
	assert.Contains(t, result, promptSectionMessaging)
	assert.Contains(t, result, promptSectionProjectManagementMonitor)
	assert.Contains(t, result, "## Available Task Subagents for Delegation\n(none configured)")
	assert.Contains(t, result, "CRITICALLY IMPORTANT: use as many Task subagents as possible at all times")
	assert.Contains(t, result, "launch every safe Task subagent concurrently")
	assert.Contains(t, result, promptSectionPostSkillsCoordinator)
	assert.Contains(t, result, promptSectionTailCoordinator)
	assert.Contains(t, result, promptSectionCoordinatorMessagingTail)
}

func TestBuildAgentMessageIncludesConfiguredSubagents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "react.md"), []byte("---\ndescription: React cleanup\n---\n"), 0o644))

	msg := buildAgentMessage(agentRunConfig{dir: dir, agent: "coordinator"}, state.Workflow{InteractionMode: state.ModeInteractive}, GoalMetadata{Agents: []string{"coordinator", "react"}})

	assert.Contains(t, msg, "react: React cleanup")
	assert.NotContains(t, delegationSection(msg), "coordinator")
	assert.Contains(t, msg, promptSectionInteractiveMode)
}

func TestPromptSkillInstructionsUseRegisteredLoaderSyntax(t *testing.T) {
	result := buildAgentMessage(agentRunConfig{dir: t.TempDir(), agent: "coordinator"}, state.Workflow{InteractionMode: state.ModeInteractive}, GoalMetadata{})

	assert.Contains(t, result, `find_skills({"name":"set-workflow-state"})`)
	assert.Contains(t, result, `skill({"name":"set-workflow-state"})`)
	assert.Contains(t, result, `find_skills({"name":""})`)
	assert.Contains(t, result, `skill({"name":"skill-name"})`)
	assert.NotContains(t, result, `CALL skills({`)
	assert.NotContains(t, result, `Use skills({`)
	assert.NotContains(t, result, `with skills({`)
}
