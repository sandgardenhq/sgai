package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestParseModelAndVariant(t *testing.T) {
	cases := []struct {
		name        string
		modelSpec   string
		wantModel   string
		wantVariant string
	}{
		{
			name:        "noVariant",
			modelSpec:   "anthropic/claude-opus-4-5",
			wantModel:   "anthropic/claude-opus-4-5",
			wantVariant: "",
		},
		{
			name:        "withSpaceBeforeParenthesis",
			modelSpec:   "anthropic/claude-opus-4-5 (high)",
			wantModel:   "anthropic/claude-opus-4-5",
			wantVariant: "high",
		},
		{
			name:        "noSpaceBeforeParenthesis",
			modelSpec:   "anthropic/claude-opus-4-5(banana)",
			wantModel:   "anthropic/claude-opus-4-5",
			wantVariant: "banana",
		},
		{
			name:        "differentProvider",
			modelSpec:   "openai/gpt-4o (creative)",
			wantModel:   "openai/gpt-4o",
			wantVariant: "creative",
		},
		{
			name:        "emptySpec",
			modelSpec:   "",
			wantModel:   "",
			wantVariant: "",
		},
		{
			name:        "multipleSpacesBeforeParenthesis",
			modelSpec:   "anthropic/claude-opus-4-5  (high)",
			wantModel:   "anthropic/claude-opus-4-5",
			wantVariant: "high",
		},
		{
			name:        "variantWithSpaces",
			modelSpec:   "anthropic/claude-opus-4-5 (high quality)",
			wantModel:   "anthropic/claude-opus-4-5",
			wantVariant: "high quality",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotModel, gotVariant := parseModelAndVariant(tc.modelSpec)
			if gotModel != tc.wantModel {
				t.Errorf("parseModelAndVariant(%q) model = %q; want %q", tc.modelSpec, gotModel, tc.wantModel)
			}
			if gotVariant != tc.wantVariant {
				t.Errorf("parseModelAndVariant(%q) variant = %q; want %q", tc.modelSpec, gotVariant, tc.wantVariant)
			}
		})
	}
}

