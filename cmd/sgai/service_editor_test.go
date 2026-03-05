package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenEditorServiceEditorNotAvailable(t *testing.T) {
	server := NewServer(t.TempDir())
	server.editorAvailable = false
	_, err := server.openEditorService(t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no editor available")
}

func TestOpenEditorFileServiceEditorNotAvailable(t *testing.T) {
	server := NewServer(t.TempDir())
	server.editorAvailable = false
	_, err := server.openEditorFileService(t.TempDir(), "GOAL.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no editor available")
}

func TestOpenEditorFileServiceFileNotFound(t *testing.T) {
	dir := t.TempDir()
	server := NewServer(t.TempDir())
	server.editorAvailable = true
	server.editor = newConfigurableEditor("echo")
	_, err := server.openEditorFileService(dir, "nonexistent.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestOpenEditorGoalServiceDelegates(t *testing.T) {
	server := NewServer(t.TempDir())
	server.editorAvailable = false
	_, err := server.openEditorGoalService(t.TempDir())
	assert.Error(t, err)
}

func TestOpenEditorProjectManagementServiceDelegates(t *testing.T) {
	server := NewServer(t.TempDir())
	server.editorAvailable = false
	_, err := server.openEditorProjectManagementService(t.TempDir())
	assert.Error(t, err)
}

func TestOpenEditorFileServiceSuccess(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("# Goal"), 0644))
	server := NewServer(t.TempDir())
	server.editorAvailable = true
	server.editor = newConfigurableEditor("echo")
	result, err := server.openEditorFileService(dir, "GOAL.md")
	assert.NoError(t, err)
	assert.True(t, result.Opened)
}
