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
