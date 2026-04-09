package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractModelFromArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"withModel", []string{"run", "--model", "claude-opus-4", "--agent", "build"}, "claude-opus-4"},
		{"noModel", []string{"run", "--agent", "build"}, ""},
		{"modelAtEnd", []string{"--model"}, ""},
		{"empty", []string{}, ""},
		{"modelFirst", []string{"--model", "gpt-4"}, "gpt-4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractModelFromArgs(tc.args)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEnsureImplicitAgentModel(t *testing.T) {
	t.Run("addsModelFromCoordinator", func(t *testing.T) {
		flowDag := &dag{Nodes: map[string]*dagNode{"retrospective": {}}}
		metadata := &GoalMetadata{Models: map[string]any{"coordinator": "claude-opus-4"}}

		ensureImplicitAgentModel(flowDag, metadata, "retrospective")

		assert.Equal(t, "claude-opus-4", metadata.Models["retrospective"])
	})

	t.Run("doesNotOverrideExisting", func(t *testing.T) {
		flowDag := &dag{Nodes: map[string]*dagNode{"retrospective": {}}}
		metadata := &GoalMetadata{Models: map[string]any{
			"coordinator":   "claude-opus-4",
			"retrospective": "gpt-4",
		}}

		ensureImplicitAgentModel(flowDag, metadata, "retrospective")

		assert.Equal(t, "gpt-4", metadata.Models["retrospective"])
	})

	t.Run("agentNotInDag", func(t *testing.T) {
		flowDag := &dag{Nodes: map[string]*dagNode{}}
		metadata := &GoalMetadata{Models: map[string]any{"coordinator": "claude-opus-4"}}

		ensureImplicitAgentModel(flowDag, metadata, "retrospective")

		_, exists := metadata.Models["retrospective"]
		assert.False(t, exists)
	})

	t.Run("noCoordinatorModel", func(t *testing.T) {
		flowDag := &dag{Nodes: map[string]*dagNode{"retrospective": {}}}
		metadata := &GoalMetadata{Models: map[string]any{}}

		ensureImplicitAgentModel(flowDag, metadata, "retrospective")

		_, exists := metadata.Models["retrospective"]
		assert.False(t, exists)
	})

	t.Run("nilModelsMap", func(t *testing.T) {
		flowDag := &dag{Nodes: map[string]*dagNode{"retrospective": {}}}
		metadata := &GoalMetadata{}

		ensureImplicitAgentModel(flowDag, metadata, "retrospective")

		assert.NotNil(t, metadata.Models)
	})
}

func TestAddRetrospectiveRedirectMessage(t *testing.T) {
	wfState := &state.Workflow{
		Messages: []state.Message{
			{ID: 1, FromAgent: "dev", ToAgent: "coordinator", Body: "done", Read: true},
		},
	}

	addRetrospectiveRedirectMessage(wfState, "coordinator")

	require.Len(t, wfState.Messages, 2)
	msg := wfState.Messages[1]
	assert.Equal(t, 2, msg.ID)
	assert.Equal(t, "coordinator", msg.FromAgent)
	assert.Equal(t, "retrospective", msg.ToAgent)
	assert.Contains(t, msg.Body, "retrospective analysis")
	assert.False(t, msg.Read)
}

func TestBlockCompletionOnPendingTodos(t *testing.T) {
	t.Run("noPendingTodos", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{coord: coord}
		newState := state.Workflow{Status: state.StatusComplete}
		wfState := state.Workflow{
			Todos: []state.TodoItem{
				{Content: "done", Status: "completed"},
				{Content: "cancelled", Status: "cancelled"},
			},
		}

		result := blockCompletionOnPendingTodos(cfg, newState, wfState)
		assert.Nil(t, result)
	})

	t.Run("hasPendingTodos", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{coord: coord, agent: "coordinator", paddedsgai: "sgai"}
		newState := state.Workflow{Status: state.StatusComplete}
		wfState := state.Workflow{
			Todos: []state.TodoItem{
				{Content: "done", Status: "completed"},
				{Content: "pending task", Status: "pending"},
			},
		}

		result := blockCompletionOnPendingTodos(cfg, newState, wfState)
		require.NotNil(t, result)
		assert.Equal(t, state.StatusWorking, result.Status)
	})
}

func TestCopyCompletionArtifactsToRetrospective(t *testing.T) {
	t.Run("noRetrospectiveDir", func(_ *testing.T) {
		cfg := multiModelConfig{retrospectiveDir: ""}
		copyCompletionArtifactsToRetrospective(cfg)
	})

	t.Run("withRetrospectiveDir", func(t *testing.T) {
		dir := t.TempDir()
		retrospectiveDir := filepath.Join(dir, "retrospective")
		require.NoError(t, os.MkdirAll(retrospectiveDir, 0755))

		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0644))

		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0755))
		pmPath := filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md")
		require.NoError(t, os.WriteFile(pmPath, []byte("# PM"), 0644))

		cfg := multiModelConfig{
			dir:              dir,
			goalPath:         goalPath,
			retrospectiveDir: retrospectiveDir,
		}

		copyCompletionArtifactsToRetrospective(cfg)

		goalCopy, errGoal := os.ReadFile(filepath.Join(retrospectiveDir, "GOAL.md"))
		require.NoError(t, errGoal)
		assert.Equal(t, "# Goal", string(goalCopy))

		pmCopy, errPM := os.ReadFile(filepath.Join(retrospectiveDir, "PROJECT_MANAGEMENT.md"))
		require.NoError(t, errPM)
		assert.Equal(t, "# PM", string(pmCopy))
	})
}

func TestTryReloadGoalMetadata(t *testing.T) {
	t.Run("fileNotExists", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		current := GoalMetadata{Flow: "coordinator -> dev"}

		result, err := tryReloadGoalMetadata(goalPath, current, &dag{Nodes: map[string]*dagNode{}})
		require.NoError(t, err)
		assert.Equal(t, "coordinator -> dev", result.Flow)
	})

	t.Run("validFrontmatter", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nflow: coordinator -> dev\nretrospective: \"true\"\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		result, err := tryReloadGoalMetadata(goalPath, GoalMetadata{}, &dag{Nodes: map[string]*dagNode{}})
		require.NoError(t, err)
		assert.Equal(t, "coordinator -> dev", result.Flow)
	})

	t.Run("invalidFrontmatter", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\n  bad yaml: [unclosed\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		_, err := tryReloadGoalMetadata(goalPath, GoalMetadata{}, &dag{Nodes: map[string]*dagNode{}})
		assert.Error(t, err)
	})
}

func TestHandleWaitingForHumanStatus(t *testing.T) {
	t.Run("withPendingQuestion", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{coord: coord, agent: "coordinator", paddedsgai: "sgai"}
		newState := state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "What do you want?",
		}

		result := handleWaitingForHumanStatus(cfg, newState)
		assert.Equal(t, state.StatusWorking, result.Status)
	})

	t.Run("withMultiChoiceQuestion", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{coord: coord, agent: "coordinator", paddedsgai: "sgai"}
		newState := state.Workflow{
			Status: state.StatusWaitingForHuman,
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "pick", Choices: []string{"A"}}},
			},
		}

		result := handleWaitingForHumanStatus(cfg, newState)
		assert.Equal(t, state.StatusWorking, result.Status)
	})

	t.Run("withoutPendingQuestion", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{coord: coord, agent: "coordinator", paddedsgai: "sgai"}
		newState := state.Workflow{
			Status: state.StatusWaitingForHuman,
		}

		result := handleWaitingForHumanStatus(cfg, newState)
		assert.Equal(t, state.StatusWorking, result.Status)
	})
}

func TestBlockCompletionOnRetrospective(t *testing.T) {
	t.Run("retrospectiveDisabled", func(t *testing.T) {
		cfg := multiModelConfig{flowDag: &dag{Nodes: map[string]*dagNode{"retrospective": {}}}}
		metadata := GoalMetadata{Retrospective: "false"}
		result := blockCompletionOnRetrospective(cfg, state.Workflow{}, metadata)
		assert.Nil(t, result)
	})

	t.Run("noRetrospectiveInDag", func(t *testing.T) {
		cfg := multiModelConfig{flowDag: &dag{Nodes: map[string]*dagNode{}}}
		metadata := GoalMetadata{Retrospective: "true"}
		result := blockCompletionOnRetrospective(cfg, state.Workflow{}, metadata)
		assert.Nil(t, result)
	})

	t.Run("retrospectiveAlreadyRan", func(t *testing.T) {
		cfg := multiModelConfig{flowDag: &dag{Nodes: map[string]*dagNode{"retrospective": {}}}}
		metadata := GoalMetadata{Retrospective: "true"}
		newState := state.Workflow{VisitCounts: map[string]int{"retrospective": 1}}
		result := blockCompletionOnRetrospective(cfg, newState, metadata)
		assert.Nil(t, result)
	})

	t.Run("blocksCompletion", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{
			coord:      coord,
			agent:      "coordinator",
			paddedsgai: "sgai",
			flowDag:    &dag{Nodes: map[string]*dagNode{"retrospective": {}}},
		}
		metadata := GoalMetadata{Retrospective: "true"}
		newState := state.Workflow{
			Status:      state.StatusComplete,
			VisitCounts: map[string]int{},
		}
		result := blockCompletionOnRetrospective(cfg, newState, metadata)
		require.NotNil(t, result)
		assert.Equal(t, state.StatusAgentDone, result.Status)
	})
}

func TestBlockCompletionOnGateScript(t *testing.T) {
	t.Run("noGateScript", func(t *testing.T) {
		cfg := multiModelConfig{}
		metadata := GoalMetadata{}
		result := blockCompletionOnGateScript(t.Context(), cfg, state.Workflow{}, metadata)
		assert.Nil(t, result)
	})
}

func TestHandleCompleteStatus(t *testing.T) {
	t.Run("nonCoordinatorAgent", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{
			coord:      coord,
			agent:      "developer",
			paddedsgai: "sgai",
			flowDag:    &dag{Nodes: map[string]*dagNode{}},
		}

		newState := state.Workflow{Status: state.StatusComplete}
		result := handleCompleteStatus(t.Context(), cfg, newState, state.Workflow{}, GoalMetadata{})
		assert.Equal(t, state.StatusAgentDone, result.Status)
	})

	t.Run("coordinatorNoPendingTodos", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{
			coord:      coord,
			agent:      "coordinator",
			paddedsgai: "sgai",
			flowDag:    &dag{Nodes: map[string]*dagNode{}},
		}

		newState := state.Workflow{Status: state.StatusComplete}
		wfState := state.Workflow{}
		metadata := GoalMetadata{Retrospective: "false"}

		result := handleCompleteStatus(t.Context(), cfg, newState, wfState, metadata)
		assert.Equal(t, state.StatusComplete, result.Status)
	})

	t.Run("blockedByPendingTodos", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{
			coord:      coord,
			agent:      "coordinator",
			paddedsgai: "sgai",
			flowDag:    &dag{Nodes: map[string]*dagNode{}},
		}

		newState := state.Workflow{Status: state.StatusComplete}
		wfState := state.Workflow{
			Todos: []state.TodoItem{{Content: "unfinished", Status: "pending", Priority: "high"}},
		}

		result := handleCompleteStatus(t.Context(), cfg, newState, wfState, GoalMetadata{Retrospective: "false"})
		assert.Equal(t, state.StatusWorking, result.Status)
	})

	t.Run("blockedByGateScript", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{
			coord:      coord,
			agent:      "coordinator",
			paddedsgai: "sgai",
			dir:        dir,
			flowDag:    &dag{Nodes: map[string]*dagNode{}},
		}

		newState := state.Workflow{Status: state.StatusComplete}
		wfState := state.Workflow{}
		metadata := GoalMetadata{
			CompletionGateScript: "false",
			Retrospective:        "false",
		}

		result := handleCompleteStatus(t.Context(), cfg, newState, wfState, metadata)
		assert.Equal(t, state.StatusWorking, result.Status)
	})

	t.Run("blockedByRetrospective", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		cfg := multiModelConfig{
			coord:      coord,
			agent:      "coordinator",
			paddedsgai: "sgai",
			flowDag: &dag{Nodes: map[string]*dagNode{
				"retrospective": {},
			}},
		}

		newState := state.Workflow{Status: state.StatusComplete, VisitCounts: map[string]int{}}
		wfState := state.Workflow{}

		result := handleCompleteStatus(t.Context(), cfg, newState, wfState, GoalMetadata{})
		assert.Equal(t, state.StatusAgentDone, result.Status)
	})
}