func TestTryReloadGoalMetadata(t *testing.T) {
	t.Run("successfulReloadReturnsNewMetadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalPath := filepath.Join(tmpDir, "GOAL.md")

		newContent := `---
interactive: "auto"
models:
  coordinator: anthropic/claude-opus-4-5
completionGateScript: make test
---
# New Goal
`
		if err := os.WriteFile(goalPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		original := GoalMetadata{
			Interactive:          "no",
			Models:               map[string]any{"coordinator": "old-model"},
			CompletionGateScript: "old-command",
		}

		got := tryReloadGoalMetadata(goalPath, original)

		if got.Interactive != "auto" {
			t.Errorf("tryReloadGoalMetadata() Interactive = %q; want %q", got.Interactive, "auto")
		}
		models := getModelsForAgent(got.Models, "coordinator")
		if len(models) != 1 || models[0] != "anthropic/claude-opus-4-5" {
			t.Errorf("tryReloadGoalMetadata() Models[coordinator] = %v; want [anthropic/claude-opus-4-5]", models)
		}
		if got.CompletionGateScript != "make test" {
			t.Errorf("tryReloadGoalMetadata() CompletionGateScript = %q; want %q", got.CompletionGateScript, "make test")
		}
	})

	t.Run("missingFilePreservesOriginalMetadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalPath := filepath.Join(tmpDir, "nonexistent.md")

		original := GoalMetadata{
			Interactive:          "auto",
			Models:               map[string]any{"coordinator": "original-model"},
			CompletionGateScript: "make test",
		}

		got := tryReloadGoalMetadata(goalPath, original)

		if got.Interactive != original.Interactive {
			t.Errorf("tryReloadGoalMetadata() Interactive = %q; want %q", got.Interactive, original.Interactive)
		}
		gotModels := getModelsForAgent(got.Models, "coordinator")
		origModels := getModelsForAgent(original.Models, "coordinator")
		if len(gotModels) != len(origModels) || (len(gotModels) > 0 && gotModels[0] != origModels[0]) {
			t.Errorf("tryReloadGoalMetadata() Models[coordinator] = %v; want %v", gotModels, origModels)
		}
		if got.CompletionGateScript != original.CompletionGateScript {
			t.Errorf("tryReloadGoalMetadata() CompletionGateScript = %q; want %q", got.CompletionGateScript, original.CompletionGateScript)
		}
	})

	t.Run("parseErrorPreservesOriginalMetadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalPath := filepath.Join(tmpDir, "GOAL.md")

		invalidContent := `---
interactive: [invalid yaml structure
models:
  - broken: yes
---
# Invalid GOAL
`
		if err := os.WriteFile(goalPath, []byte(invalidContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		original := GoalMetadata{
			Interactive: "no",
			Models:      map[string]any{"coordinator": "preserve-me"},
			Flow:        "preserved-flow",
		}

		got := tryReloadGoalMetadata(goalPath, original)

		if got.Interactive != original.Interactive {
			t.Errorf("tryReloadGoalMetadata() Interactive = %q; want %q", got.Interactive, original.Interactive)
		}
		gotModels := getModelsForAgent(got.Models, "coordinator")
		origModels := getModelsForAgent(original.Models, "coordinator")
		if len(gotModels) != len(origModels) || (len(gotModels) > 0 && gotModels[0] != origModels[0]) {
			t.Errorf("tryReloadGoalMetadata() Models[coordinator] = %v; want %v", gotModels, origModels)
		}
		if got.Flow != original.Flow {
			t.Errorf("tryReloadGoalMetadata() Flow = %q; want %q", got.Flow, original.Flow)
		}
	})

	t.Run("noFrontmatterReturnsDefaultMetadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalPath := filepath.Join(tmpDir, "GOAL.md")

		noFrontmatterContent := `# Plain Goal
No frontmatter here.
`
		if err := os.WriteFile(goalPath, []byte(noFrontmatterContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		original := GoalMetadata{
			Interactive: "yes",
			Models:      map[string]any{"agent": "old-model"},
		}

		got := tryReloadGoalMetadata(goalPath, original)

		if got.Interactive != "" {
			t.Errorf("tryReloadGoalMetadata() Interactive = %q; want empty string", got.Interactive)
		}
		if len(got.Models) != 0 {
			t.Errorf("tryReloadGoalMetadata() Models should be empty, got %v", got.Models)
		}
	})
}

func TestCanResumeWorkflow(t *testing.T) {
	checksum := "abc123"

	t.Run("workingWithMatchingChecksum", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusWorking, GoalChecksum: checksum}
		if !canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = true for working status with matching checksum")
		}
	})

	t.Run("agentDoneWithMatchingChecksum", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusAgentDone, GoalChecksum: checksum}
		if !canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = true for agent-done status")
		}
	})

	t.Run("waitingForHumanWithMatchingChecksum", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusWaitingForHuman, GoalChecksum: checksum}
		if !canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = true for waiting-for-human status")
		}
	})

	t.Run("completeWithMatchingChecksum", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusComplete, GoalChecksum: checksum}
		if canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = false for complete status")
		}
	})

	t.Run("emptyStatusWithMatchingChecksum", func(t *testing.T) {
		wf := state.Workflow{Status: "", GoalChecksum: checksum}
		if canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = false for empty status")
		}
	})

	t.Run("freshFlagAlwaysFalse", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusWorking, GoalChecksum: checksum}
		if canResumeWorkflow(wf, true, checksum) {
			t.Error("expected canResumeWorkflow = false when fresh=true")
		}
	})

	t.Run("mismatchedChecksumAlwaysFalse", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusWorking, GoalChecksum: "different"}
		if canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = false when checksums mismatch")
		}
	})

	t.Run("freshFlagWithWaitingForHuman", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusWaitingForHuman, GoalChecksum: checksum}
		if canResumeWorkflow(wf, true, checksum) {
			t.Error("expected canResumeWorkflow = false when fresh=true even for waiting-for-human")
		}
	})

	t.Run("emptyChecksumMatchesEmpty", func(t *testing.T) {
		wf := state.Workflow{Status: state.StatusWorking, GoalChecksum: ""}
		if !canResumeWorkflow(wf, false, "") {
			t.Error("expected canResumeWorkflow = true when both checksums are empty")
		}
	})

	t.Run("arbitraryStatusWithMatchingChecksum", func(t *testing.T) {
		wf := state.Workflow{Status: "some-unknown-status", GoalChecksum: checksum}
		if canResumeWorkflow(wf, false, checksum) {
			t.Error("expected canResumeWorkflow = false for unknown status")
		}
	})
}
