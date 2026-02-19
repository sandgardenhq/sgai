package main

import (
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

func TestApplyWorkGateApproval(t *testing.T) {
	t.Run("locksAutoModeAfterApproval", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "state.json")
		wf := state.Workflow{WorkGateApproved: true}

		if shouldContinue := applyWorkGateApproval(&wf, statePath, "sgai"); shouldContinue {
			t.Fatal("applyWorkGateApproval should not request loop continue on successful save")
		}
		if !wf.InteractiveAutoLock {
			t.Fatal("interactive auto lock should be enabled")
		}
		if wf.WorkGateApproved {
			t.Fatal("work gate approved flag should be cleared")
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

func TestStartedInteractivePreservedOnInit(t *testing.T) {
	t.Run("preservedWhenNotResuming", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		original := state.Workflow{
			StartedInteractive:  true,
			InteractiveAutoLock: true,
		}
		if errSave := state.Save(stPath, original); errSave != nil {
			t.Fatal(errSave)
		}

		loaded, errLoad := state.Load(stPath)
		if errLoad != nil {
			t.Fatal(errLoad)
		}

		preservedAutoLock := loaded.InteractiveAutoLock
		preservedStartedInteractive := loaded.StartedInteractive

		newState := state.Workflow{
			Status:              state.StatusWorking,
			InteractiveAutoLock: preservedAutoLock,
			StartedInteractive:  preservedStartedInteractive,
		}

		if !newState.StartedInteractive {
			t.Error("StartedInteractive should be preserved during reinitialization")
		}
		if !newState.InteractiveAutoLock {
			t.Error("InteractiveAutoLock should be preserved during reinitialization")
		}
	})
}

func TestUnlockInteractiveForRetrospective(t *testing.T) {
	t.Run("interactiveModeUnlocksForRetrospective", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			StartedInteractive:  true,
			InteractiveAutoLock: true,
			Status:              state.StatusWorking,
		}
		if errSave := state.Save(stPath, wf); errSave != nil {
			t.Fatal(errSave)
		}

		unlockInteractiveForRetrospective(&wf, "retrospective", stPath, "test")

		if wf.InteractiveAutoLock {
			t.Error("InteractiveAutoLock should be false after retrospective unlock")
		}

		loaded, errLoad := state.Load(stPath)
		if errLoad != nil {
			t.Fatal(errLoad)
		}
		if loaded.InteractiveAutoLock {
			t.Error("InteractiveAutoLock should be false on disk after retrospective unlock")
		}
	})

	t.Run("selfDriveModeStaysLocked", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			StartedInteractive:  false,
			InteractiveAutoLock: true,
			Status:              state.StatusWorking,
		}
		if errSave := state.Save(stPath, wf); errSave != nil {
			t.Fatal(errSave)
		}

		unlockInteractiveForRetrospective(&wf, "retrospective", stPath, "test")

		if !wf.InteractiveAutoLock {
			t.Error("InteractiveAutoLock should stay true in self-drive mode")
		}
	})

	t.Run("nonRetrospectiveAgentStaysLocked", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			StartedInteractive:  true,
			InteractiveAutoLock: true,
			Status:              state.StatusWorking,
		}
		if errSave := state.Save(stPath, wf); errSave != nil {
			t.Fatal(errSave)
		}

		unlockInteractiveForRetrospective(&wf, "coordinator", stPath, "test")

		if !wf.InteractiveAutoLock {
			t.Error("InteractiveAutoLock should stay true for non-retrospective agents")
		}
	})
}

func TestStartedInteractiveRoundTrip(t *testing.T) {
	t.Run("savedAndLoadedCorrectly", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			StartedInteractive:  true,
			InteractiveAutoLock: true,
			Status:              state.StatusWorking,
		}
		if errSave := state.Save(stPath, wf); errSave != nil {
			t.Fatal(errSave)
		}

		loaded, errLoad := state.Load(stPath)
		if errLoad != nil {
			t.Fatal(errLoad)
		}
		if !loaded.StartedInteractive {
			t.Error("StartedInteractive should be true after round-trip")
		}
		if !loaded.InteractiveAutoLock {
			t.Error("InteractiveAutoLock should be true after round-trip")
		}
	})

	t.Run("falseOmittedFromJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "state.json")

		wf := state.Workflow{
			StartedInteractive: false,
			Status:             state.StatusWorking,
		}
		if errSave := state.Save(stPath, wf); errSave != nil {
			t.Fatal(errSave)
		}

		data, errRead := os.ReadFile(stPath)
		if errRead != nil {
			t.Fatal(errRead)
		}
		if strings.Contains(string(data), "startedInteractive") {
			t.Error("startedInteractive should be omitted when false (omitempty)")
		}
	})
}
