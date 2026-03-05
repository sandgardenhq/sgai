package main

import (
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
