package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeStateService(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.composeStateService(wsDir)

	assert.Equal(t, "test-ws", result.Workspace)
	assert.Equal(t, defaultCoordinatorModel, result.State.Model)
}

func TestComposeSaveServiceWritesAgentsAndModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.composeDraftService(wsDir, composerState{Model: "openai/gpt-5.5 (xhigh)", Agents: []composerAgentConf{{Name: "coordinator", Selected: true}, {Name: "go", Selected: true}}}, wizardState{})

	result, errSave := server.composeSaveService(wsDir, "")

	require.NoError(t, errSave)
	assert.True(t, result.Saved)
	goalContent, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Contains(t, string(goalContent), "agents:\n  - \"go\"")
	assert.NotContains(t, string(goalContent), "coordinator")
	assert.Contains(t, string(goalContent), "model: \"openai/gpt-5.5 (xhigh)\"")
}

func TestComposePreviewServiceReturnsAgentsAndModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.composeDraftService(wsDir, composerState{Model: "openai/gpt-5.5 (xhigh)", Agents: []composerAgentConf{{Name: "coordinator", Selected: true}, {Name: "go", Selected: true}}}, wizardState{})

	result, errPreview := server.composePreviewService(wsDir)

	require.NoError(t, errPreview)
	assert.Contains(t, result.Content, "agents:\n  - \"go\"")
	assert.NotContains(t, result.Content, "coordinator")
	assert.Contains(t, result.Content, "model: \"openai/gpt-5.5 (xhigh)\"")
}

func TestComposeDraftService(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.composeDraftService(wsDir, composerState{Description: "Test description", Tasks: "Test tasks", Retrospective: true}, wizardState{})

	assert.True(t, result.Saved)
	stateResult := server.composeStateService(wsDir)
	assert.Equal(t, "Test description", stateResult.State.Description)
	assert.True(t, stateResult.State.Retrospective)
}