func TestRedirectToPendingMessageAgent(t *testing.T) {
	t.Run("noMessages", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		wfState := state.Workflow{}
		result := redirectToPendingMessageAgent(&wfState, coord, "sgai")
		assert.False(t, result)
	})

	t.Run("allMessagesRead", func(t *testing.T) {
		dir := t.TempDir()
		coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
		require.NoError(t, err)

		wfState := state.Workflow{
			Messages: []state.Message{
				{ID: 1, ToAgent: "dev", Read: true},
			},
		}
		result := redirectToPendingMessageAgent(&wfState, coord, "sgai")
		assert.False(t, result)
	})

	t.Run("unreadMessageRedirects", func(t *testing.T) {
		dir := t.TempDir()
		statePath := filepath.Join(dir, "state.json")
		coord, err := state.NewCoordinatorWith(statePath, state.Workflow{})
		require.NoError(t, err)

		wfState := state.Workflow{
			VisitCounts: map[string]int{},
			Messages: []state.Message{
				{ID: 1, ToAgent: "developer", Read: false},
			},
		}
		result := redirectToPendingMessageAgent(&wfState, coord, "sgai")
		assert.True(t, result)
		assert.Equal(t, "developer", wfState.CurrentAgent)
		assert.Equal(t, state.StatusWorking, wfState.Status)
	})
}

func TestBuildAgentArgsVariants(t *testing.T) {
	cases := []struct {
		name      string
		agent     string
		baseAgent string
		modelSpec string
		sessionID string
		wantModel bool
	}{
		{"basic", "coordinator", "coordinator", "", "", false},
		{"withModel", "builder", "builder", "claude-opus-4", "", true},
		{"withModelVariant", "builder", "builder", "claude-opus-4/fast", "", true},
		{"withSession", "builder", "builder", "", "sess-123", false},
		{"withModelAndSession", "builder", "builder", "gpt-4", "sess-1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := buildAgentArgs(tc.agent, tc.baseAgent, tc.modelSpec, tc.sessionID)
			assert.Contains(t, args, "run")
			assert.Contains(t, args, "--agent")
			if tc.wantModel {
				assert.Contains(t, args, "--model")
			}
			if tc.sessionID != "" {
				assert.Contains(t, args, "--session")
			}
		})
	}
}

func TestBuildAgentMessageWithPendingMessages(t *testing.T) {
	dag := buildTestDag(map[string][]string{"coordinator": {"builder"}}, []string{"coordinator"})
	cfg := multiModelConfig{
		dir:     "/tmp/test",
		agent:   "builder",
		flowDag: dag,
	}
	wfState := state.Workflow{
		Messages: []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "builder", Body: "Do work", Read: false},
		},
		VisitCounts: map[string]int{"builder": 1},
		Todos: []state.TodoItem{
			{Content: "task 1", Status: "pending", Priority: "high"},
		},
		CurrentAgent: "builder",
	}
	metadata := GoalMetadata{}

	msg := buildAgentMessage(cfg, wfState, metadata)
	assert.Contains(t, msg, "PENDING MESSAGE")
	assert.Contains(t, msg, "pending TODO items")
}

func TestBuildAgentMessageOutboxPendingMessages(t *testing.T) {
	dag := buildTestDag(map[string][]string{"coordinator": {"builder"}}, []string{"coordinator"})
	cfg := multiModelConfig{
		dir:     "/tmp/test",
		agent:   "builder",
		flowDag: dag,
	}
	wfState := state.Workflow{
		Messages: []state.Message{
			{ID: 1, FromAgent: "builder", ToAgent: "reviewer", Body: "Review please", Read: false},
		},
		VisitCounts: map[string]int{"builder": 1},
	}
	metadata := GoalMetadata{}

	msg := buildAgentMessage(cfg, wfState, metadata)
	assert.Contains(t, msg, "yield control")
}

func TestBuildAgentMessageWithMultiModel(t *testing.T) {
	dag := buildTestDag(map[string][]string{"coordinator": {"builder"}}, []string{"coordinator"})
	cfg := multiModelConfig{
		dir:     "/tmp/test",
		agent:   "builder",
		flowDag: dag,
	}
	wfState := state.Workflow{
		Messages:     []state.Message{},
		VisitCounts:  map[string]int{"builder": 1},
		CurrentModel: "model-1",
	}
	metadata := GoalMetadata{
		Models: map[string]any{
			"builder": []any{"model-1", "model-2"},
		},
	}

	msg := buildAgentMessage(cfg, wfState, metadata)
	assert.NotEmpty(t, msg)
}

func TestInitializeWorkspaceDirAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	err := initializeWorkspaceDir(dir)
	assert.NoError(t, err)
}

func TestInitializeWorkspaceDirFreshSetup(t *testing.T) {
	dir := t.TempDir()
	err := initializeWorkspaceDir(dir)
	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(dir, ".sgai"))
}

func TestFormatDurationVariants(t *testing.T) {
	assert.Equal(t, "45s", formatDuration(45*time.Second))
	assert.Equal(t, "5m 30s", formatDuration(5*time.Minute+30*time.Second))
	assert.Equal(t, "135m 0s", formatDuration(2*time.Hour+15*time.Minute))
}

func TestIsFalsishVariants(t *testing.T) {
	assert.True(t, isFalsish("false"))
	assert.True(t, isFalsish("no"))
	assert.True(t, isFalsish("0"))
	assert.True(t, isFalsish("FALSE"))
	assert.False(t, isFalsish("true"))
	assert.False(t, isFalsish("yes"))
}

func TestRetrospectiveEnabledVariants(t *testing.T) {
	assert.True(t, retrospectiveEnabled(GoalMetadata{}))
	assert.False(t, retrospectiveEnabled(GoalMetadata{Retrospective: "false"}))
	assert.True(t, retrospectiveEnabled(GoalMetadata{Retrospective: "true"}))
}

func TestFormatElapsedOutput(t *testing.T) {
	got := formatElapsed(time.Now().Add(-5 * time.Minute))
	assert.Contains(t, got, "05:0")
}

func TestIsExistingDirectoryVariants(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, isExistingDirectory(dir))
	assert.False(t, isExistingDirectory("/nonexistent/12345"))
}

func TestPrintUsageDoesNotPanic(_ *testing.T) {
	printUsage()
}

func TestComputeGoalChecksumDeterminism(t *testing.T) {
	dir := t.TempDir()
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("---\ntitle: test\n---\n# Goal"), 0o644))

	h1, err1 := computeGoalChecksum(goalPath)
	h2, err2 := computeGoalChecksum(goalPath)
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, h1, h2)
}

func TestResolveBaseAgentCases(t *testing.T) {
	alias := map[string]string{}
	assert.Equal(t, "coordinator", resolveBaseAgent(alias, "coordinator"))
	assert.Equal(t, "builder", resolveBaseAgent(alias, "builder"))
	assert.Equal(t, "retrospective", resolveBaseAgent(alias, "retrospective"))
}

func TestResolveBaseAgentWithAlias(t *testing.T) {
	alias := map[string]string{"custom-agent": "coordinator"}
	assert.Equal(t, "coordinator", resolveBaseAgent(alias, "custom-agent"))
	assert.Equal(t, "builder", resolveBaseAgent(alias, "builder"))
}

func TestFindFirstPendingMessageAgentVariants(t *testing.T) {
	t.Run("noMessages", func(t *testing.T) {
		assert.Empty(t, findFirstPendingMessageAgent(state.Workflow{}))
	})

	t.Run("allRead", func(t *testing.T) {
		wf := state.Workflow{Messages: []state.Message{{ToAgent: "builder", Read: true}}}
		assert.Empty(t, findFirstPendingMessageAgent(wf))
	})

	t.Run("unreadForAgent", func(t *testing.T) {
		wf := state.Workflow{
			Messages:     []state.Message{{ToAgent: "builder", Read: false}},
			CurrentAgent: "coordinator",
		}
		assert.Equal(t, "builder", findFirstPendingMessageAgent(wf))
	})
}

func TestValidateModelsPartial(t *testing.T) {
	t.Run("emptyModels", func(t *testing.T) {
		err := validateModels(nil)
		assert.NoError(t, err)
	})

	t.Run("singleValidModel", func(t *testing.T) {
		if _, err := exec.LookPath("opencode"); err != nil {
			t.Skip("opencode not found in PATH")
		}
		models := map[string]any{"coordinator": "anthropic/claude-opus-4-6"}
		err := validateModels(models)
		assert.NoError(t, err)
	})

	t.Run("invalidModel", func(t *testing.T) {
		if _, err := exec.LookPath("opencode"); err != nil {
			t.Skip("opencode not found in PATH")
		}
		models := map[string]any{"coordinator": "totally-fake-model-xyz"}
		err := validateModels(models)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid model")
	})

	t.Run("listWithValidModels", func(t *testing.T) {
		if _, err := exec.LookPath("opencode"); err != nil {
			t.Skip("opencode not found in PATH")
		}
		models := map[string]any{"coordinator": []any{"anthropic/claude-opus-4-6", "anthropic/claude-sonnet-4-6"}}
		err := validateModels(models)
		assert.NoError(t, err)
	})

	t.Run("listWithInvalidModel", func(t *testing.T) {
		if _, err := exec.LookPath("opencode"); err != nil {
			t.Skip("opencode not found in PATH")
		}
		models := map[string]any{"coordinator": []any{"anthropic/claude-opus-4-6", "fake-model-abc"}}
		err := validateModels(models)
		assert.Error(t, err)
	})
}

func TestSaveState(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	statePath := filepath.Join(sgaiDir, "state.json")

	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusWorking,
	})
	require.NoError(t, errCoord)

	wf := state.Workflow{Status: state.StatusComplete, Task: "done"}
	saveState(coord, wf)

	updated := coord.State()
	assert.Equal(t, state.StatusComplete, updated.Status)
}

func TestCopyLayerSubfolder(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	subDir := filepath.Join(srcDir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o644))

	require.NoError(t, copyLayerSubfolder(srcDir, dstDir, "sub"))

	data, errRead := os.ReadFile(filepath.Join(dstDir, "sub", "file.txt"))
	require.NoError(t, errRead)
	assert.Equal(t, "content", string(data))
}

func TestCopyLayerSubfolderNonExistent(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	require.NoError(t, copyLayerSubfolder(srcDir, dstDir, "nonexistent"))
	_, err := os.Stat(filepath.Join(dstDir, "nonexistent"))
	assert.True(t, os.IsNotExist(err))
}

func TestApplyLayerFolderOverlayWithSkills(t *testing.T) {
	baseDir := t.TempDir()

	overlayDir := filepath.Join(baseDir, "sgai")
	skillDir := filepath.Join(overlayDir, "skills", "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0o644))

	sgaiDir := filepath.Join(baseDir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))

	err := applyLayerFolderOverlay(baseDir)
	require.NoError(t, err)

	data, errRead := os.ReadFile(filepath.Join(sgaiDir, "skills", "my-skill", "SKILL.md"))
	require.NoError(t, errRead)
	assert.Equal(t, "# Skill", string(data))
}

func TestExtractBodyEmptyFrontmatter(t *testing.T) {
	input := []byte("---\n---\n# Body content")
	result := extractBody(input)
	assert.Equal(t, "# Body content", string(result))
}

func TestExtractBodyNoFrontmatter(t *testing.T) {
	input := []byte("Just plain text")
	result := extractBody(input)
	assert.Equal(t, "Just plain text", string(result))
}

func TestExtractFrontmatterDescriptionEmpty(t *testing.T) {
	result := extractFrontmatterDescription("")
	assert.Empty(t, result)
}

func TestExtractFrontmatterDescriptionValid(t *testing.T) {
	content := "---\ndescription: A great skill\n---\n# Skill"
	result := extractFrontmatterDescription(content)
	assert.Equal(t, "A great skill", result)
}

func TestCopyFileAtomicSuccessPath(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.txt")
	dstPath := filepath.Join(dir, "subdir", "dest.txt")
	require.NoError(t, os.WriteFile(srcPath, []byte("hello world"), 0644))
	require.NoError(t, copyFileAtomic(srcPath, dstPath))
	data, errRead := os.ReadFile(dstPath)
	require.NoError(t, errRead)
	assert.Equal(t, "hello world", string(data))
}

