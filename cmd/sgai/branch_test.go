package main

import (
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
)

func TestInteractiveBranchToolsAllowed(t *testing.T) {
	b := &interactiveBranch{}
	assert.True(t, b.toolsAllowed())
}

func TestSelfDriveBranchToolsAllowed(t *testing.T) {
	b := &selfDriveBranch{}
	assert.False(t, b.toolsAllowed())
}

func TestContinuousBranchToolsAllowed(t *testing.T) {
	b := &continuousBranch{}
	assert.False(t, b.toolsAllowed())
}

func TestDispatchBranch(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{
			name:     "continuousMode",
			mode:     state.ModeContinuous,
			expected: "*main.continuousBranch",
		},
		{
			name:     "selfDriveMode",
			mode:     state.ModeSelfDrive,
			expected: "*main.selfDriveBranch",
		},
		{
			name:     "brainstormingMode",
			mode:     state.ModeBrainstorming,
			expected: "*main.interactiveBranch",
		},
		{
			name:     "unknownMode",
			mode:     "unknown",
			expected: "*main.interactiveBranch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch := dispatchBranch(tt.mode)
			assert.NotNil(t, branch)
			assert.IsType(t, branch, dispatchBranch(tt.mode))
		})
	}
}
