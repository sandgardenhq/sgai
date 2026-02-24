package main

import (
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestDispatchBranch(t *testing.T) {
	t.Run("continuousModeReturnsContinuousBranch", func(t *testing.T) {
		branch := dispatchBranch(state.ModeContinuous)
		if _, ok := branch.(*continuousBranch); !ok {
			t.Errorf("expected *continuousBranch, got %T", branch)
		}
	})

	t.Run("selfDriveModeReturnsSelfDriveBranch", func(t *testing.T) {
		branch := dispatchBranch(state.ModeSelfDrive)
		if _, ok := branch.(*selfDriveBranch); !ok {
			t.Errorf("expected *selfDriveBranch, got %T", branch)
		}
	})

	t.Run("brainstormingModeReturnsInteractiveBranch", func(t *testing.T) {
		branch := dispatchBranch(state.ModeBrainstorming)
		if _, ok := branch.(*interactiveBranch); !ok {
			t.Errorf("expected *interactiveBranch, got %T", branch)
		}
	})

	t.Run("buildingModeReturnsInteractiveBranch", func(t *testing.T) {
		branch := dispatchBranch(state.ModeBuilding)
		if _, ok := branch.(*interactiveBranch); !ok {
			t.Errorf("expected *interactiveBranch, got %T", branch)
		}
	})

	t.Run("emptyModeReturnsInteractiveBranch", func(t *testing.T) {
		branch := dispatchBranch("")
		if _, ok := branch.(*interactiveBranch); !ok {
			t.Errorf("expected *interactiveBranch, got %T", branch)
		}
	})

	t.Run("unknownModeReturnsInteractiveBranch", func(t *testing.T) {
		branch := dispatchBranch("unknown-mode")
		if _, ok := branch.(*interactiveBranch); !ok {
			t.Errorf("expected *interactiveBranch, got %T", branch)
		}
	})
}

func TestDispatchBranchToolsAllowed(t *testing.T) {
	t.Run("continuousBranchDisallowsTools", func(t *testing.T) {
		branch := dispatchBranch(state.ModeContinuous)
		if branch.toolsAllowed() {
			t.Error("continuous branch should not allow tools")
		}
	})

	t.Run("selfDriveBranchDisallowsTools", func(t *testing.T) {
		branch := dispatchBranch(state.ModeSelfDrive)
		if branch.toolsAllowed() {
			t.Error("self-drive branch should not allow tools")
		}
	})

	t.Run("interactiveBranchAllowsTools", func(t *testing.T) {
		branch := dispatchBranch(state.ModeBrainstorming)
		if !branch.toolsAllowed() {
			t.Error("interactive branch should allow tools")
		}
	})
}
