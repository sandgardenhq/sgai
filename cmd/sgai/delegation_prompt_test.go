package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeCoordinatorPromptTemplateCoordinatorSections(t *testing.T) {
	result := composeCoordinatorPromptTemplate("coordinator")

	assert.Contains(t, result, promptSectionPreamble)
	assert.Contains(t, result, promptSectionHumanCommDirect)
	assert.Contains(t, result, promptSectionMessaging)
	assert.Contains(t, result, promptSectionProjectManagementMonitor)
	assert.Contains(t, result, promptSectionDelegation)
	assert.Contains(t, result, promptSectionPostSkillsCoordinator)
	assert.Contains(t, result, promptSectionTailCoordinator)
	assert.Contains(t, result, promptSectionCoordinatorMessagingTail)
}

func TestBuildCoordinatorDelegationMessageIncludesConfiguredSubagents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "react.md"), []byte("---\ndescription: React cleanup\n---\n"), 0o644))

	msg := buildCoordinatorDelegationMessage([]string{"coordinator", "react"}, map[string]int{"coordinator": 2, "react": 1}, dir, state.ModeBuilding)

	assert.Contains(t, msg, "react: React cleanup")
	assert.NotContains(t, delegationSection(msg), "coordinator")
	assert.Contains(t, msg, "coordinator: 2 visits")
	assert.Contains(t, msg, "react: 1 visits")
	assert.Contains(t, msg, promptSectionBuildingMode)
}

func TestPromptSkillInstructionsUseRegisteredLoaderSyntax(t *testing.T) {
	result := composeCoordinatorPromptTemplate("coordinator")

	assert.Contains(t, result, `find_skills({"name":"set-workflow-state"})`)
	assert.Contains(t, result, `skill({"name":"set-workflow-state"})`)
	assert.Contains(t, result, `find_skills({"name":""})`)
	assert.Contains(t, result, `skill({"name":"skill-name"})`)
	assert.NotContains(t, result, `CALL skills({`)
	assert.NotContains(t, result, `Use skills({`)
	assert.NotContains(t, result, `with skills({`)
}