func TestCopyFileAtomicMissingSrcError(t *testing.T) {
	dir := t.TempDir()
	err := copyFileAtomic(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dest"))
	assert.Error(t, err)
}

func TestCopyFinalStateToRetrospectiveWithFiles(t *testing.T) {
	dir := t.TempDir()
	retroDir := filepath.Join(dir, "retro")
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.MkdirAll(retroDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"complete"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0644))

	require.NoError(t, copyFinalStateToRetrospective(dir, retroDir))

	stateData, errState := os.ReadFile(filepath.Join(retroDir, "state.json"))
	require.NoError(t, errState)
	assert.Contains(t, string(stateData), "complete")

	pmData, errPM := os.ReadFile(filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
	require.NoError(t, errPM)
	assert.Equal(t, "# PM", string(pmData))
}

func TestCopyFinalStateToRetrospectiveNoFilesDoesNotFail(t *testing.T) {
	dir := t.TempDir()
	retroDir := filepath.Join(dir, "retro")
	require.NoError(t, os.MkdirAll(retroDir, 0755))
	require.NoError(t, copyFinalStateToRetrospective(dir, retroDir))
}

func TestInitializeJJTest(t *testing.T) {
	dir := t.TempDir()
	err := initializeJJ(dir)
	assert.NoError(t, err)
}

func TestIsExistingDirectory(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		expected  bool
	}{
		{
			name: "existingDirectory",
			setupFunc: func(t *testing.T, path string) {
				require.NoError(t, os.MkdirAll(path, 0755))
			},
			expected: true,
		},
		{
			name: "nonexistentPath",
			setupFunc: func(_ *testing.T, _ string) {
			},
			expected: false,
		},
		{
			name: "existingFile",
			setupFunc: func(t *testing.T, path string) {
				dir := filepath.Dir(path)
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(path, []byte("content"), 0644))
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testPath := filepath.Join(dir, "test")
			tt.setupFunc(t, testPath)
			result := isExistingDirectory(testPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsProtectedFile(t *testing.T) {
	tests := []struct {
		name      string
		subfolder string
		relPath   string
		expected  bool
	}{
		{
			name:      "protectedCoordinator",
			subfolder: "agent",
			relPath:   "coordinator.md",
			expected:  true,
		},
		{
			name:      "nonProtectedAgent",
			subfolder: "agent",
			relPath:   "other.md",
			expected:  false,
		},
		{
			name:      "nonProtectedSubfolder",
			subfolder: "skills",
			relPath:   "coordinator.md",
			expected:  false,
		},
		{
			name:      "emptyPath",
			subfolder: "agent",
			relPath:   "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isProtectedFile(tt.subfolder, tt.relPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsExecNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "execNotFound",
			err:      &exec.Error{Name: "test", Err: exec.ErrNotFound},
			expected: true,
		},
		{
			name:     "otherError",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "wrappedExecNotFound",
			err:      errors.Join(errors.New("wrapper"), &exec.Error{Name: "test", Err: exec.ErrNotFound}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExecNotFound(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseYAMLFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, GoalMetadata)
	}{
		{
			name: "validFrontmatter",
			content: `---
flow: |
  "agent1" -> "agent2"
models:
  "agent1": "model1"
---
# Goal`,
			wantErr: false,
			validate: func(t *testing.T, m GoalMetadata) {
				assert.Contains(t, m.Flow, "agent1")
				assert.Equal(t, "model1", m.Models["agent1"])
			},
		},
		{
			name:    "noFrontmatter",
			content: "# Just a goal",
			wantErr: false,
			validate: func(t *testing.T, m GoalMetadata) {
				assert.Equal(t, "", m.Flow)
			},
		},
		{
			name: "unclosedFrontmatter",
			content: `---
flow: "test"
# no closing`,
			wantErr:     true,
			errContains: "no closing",
		},
		{
			name: "invalidYAML",
			content: `---
flow: [invalid yaml
---
# Goal`,
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name: "emptyFrontmatter",
			content: `---
---
# Goal`,
			wantErr: false,
			validate: func(t *testing.T, m GoalMetadata) {
				assert.Equal(t, "", m.Flow)
			},
		},
		{
			name: "withRetrospective",
			content: `---
flow: "test"
retrospective: "true"
---
# Goal`,
			wantErr: false,
			validate: func(t *testing.T, m GoalMetadata) {
				assert.Equal(t, "true", m.Retrospective)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := parseYAMLFrontmatter([]byte(tt.content))

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, metadata)
			}
		})
	}
}

func TestRetrospectiveEnabled(t *testing.T) {
	tests := []struct {
		name     string
		metadata GoalMetadata
		expected bool
	}{
		{
			name:     "trueString",
			metadata: GoalMetadata{Retrospective: "true"},
			expected: true,
		},
		{
			name:     "falseString",
			metadata: GoalMetadata{Retrospective: "false"},
			expected: false,
		},
		{
			name:     "emptyString",
			metadata: GoalMetadata{Retrospective: ""},
			expected: true,
		},
		{
			name:     "yesString",
			metadata: GoalMetadata{Retrospective: "yes"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retrospectiveEnabled(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindFirstPendingMessageAgent(t *testing.T) {
	tests := []struct {
		name     string
		workflow state.Workflow
		expected string
	}{
		{
			name:     "noMessages",
			workflow: state.Workflow{Messages: []state.Message{}},
			expected: "",
		},
		{
			name: "allRead",
			workflow: state.Workflow{
				Messages: []state.Message{
					{ToAgent: "agent1", Read: true},
					{ToAgent: "agent2", Read: true},
				},
			},
			expected: "",
		},
		{
			name: "firstUnread",
			workflow: state.Workflow{
				Messages: []state.Message{
					{ToAgent: "agent1", Read: true},
					{ToAgent: "agent2", Read: false},
					{ToAgent: "agent3", Read: false},
				},
			},
			expected: "agent2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findFirstPendingMessageAgent(tt.workflow)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFrontmatterDescription(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "withDescription",
			content: `---
description: Test description
---
# Content`,
			expected: "Test description",
		},
		{
			name: "noDescription",
			content: `---
name: Test
---
# Content`,
			expected: "",
		},
		{
			name:     "noFrontmatter",
			content:  `# Just content`,
			expected: "",
		},
		{
			name:     "emptyContent",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFrontmatterDescription(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFrontmatterMap(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "simple",
			content: `---
key1: value1
key2: value2
---
# Content`,
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:     "noFrontmatter",
			content:  `# Just content`,
			expected: map[string]string{},
		},
		{
			name: "emptyFrontmatter",
			content: `---
---
# Content`,
			expected: map[string]string{},
		},
		{
			name: "quotedValue",
			content: `---
key: "quoted value"
---
# Content`,
			expected: map[string]string{
				"key": "\"quoted value\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFrontmatterMap([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetModelsForAgent(t *testing.T) {
	tests := []struct {
		name     string
		models   map[string]any
		agent    string
		expected []string
	}{
		{
			name:     "agentNotInModels",
			models:   map[string]any{"agent1": "model1"},
			agent:    "agent2",
			expected: nil,
		},
		{
			name:     "singleModel",
			models:   map[string]any{"agent1": "model1"},
			agent:    "agent1",
			expected: []string{"model1"},
		},
		{
			name:     "emptyStringModel",
			models:   map[string]any{"agent1": ""},
			agent:    "agent1",
			expected: nil,
		},
		{
			name: "multipleModels",
			models: map[string]any{
				"agent1": []any{"model1", "model2"},
			},
			agent:    "agent1",
			expected: []string{"model1", "model2"},
		},
		{
			name: "mixedTypesInArray",
			models: map[string]any{
				"agent1": []any{"model1", 123, "model2"},
			},
			agent:    "agent1",
			expected: []string{"model1", "model2"},
		},
		{
			name: "emptyArray",
			models: map[string]any{
				"agent1": []any{},
			},
			agent:    "agent1",
			expected: []string{},
		},
		{
			name: "arrayWithEmptyStrings",
			models: map[string]any{
				"agent1": []any{"", "model1", ""},
			},
			agent:    "agent1",
			expected: []string{"model1"},
		},
		{
			name:     "nilModels",
			models:   nil,
			agent:    "agent1",
			expected: nil,
		},
		{
			name:     "invalidType",
			models:   map[string]any{"agent1": 123},
			agent:    "agent1",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModelsForAgent(tt.models, tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNextMessageID(t *testing.T) {
	tests := []struct {
		name     string
		messages []state.Message
		expected int
	}{
		{
			name:     "empty",
			messages: []state.Message{},
			expected: 1,
		},
		{
			name: "singleMessage",
			messages: []state.Message{
				{ID: 1},
			},
			expected: 2,
		},
		{
			name: "multipleMessages",
			messages: []state.Message{
				{ID: 1},
				{ID: 2},
				{ID: 3},
			},
			expected: 4,
		},
		{
			name: "nonSequential",
			messages: []state.Message{
				{ID: 1},
				{ID: 5},
				{ID: 3},
			},
			expected: 6,
		},
		{
			name: "zeroID",
			messages: []state.Message{
				{ID: 0},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextMessageID(tt.messages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddEnvironmentMessage(t *testing.T) {
	wf := &state.Workflow{
		Messages: []state.Message{},
	}

	addEnvironmentMessage(wf, "agent1", "test message")

	assert.Len(t, wf.Messages, 1)
	assert.Equal(t, 1, wf.Messages[0].ID)
	assert.Equal(t, "environment", wf.Messages[0].FromAgent)
	assert.Equal(t, "agent1", wf.Messages[0].ToAgent)
	assert.Equal(t, "test message", wf.Messages[0].Body)
	assert.False(t, wf.Messages[0].Read)
	assert.NotEmpty(t, wf.Messages[0].CreatedAt)

	addEnvironmentMessage(wf, "agent2", "another message")

	assert.Len(t, wf.Messages, 2)
	assert.Equal(t, 2, wf.Messages[1].ID)
	assert.Equal(t, "agent2", wf.Messages[1].ToAgent)
}

func TestHasMessagesForModel(t *testing.T) {
	tests := []struct {
		name     string
		messages []state.Message
		modelID  string
		expected bool
	}{
		{
			name:     "emptyMessages",
			messages: []state.Message{},
			modelID:  "agent1:model1",
			expected: false,
		},
		{
			name: "messageForModel",
			messages: []state.Message{
				{ToAgent: "agent1:model1", Read: false},
			},
			modelID:  "agent1:model1",
			expected: true,
		},
		{
			name: "messageForAgentOnly",
			messages: []state.Message{
				{ToAgent: "agent1", Read: false},
			},
			modelID:  "agent1:model1",
			expected: true,
		},
		{
			name: "messageAlreadyRead",
			messages: []state.Message{
				{ToAgent: "agent1:model1", Read: true},
			},
			modelID:  "agent1:model1",
			expected: false,
		},
		{
			name: "messageForDifferentAgent",
			messages: []state.Message{
				{ToAgent: "agent2:model1", Read: false},
			},
			modelID:  "agent1:model1",
			expected: false,
		},
		{
			name: "mixedMessages",
			messages: []state.Message{
				{ToAgent: "agent1:model1", Read: true},
				{ToAgent: "agent1:model2", Read: false},
			},
			modelID:  "agent1:model1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasMessagesForModel(tt.messages, tt.modelID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPendingMessagesForAnyModel(t *testing.T) {
	tests := []struct {
		name     string
		messages []state.Message
		models   []string
		agent    string
		expected bool
	}{
		{
			name:     "emptyMessages",
			messages: []state.Message{},
			models:   []string{"model1", "model2"},
			agent:    "agent1",
			expected: false,
		},
		{
			name: "messageForFirstModel",
			messages: []state.Message{
				{ToAgent: "agent1:model1", Read: false},
			},
			models:   []string{"model1", "model2"},
			agent:    "agent1",
			expected: true,
		},
		{
			name: "messageForSecondModel",
			messages: []state.Message{
				{ToAgent: "agent1:model2", Read: false},
			},
			models:   []string{"model1", "model2"},
			agent:    "agent1",
			expected: true,
		},
		{
			name: "allMessagesRead",
			messages: []state.Message{
				{ToAgent: "agent1:model1", Read: true},
				{ToAgent: "agent1:model2", Read: true},
			},
			models:   []string{"model1", "model2"},
			agent:    "agent1",
			expected: false,
		},
		{
			name: "messageForDifferentAgent",
			messages: []state.Message{
				{ToAgent: "agent2:model1", Read: false},
			},
			models:   []string{"model1", "model2"},
			agent:    "agent1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPendingMessagesForAnyModel(tt.messages, tt.models, tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncModelStatuses(t *testing.T) {
	tests := []struct {
		name            string
		existingStatus  map[string]string
		models          []string
		agent           string
		expectedStatus  map[string]string
		expectedDeleted int
	}{
		{
			name:           "nilStatus",
			existingStatus: nil,
			models:         []string{"model1", "model2"},
			agent:          "agent1",
			expectedStatus: map[string]string{
				"agent1:model1": "model-working",
				"agent1:model2": "model-working",
			},
		},
		{
			name: "addNewModels",
			existingStatus: map[string]string{
				"agent1:model1": "model-working",
			},
			models: []string{"model1", "model2"},
			agent:  "agent1",
			expectedStatus: map[string]string{
				"agent1:model1": "model-working",
				"agent1:model2": "model-working",
			},
		},
		{
			name: "removeOldModels",
			existingStatus: map[string]string{
				"agent1:model1": "model-working",
				"agent1:model2": "model-done",
			},
			models: []string{"model1"},
			agent:  "agent1",
			expectedStatus: map[string]string{
				"agent1:model1": "model-working",
			},
		},
		{
			name: "preserveOtherAgentStatuses",
			existingStatus: map[string]string{
				"agent1:model1": "model-working",
				"agent2:model1": "model-done",
			},
			models: []string{"model1"},
			agent:  "agent1",
			expectedStatus: map[string]string{
				"agent1:model1": "model-working",
				"agent2:model1": "model-done",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncModelStatuses(tt.existingStatus, tt.models, tt.agent)
			assert.Equal(t, tt.expectedStatus, result)
		})
	}
}

func TestCleanupModelStatuses(t *testing.T) {
	wf := &state.Workflow{
		ModelStatuses: map[string]string{
			"agent1/model1": "model-working",
			"agent1/model2": "model-done",
		},
		CurrentModel: "agent1/model1",
	}

	cleanupModelStatuses(wf)

	assert.Nil(t, wf.ModelStatuses)
	assert.Empty(t, wf.CurrentModel)
}

func TestFormatModelID(t *testing.T) {
	tests := []struct {
		name      string
		agent     string
		modelSpec string
		expected  string
	}{
		{
			name:      "simple",
			agent:     "agent1",
			modelSpec: "model1",
			expected:  "agent1:model1",
		},
		{
			name:      "emptyAgent",
			agent:     "",
			modelSpec: "model1",
			expected:  ":model1",
		},
		{
			name:      "emptyModel",
			agent:     "agent1",
			modelSpec: "",
			expected:  "agent1:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatModelID(tt.agent, tt.modelSpec)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractAgentFromModelID(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected string
	}{
		{
			name:     "withColon",
			modelID:  "agent1:model1",
			expected: "agent1",
		},
		{
			name:     "noColon",
			modelID:  "agent1",
			expected: "agent1",
		},
		{
			name:     "empty",
			modelID:  "",
			expected: "",
		},
		{
			name:     "multipleColons",
			modelID:  "agent1:model1:variant",
			expected: "agent1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAgentFromModelID(tt.modelID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllModelsDone(t *testing.T) {
	tests := []struct {
		name          string
		modelStatuses map[string]string
		expected      bool
	}{
		{
			name:          "empty",
			modelStatuses: map[string]string{},
			expected:      true,
		},
		{
			name: "allDone",
			modelStatuses: map[string]string{
				"model1": "model-done",
				"model2": "model-done",
			},
			expected: true,
		},
		{
			name: "allDoneOrError",
			modelStatuses: map[string]string{
				"model1": "model-done",
				"model2": "model-error",
			},
			expected: true,
		},
		{
			name: "oneRunning",
			modelStatuses: map[string]string{
				"model1": "model-done",
				"model2": "model-running",
			},
			expected: false,
		},
		{
			name: "allRunning",
			modelStatuses: map[string]string{
				"model1": "model-running",
				"model2": "model-running",
			},
			expected: false,
		},
		{
			name: "oneWorking",
			modelStatuses: map[string]string{
				"model1": "model-working",
				"model2": "model-done",
			},
			expected: false,
		},
		{
			name:          "nilStatuses",
			modelStatuses: nil,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allModelsDone(tt.modelStatuses)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAgentPrefix(t *testing.T) {
	tests := []struct {
		name            string
		dir             string
		paddedAgentName string
		iteration       int
		expected        string
	}{
		{
			name:            "simple",
			dir:             "/path/to/workspace",
			paddedAgentName: "agent1  ",
			iteration:       1,
			expected:        "[workspace][agent1  :0001]",
		},
		{
			name:            "largeIteration",
			dir:             "/path/to/workspace",
			paddedAgentName: "agent2",
			iteration:       12345,
			expected:        "[workspace][agent2:12345]",
		},
		{
			name:            "rootDir",
			dir:             "/workspace",
			paddedAgentName: "agent",
			iteration:       0,
			expected:        "[workspace][agent:0000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAgentPrefix(tt.dir, tt.paddedAgentName, tt.iteration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAgentArgs(t *testing.T) {
	tests := []struct {
		name      string
		agent     string
		baseAgent string
		modelSpec string
		sessionID string
		expected  []string
	}{
		{
			name:      "simpleAgent",
			agent:     "agent1",
			baseAgent: "agent1",
			modelSpec: "",
			sessionID: "",
			expected:  []string{"run", "--format=json", "--agent", "agent1", "--title", "agent1"},
		},
		{
			name:      "withModel",
			agent:     "agent1",
			baseAgent: "agent1",
			modelSpec: "gpt-4",
			sessionID: "",
			expected:  []string{"run", "--format=json", "--agent", "agent1", "--model", "gpt-4", "--title", "agent1 [gpt-4]"},
		},
		{
			name:      "withModelAndVariant",
			agent:     "agent1",
			baseAgent: "agent1",
			modelSpec: "gpt-4:latest",
			sessionID: "",
			expected:  []string{"run", "--format=json", "--agent", "agent1", "--model", "gpt-4:latest", "--title", "agent1 [gpt-4:latest]"},
		},
		{
			name:      "withSession",
			agent:     "agent1",
			baseAgent: "agent1",
			modelSpec: "",
			sessionID: "session-123",
			expected:  []string{"run", "--format=json", "--agent", "agent1", "--session", "session-123", "--title", "agent1"},
		},
		{
			name:      "withAll",
			agent:     "agent1",
			baseAgent: "base-agent",
			modelSpec: "gpt-4",
			sessionID: "session-123",
			expected:  []string{"run", "--format=json", "--agent", "base-agent", "--model", "gpt-4", "--session", "session-123", "--title", "agent1 [gpt-4]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAgentArgs(tt.agent, tt.baseAgent, tt.modelSpec, tt.sessionID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseModelAndVariant(t *testing.T) {
	tests := []struct {
		name          string
		modelSpec     string
		expectedModel string
		expectedVar   string
	}{
		{
			name:          "modelWithVariant",
			modelSpec:     "anthropic/claude-opus-4-6 (max)",
			expectedModel: "anthropic/claude-opus-4-6",
			expectedVar:   "max",
		},
		{
			name:          "modelWithoutVariant",
			modelSpec:     "anthropic/claude-opus-4-6",
			expectedModel: "anthropic/claude-opus-4-6",
			expectedVar:   "",
		},
		{
			name:          "empty",
			modelSpec:     "",
			expectedModel: "",
			expectedVar:   "",
		},
		{
			name:          "variantOnly",
			modelSpec:     "(max)",
			expectedModel: "(max)",
			expectedVar:   "",
		},
		{
			name:          "multipleParentheses",
			modelSpec:     "model (variant) (extra)",
			expectedModel: "model (variant)",
			expectedVar:   "extra",
		},
		{
			name:          "variantWithSpaces",
			modelSpec:     "model (variant with spaces)",
			expectedModel: "model",
			expectedVar:   "variant with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, variant := parseModelAndVariant(tt.modelSpec)
			assert.Equal(t, tt.expectedModel, model)
			assert.Equal(t, tt.expectedVar, variant)
		})
	}
}

func TestUpdateProjectManagementWithRetrospectiveDir(t *testing.T) {
	tests := []struct {
		name               string
		existingContent    string
		retrospectiveDir   string
		expectedContains   []string
		expectedNotContain []string
	}{
		{
			name:             "newFileNoExistingContent",
			existingContent:  "",
			retrospectiveDir: ".sgai/retrospectives/2026-03-05-10-00.abc1",
			expectedContains: []string{"---", "Retrospective Session: .sgai/retrospectives/2026-03-05-10-00.abc1"},
		},
		{
			name:             "existingContentWithoutHeader",
			existingContent:  "## Some existing content\n\nHello world\n",
			retrospectiveDir: ".sgai/retrospectives/2026-03-05-10-00.abc1",
			expectedContains: []string{"---", "Retrospective Session:", "## Some existing content"},
		},
		{
			name:               "replaceExistingRetrospectiveHeader",
			existingContent:    "---\nRetrospective Session: .sgai/retrospectives/old-session\n---\n\n## Old content\n",
			retrospectiveDir:   ".sgai/retrospectives/new-session",
			expectedContains:   []string{"Retrospective Session: .sgai/retrospectives/new-session", "## Old content"},
			expectedNotContain: []string{"old-session"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pmPath := filepath.Join(tmpDir, ".sgai", "PROJECT_MANAGEMENT.md")

			if tt.existingContent != "" {
				require.NoError(t, os.MkdirAll(filepath.Dir(pmPath), 0755))
				require.NoError(t, os.WriteFile(pmPath, []byte(tt.existingContent), 0644))
			}

			err := updateProjectManagementWithRetrospectiveDir(pmPath, tt.retrospectiveDir)
			require.NoError(t, err)

			content, err := os.ReadFile(pmPath)
			require.NoError(t, err)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, string(content), expected)
			}
			for _, notExpected := range tt.expectedNotContain {
				assert.NotContains(t, string(content), notExpected)
			}
		})
	}
}

func TestExtractRetrospectiveDirFromProjectManagement(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "validHeader",
			content:  "---\nRetrospective Session: .sgai/retrospectives/2026-03-05-10-00.abc1\n---\n\n## Content\n",
			expected: ".sgai/retrospectives/2026-03-05-10-00.abc1",
		},
		{
			name:     "noHeader",
			content:  "## No header here\n",
			expected: "",
		},
		{
			name:     "emptyFile",
			content:  "",
			expected: "",
		},
		{
			name:     "headerWithoutRetrospectiveSession",
			content:  "---\nTitle: Some Title\n---\n\n## Content\n",
			expected: "",
		},
		{
			name:     "nonExistentFile",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pmPath := filepath.Join(tmpDir, "PROJECT_MANAGEMENT.md")

			if tt.name == "nonExistentFile" {
				result := extractRetrospectiveDirFromProjectManagement(filepath.Join(tmpDir, "nonexistent.md"))
				assert.Equal(t, "", result)
				return
			}

			require.NoError(t, os.WriteFile(pmPath, []byte(tt.content), 0644))
			result := extractRetrospectiveDirFromProjectManagement(pmPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanResumeWorkflow(t *testing.T) {
	tests := []struct {
		name            string
		wfState         state.Workflow
		currentChecksum string
		expected        bool
	}{
		{
			name:            "matchingChecksumWorkingStatus",
			wfState:         state.Workflow{GoalChecksum: "abc123", Status: state.StatusWorking},
			currentChecksum: "abc123",
			expected:        true,
		},
		{
			name:            "matchingChecksumAgentDone",
			wfState:         state.Workflow{GoalChecksum: "abc123", Status: state.StatusAgentDone},
			currentChecksum: "abc123",
			expected:        true,
		},
		{
			name:            "matchingChecksumHumanPending",
			wfState:         state.Workflow{GoalChecksum: "abc123", Status: state.StatusWaitingForHuman},
			currentChecksum: "abc123",
			expected:        true,
		},
		{
			name:            "mismatchedChecksum",
			wfState:         state.Workflow{GoalChecksum: "abc123", Status: state.StatusWorking},
			currentChecksum: "different",
			expected:        false,
		},
		{
			name:            "completeStatus",
			wfState:         state.Workflow{GoalChecksum: "abc123", Status: state.StatusComplete},
			currentChecksum: "abc123",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canResumeWorkflow(tt.wfState, tt.currentChecksum)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCopyFileAtomic(t *testing.T) {
	t.Run("successfulCopy", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "dest", "copied.txt")

		require.NoError(t, os.WriteFile(srcPath, []byte("hello world"), 0644))

		err := copyFileAtomic(srcPath, dstPath)
		require.NoError(t, err)

		content, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(content))
	})

	t.Run("sourceDoesNotExist", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := copyFileAtomic(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
		assert.Error(t, err)
	})
}

func TestCopyFinalStateToRetrospective(t *testing.T) {
	t.Run("copiesBothFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		sgaiDir := filepath.Join(tmpDir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"complete"}`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("## PM Content"), 0644))

		retroDir := filepath.Join(tmpDir, "retro")
		require.NoError(t, os.MkdirAll(retroDir, 0755))

		err := copyFinalStateToRetrospective(tmpDir, retroDir)
		require.NoError(t, err)

		stateContent, err := os.ReadFile(filepath.Join(retroDir, "state.json"))
		require.NoError(t, err)
		assert.Contains(t, string(stateContent), "complete")

		pmContent, err := os.ReadFile(filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
		require.NoError(t, err)
		assert.Contains(t, string(pmContent), "PM Content")
	})

	t.Run("missingFilesNoError", func(t *testing.T) {
		tmpDir := t.TempDir()
		retroDir := filepath.Join(tmpDir, "retro")
		require.NoError(t, os.MkdirAll(retroDir, 0755))

		err := copyFinalStateToRetrospective(tmpDir, retroDir)
		require.NoError(t, err)
	})
}

func TestApplyLayerFolderOverlay(t *testing.T) {
	t.Run("noLayerDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := applyLayerFolderOverlay(tmpDir)
		require.NoError(t, err)
	})

	t.Run("copiesSkillsOverlay", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcSkillDir := filepath.Join(tmpDir, "sgai", "skills", "my-skill")
		require.NoError(t, os.MkdirAll(srcSkillDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcSkillDir, "SKILL.md"), []byte("# Skill Content"), 0644))

		dstDir := filepath.Join(tmpDir, ".sgai")
		require.NoError(t, os.MkdirAll(dstDir, 0755))

		err := applyLayerFolderOverlay(tmpDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tmpDir, ".sgai", "skills", "my-skill", "SKILL.md"))
		require.NoError(t, err)
		assert.Equal(t, "# Skill Content", string(content))
	})

	t.Run("protectsCoordinatorMD", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcAgentDir := filepath.Join(tmpDir, "sgai", "agent")
		require.NoError(t, os.MkdirAll(srcAgentDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(srcAgentDir, "coordinator.md"), []byte("SHOULD NOT COPY"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(srcAgentDir, "developer.md"), []byte("SHOULD COPY"), 0644))

		dstAgentDir := filepath.Join(tmpDir, ".sgai", "agent")
		require.NoError(t, os.MkdirAll(dstAgentDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dstAgentDir, "coordinator.md"), []byte("ORIGINAL"), 0644))

		err := applyLayerFolderOverlay(tmpDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(dstAgentDir, "coordinator.md"))
		require.NoError(t, err)
		assert.Equal(t, "ORIGINAL", string(content))

		content, err = os.ReadFile(filepath.Join(dstAgentDir, "developer.md"))
		require.NoError(t, err)
		assert.Equal(t, "SHOULD COPY", string(content))
	})
}

func TestAgentHasUnreadOutgoingMessages(t *testing.T) {
	tests := []struct {
		name      string
		messages  []state.Message
		agentName string
		expected  bool
	}{
		{
			name:      "noMessages",
			messages:  []state.Message{},
			agentName: "test-agent",
			expected:  false,
		},
		{
			name: "hasUnreadOutgoing",
			messages: []state.Message{
				{ID: 1, FromAgent: "test-agent", ToAgent: "coordinator", Body: "hello", Read: false},
			},
			agentName: "test-agent",
			expected:  true,
		},
		{
			name: "allOutgoingRead",
			messages: []state.Message{
				{ID: 1, FromAgent: "test-agent", ToAgent: "coordinator", Body: "hello", Read: true},
			},
			agentName: "test-agent",
			expected:  false,
		},
		{
			name: "unreadFromOtherAgent",
			messages: []state.Message{
				{ID: 1, FromAgent: "other-agent", ToAgent: "coordinator", Body: "hello", Read: false},
			},
			agentName: "test-agent",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := state.Workflow{Messages: tt.messages}
			result := agentHasUnreadOutgoingMessages(wf, tt.agentName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAgentMessage(t *testing.T) {
	dag, err := parseFlow("\"agent1\" -> \"agent2\"\n", "")
	require.NoError(t, err)

	tests := []struct {
		name         string
		cfg          multiModelConfig
		wfState      state.Workflow
		metadata     GoalMetadata
		wantContains []string
	}{
		{
			name: "withPendingMessages",
			cfg: multiModelConfig{
				agent:   "agent1",
				flowDag: dag,
				dir:     t.TempDir(),
			},
			wfState: state.Workflow{
				Status:      state.StatusWorking,
				VisitCounts: map[string]int{"agent1": 1},
				Messages: []state.Message{
					{ID: 1, FromAgent: "coordinator", ToAgent: "agent1", Body: "do work", Read: false},
				},
			},
			metadata:     GoalMetadata{},
			wantContains: []string{"YOU HAVE 1 PENDING MESSAGE(S)"},
		},
		{
			name: "withPendingTodos",
			cfg: multiModelConfig{
				agent:   "agent1",
				flowDag: dag,
				dir:     t.TempDir(),
			},
			wfState: state.Workflow{
				Status:      state.StatusWorking,
				VisitCounts: map[string]int{"agent1": 1},
				Messages:    []state.Message{},
				Todos: []state.TodoItem{
					{Content: "pending task", Status: "pending", Priority: "high"},
				},
			},
			metadata:     GoalMetadata{},
			wantContains: []string{"1 pending TODO items"},
		},
		{
			name: "withUnreadOutboxMessages",
			cfg: multiModelConfig{
				agent:   "agent1",
				flowDag: dag,
				dir:     t.TempDir(),
			},
			wfState: state.Workflow{
				Status:      state.StatusWorking,
				VisitCounts: map[string]int{"agent1": 1},
				Messages: []state.Message{
					{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "review this", Read: false},
				},
			},
			metadata:     GoalMetadata{},
			wantContains: []string{"messages that haven't been read yet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAgentMessage(tt.cfg, tt.wfState, tt.metadata)
			for _, expected := range tt.wantContains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestBuildAgentEnv(t *testing.T) {
	cfg := multiModelConfig{
		agent: "test-agent",
		dir:   "/tmp/test-workspace",
	}
	wfState := state.Workflow{
		InteractionMode: state.ModeSelfDrive,
	}

	env := buildAgentEnv(cfg, wfState, "")
	envMap := make(map[string]string)
	for _, e := range env {
		if len(e) > 0 {
			for i := 0; i < len(e); i++ {
				if e[i] == '=' {
					envMap[e[:i]] = e[i+1:]
					break
				}
			}
		}
	}

	assert.Equal(t, filepath.Join("/tmp/test-workspace", ".sgai"), envMap["OPENCODE_CONFIG_DIR"])
	assert.Equal(t, "auto", envMap["SGAI_MCP_INTERACTIVE"])
	assert.Equal(t, "test-agent", envMap["SGAI_AGENT_IDENTITY"])
}

func TestBuildAgentEnvWithModel(t *testing.T) {
	cfg := multiModelConfig{
		agent: "test-agent",
		dir:   "/tmp/test-workspace",
	}
	wfState := state.Workflow{}

	env := buildAgentEnv(cfg, wfState, "anthropic/claude-opus-4-6 (max)")

	identityValues := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				identityValues[e[:i]] = e[i+1:]
				break
			}
		}
	}

	assert.Contains(t, identityValues["SGAI_AGENT_IDENTITY"], "anthropic/claude-opus-4-6")
}

func TestMarkCurrentAgentInSequence(t *testing.T) {
	tests := []struct {
		name         string
		initialSeq   []state.AgentSequenceEntry
		currentAgent string
		expectedLen  int
		expectedLast string
	}{
		{
			name:         "emptySequence",
			initialSeq:   nil,
			currentAgent: "agent1",
			expectedLen:  1,
			expectedLast: "agent1",
		},
		{
			name: "sameAgentAsLast",
			initialSeq: []state.AgentSequenceEntry{
				{Agent: "agent1", IsCurrent: false},
			},
			currentAgent: "agent1",
			expectedLen:  1,
			expectedLast: "agent1",
		},
		{
			name: "differentAgent",
			initialSeq: []state.AgentSequenceEntry{
				{Agent: "agent1", IsCurrent: true},
			},
			currentAgent: "agent2",
			expectedLen:  2,
			expectedLast: "agent2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := &state.Workflow{AgentSequence: tt.initialSeq}
			markCurrentAgentInSequence(wf, tt.currentAgent)
			assert.Len(t, wf.AgentSequence, tt.expectedLen)
			last := wf.AgentSequence[len(wf.AgentSequence)-1]
			assert.Equal(t, tt.expectedLast, last.Agent)
			assert.True(t, last.IsCurrent)
		})
	}
}

func TestAddAgentHandoffProgress(t *testing.T) {
	wf := &state.Workflow{
		Progress: []state.ProgressEntry{},
	}

	addAgentHandoffProgress(wf, "backend-developer")

	assert.Len(t, wf.Progress, 1)
	assert.Equal(t, "sgai", wf.Progress[0].Agent)
	assert.Contains(t, wf.Progress[0].Description, "Handing off to backend-developer")
}

func TestShouldLogAgent(t *testing.T) {
	t.Run("defaultTrueWhenNoFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.True(t, shouldLogAgent(tmpDir, "nonexistent"))
	})

	t.Run("trueWhenLogIsTrue", func(t *testing.T) {
		tmpDir := t.TempDir()
		agentDir := filepath.Join(tmpDir, ".sgai", "agent")
		require.NoError(t, os.MkdirAll(agentDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(agentDir, "test-agent.md"), []byte("---\nlog: true\n---\n# Agent"), 0644))

		assert.True(t, shouldLogAgent(tmpDir, "test-agent"))
	})

	t.Run("falseWhenLogIsFalse", func(t *testing.T) {
		tmpDir := t.TempDir()
		agentDir := filepath.Join(tmpDir, ".sgai", "agent")
		require.NoError(t, os.MkdirAll(agentDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(agentDir, "test-agent.md"), []byte("---\nlog: false\n---\n# Agent"), 0644))

		assert.False(t, shouldLogAgent(tmpDir, "test-agent"))
	})
}

func TestParseAgentSnippets(t *testing.T) {
	t.Run("noAgentFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		result := parseAgentSnippets(tmpDir, "nonexistent")
		assert.Nil(t, result)
	})

	t.Run("agentWithSnippets", func(t *testing.T) {
		tmpDir := t.TempDir()
		agentDir := filepath.Join(tmpDir, ".sgai", "agent")
		require.NoError(t, os.MkdirAll(agentDir, 0755))
		content := "---\nlog: true\nsnippets:\n  - go/http-server\n  - go/json-encode\n---\n# Agent"
		require.NoError(t, os.WriteFile(filepath.Join(agentDir, "developer.md"), []byte(content), 0644))

		result := parseAgentSnippets(tmpDir, "developer")
		assert.Equal(t, []string{"go/http-server", "go/json-encode"}, result)
	})
}

func TestParseAgentFileMetadata(t *testing.T) {
	t.Run("noFrontmatter", func(t *testing.T) {
		tmpDir := t.TempDir()
		agentDir := filepath.Join(tmpDir, ".sgai", "agent")
		require.NoError(t, os.MkdirAll(agentDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(agentDir, "test.md"), []byte("# No frontmatter"), 0644))

		_, ok := parseAgentFileMetadata(tmpDir, "test")
		assert.False(t, ok)
	})
}

func TestValidateModels(t *testing.T) {
	t.Run("emptyModels", func(t *testing.T) {
		err := validateModels(map[string]any{})
		assert.NoError(t, err)
	})

	t.Run("nilModels", func(t *testing.T) {
		err := validateModels(nil)
		assert.NoError(t, err)
	})
}

func TestReadNewestForkGoal(t *testing.T) {
	t.Run("emptyForks", func(t *testing.T) {
		result := readNewestForkGoal([]workspaceInfo{})
		assert.Empty(t, result)
	})

	t.Run("forkWithGoal", func(t *testing.T) {
		tmpDir := t.TempDir()
		forkDir := filepath.Join(tmpDir, "fork1")
		require.NoError(t, os.MkdirAll(forkDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte("# Fork Goal"), 0644))

		forks := []workspaceInfo{{Directory: forkDir, DirName: "fork1"}}
		result := readNewestForkGoal(forks)
		assert.Equal(t, "# Fork Goal", result)
	})

	t.Run("forkWithEmptyGoal", func(t *testing.T) {
		tmpDir := t.TempDir()
		forkDir := filepath.Join(tmpDir, "fork1")
		require.NoError(t, os.MkdirAll(forkDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte("   "), 0644))

		forks := []workspaceInfo{{Directory: forkDir, DirName: "fork1"}}
		result := readNewestForkGoal(forks)
		assert.Empty(t, result)
	})

	t.Run("multipleForksSortsByNewest", func(t *testing.T) {
		tmpDir := t.TempDir()

		fork1Dir := filepath.Join(tmpDir, "fork1")
		require.NoError(t, os.MkdirAll(fork1Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(fork1Dir, "GOAL.md"), []byte("# Old Goal"), 0644))

		fork2Dir := filepath.Join(tmpDir, "fork2")
		require.NoError(t, os.MkdirAll(fork2Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(fork2Dir, "GOAL.md"), []byte("# New Goal"), 0644))

		forks := []workspaceInfo{
			{Directory: fork1Dir, DirName: "fork1"},
			{Directory: fork2Dir, DirName: "fork2"},
		}
		result := readNewestForkGoal(forks)
		assert.NotEmpty(t, result)
	})
}

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectOK     bool
		expectedYAML string
	}{
		{
			name: "validFrontmatter",
			content: `---
key: value
---
content`,
			expectOK:     true,
			expectedYAML: "key: value\n",
		},
		{
			name:     "noFrontmatter",
			content:  "just content",
			expectOK: false,
		},
		{
			name: "unclosedFrontmatter",
			content: `---
key: value
content`,
			expectOK: false,
		},
		{
			name: "emptyFrontmatter",
			content: `---
---
content`,
			expectOK:     true,
			expectedYAML: "",
		},
		{
			name: "multilineFrontmatter",
			content: `---
key1: value1
key2: value2
---
content`,
			expectOK:     true,
			expectedYAML: "key1: value1\nkey2: value2\n",
		},
		{
			name: "noNewlineAfterDelimiter",
			content: `---key: value
---
content`,
			expectOK:     true,
			expectedYAML: "key: value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent, ok := splitFrontmatter([]byte(tt.content))
			assert.Equal(t, tt.expectOK, ok)
			if tt.expectOK {
				assert.Equal(t, tt.expectedYAML, string(yamlContent))
			}
		})
	}
}

func TestIsTodoTool(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		expected bool
	}{
		{
			name:     "todowrite",
			tool:     "todowrite",
			expected: true,
		},
		{
			name:     "todoread",
			tool:     "todoread",
			expected: true,
		},
		{
			name:     "sgaiProjectTodowrite",
			tool:     "sgai_project_todowrite",
			expected: true,
		},
		{
			name:     "sgaiProjectTodoread",
			tool:     "sgai_project_todoread",
			expected: true,
		},
		{
			name:     "otherTool",
			tool:     "bash",
			expected: false,
		},
		{
			name:     "emptyTool",
			tool:     "",
			expected: false,
		},
		{
			name:     "similarTool",
			tool:     "todowrites",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTodoTool(tt.tool)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTodoStatusSymbol(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "pending",
			status:   "pending",
			expected: "○",
		},
		{
			name:     "inProgress",
			status:   "in_progress",
			expected: "◐",
		},
		{
			name:     "completed",
			status:   "completed",
			expected: "●",
		},
		{
			name:     "cancelled",
			status:   "cancelled",
			expected: "✕",
		},
		{
			name:     "unknown",
			status:   "unknown",
			expected: "○",
		},
		{
			name:     "empty",
			status:   "",
			expected: "○",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := todoStatusSymbol(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripMCPTodoPrefix(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "withTodosPrefix",
			output:   "todos\n[{\"content\": \"test\"}]",
			expected: "[{\"content\": \"test\"}]",
		},
		{
			name:     "withTodoPrefix",
			output:   "todo\n[{\"content\": \"test\"}]",
			expected: "[{\"content\": \"test\"}]",
		},
		{
			name:     "withSpacedTodosPrefix",
			output:   "  todos  \n[{\"content\": \"test\"}]",
			expected: "[{\"content\": \"test\"}]",
		},
		{
			name:     "withoutPrefix",
			output:   "[{\"content\": \"test\"}]",
			expected: "[{\"content\": \"test\"}]",
		},
		{
			name:     "emptyOutput",
			output:   "",
			expected: "",
		},
		{
			name:     "noNewline",
			output:   "todos",
			expected: "todos",
		},
		{
			name:     "wrongPrefix",
			output:   "other\n[{\"content\": \"test\"}]",
			expected: "other\n[{\"content\": \"test\"}]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMCPTodoPrefix(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatToolCall(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		input    map[string]any
		expected string
	}{
		{
			name:     "emptyInput",
			tool:     "bash",
			input:    map[string]any{},
			expected: "bash",
		},
		{
			name:     "stringInput",
			tool:     "read",
			input:    map[string]any{"filePath": "/path/to/file"},
			expected: "read(filePath: '/path/to/file')",
		},
		{
			name:     "boolInput",
			tool:     "edit",
			input:    map[string]any{"replaceAll": true},
			expected: "edit(replaceAll: true)",
		},
		{
			name:     "floatInput",
			tool:     "tool",
			input:    map[string]any{"count": float64(42)},
			expected: "tool(count: 42)",
		},
		{
			name:     "truncatesLongString",
			tool:     "tool",
			input:    map[string]any{"content": "this is a very long string that should be truncated because it exceeds the limit"},
			expected: "tool(content: 'this is a very long string that should be trunc...')",
		},
		{
			name:     "doesNotTruncateFilePath",
			tool:     "read",
			input:    map[string]any{"filePath": "/this/is/a/very/long/path/that/should/not/be/truncated/at/all"},
			expected: "read(filePath: '/this/is/a/very/long/path/that/should/not/be/truncated/at/all')",
		},
		{
			name:     "escapesNewlines",
			tool:     "tool",
			input:    map[string]any{"text": "line1\nline2"},
			expected: "tool(text: 'line1\\nline2')",
		},
		{
			name:     "escapesTabs",
			tool:     "tool",
			input:    map[string]any{"text": "col1\tcol2"},
			expected: "tool(text: 'col1\\tcol2')",
		},
		{
			name:     "multipleInputs",
			tool:     "tool",
			input:    map[string]any{"a": "val1", "b": true},
			expected: "tool(a: 'val1', b: true)",
		},
		{
			name:     "intInput",
			tool:     "tool",
			input:    map[string]any{"count": 42},
			expected: "tool(count: 42)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolCall(tt.tool, tt.input)
			if tt.name == "multipleInputs" {
				assert.Contains(t, result, "a: 'val1'")
				assert.Contains(t, result, "b: true")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsFalsish(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "no",
			input:    "no",
			expected: true,
		},
		{
			name:     "false",
			input:    "false",
			expected: true,
		},
		{
			name:     "zero",
			input:    "0",
			expected: true,
		},
		{
			name:     "off",
			input:    "off",
			expected: true,
		},
		{
			name:     "yes",
			input:    "yes",
			expected: false,
		},
		{
			name:     "true",
			input:    "true",
			expected: false,
		},
		{
			name:     "one",
			input:    "1",
			expected: false,
		},
		{
			name:     "empty",
			input:    "",
			expected: false,
		},
		{
			name:     "uppercaseFalse",
			input:    "FALSE",
			expected: true,
		},
		{
			name:     "spacedFalse",
			input:    " false ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFalsish(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveBaseAgent(t *testing.T) {
	tests := []struct {
		name      string
		alias     map[string]string
		agentName string
		expected  string
	}{
		{
			name:      "noAlias",
			alias:     nil,
			agentName: "agent1",
			expected:  "agent1",
		},
		{
			name: "hasAlias",
			alias: map[string]string{
				"agent-lite": "agent",
			},
			agentName: "agent-lite",
			expected:  "agent",
		},
		{
			name: "noMatchingAlias",
			alias: map[string]string{
				"other-agent": "agent",
			},
			agentName: "agent1",
			expected:  "agent1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveBaseAgent(tt.alias, tt.agentName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zeroDuration",
			duration: 0,
			expected: "[00:00:00.000]",
		},
		{
			name:     "oneSecond",
			duration: time.Second,
			expected: "[00:00:01.000]",
		},
		{
			name:     "oneMinute",
			duration: time.Minute,
			expected: "[00:01:00.000]",
		},
		{
			name:     "oneHour",
			duration: time.Hour,
			expected: "[01:00:00.000]",
		},
		{
			name:     "mixedDuration",
			duration: time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond,
			expected: "[01:02:03.004]",
		},
		{
			name:     "millisecondsOnly",
			duration: 123 * time.Millisecond,
			expected: "[00:00:00.123]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now().Add(-tt.duration)
			result := formatElapsed(start)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountPendingTodos(t *testing.T) {
	tests := []struct {
		name     string
		wfState  state.Workflow
		agent    string
		expected int
	}{
		{
			name:     "coordinatorReturnsZero",
			wfState:  state.Workflow{},
			agent:    "coordinator",
			expected: 0,
		},
		{
			name: "emptyTodos",
			wfState: state.Workflow{
				Todos: []state.TodoItem{},
			},
			agent:    "agent1",
			expected: 0,
		},
		{
			name: "pendingTodo",
			wfState: state.Workflow{
				Todos: []state.TodoItem{
					{Content: "Task 1", Status: "pending"},
				},
			},
			agent:    "agent1",
			expected: 1,
		},
		{
			name: "inProgressTodo",
			wfState: state.Workflow{
				Todos: []state.TodoItem{
					{Content: "Task 1", Status: "in_progress"},
				},
			},
			agent:    "agent1",
			expected: 1,
		},
		{
			name: "completedTodo",
			wfState: state.Workflow{
				Todos: []state.TodoItem{
					{Content: "Task 1", Status: "completed"},
				},
			},
			agent:    "agent1",
			expected: 0,
		},
		{
			name: "cancelledTodo",
			wfState: state.Workflow{
				Todos: []state.TodoItem{
					{Content: "Task 1", Status: "cancelled"},
				},
			},
			agent:    "agent1",
			expected: 0,
		},
		{
			name: "mixedTodos",
			wfState: state.Workflow{
				Todos: []state.TodoItem{
					{Content: "Task 1", Status: "pending"},
					{Content: "Task 2", Status: "completed"},
					{Content: "Task 3", Status: "in_progress"},
					{Content: "Task 4", Status: "cancelled"},
				},
			},
			agent:    "agent1",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countPendingTodos(tt.wfState, tt.agent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCompletionGateScriptFailureMessage(t *testing.T) {
	script := "make test"
	output := "FAIL: TestSomething\n--- Expected: 1, Actual: 2"

	result := formatCompletionGateScriptFailureMessage(script, output)

	assert.Contains(t, result, "From: environment")
	assert.Contains(t, result, "To: coordinator")
	assert.Contains(t, result, "Subject: computable definition of success has failed")
	assert.Contains(t, result, script)
	assert.Contains(t, result, output)
}

func TestInitVisitCounts(t *testing.T) {
	tests := []struct {
		name     string
		agents   []string
		expected map[string]int
	}{
		{
			name:     "emptyAgents",
			agents:   []string{},
			expected: map[string]int{},
		},
		{
			name:   "singleAgent",
			agents: []string{"agent1"},
			expected: map[string]int{
				"agent1": 0,
			},
		},
		{
			name:   "multipleAgents",
			agents: []string{"agent1", "agent2", "agent3"},
			expected: map[string]int{
				"agent1": 0,
				"agent2": 0,
				"agent3": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := initVisitCounts(tt.agents)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateRetrospectiveDirName(t *testing.T) {
	result := generateRetrospectiveDirName()

	assert.Len(t, result, 21)

	now := time.Now()
	expectedPrefix := now.Format("2006-01-02-15-04")
	assert.True(t, strings.HasPrefix(result, expectedPrefix+"."), "expected prefix %s, got %s", expectedPrefix, result)

	suffix := result[len(result)-4:]
	for _, c := range suffix {
		assert.True(t, (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'), "suffix should be alphanumeric, got %c", c)
	}
}

func TestDotSGAILinePresent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "present",
			content:  "some content\n/.sgai\nmore content",
			expected: true,
		},
		{
			name:     "presentWithSpaces",
			content:  "some content\n  /.sgai  \nmore content",
			expected: true,
		},
		{
			name:     "notPresent",
			content:  "some content\n/.sgai-other\nmore content",
			expected: false,
		},
		{
			name:     "emptyContent",
			content:  "",
			expected: false,
		},
		{
			name:     "partialMatch",
			content:  "/.sg",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dotSGAILinePresent([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONPrettyWriterWrite(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " [test] ",
		w:         &buf,
		startTime: time.Now(),
	}

	event := streamEvent{Type: "text", Part: part{Text: "hello world"}}
	data, err := json.Marshal(event)
	require.NoError(t, err)
	data = append(data, '\n')

	n, errWrite := w.Write(data)
	assert.NoError(t, errWrite)
	assert.Equal(t, len(data), n)

	w.Flush()
	assert.Contains(t, buf.String(), "hello world")
}

func TestJSONPrettyWriterProcessEventText(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{Type: "text", Part: part{Text: "some text"}})
	assert.Equal(t, "some text", w.currentText.String())

	w.Flush()
	assert.Contains(t, buf.String(), "some text")
}

func TestJSONPrettyWriterProcessEventToolPending(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{
		Type: "tool",
		Part: part{
			Tool: "mcp_bash",
			State: &toolState{
				Status: "pending",
				Input:  map[string]any{"command": "ls"},
			},
		},
	})

	output := buf.String()
	assert.Contains(t, output, "mcp_bash")
}

func TestJSONPrettyWriterProcessEventToolRunning(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{
		Type: "tool",
		Part: part{
			Tool: "mcp_read",
			State: &toolState{
				Status: "running",
				Input:  map[string]any{"filePath": "/some/path"},
			},
		},
	})

	output := buf.String()
	assert.Contains(t, output, "mcp_read")
	assert.Contains(t, output, "...")
}

func TestJSONPrettyWriterProcessEventToolCompleted(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{
		Type: "tool",
		Part: part{
			Tool: "mcp_bash",
			State: &toolState{
				Status: "completed",
				Input:  map[string]any{"command": "echo hello"},
				Output: "hello\nworld",
			},
		},
	})

	output := buf.String()
	assert.Contains(t, output, "mcp_bash")
	assert.Contains(t, output, "→")
}

func TestJSONPrettyWriterProcessEventToolCompletedTodo(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	todos := `[{"content":"task1","status":"completed","priority":"high"},{"content":"task2","status":"pending","priority":"medium"}]`
	w.processEvent(streamEvent{
		Type: "tool",
		Part: part{
			Tool: "todowrite",
			State: &toolState{
				Status: "completed",
				Input:  map[string]any{},
				Output: todos,
			},
		},
	})

	output := buf.String()
	assert.Contains(t, output, "●")
	assert.Contains(t, output, "task1")
	assert.Contains(t, output, "○")
	assert.Contains(t, output, "task2")
}

func TestJSONPrettyWriterProcessEventToolError(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{
		Type: "tool",
		Part: part{
			Tool: "mcp_bash",
			State: &toolState{
				Status: "error",
				Input:  map[string]any{},
				Error:  "permission denied",
			},
		},
	})

	output := buf.String()
	assert.Contains(t, output, "ERROR:")
	assert.Contains(t, output, "permission denied")
}

func TestJSONPrettyWriterProcessEventStepStart(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.currentText.WriteString("buffered text")
	w.processEvent(streamEvent{Type: "step_start"})

	assert.Equal(t, 1, w.stepCounter)
	assert.Contains(t, buf.String(), "buffered text")
}

func TestJSONPrettyWriterProcessEventReasoning(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{Type: "reasoning", Part: part{Text: "thinking about it"}})
	assert.Contains(t, buf.String(), "[thinking]")
}

func TestJSONPrettyWriterProcessEventUnknownType(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{Type: "custom_event"})
	assert.Contains(t, buf.String(), "[custom_event]")
}

func TestJSONPrettyWriterSessionIDCapture(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{Type: "text", SessionID: "sess-123", Part: part{Text: "hi"}})
	assert.Equal(t, "sess-123", w.sessionID)
}

func TestJSONPrettyWriterFlushEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.Flush()
	assert.Empty(t, buf.String())
}

func TestJSONPrettyWriterProcessBufferMultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	event1, _ := json.Marshal(streamEvent{Type: "text", Part: part{Text: "hello"}})
	event2, _ := json.Marshal(streamEvent{Type: "text", Part: part{Text: " world"}})
	data := string(event1) + "\n" + string(event2) + "\n"

	_, _ = w.Write([]byte(data))
	w.Flush()

	assert.Contains(t, buf.String(), "hello world")
}

func TestJSONPrettyWriterProcessBufferPartialLine(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	event, _ := json.Marshal(streamEvent{Type: "text", Part: part{Text: "partial"}})
	data := string(event)

	_, _ = w.Write([]byte(data[:10]))
	assert.Empty(t, buf.String())

	_, _ = w.Write([]byte(data[10:] + "\n"))
	w.Flush()
	assert.Contains(t, buf.String(), "partial")
}

func TestJSONPrettyWriterRecordStepCost(t *testing.T) {
	dir := t.TempDir()
	coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
	require.NoError(t, err)

	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:       " ",
		w:            &buf,
		startTime:    time.Now(),
		coord:        coord,
		currentAgent: "test-agent",
		stepCounter:  1,
	}

	w.recordStepCost(part{
		Cost: 0.05,
		Tokens: partTokens{
			Input:  100,
			Output: 50,
		},
	}, time.Now().UnixMilli())

	wfState := coord.State()
	assert.InDelta(t, 0.05, wfState.Cost.TotalCost, 0.001)
	assert.Equal(t, 100, wfState.Cost.TotalTokens.Input)
	assert.Equal(t, 50, wfState.Cost.TotalTokens.Output)
	assert.Len(t, wfState.Cost.ByAgent, 1)
	assert.Equal(t, "test-agent", wfState.Cost.ByAgent[0].Agent)
}

func TestJSONPrettyWriterRecordStepCostMultipleSteps(t *testing.T) {
	dir := t.TempDir()
	coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
	require.NoError(t, err)

	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:       " ",
		w:            &buf,
		startTime:    time.Now(),
		coord:        coord,
		currentAgent: "test-agent",
		stepCounter:  1,
	}

	w.recordStepCost(part{Cost: 0.01, Tokens: partTokens{Input: 10, Output: 5}}, time.Now().UnixMilli())
	w.stepCounter++
	w.recordStepCost(part{Cost: 0.02, Tokens: partTokens{Input: 20, Output: 10}}, time.Now().UnixMilli())

	wfState := coord.State()
	assert.InDelta(t, 0.03, wfState.Cost.TotalCost, 0.001)
	assert.Len(t, wfState.Cost.ByAgent[0].Steps, 2)
}

func TestJSONPrettyWriterRecordStepCostNilCoord(_ *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:       " ",
		w:            &buf,
		startTime:    time.Now(),
		coord:        nil,
		currentAgent: "test-agent",
	}

	w.recordStepCost(part{Cost: 0.05}, time.Now().UnixMilli())
}

func TestJSONPrettyWriterRecordStepCostEmptyAgent(t *testing.T) {
	dir := t.TempDir()
	coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
	require.NoError(t, err)

	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:       " ",
		w:            &buf,
		startTime:    time.Now(),
		coord:        coord,
		currentAgent: "",
	}

	w.recordStepCost(part{Cost: 0.05}, time.Now().UnixMilli())
	wfState := coord.State()
	assert.InDelta(t, 0.0, wfState.Cost.TotalCost, 0.001)
}

func TestJSONPrettyWriterRecordStepCostZeroValues(t *testing.T) {
	dir := t.TempDir()
	coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
	require.NoError(t, err)

	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:       " ",
		w:            &buf,
		startTime:    time.Now(),
		coord:        coord,
		currentAgent: "agent",
	}

	w.recordStepCost(part{Cost: 0, Tokens: partTokens{}}, time.Now().UnixMilli())
	wfState := coord.State()
	assert.InDelta(t, 0.0, wfState.Cost.TotalCost, 0.001)
}

func TestJSONPrettyWriterFormatTodoOutput(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	todos := `[{"content":"write tests","status":"in_progress","priority":"high"},{"content":"fix bug","status":"completed","priority":"medium"},{"content":"deploy","status":"cancelled","priority":"low"}]`
	w.formatTodoOutput(todos)

	output := buf.String()
	assert.Contains(t, output, "◐")
	assert.Contains(t, output, "write tests")
	assert.Contains(t, output, "●")
	assert.Contains(t, output, "fix bug")
	assert.Contains(t, output, "✕")
	assert.Contains(t, output, "deploy")
}

func TestJSONPrettyWriterFormatTodoOutputInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.formatTodoOutput("not json at all")

	output := buf.String()
	assert.Contains(t, output, "→")
	assert.Contains(t, output, "not json at all")
}

func TestJSONPrettyWriterFormatTodoOutputWithPrefix(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	todos := "Updated todos\n" + `[{"content":"task","status":"pending","priority":"high"}]`
	w.formatTodoOutput(todos)

	output := buf.String()
	assert.Contains(t, output, "○")
	assert.Contains(t, output, "task")
}

func TestJSONPrettyWriterProcessEventStepFinish(t *testing.T) {
	dir := t.TempDir()
	coord, err := state.NewCoordinatorWith(filepath.Join(dir, "state.json"), state.Workflow{})
	require.NoError(t, err)

	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:       " ",
		w:            &buf,
		startTime:    time.Now(),
		coord:        coord,
		currentAgent: "test-agent",
		stepCounter:  1,
	}

	w.currentText.WriteString("some text")
	w.processEvent(streamEvent{
		Type:      "step_finish",
		Timestamp: time.Now().UnixMilli(),
		Part: part{
			Cost:   0.1,
			Tokens: partTokens{Input: 500, Output: 200},
		},
	})

	assert.Contains(t, buf.String(), "some text")
	wfState := coord.State()
	assert.InDelta(t, 0.1, wfState.Cost.TotalCost, 0.01)
}

func TestJSONPrettyWriterToolNilState(t *testing.T) {
	var buf bytes.Buffer
	w := &jsonPrettyWriter{
		prefix:    " ",
		w:         &buf,
		startTime: time.Now(),
	}

	w.processEvent(streamEvent{
		Type: "tool",
		Part: part{
			Tool:  "mcp_bash",
			State: nil,
		},
	})

	assert.Empty(t, buf.String())
}

func TestPrefixWriter(t *testing.T) {
	var buf bytes.Buffer
	w := &prefixWriter{
		prefix:    " [test] ",
		w:         &buf,
		startTime: time.Now(),
	}

	n, err := w.Write([]byte("hello\nworld\n"))
	assert.NoError(t, err)
	assert.Equal(t, 12, n)

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	assert.Len(t, lines, 2)
	for _, line := range lines {
		assert.Contains(t, line, "[test]")
	}
	assert.Contains(t, output, "hello")
	assert.Contains(t, output, "world")
}

func TestPrefixWriterSingleLine(t *testing.T) {
	var buf bytes.Buffer
	w := &prefixWriter{
		prefix:    " [p] ",
		w:         &buf,
		startTime: time.Now(),
	}

	_, _ = w.Write([]byte("single line\n"))
	assert.Contains(t, buf.String(), "single line")
}

func TestCopyCompletionArtifactsToRetrospectiveNoDir(_ *testing.T) {
	cfg := multiModelConfig{retrospectiveDir: ""}
	copyCompletionArtifactsToRetrospective(cfg)
}

func TestInitializeWorkspaceDirExisting(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0755))
	err := initializeWorkspaceDir(dir)
	assert.NoError(t, err)
}

func TestCopyFileAtomicSuccess(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "source.txt")
	dstFile := filepath.Join(dstDir, "dest.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("hello world"), 0644))
	err := copyFileAtomic(srcFile, dstFile)
	require.NoError(t, err)
	data, errRead := os.ReadFile(dstFile)
	require.NoError(t, errRead)
	assert.Equal(t, "hello world", string(data))
}

func TestCopyFileAtomicMissingSource(t *testing.T) {
	err := copyFileAtomic("/nonexistent/source.txt", filepath.Join(t.TempDir(), "dest.txt"))
	assert.Error(t, err)
}

func TestCopyFileAtomicCreatesDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "source.txt")
	dstFile := filepath.Join(dstDir, "subdir", "dest.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("nested"), 0644))
	err := copyFileAtomic(srcFile, dstFile)
	require.NoError(t, err)
	data, errRead := os.ReadFile(dstFile)
	require.NoError(t, errRead)
	assert.Equal(t, "nested", string(data))
}

func TestCopyFinalStateToRetrospectiveNoFilesNewBatch(t *testing.T) {
	dir := t.TempDir()
	retroDir := t.TempDir()
	err := copyFinalStateToRetrospective(dir, retroDir)
	assert.NoError(t, err)
}

func TestRunCompletionGateScriptSuccess(t *testing.T) {
	dir := t.TempDir()
	output, err := runCompletionGateScript(context.Background(), dir, "echo 'hello'")
	require.NoError(t, err)
	assert.Contains(t, output, "hello")
}

func TestRunCompletionGateScriptFailure(t *testing.T) {
	dir := t.TempDir()
	_, err := runCompletionGateScript(context.Background(), dir, "exit 1")
	assert.Error(t, err)
}

func TestSaveStateCoord(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	stateFile := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	wf := coord.State()
	saveState(coord, wf)
	assert.FileExists(t, stateFile)
}

func TestComputeGoalChecksumSuccess(t *testing.T) {
	dir := t.TempDir()
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# My Goal"), 0644))
	checksum, err := computeGoalChecksum(goalPath)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 64)
}

func TestComputeGoalChecksumMissing(t *testing.T) {
	_, err := computeGoalChecksum("/nonexistent/GOAL.md")
	assert.Error(t, err)
}

func TestOpenRetrospectiveLogsSuccess(t *testing.T) {
	dir := t.TempDir()
	stdoutLog, stderrLog, err := openRetrospectiveLogs(dir)
	require.NoError(t, err)
	require.NotNil(t, stdoutLog)
	require.NotNil(t, stderrLog)
	t.Cleanup(func() {
		_ = stdoutLog.Close()
		_ = stderrLog.Close()
	})
	assert.FileExists(t, filepath.Join(dir, "stdout.log"))
	assert.FileExists(t, filepath.Join(dir, "stderr.log"))
}

func TestOpenRetrospectiveLogsInvalidDir(t *testing.T) {
	_, _, err := openRetrospectiveLogs("/nonexistent/path/to/retro")
	assert.Error(t, err)
}

func TestOpenRetrospectiveLogsWritable(t *testing.T) {
	dir := t.TempDir()
	stdoutLog, stderrLog, err := openRetrospectiveLogs(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = stdoutLog.Close()
		_ = stderrLog.Close()
	})
	_, errWrite := stdoutLog.Write([]byte("stdout test\n"))
	assert.NoError(t, errWrite)
	_, errWrite2 := stderrLog.Write([]byte("stderr test\n"))
	assert.NoError(t, errWrite2)
}

func TestHandleCompleteStatusCoordinatorNoBlockers(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{})
	require.NoError(t, errCoord)

	d := &dag{Nodes: map[string]*dagNode{}}
	cfg := multiModelConfig{paddedsgai: "test", coord: coord, dir: dir, agent: "coordinator", flowDag: d, goalPath: filepath.Join(dir, "GOAL.md")}
	require.NoError(t, os.WriteFile(cfg.goalPath, []byte("# Goal"), 0644))
	result := handleCompleteStatus(context.Background(), cfg, state.Workflow{Status: state.StatusComplete}, state.Workflow{}, GoalMetadata{})
	assert.Equal(t, state.StatusComplete, result.Status)
}

func TestTerminateProcessGroupOnCancelWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.Command("sleep", "30")
	require.NoError(t, cmd.Start())

	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()

	cancel()
	terminateProcessGroupOnCancel(ctx, cmd, exited)
	<-exited
}

func TestExportSessionMissingBinary(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0755))
	err := exportSession(dir, "session-1", filepath.Join(dir, "output.json"))
	assert.Error(t, err)
}

func TestRunCompletionGateScriptCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runCompletionGateScript(ctx, t.TempDir(), "echo hello")
	assert.Error(t, err)
}

func TestFormatCompletionGateScriptFailureMessageContent(t *testing.T) {
	msg := formatCompletionGateScriptFailureMessage("make test", "FAIL: tests failed")
	assert.Contains(t, msg, "make test")
	assert.Contains(t, msg, "FAIL: tests failed")
}

func TestParseAgentFileMetadataValidFile(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	agentFile := filepath.Join(agentDir, "test-agent.md")
	content := "---\nlog: true\nsnippets:\n  - go\n---\n# Test Agent\nAgent instructions here"
	require.NoError(t, os.WriteFile(agentFile, []byte(content), 0644))

	meta, ok := parseAgentFileMetadata(dir, "test-agent")
	assert.True(t, ok)
	assert.True(t, meta.Log)
	assert.Contains(t, meta.Snippets, "go")
}

func TestParseAgentFileMetadataMissing(t *testing.T) {
	_, ok := parseAgentFileMetadata(t.TempDir(), "nonexistent")
	assert.False(t, ok)
}

func TestParseAgentFileMetadataNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "test.md"), []byte("no frontmatter"), 0644))

	_, ok := parseAgentFileMetadata(dir, "test")
	assert.False(t, ok)
}

func TestShouldLogAgentDefault(t *testing.T) {
	result := shouldLogAgent(t.TempDir(), "nonexistent")
	assert.True(t, result)
}

func TestShouldLogAgentExplicit(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "quiet.md"), []byte("---\nlog: false\n---\n"), 0644))

	result := shouldLogAgent(dir, "quiet")
	assert.False(t, result)
}

func TestParseAgentSnippetsEmpty(t *testing.T) {
	result := parseAgentSnippets(t.TempDir(), "nonexistent")
	assert.Nil(t, result)
}

func TestParseAgentSnippetsPopulated(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "dev.md"), []byte("---\nsnippets:\n  - go\n  - react\n---\n"), 0644))

	result := parseAgentSnippets(dir, "dev")
	assert.Equal(t, []string{"go", "react"}, result)
}

func TestBlockCompletionOnGateScriptPassingScript(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, ".sgai", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(sp), 0755))
	coord := state.NewCoordinatorEmpty(sp)

	cfg := multiModelConfig{
		paddedsgai: "sgai",
		agent:      "test",
		dir:        dir,
		coord:      coord,
	}
	metadata := GoalMetadata{CompletionGateScript: "true"}
	wfState := state.Workflow{Status: state.StatusComplete}
	result := blockCompletionOnGateScript(context.Background(), cfg, wfState, metadata)
	assert.Nil(t, result)
}

func TestBlockCompletionOnGateScriptFailingScript(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, ".sgai", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(sp), 0755))
	coord := state.NewCoordinatorEmpty(sp)

	cfg := multiModelConfig{
		paddedsgai: "sgai",
		agent:      "test",
		dir:        dir,
		coord:      coord,
	}
	metadata := GoalMetadata{CompletionGateScript: "false"}
	wfState := state.Workflow{Status: state.StatusComplete}
	result := blockCompletionOnGateScript(context.Background(), cfg, wfState, metadata)
	require.NotNil(t, result)
	assert.Equal(t, state.StatusWorking, result.Status)
}

func TestCopyCompletionArtifactsWithPM(t *testing.T) {
	dir := t.TempDir()
	retroDir := t.TempDir()
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("# Goal"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0644))

	cfg := multiModelConfig{
		dir:              dir,
		goalPath:         goalPath,
		retrospectiveDir: retroDir,
	}
	copyCompletionArtifactsToRetrospective(cfg)

	_, errGoal := os.Stat(filepath.Join(retroDir, "GOAL.md"))
	assert.NoError(t, errGoal)
	_, errPM := os.Stat(filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
	assert.NoError(t, errPM)
}

func TestHandleWaitingForHumanStatusWithMessage(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, ".sgai", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(sp), 0755))
	coord := state.NewCoordinatorEmpty(sp)

	cfg := multiModelConfig{
		coord:      coord,
		agent:      "test",
		paddedsgai: "sgai",
	}
	wfState := state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "question for user",
	}
	result := handleWaitingForHumanStatus(cfg, wfState)
	assert.Equal(t, state.StatusWorking, result.Status)
}

func TestHandleWorkingLoopReset(t *testing.T) {
	cfg := multiModelConfig{
		paddedsgai: "sgai",
		agent:      "test",
	}
	sessionID := "session-1"
	result := handleWorkingLoop(cfg, &sessionID, maxConsecutiveWorkingIterations-1)
	assert.Equal(t, 0, result)
	assert.Empty(t, sessionID)
}

func TestSaveStatePersistence(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, ".sgai", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(sp), 0755))
	coord := state.NewCoordinatorEmpty(sp)

	wf := state.Workflow{Status: state.StatusWorking, Task: "testing"}
	saveState(coord, wf)

	loaded := coord.State()
	assert.Equal(t, state.StatusWorking, loaded.Status)
	assert.Equal(t, "testing", loaded.Task)
}

func TestCopyFileAtomicMissingSrc(t *testing.T) {
	err := copyFileAtomic("/nonexistent/source.txt", filepath.Join(t.TempDir(), "dest.txt"))
	assert.Error(t, err)
}

func TestCopyFinalStateToRetrospectiveNoFiles(t *testing.T) {
	dir := t.TempDir()
	retroDir := filepath.Join(t.TempDir(), "retro")
	err := copyFinalStateToRetrospective(dir, retroDir)
	require.NoError(t, err)
}

func TestApplyLayerFolderOverlayNoLayerDir(t *testing.T) {
	dir := t.TempDir()
	err := applyLayerFolderOverlay(dir)
	assert.NoError(t, err)
}

func TestApplyLayerFolderOverlayWithFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "skills"), 0755))
	layerDir := filepath.Join(dir, "sgai", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(layerDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(layerDir, "SKILL.md"), []byte("# My Skill"), 0644))
	err := applyLayerFolderOverlay(dir)
	assert.NoError(t, err)
	content, errRead := os.ReadFile(filepath.Join(dir, ".sgai", "skills", "my-skill", "SKILL.md"))
	require.NoError(t, errRead)
	assert.Equal(t, "# My Skill", string(content))
}

func TestApplyLayerFolderOverlayProtectedCoordinator(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "agent"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sgai", "agent", "coordinator.md"), []byte("# Original"), 0644))
	layerDir := filepath.Join(dir, "sgai", "agent")
	require.NoError(t, os.MkdirAll(layerDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(layerDir, "coordinator.md"), []byte("# Overlay"), 0644))
	err := applyLayerFolderOverlay(dir)
	assert.NoError(t, err)
	content, errRead := os.ReadFile(filepath.Join(dir, ".sgai", "agent", "coordinator.md"))
	require.NoError(t, errRead)
	assert.Equal(t, "# Original", string(content))
}

func TestCopyLayerSubfolderWithProtected(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src", "agent")
	dstDir := filepath.Join(dir, "dst", "agent")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.MkdirAll(dstDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "coordinator.md"), []byte("# Override"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "builder.md"), []byte("# Builder"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dstDir, "coordinator.md"), []byte("# Original"), 0644))
	err := copyLayerSubfolder(srcDir, dstDir, "agent")
	require.NoError(t, err)
	coordContent, _ := os.ReadFile(filepath.Join(dstDir, "coordinator.md"))
	assert.Equal(t, "# Original", string(coordContent))
	builderContent, _ := os.ReadFile(filepath.Join(dstDir, "builder.md"))
	assert.Equal(t, "# Builder", string(builderContent))
}

