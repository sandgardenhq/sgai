package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelFromGoalUsesTopLevelModel(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\nmodel: openai/gpt-5.5 (xhigh)\n---\n# Goal"), 0o644))

	assert.Equal(t, "openai/gpt-5.5 (xhigh)", modelFromGoal(dir))
}
