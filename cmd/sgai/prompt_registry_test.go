package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
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
			name:          "building",
			mode:          state.ModeBuilding,
			wantNonEmpty:  true,
			wantCoordPlan: true,
		},
		{
			name:          "retrospective",
			mode:          state.ModeRetrospective,
			wantNonEmpty:  true,
			wantCoordPlan: true,
		},
		{
			name:          "brainstorming",
			mode:          state.ModeBrainstorming,
			wantNonEmpty:  true,
			wantCoordPlan: false,
		},
		{
			name:          "unknown",
			mode:          "unknown-mode",
			wantNonEmpty:  true,
			wantCoordPlan: false,
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
			assert.NotContains(t, prompt, "default: enabled when absent or truish")
			assert.Contains(t, prompt, "default: disabled when absent or empty")
		})
	}
}

func TestBuildingModePromptDoesNotSayRetrospectiveIsMandatory(t *testing.T) {
	_, coordPlan := modeSectionForMode(state.ModeBuilding)

	assert.NotContains(t, coordPlan, "retrospective step is MANDATORY")
	assert.NotContains(t, coordPlan, "Do NOT mark status:complete until the retrospective has finished")
}

func TestCoordinatorDelegationPromptStatesCoordinatorOwnedSubagentDelegation(t *testing.T) {
	prompt := buildCoordinatorDelegationMessage([]string{"coordinator", "go-reviewer"}, map[string]int{"coordinator": 1, "go-reviewer": 0}, t.TempDir(), state.ModeBuilding)

	assert.Contains(t, prompt, "SGAI runs this top-level OpenCode session as the coordinator")
	assert.Contains(t, prompt, "Delegate to available subagents through OpenCode's subagent/delegation mechanisms")
	assert.Contains(t, prompt, "Available OpenCode Subagents for Delegation")
	assert.Contains(t, prompt, "go-reviewer")
	assert.Contains(t, prompt, ".sgai/PROJECT_MANAGEMENT.md is the shared ledger for inter-agent state, handoffs, blockers, questions, and completion evidence")
	assert.Contains(t, prompt, "do not use navigate to cycle SGAI through the GOAL agents")
}

func TestNonCoordinatorPromptReturnsThroughDelegationAndLedgerHandoff(t *testing.T) {
	prompt := composeCoordinatorPromptTemplate("go")

	assert.Contains(t, prompt, "append QUESTION: <your question> to PROJECT_MANAGEMENT.md")
	assert.Contains(t, prompt, "return through OpenCode's subagent/delegation mechanism")
	assert.Contains(t, prompt, "only the coordinator can mark checkboxes")
	assert.Contains(t, prompt, "append status updates to .sgai/PROJECT_MANAGEMENT.md")
}

func TestModePromptsDoNotPresentRetrospectiveAsMandatory(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "building", mode: state.ModeBuilding},
		{name: "retrospective", mode: state.ModeRetrospective},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modeSection, coordPlan := modeSectionForMode(tt.mode)
			prompt := modeSection + "\n" + coordPlan

			assert.NotContains(t, prompt, "mandatory")
			assert.NotContains(t, prompt, "Do NOT skip the retrospective")
		})
	}
}
