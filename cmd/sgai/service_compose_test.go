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
	assert.NotNil(t, result.State)
}

func TestComposeSaveService(t *testing.T) {
	t.Run("savesGoal", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")

		result, err := server.composeSaveService(wsDir, "")
		require.NoError(t, err)
		assert.True(t, result.Saved)
		assert.Equal(t, "test-ws", result.Workspace)

		goalPath := filepath.Join(wsDir, "GOAL.md")
		_, errStat := os.Stat(goalPath)
		assert.NoError(t, errStat)
	})

	t.Run("etagMismatch", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# existing"), 0644))

		_, err := server.composeSaveService(wsDir, `"wrong-etag"`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "modified")
	})

	t.Run("etagMatch", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		content := []byte("# existing")
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), content, 0644))

		etag := computeEtag(content)
		result, err := server.composeSaveService(wsDir, etag)
		require.NoError(t, err)
		assert.True(t, result.Saved)
	})
}

func TestComposeTemplatesService(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.composeTemplatesService()
	assert.NotEmpty(t, result.Templates)
}

func TestComposePreviewService(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result, err := server.composePreviewService(wsDir)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)
}

func TestComposeStateServiceWithInvalidFlow(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "flow-err-ws")
	server.composeDraftService(wsDir, composerState{
		Description: "Test",
		Flow:        `digraph G { "a" -> }`,
	}, wizardState{})
	result := server.composeStateService(wsDir)
	assert.NotEmpty(t, result.FlowError)
}

func TestComposePreviewServiceWithInvalidFlow(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "preview-flow-ws")
	server.composeDraftService(wsDir, composerState{
		Description: "Test",
		Flow:        `digraph G { "a" -> }`,
	}, wizardState{})
	result, err := server.composePreviewService(wsDir)
	require.NoError(t, err)
	assert.NotEmpty(t, result.FlowError)
}

func TestComposePreviewServiceWithValidFlow(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "preview-valid-ws")
	server.composeDraftService(wsDir, composerState{
		Description: "Test",
		Flow:        `"a" -> "b"`,
	}, wizardState{})
	result, err := server.composePreviewService(wsDir)
	require.NoError(t, err)
	assert.Empty(t, result.FlowError)
	assert.NotEmpty(t, result.Content)
}

func TestComposeStateServiceWithValidFlow(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "state-valid-ws")
	server.composeDraftService(wsDir, composerState{
		Description: "Test",
		Flow:        `"x" -> "y"`,
	}, wizardState{})
	result := server.composeStateService(wsDir)
	assert.Empty(t, result.FlowError)
}

func TestComposeDraftService(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	result := server.composeDraftService(wsDir, composerState{
		Description: "Test description",
		Tasks:       "Test tasks",
	}, wizardState{})
	assert.True(t, result.Saved)

	stateResult := server.composeStateService(wsDir)
	assert.Equal(t, "Test description", stateResult.State.Description)
}
