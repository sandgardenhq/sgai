package main

import (
	"os"
	"path/filepath"
	"testing"
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
