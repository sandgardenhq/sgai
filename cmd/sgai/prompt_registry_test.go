package main

import (
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestComposePromptSharedSectionsInAllModes(t *testing.T) {
	modes := []string{
		state.ModeBrainstorming,
		state.ModeBuilding,
		state.ModeSelfDrive,
		state.ModeContinuous,
	}

	sharedSections := []string{
		flowSectionPreamble,
		flowSectionMessaging,
		flowSectionWorkFocus,
		flowSectionGuidelines,
		flowSectionCommonTail,
	}

	for _, mode := range modes {
		modeSection, coordPlan := modeSectionForMode(mode)
		msg := composePrompt(promptOptions{
			agent:           "coordinator",
			modeSection:     modeSection,
			coordinatorPlan: coordPlan,
		})
		for _, section := range sharedSections {
			if !strings.Contains(msg, section) {
				t.Errorf("mode %q missing shared section: %q", mode, truncateForTest(section))
			}
		}
	}
}

func TestComposePromptModeSectionInjected(t *testing.T) {
	t.Run("selfDriveInjectsModeSelfDrive", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeSelfDrive)
		msg := composePrompt(promptOptions{
			agent:           "coordinator",
			modeSection:     modeSection,
			coordinatorPlan: coordPlan,
		})
		if !strings.Contains(msg, "SELF-DRIVE MODE ACTIVE") {
			t.Error("self-drive mode should inject SELF-DRIVE MODE ACTIVE")
		}
	})

	t.Run("buildingInjectsBuildingMode", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeBuilding)
		msg := composePrompt(promptOptions{
			agent:           "coordinator",
			modeSection:     modeSection,
			coordinatorPlan: coordPlan,
		})
		if !strings.Contains(msg, "BUILDING MODE ACTIVE") {
			t.Error("building mode should inject BUILDING MODE ACTIVE")
		}
	})

	t.Run("continuousInjectsContinuousMode", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeContinuous)
		msg := composePrompt(promptOptions{
			agent:           "coordinator",
			modeSection:     modeSection,
			coordinatorPlan: coordPlan,
		})
		if !strings.Contains(msg, "CONTINUOUS MODE ACTIVE") {
			t.Error("continuous mode should inject CONTINUOUS MODE ACTIVE")
		}
	})

	t.Run("brainstormingInjectsBrainstormingMode", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeBrainstorming)
		msg := composePrompt(promptOptions{
			agent:           "coordinator",
			modeSection:     modeSection,
			coordinatorPlan: coordPlan,
		})
		if !strings.Contains(msg, "ASK ME QUESTIONS BEFORE BUILDING") {
			t.Error("brainstorming mode should inject ASK ME QUESTIONS BEFORE BUILDING")
		}
	})
}

func TestComposePromptCoordinatorPlanOnlyForCoordinator(t *testing.T) {
	modeSection, coordPlan := modeSectionForMode(state.ModeSelfDrive)

	coordMsg := composePrompt(promptOptions{
		agent:           "coordinator",
		modeSection:     modeSection,
		coordinatorPlan: coordPlan,
	})
	if !strings.Contains(coordMsg, "delegate work to specialized agents") {
		t.Error("coordinator self-drive message should contain delegation instructions")
	}

	agentMsg := composePrompt(promptOptions{
		agent:           "backend-go-developer",
		modeSection:     modeSection,
		coordinatorPlan: coordPlan,
	})
	if strings.Contains(agentMsg, "delegate work to specialized agents") {
		t.Error("non-coordinator self-drive message should not contain delegation instructions")
	}
}

func TestModeSectionForMode(t *testing.T) {
	t.Run("selfDriveReturnsCorrectSections", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeSelfDrive)
		if modeSection != flowSectionSelfDriveMode {
			t.Errorf("expected flowSectionSelfDriveMode, got different section")
		}
		if coordPlan != flowSectionSelfDriveModeCoordinator {
			t.Errorf("expected flowSectionSelfDriveModeCoordinator, got different plan")
		}
	})

	t.Run("buildingReturnsCorrectSections", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeBuilding)
		if modeSection != flowSectionBuildingMode {
			t.Errorf("expected flowSectionBuildingMode, got different section")
		}
		if coordPlan != flowSectionBuildingModeCoordinator {
			t.Errorf("expected flowSectionBuildingModeCoordinator, got different plan")
		}
	})

	t.Run("continuousReturnsCorrectSections", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeContinuous)
		if modeSection != flowSectionContinuousMode {
			t.Errorf("expected flowSectionContinuousMode, got different section")
		}
		if coordPlan != flowSectionContinuousModeCoordinator {
			t.Errorf("expected flowSectionContinuousModeCoordinator, got different plan")
		}
	})

	t.Run("brainstormingReturnsDefaultSection", func(t *testing.T) {
		modeSection, coordPlan := modeSectionForMode(state.ModeBrainstorming)
		if modeSection != flowSectionBrainstormingMode {
			t.Errorf("expected flowSectionBrainstormingMode, got different section")
		}
		if coordPlan != "" {
			t.Errorf("expected empty coordinator plan for brainstorming, got %q", coordPlan)
		}
	})
}
