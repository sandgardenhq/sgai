package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnpackSkeletonRefreshesManagedFilesAndPreservesRuntimeFiles(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	agentDir := filepath.Join(sgaiDir, "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "coordinator.md"), []byte("custom coordinator"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"working"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("project notes"), 0644))

	require.NoError(t, unpackSkeleton(dir))

	coordinator, errRead := os.ReadFile(filepath.Join(agentDir, "coordinator.md"))
	require.NoError(t, errRead)
	assert.NotEqual(t, "custom coordinator", string(coordinator))
	assert.FileExists(t, filepath.Join(sgaiDir, "skills", "set-workflow-state", "SKILL.md"))
	content, errRead := os.ReadFile(filepath.Join(sgaiDir, "state.json"))
	require.NoError(t, errRead)
	assert.Equal(t, `{"status":"working"}`, string(content))
	content, errRead = os.ReadFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"))
	require.NoError(t, errRead)
	assert.Equal(t, "project notes", string(content))
}
