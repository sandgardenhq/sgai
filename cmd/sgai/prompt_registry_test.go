package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModeSectionForMode(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		wantNonEmpty  bool
		wantCoordPlan bool
	}{
		{
			name:          "selfDrive",
			mode:          state.ModeSelfDrive,
			wantNonEmpty:  true,
			wantCoordPlan: true,
		},
		{
			name:          "continuous",
			mode:          state.ModeContinuous,
			wantNonEmpty:  true,
			wantCoordPlan: true,
		},
		{
			name:          "interactive",
			mode:          state.ModeInteractive,
			wantNonEmpty:  true,
			wantCoordPlan: true,
		},
		{
			name:          "unknown",
			mode:          "unknown-mode",
			wantNonEmpty:  true,
			wantCoordPlan: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modeSection, coordPlan := modeSectionForMode(tt.mode)
			if tt.wantNonEmpty {
				assert.NotEmpty(t, modeSection)
			}
			if tt.wantCoordPlan {
				assert.NotEmpty(t, coordPlan)
			} else {
				assert.Empty(t, coordPlan)
			}
		})
	}
}

func TestCoordinatorPromptsSayRetrospectiveIsOptIn(t *testing.T) {
	paths := []string{
		filepath.Join("skel", ".sgai", "agent", "coordinator.md"),
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			content, errRead := os.ReadFile(path)
			assert.NoError(t, errRead)

			prompt := string(content)
			assert.Contains(t, prompt, "default: disabled when absent or empty")
		})
	}
}

func TestApprovedExecutionModesSuppressWorkflowChoiceQuestions(t *testing.T) {
	for _, mode := range []string{state.ModeInteractive, state.ModeSelfDrive} {
		t.Run(mode, func(t *testing.T) {
			modeSection, _ := modeSectionForMode(mode)

			assert.Contains(t, modeSection, "Do NOT ask workflow-choice questions")
			assert.Contains(t, modeSection, "task decomposition")
			assert.Contains(t, modeSection, "direct implementation")
			assert.Contains(t, modeSection, "plan mode")
		})
	}
}

func TestCoordinatorDelegationPromptStatesCoordinatorOwnedSubagentDelegation(t *testing.T) {
	prompt := buildCoordinatorDelegationMessage([]string{"coordinator", "go-reviewer"}, t.TempDir(), state.ModeInteractive)

	assert.Contains(t, prompt, "SGAI runs this top-level session as the coordinator")
	assert.Contains(t, prompt, "Delegate by calling the Task tool")
	assert.Contains(t, prompt, "Do not use bash, shell commands, opencode, or opencode run to delegate work")
	assert.Contains(t, prompt, "Available Task Subagents for Delegation")
	assert.Contains(t, prompt, "go-reviewer")
	assert.Contains(t, prompt, ".sgai/PROJECT_MANAGEMENT.md is the shared ledger for inter-agent state, handoffs, blockers, questions, and completion evidence")
	assert.Contains(t, prompt, "delegate with the Task tool only")
}

func TestCoordinatorSkeletonDeniesOnlyOpencodeBashDelegation(t *testing.T) {
	content, errRead := os.ReadFile("skel/.sgai/agent/coordinator.md")
	require.NoError(t, errRead)
	prompt := string(content)

	assert.NotContains(t, prompt, "\n  bash: deny\n")
	assert.Contains(t, prompt, "  bash:\n")
	assert.Contains(t, prompt, "    opencode: deny")
	assert.Contains(t, prompt, "    \"opencode *\": deny")
	assert.Contains(t, prompt, "    \"*/opencode\": deny")
	assert.Contains(t, prompt, "    \"*/opencode *\": deny")
}
