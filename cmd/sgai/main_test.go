package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
			modelSpec:   "anthropic/claude-opus-4-6",
			wantModel:   "anthropic/claude-opus-4-6",
			wantVariant: "",
		},
		{
			name:        "withSpaceBeforeParenthesis",
			modelSpec:   "anthropic/claude-opus-4-6 (high)",
			wantModel:   "anthropic/claude-opus-4-6",
			wantVariant: "high",
		},
		{
			name:        "noSpaceBeforeParenthesis",
			modelSpec:   "anthropic/claude-opus-4-6(banana)",
			wantModel:   "anthropic/claude-opus-4-6",
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
			modelSpec:   "anthropic/claude-opus-4-6  (high)",
			wantModel:   "anthropic/claude-opus-4-6",
			wantVariant: "high",
		},
		{
			name:        "variantWithSpaces",
			modelSpec:   "anthropic/claude-opus-4-6 (high quality)",
			wantModel:   "anthropic/claude-opus-4-6",
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
models:
  coordinator: anthropic/claude-opus-4-6
completionGateScript: make test
---
# New Goal
`
		if err := os.WriteFile(goalPath, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		original := GoalMetadata{
			Models:               map[string]any{"coordinator": "old-model"},
			CompletionGateScript: "old-command",
		}

		got, errReloadGoalMetadata := tryReloadGoalMetadata(goalPath, original)
		if errReloadGoalMetadata != nil {
			t.Fatalf("tryReloadGoalMetadata() unexpected error: %v", errReloadGoalMetadata)
		}

		models := getModelsForAgent(got.Models, "coordinator")
		if len(models) != 1 || models[0] != "anthropic/claude-opus-4-6" {
			t.Errorf("tryReloadGoalMetadata() Models[coordinator] = %v; want [anthropic/claude-opus-4-6]", models)
		}
		if got.CompletionGateScript != "make test" {
			t.Errorf("tryReloadGoalMetadata() CompletionGateScript = %q; want %q", got.CompletionGateScript, "make test")
		}
	})

	t.Run("missingFilePreservesOriginalMetadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalPath := filepath.Join(tmpDir, "nonexistent.md")

		original := GoalMetadata{
			Models:               map[string]any{"coordinator": "original-model"},
			CompletionGateScript: "make test",
		}

		got, errReloadGoalMetadata := tryReloadGoalMetadata(goalPath, original)
		if errReloadGoalMetadata != nil {
			t.Fatalf("tryReloadGoalMetadata() unexpected error: %v", errReloadGoalMetadata)
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

	t.Run("parseErrorReturnsError", func(t *testing.T) {
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
			Models: map[string]any{"coordinator": "preserve-me"},
			Flow:   "preserved-flow",
		}

		_, errReloadGoalMetadata := tryReloadGoalMetadata(goalPath, original)
		if errReloadGoalMetadata == nil {
			t.Fatal("tryReloadGoalMetadata() expected error for invalid frontmatter")
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
			Models: map[string]any{"agent": "old-model"},
		}

		got, errReloadGoalMetadata := tryReloadGoalMetadata(goalPath, original)
		if errReloadGoalMetadata != nil {
			t.Fatalf("tryReloadGoalMetadata() unexpected error: %v", errReloadGoalMetadata)
		}

		if len(got.Models) != 0 {
			t.Errorf("tryReloadGoalMetadata() Models should be empty, got %v", got.Models)
		}
	})
}

func TestWorkGateTransitionsToBuildingMode(t *testing.T) {
	t.Run("brainstormingTransitionsToBuilding", func(t *testing.T) {
		wf := state.Workflow{InteractionMode: state.ModeBrainstorming}

		if wf.InteractionMode == state.ModeBrainstorming {
			wf.InteractionMode = state.ModeBuilding
		}

		if wf.InteractionMode != state.ModeBuilding {
			t.Fatalf("InteractionMode should be %q, got %q", state.ModeBuilding, wf.InteractionMode)
		}
		if wf.InteractionMode == state.ModeSelfDrive {
			t.Fatal("building mode should not be self-drive")
		}
		if wf.ToolsAllowed() {
			t.Fatal("building mode should not allow tools")
		}
	})

	t.Run("selfDriveDoesNotTransition", func(t *testing.T) {
		wf := state.Workflow{InteractionMode: state.ModeSelfDrive}

		if wf.InteractionMode == state.ModeBrainstorming {
			wf.InteractionMode = state.ModeBuilding
		}

		if wf.InteractionMode != state.ModeSelfDrive {
			t.Fatalf("InteractionMode should remain %q, got %q", state.ModeSelfDrive, wf.InteractionMode)
		}
	})
}

func TestFindFirstPendingMessageAgent(t *testing.T) {
	t.Run("noMessages", func(t *testing.T) {
		wf := state.Workflow{}
		got := findFirstPendingMessageAgent(wf)
		if got != "" {
			t.Errorf("findFirstPendingMessageAgent() = %q; want empty string", got)
		}
	})

	t.Run("allRead", func(t *testing.T) {
		wf := state.Workflow{
			Messages: []state.Message{
				{ToAgent: "agent-a", Read: true},
				{ToAgent: "agent-b", Read: true},
			},
		}
		got := findFirstPendingMessageAgent(wf)
		if got != "" {
			t.Errorf("findFirstPendingMessageAgent() = %q; want empty string", got)
		}
	})

	t.Run("plainAgentName", func(t *testing.T) {
		wf := state.Workflow{
			Messages: []state.Message{
				{ToAgent: "agent-a", Read: true},
				{ToAgent: "agent-b", Read: false},
			},
		}
		got := findFirstPendingMessageAgent(wf)
		if got != "agent-b" {
			t.Errorf("findFirstPendingMessageAgent() = %q; want %q", got, "agent-b")
		}
	})

	t.Run("modelQualifiedID", func(t *testing.T) {
		wf := state.Workflow{
			Messages: []state.Message{
				{ToAgent: "project-critic-council:openai/gpt-5.2", Read: false},
			},
		}
		got := findFirstPendingMessageAgent(wf)
		if got != "project-critic-council" {
			t.Errorf("findFirstPendingMessageAgent() = %q; want %q", got, "project-critic-council")
		}
	})

	t.Run("firstUnreadWins", func(t *testing.T) {
		wf := state.Workflow{
			Messages: []state.Message{
				{ToAgent: "agent-a", Read: true},
				{ToAgent: "agent-b:openai/gpt-5.2", Read: false},
				{ToAgent: "agent-c", Read: false},
			},
		}
		got := findFirstPendingMessageAgent(wf)
		if got != "agent-b" {
			t.Errorf("findFirstPendingMessageAgent() = %q; want %q", got, "agent-b")
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

func TestAgentFilesHaveNoModelVariants(t *testing.T) {
	agentsFS, err := fs.Sub(skelFS, "skel/.sgai/agent")
	if err != nil {
		t.Fatal("failed to access skeleton agents FS:", err)
	}

	err = fs.WalkDir(agentsFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		content, errRead := fs.ReadFile(agentsFS, path)
		if errRead != nil {
			t.Errorf("failed to read %s: %v", path, errRead)
			return nil
		}
		fm := parseFrontmatterMap(content)
		modelVal := strings.Trim(fm["model"], "\"")
		if modelVal == "" {
			return nil
		}
		_, variant := parseModelAndVariant(modelVal)
		if variant != "" {
			t.Errorf("agent file %s has model with variant %q: %q (variants must be specified in GOAL.md models section, not agent files)",
				path, variant, modelVal)
		}
		return nil
	})
	if err != nil {
		t.Fatal("failed to walk agent files:", err)
	}
}

func TestIsTruish(t *testing.T) {
	truthy := []string{"yes", "YES", "Yes", "true", "TRUE", "True", "1", "on", "ON", " yes ", " true "}
	for _, v := range truthy {
		if !isTruish(v) {
			t.Errorf("isTruish(%q) = false; want true", v)
		}
	}

	falsy := []string{"no", "false", "0", "off", "", "maybe", "random"}
	for _, v := range falsy {
		if isTruish(v) {
			t.Errorf("isTruish(%q) = true; want false", v)
		}
	}
}

func TestIsFalsish(t *testing.T) {
	falsy := []string{"no", "NO", "No", "false", "FALSE", "False", "0", "off", "OFF", " no ", " false "}
	for _, v := range falsy {
		if !isFalsish(v) {
			t.Errorf("isFalsish(%q) = false; want true", v)
		}
	}

	notFalsy := []string{"yes", "true", "1", "on", "", "maybe"}
	for _, v := range notFalsy {
		if isFalsish(v) {
			t.Errorf("isFalsish(%q) = true; want false", v)
		}
	}
}

func TestRetrospectiveEnabled(t *testing.T) {
	t.Run("defaultEnabled", func(t *testing.T) {
		metadata := GoalMetadata{}
		if !retrospectiveEnabled(metadata) {
			t.Error("retrospectiveEnabled should be true when Retrospective is empty (default)")
		}
	})

	t.Run("explicitlyEnabled", func(t *testing.T) {
		metadata := GoalMetadata{Retrospective: "yes"}
		if !retrospectiveEnabled(metadata) {
			t.Error("retrospectiveEnabled should be true when Retrospective is 'yes'")
		}
	})

	t.Run("explicitlyDisabled", func(t *testing.T) {
		metadata := GoalMetadata{Retrospective: "no"}
		if retrospectiveEnabled(metadata) {
			t.Error("retrospectiveEnabled should be false when Retrospective is 'no'")
		}
	})

	t.Run("disabledWithFalse", func(t *testing.T) {
		metadata := GoalMetadata{Retrospective: "false"}
		if retrospectiveEnabled(metadata) {
			t.Error("retrospectiveEnabled should be false when Retrospective is 'false'")
		}
	})

	t.Run("disabledWithOff", func(t *testing.T) {
		metadata := GoalMetadata{Retrospective: "off"}
		if retrospectiveEnabled(metadata) {
			t.Error("retrospectiveEnabled should be false when Retrospective is 'off'")
		}
	})

	t.Run("unknownValueTreatedAsEnabled", func(t *testing.T) {
		metadata := GoalMetadata{Retrospective: "maybe"}
		if !retrospectiveEnabled(metadata) {
			t.Error("retrospectiveEnabled should be true for unknown values")
		}
	})
}

func TestGoalMetadataRetrospectiveParsing(t *testing.T) {
	t.Run("parsesRetrospectiveYes", func(t *testing.T) {
		content := []byte("---\nretrospective: yes\nflow: |\n  a -> b\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !retrospectiveEnabled(metadata) {
			t.Errorf("retrospectiveEnabled should be true for frontmatter 'yes', got Retrospective=%q", metadata.Retrospective)
		}
	})

	t.Run("parsesRetrospectiveNo", func(t *testing.T) {
		content := []byte("---\nretrospective: no\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if retrospectiveEnabled(metadata) {
			t.Errorf("retrospectiveEnabled should be false for frontmatter 'no', got Retrospective=%q", metadata.Retrospective)
		}
	})

	t.Run("emptyWhenAbsent", func(t *testing.T) {
		content := []byte("---\nflow: |\n  a -> b\n---\n\nGoal.\n")
		metadata, err := parseYAMLFrontmatter(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metadata.Retrospective != "" {
			t.Errorf("Retrospective = %q; want empty", metadata.Retrospective)
		}
	})
}

func TestEnsureImplicitRetrospectiveModel(t *testing.T) {
	t.Run("implicitGetsCoordinatorModel", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":   {Name: "coordinator"},
				"retrospective": {Name: "retrospective"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator": "anthropic/claude-opus-4-6 (max)",
			},
		}

		ensureImplicitRetrospectiveModel(flowDag, &metadata)

		got, exists := metadata.Models["retrospective"]
		if !exists {
			t.Fatal("expected retrospective to exist in Models")
		}
		want := "anthropic/claude-opus-4-6 (max)"
		if got != want {
			t.Errorf("retrospective model = %v; want %v", got, want)
		}
	})

	t.Run("explicitModelNotOverridden", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":   {Name: "coordinator"},
				"retrospective": {Name: "retrospective"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator":   "anthropic/claude-opus-4-6 (max)",
				"retrospective": "anthropic/claude-sonnet-4-5",
			},
		}

		ensureImplicitRetrospectiveModel(flowDag, &metadata)

		got := metadata.Models["retrospective"]
		if got != "anthropic/claude-sonnet-4-5" {
			t.Errorf("retrospective model = %v; want anthropic/claude-sonnet-4-5", got)
		}
	})

	t.Run("coordinatorHasNoModel", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":   {Name: "coordinator"},
				"retrospective": {Name: "retrospective"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{},
		}

		ensureImplicitRetrospectiveModel(flowDag, &metadata)

		if _, exists := metadata.Models["retrospective"]; exists {
			t.Error("expected retrospective to NOT exist in Models when coordinator has no model")
		}
	})

	t.Run("nilModelsMap", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":   {Name: "coordinator"},
				"retrospective": {Name: "retrospective"},
			},
		}
		metadata := GoalMetadata{
			Models: nil,
		}

		ensureImplicitRetrospectiveModel(flowDag, &metadata)

		if metadata.Models == nil {
			t.Fatal("expected Models to be initialized")
		}
	})

	t.Run("retrospectiveNotInDag", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator": {Name: "coordinator"},
				"planner":     {Name: "planner"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator": "anthropic/claude-opus-4-6 (max)",
			},
		}

		ensureImplicitRetrospectiveModel(flowDag, &metadata)

		if _, exists := metadata.Models["retrospective"]; exists {
			t.Error("expected retrospective to NOT exist in Models when not in DAG")
		}
	})
}

func TestEnsureImplicitProjectCriticCouncilModel(t *testing.T) {
	t.Run("implicitGetsCoordinatorModelWithVariant", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":            {Name: "coordinator"},
				"project-critic-council": {Name: "project-critic-council"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator": "anthropic/claude-opus-4-6 (max)",
			},
		}

		ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

		got, exists := metadata.Models["project-critic-council"]
		if !exists {
			t.Fatal("expected project-critic-council to exist in Models")
		}
		want := "anthropic/claude-opus-4-6 (max)"
		if got != want {
			t.Errorf("project-critic-council model = %v; want %v", got, want)
		}
	})

	t.Run("explicitModelNotOverridden", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":            {Name: "coordinator"},
				"project-critic-council": {Name: "project-critic-council"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator":            "anthropic/claude-opus-4-6 (max)",
				"project-critic-council": []any{"anthropic/claude-opus-4-6", "openai/gpt-5.2"},
			},
		}

		ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

		got := metadata.Models["project-critic-council"]
		gotSlice, ok := got.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", got)
		}
		if len(gotSlice) != 2 {
			t.Fatalf("expected 2 models, got %d", len(gotSlice))
		}
		if gotSlice[0] != "anthropic/claude-opus-4-6" || gotSlice[1] != "openai/gpt-5.2" {
			t.Errorf("project-critic-council models = %v; want [anthropic/claude-opus-4-6 openai/gpt-5.2]", gotSlice)
		}
	})

	t.Run("coordinatorHasNoModel", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":            {Name: "coordinator"},
				"project-critic-council": {Name: "project-critic-council"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{},
		}

		ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

		if _, exists := metadata.Models["project-critic-council"]; exists {
			t.Error("expected project-critic-council to NOT exist in Models when coordinator has no model")
		}
	})

	t.Run("nilModelsMap", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":            {Name: "coordinator"},
				"project-critic-council": {Name: "project-critic-council"},
			},
		}
		metadata := GoalMetadata{
			Models: nil,
		}

		ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

		if metadata.Models == nil {
			t.Fatal("expected Models to be initialized")
		}
		if _, exists := metadata.Models["project-critic-council"]; exists {
			t.Error("expected project-critic-council to NOT exist in Models when coordinator has no model")
		}
	})

	t.Run("projectCriticCouncilNotInDag", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator": {Name: "coordinator"},
				"planner":     {Name: "planner"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator": "anthropic/claude-opus-4-6 (max)",
			},
		}

		ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

		if _, exists := metadata.Models["project-critic-council"]; exists {
			t.Error("expected project-critic-council to NOT exist in Models when not in DAG")
		}
	})

	t.Run("coordinatorModelWithoutVariant", func(t *testing.T) {
		flowDag := &dag{
			Nodes: map[string]*dagNode{
				"coordinator":            {Name: "coordinator"},
				"project-critic-council": {Name: "project-critic-council"},
			},
		}
		metadata := GoalMetadata{
			Models: map[string]any{
				"coordinator": "anthropic/claude-opus-4-6",
			},
		}

		ensureImplicitProjectCriticCouncilModel(flowDag, &metadata)

		got, exists := metadata.Models["project-critic-council"]
		if !exists {
			t.Fatal("expected project-critic-council to exist in Models")
		}
		want := "anthropic/claude-opus-4-6"
		if got != want {
			t.Errorf("project-critic-council model = %v; want %v", got, want)
		}
	})
}

func TestInteractionModePreservedOnInit(t *testing.T) {
	t.Run("preservedWhenNotResuming", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		original := state.Workflow{
			InteractionMode: state.ModeBuilding,
		}
		coord, errCoordOrig := state.NewCoordinatorWith(stPath, original)
		if errCoordOrig != nil {
			t.Fatal(errCoordOrig)
		}

		loaded := coord.State()
		preservedMode := loaded.InteractionMode

		newState := state.Workflow{
			Status:          state.StatusWorking,
			InteractionMode: preservedMode,
		}

		if newState.InteractionMode != state.ModeBuilding {
			t.Errorf("InteractionMode should be preserved during reinitialization, got %q", newState.InteractionMode)
		}
	})
}

func TestUnlockInteractiveForRetrospective(t *testing.T) {
	t.Run("buildingTransitionsToRetrospective", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			InteractionMode: state.ModeBuilding,
			Status:          state.StatusWorking,
		}
		coord := state.NewCoordinatorEmpty(stPath)
		if errUpdate := coord.UpdateState(func(s *state.Workflow) { *s = wf }); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		unlockInteractiveForRetrospective(&wf, "retrospective", coord, "test")

		if wf.InteractionMode != state.ModeRetrospective {
			t.Errorf("InteractionMode should be %q after retrospective unlock, got %q", state.ModeRetrospective, wf.InteractionMode)
		}

		loaded := coord.State()
		if loaded.InteractionMode != state.ModeRetrospective {
			t.Errorf("InteractionMode should be %q on disk after retrospective unlock, got %q", state.ModeRetrospective, loaded.InteractionMode)
		}
	})

	t.Run("selfDriveStaysSelfDrive", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			InteractionMode: state.ModeSelfDrive,
			Status:          state.StatusWorking,
		}
		coord := state.NewCoordinatorEmpty(stPath)
		if errUpdate := coord.UpdateState(func(s *state.Workflow) { *s = wf }); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		unlockInteractiveForRetrospective(&wf, "retrospective", coord, "test")

		if wf.InteractionMode != state.ModeSelfDrive {
			t.Errorf("InteractionMode should stay %q in self-drive mode, got %q", state.ModeSelfDrive, wf.InteractionMode)
		}
	})

	t.Run("nonRetrospectiveAgentNoTransition", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			InteractionMode: state.ModeBuilding,
			Status:          state.StatusWorking,
		}
		coord := state.NewCoordinatorEmpty(stPath)
		if errUpdate := coord.UpdateState(func(s *state.Workflow) { *s = wf }); errUpdate != nil {
			t.Fatal(errUpdate)
		}

		unlockInteractiveForRetrospective(&wf, "coordinator", coord, "test")

		if wf.InteractionMode != state.ModeBuilding {
			t.Errorf("InteractionMode should stay %q for non-retrospective agents, got %q", state.ModeBuilding, wf.InteractionMode)
		}
	})
}

func TestInteractionModeRoundTrip(t *testing.T) {
	t.Run("savedAndLoadedCorrectly", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			InteractionMode: state.ModeBuilding,
			Status:          state.StatusWorking,
		}
		savedCoord, errSave := state.NewCoordinatorWith(stPath, wf)
		if errSave != nil {
			t.Fatal(errSave)
		}

		loaded := savedCoord.State()
		if loaded.InteractionMode != state.ModeBuilding {
			t.Errorf("InteractionMode should be %q after round-trip, got %q", state.ModeBuilding, loaded.InteractionMode)
		}
	})

	t.Run("emptyOmittedFromJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			InteractionMode: "",
			Status:          state.StatusWorking,
		}
		if _, errSave := state.NewCoordinatorWith(stPath, wf); errSave != nil {
			t.Fatal(errSave)
		}

		data, errRead := os.ReadFile(stPath)
		if errRead != nil {
			t.Fatal(errRead)
		}
		if strings.Contains(string(data), "interactionMode") {
			t.Error("interactionMode should be omitted when empty (omitempty)")
		}
	})
}

func TestBuildAgentStdoutWriter(t *testing.T) {
	t.Run("noLogWriterNoStdoutLog", func(t *testing.T) {
		w := buildAgentStdoutWriter(nil, nil)
		if w != os.Stdout {
			t.Error("expected os.Stdout when no writers provided")
		}
	})

	t.Run("withLogWriterOnly", func(t *testing.T) {
		var buf bytes.Buffer
		w := buildAgentStdoutWriter(&buf, nil)
		if _, errWrite := io.WriteString(w, "test"); errWrite != nil {
			t.Fatal(errWrite)
		}
		if buf.String() != "test" {
			t.Errorf("expected logWriter to receive output, got %q", buf.String())
		}
	})

	t.Run("withStdoutLogOnly", func(t *testing.T) {
		var buf bytes.Buffer
		w := buildAgentStdoutWriter(nil, &buf)
		if _, errWrite := io.WriteString(w, "hello"); errWrite != nil {
			t.Fatal(errWrite)
		}
		if buf.String() != "hello" {
			t.Errorf("expected stdoutLog to receive output, got %q", buf.String())
		}
	})

	t.Run("withBothWriters", func(t *testing.T) {
		var logBuf, stdoutBuf bytes.Buffer
		w := buildAgentStdoutWriter(&logBuf, &stdoutBuf)
		if _, errWrite := io.WriteString(w, "data"); errWrite != nil {
			t.Fatal(errWrite)
		}
		if logBuf.String() != "data" {
			t.Errorf("expected logWriter to receive output, got %q", logBuf.String())
		}
		if stdoutBuf.String() != "data" {
			t.Errorf("expected stdoutLog to receive output, got %q", stdoutBuf.String())
		}
	})
}

func TestBuildAgentStderrWriter(t *testing.T) {
	t.Run("noLogWriterNoStderrLog", func(t *testing.T) {
		w := buildAgentStderrWriter(nil, nil)
		if w != os.Stderr {
			t.Error("expected os.Stderr when no writers provided")
		}
	})

	t.Run("withBothWriters", func(t *testing.T) {
		var logBuf, stderrBuf bytes.Buffer
		w := buildAgentStderrWriter(&logBuf, &stderrBuf)
		if _, errWrite := io.WriteString(w, "errdata"); errWrite != nil {
			t.Fatal(errWrite)
		}
		if logBuf.String() != "errdata" {
			t.Errorf("expected logWriter to receive output, got %q", logBuf.String())
		}
		if stderrBuf.String() != "errdata" {
			t.Errorf("expected stderrLog to receive output, got %q", stderrBuf.String())
		}
	})
}

func TestNonCoordinatorCannotCompleteWorkflow(t *testing.T) {
	t.Run("retrospectiveAgentCannotComplete", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			Status:          state.StatusComplete,
			InteractionMode: state.ModeRetrospective,
		}
		savedCoord, errSave := state.NewCoordinatorWith(stPath, wf)
		if errSave != nil {
			t.Fatal(errSave)
		}

		loaded := savedCoord.State()

		if loaded.Status != state.StatusComplete {
			t.Fatalf("expected StatusComplete in state file, got %q", loaded.Status)
		}

		if loaded.Status == state.StatusComplete && "retrospective" != "coordinator" {
			loaded.Status = state.StatusAgentDone
		}

		if loaded.Status != state.StatusAgentDone {
			t.Errorf("non-coordinator agent status=complete should be treated as agent-done, got %q", loaded.Status)
		}
	})
}

func TestContinuousModePromptObservabilityAgentName(t *testing.T) {
	t.Run("updateContinuousModeStateUsesCorrectAgent", func(t *testing.T) {
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		if err := os.MkdirAll(sgaiDir, 0755); err != nil {
			t.Fatal(err)
		}

		stateJSONPath := filepath.Join(sgaiDir, "state.json")
		wfState := state.Workflow{
			Status:   state.StatusWorking,
			Messages: []state.Message{},
			Progress: []state.ProgressEntry{},
		}
		coord, errCoord := state.NewCoordinatorWith(stateJSONPath, wfState)
		if errCoord != nil {
			t.Fatal(errCoord)
		}

		const continuousModeAgentName = "continuous-mode"
		updateContinuousModeState(coord, "Running continuous mode prompt...", continuousModeAgentName, "started")

		loaded := coord.State()

		if loaded.CurrentAgent != continuousModeAgentName {
			t.Errorf("expected CurrentAgent %q, got %q", continuousModeAgentName, loaded.CurrentAgent)
		}
		if len(loaded.Progress) == 0 {
			t.Fatal("expected at least one progress entry")
		}
		if loaded.Progress[len(loaded.Progress)-1].Agent != continuousModeAgentName {
			t.Errorf("expected progress Agent %q, got %q", continuousModeAgentName, loaded.Progress[len(loaded.Progress)-1].Agent)
		}
	})
}