func TestInitializeJJForkWorkspace(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".jj"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".jj", "repo"), []byte("/some/other/path"), 0644))
	err := initializeJJ(dir)
	assert.NoError(t, err)
}

func TestComputeGoalChecksumDifferentBody(t *testing.T) {
	dir := t.TempDir()
	goalPath := filepath.Join(dir, "GOAL.md")
	require.NoError(t, os.WriteFile(goalPath, []byte("---\ntitle: test\n---\n# Content A"), 0644))
	hash1, err1 := computeGoalChecksum(goalPath)
	require.NoError(t, err1)
	require.NoError(t, os.WriteFile(goalPath, []byte("---\ntitle: test\n---\n# Content B"), 0644))
	hash2, err2 := computeGoalChecksum(goalPath)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2)
}

func TestExtractBodyWithFrontmatter(t *testing.T) {
	content := []byte("---\ntitle: test\n---\n# Body content")
	result := extractBody(content)
	assert.Equal(t, "# Body content", string(result))
}

func TestExtractBodyUnclosedFrontmatter(t *testing.T) {
	content := []byte("---\ntitle: test\n# Body content")
	result := extractBody(content)
	assert.Equal(t, content, result)
}

func TestDotSGAILinePresentVariants(t *testing.T) {
	assert.True(t, dotSGAILinePresent([]byte("/.sgai\n")))
	assert.True(t, dotSGAILinePresent([]byte("other\n/.sgai\nmore")))
	assert.False(t, dotSGAILinePresent([]byte("other\n")))
	assert.False(t, dotSGAILinePresent(nil))
}

func TestCopyFileAtomicCreatesDstDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "subdir", "nested", "dst.txt")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0644))
	require.NoError(t, copyFileAtomic(src, dst))
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}

func TestCopyFinalStateToRetrospectiveSuccess(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"complete"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0644))

	retroDir := filepath.Join(t.TempDir(), "retro")
	require.NoError(t, os.MkdirAll(retroDir, 0755))

	err := copyFinalStateToRetrospective(dir, retroDir)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(retroDir, "state.json"))
	assert.FileExists(t, filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
}

func TestCopyProjectManagementToRetrospectiveWithPM(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("# PM content"), 0644))
	retroDir := t.TempDir()
	copyProjectManagementToRetrospective(dir, retroDir)
	assert.FileExists(t, filepath.Join(retroDir, "PROJECT_MANAGEMENT.md"))
}
