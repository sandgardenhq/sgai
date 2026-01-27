package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
)

func TestGetModelsForAgent(t *testing.T) {
	cases := []struct {
		name   string
		models map[string]any
		agent  string
		want   []string
	}{
		{
			name:   "singleString",
			models: map[string]any{"agent1": "model-a"},
			agent:  "agent1",
			want:   []string{"model-a"},
		},
		{
			name:   "emptyString",
			models: map[string]any{"agent1": ""},
			agent:  "agent1",
			want:   nil,
		},
		{
			name:   "stringArray",
			models: map[string]any{"agent1": []any{"model-a", "model-b"}},
			agent:  "agent1",
			want:   []string{"model-a", "model-b"},
		},
		{
			name:   "emptyArray",
			models: map[string]any{"agent1": []any{}},
			agent:  "agent1",
			want:   []string{},
		},
		{
			name:   "agentNotFound",
			models: map[string]any{"agent1": "model-a"},
			agent:  "agent2",
			want:   nil,
		},
		{
			name:   "nilModels",
			models: nil,
			agent:  "agent1",
			want:   nil,
		},
		{
			name:   "arrayWithEmptyStrings",
			models: map[string]any{"agent1": []any{"model-a", "", "model-b"}},
			agent:  "agent1",
			want:   []string{"model-a", "model-b"},
		},
		{
			name:   "mixedArrayTypes",
			models: map[string]any{"agent1": []any{"model-a", 123, "model-b"}},
			agent:  "agent1",
			want:   []string{"model-a", "model-b"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := getModelsForAgent(tc.models, tc.agent)
			if !slices.Equal(got, tc.want) {
				t.Errorf("getModelsForAgent() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestFormatModelID(t *testing.T) {
	cases := []struct {
		name      string
		agent     string
		modelSpec string
		want      string
	}{
		{
			name:      "simpleModel",
			agent:     "backend-go-developer",
			modelSpec: "anthropic/claude-opus-4-5",
			want:      "backend-go-developer:anthropic/claude-opus-4-5",
		},
		{
			name:      "modelWithVariant",
			agent:     "backend-go-developer",
			modelSpec: "anthropic/claude-opus-4-5 (max)",
			want:      "backend-go-developer:anthropic/claude-opus-4-5 (max)",
		},
		{
			name:      "coordinator",
			agent:     "coordinator",
			modelSpec: "anthropic/claude-sonnet-4-5",
			want:      "coordinator:anthropic/claude-sonnet-4-5",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatModelID(tc.agent, tc.modelSpec)
			if got != tc.want {
				t.Errorf("formatModelID(%q, %q) = %q; want %q", tc.agent, tc.modelSpec, got, tc.want)
			}
		})
	}
}

func TestExtractAgentFromModelID(t *testing.T) {
	cases := []struct {
		name    string
		modelID string
		want    string
	}{
		{
			name:    "standardModelID",
			modelID: "backend-go-developer:anthropic/claude-opus-4-5",
			want:    "backend-go-developer",
		},
		{
			name:    "modelIDWithVariant",
			modelID: "backend-go-developer:anthropic/claude-opus-4-5 (max)",
			want:    "backend-go-developer",
		},
		{
			name:    "noColon",
			modelID: "backend-go-developer",
			want:    "backend-go-developer",
		},
		{
			name:    "emptyString",
			modelID: "",
			want:    "",
		},
		{
			name:    "colonAtStart",
			modelID: ":model-spec",
			want:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractAgentFromModelID(tc.modelID)
			if got != tc.want {
				t.Errorf("extractAgentFromModelID(%q) = %q; want %q", tc.modelID, got, tc.want)
			}
		})
	}
}

func TestSelectModelForAgent(t *testing.T) {
	cases := []struct {
		name   string
		models map[string]any
		agent  string
		want   string
	}{
		{
			name:   "singleModel",
			models: map[string]any{"agent1": "model-a"},
			agent:  "agent1",
			want:   "model-a",
		},
		{
			name:   "multipleModelsReturnsFirst",
			models: map[string]any{"agent1": []any{"model-a", "model-b"}},
			agent:  "agent1",
			want:   "model-a",
		},
		{
			name:   "agentNotFound",
			models: map[string]any{"agent1": "model-a"},
			agent:  "agent2",
			want:   "",
		},
		{
			name:   "emptyModels",
			models: map[string]any{},
			agent:  "agent1",
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := selectModelForAgent(tc.models, tc.agent)
			if got != tc.want {
				t.Errorf("selectModelForAgent(%v, %q) = %q; want %q", tc.models, tc.agent, got, tc.want)
			}
		})
	}
}

func TestAllModelsDone(t *testing.T) {
	cases := []struct {
		name          string
		modelStatuses map[string]string
		want          bool
	}{
		{
			name:          "nilMap",
			modelStatuses: nil,
			want:          true,
		},
		{
			name:          "emptyMap",
			modelStatuses: map[string]string{},
			want:          true,
		},
		{
			name: "allDone",
			modelStatuses: map[string]string{
				"agent1:model-a": "model-done",
				"agent1:model-b": "model-done",
			},
			want: true,
		},
		{
			name: "mixedDoneAndError",
			modelStatuses: map[string]string{
				"agent1:model-a": "model-done",
				"agent1:model-b": "model-error",
			},
			want: true,
		},
		{
			name: "oneWorking",
			modelStatuses: map[string]string{
				"agent1:model-a": "model-done",
				"agent1:model-b": "model-working",
			},
			want: false,
		},
		{
			name: "allWorking",
			modelStatuses: map[string]string{
				"agent1:model-a": "model-working",
				"agent1:model-b": "model-working",
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := allModelsDone(tc.modelStatuses)
			if got != tc.want {
				t.Errorf("allModelsDone(%v) = %v; want %v", tc.modelStatuses, got, tc.want)
			}
		})
	}
}

func TestHasMessagesForModel(t *testing.T) {
	cases := []struct {
		name     string
		messages []state.Message
		modelID  string
		want     bool
	}{
		{
			name:     "noMessages",
			messages: nil,
			modelID:  "agent1:model-a",
			want:     false,
		},
		{
			name: "messageToModelID",
			messages: []state.Message{
				{ToAgent: "agent1:model-a", Read: false},
			},
			modelID: "agent1:model-a",
			want:    true,
		},
		{
			name: "messageToAgentName",
			messages: []state.Message{
				{ToAgent: "agent1", Read: false},
			},
			modelID: "agent1:model-a",
			want:    true,
		},
		{
			name: "readMessageIgnored",
			messages: []state.Message{
				{ToAgent: "agent1:model-a", Read: true},
			},
			modelID: "agent1:model-a",
			want:    false,
		},
		{
			name: "messageToOtherAgent",
			messages: []state.Message{
				{ToAgent: "agent2:model-a", Read: false},
			},
			modelID: "agent1:model-a",
			want:    false,
		},
		{
			name: "mixedMessages",
			messages: []state.Message{
				{ToAgent: "agent2:model-a", Read: false},
				{ToAgent: "agent1:model-a", Read: true},
				{ToAgent: "agent1", Read: false},
			},
			modelID: "agent1:model-a",
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasMessagesForModel(tc.messages, tc.modelID)
			if got != tc.want {
				t.Errorf("hasMessagesForModel() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestHasPendingMessagesForAnyModel(t *testing.T) {
	cases := []struct {
		name     string
		messages []state.Message
		models   []string
		agent    string
		want     bool
	}{
		{
			name:     "noMessages",
			messages: nil,
			models:   []string{"model-a", "model-b"},
			agent:    "agent1",
			want:     false,
		},
		{
			name: "messageForFirstModel",
			messages: []state.Message{
				{ToAgent: "agent1:model-a", Read: false},
			},
			models: []string{"model-a", "model-b"},
			agent:  "agent1",
			want:   true,
		},
		{
			name: "messageForSecondModel",
			messages: []state.Message{
				{ToAgent: "agent1:model-b", Read: false},
			},
			models: []string{"model-a", "model-b"},
			agent:  "agent1",
			want:   true,
		},
		{
			name: "allMessagesRead",
			messages: []state.Message{
				{ToAgent: "agent1:model-a", Read: true},
				{ToAgent: "agent1:model-b", Read: true},
			},
			models: []string{"model-a", "model-b"},
			agent:  "agent1",
			want:   false,
		},
		{
			name: "messageForDifferentAgent",
			messages: []state.Message{
				{ToAgent: "agent2:model-a", Read: false},
			},
			models: []string{"model-a", "model-b"},
			agent:  "agent1",
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasPendingMessagesForAnyModel(tc.messages, tc.models, tc.agent)
			if got != tc.want {
				t.Errorf("hasPendingMessagesForAnyModel() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestSyncModelStatuses(t *testing.T) {
	cases := []struct {
		name          string
		modelStatuses map[string]string
		models        []string
		agent         string
		want          map[string]string
	}{
		{
			name:          "nilStatuses",
			modelStatuses: nil,
			models:        []string{"model-a", "model-b"},
			agent:         "agent1",
			want: map[string]string{
				"agent1:model-a": "model-working",
				"agent1:model-b": "model-working",
			},
		},
		{
			name:          "emptyStatuses",
			modelStatuses: map[string]string{},
			models:        []string{"model-a", "model-b"},
			agent:         "agent1",
			want: map[string]string{
				"agent1:model-a": "model-working",
				"agent1:model-b": "model-working",
			},
		},
		{
			name: "preserveExisting",
			modelStatuses: map[string]string{
				"agent1:model-a": "model-done",
			},
			models: []string{"model-a", "model-b"},
			agent:  "agent1",
			want: map[string]string{
				"agent1:model-a": "model-done",
				"agent1:model-b": "model-working",
			},
		},
		{
			name: "removeOrphaned",
			modelStatuses: map[string]string{
				"agent1:model-a": "model-done",
				"agent1:model-c": "model-working",
			},
			models: []string{"model-a", "model-b"},
			agent:  "agent1",
			want: map[string]string{
				"agent1:model-a": "model-done",
				"agent1:model-b": "model-working",
			},
		},
		{
			name: "preserveOtherAgentStatuses",
			modelStatuses: map[string]string{
				"agent2:model-x": "model-done",
			},
			models: []string{"model-a"},
			agent:  "agent1",
			want: map[string]string{
				"agent2:model-x": "model-done",
				"agent1:model-a": "model-working",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := syncModelStatuses(tc.modelStatuses, tc.models, tc.agent)
			if len(got) != len(tc.want) {
				t.Errorf("syncModelStatuses() returned %d entries; want %d", len(got), len(tc.want))
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("syncModelStatuses()[%q] = %q; want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestCleanupModelStatuses(t *testing.T) {
	wfState := &state.Workflow{
		ModelStatuses: map[string]string{
			"agent1:model-a": "model-done",
			"agent1:model-b": "model-done",
		},
		CurrentModel: "agent1:model-a",
	}

	cleanupModelStatuses(wfState)

	if wfState.ModelStatuses != nil {
		t.Errorf("cleanupModelStatuses() did not set ModelStatuses to nil")
	}
	if wfState.CurrentModel != "" {
		t.Errorf("cleanupModelStatuses() did not clear CurrentModel")
	}
}

func TestExtractAgentNameFromTarget(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   string
	}{
		{
			name:   "agentNameOnly",
			target: "backend-go-developer",
			want:   "backend-go-developer",
		},
		{
			name:   "modelID",
			target: "backend-go-developer:anthropic/claude-opus-4-5",
			want:   "backend-go-developer",
		},
		{
			name:   "modelIDWithVariant",
			target: "backend-go-developer:anthropic/claude-opus-4-5 (max)",
			want:   "backend-go-developer",
		},
		{
			name:   "emptyString",
			target: "",
			want:   "",
		},
		{
			name:   "colonAtStart",
			target: ":model-spec",
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractAgentNameFromTarget(tc.target)
			if got != tc.want {
				t.Errorf("extractAgentNameFromTarget(%q) = %q; want %q", tc.target, got, tc.want)
			}
		})
	}
}

func TestMessageMatchesRecipient(t *testing.T) {
	cases := []struct {
		name         string
		msg          state.Message
		currentAgent string
		currentModel string
		want         bool
	}{
		{
			name:         "matchesAgentName",
			msg:          state.Message{ToAgent: "backend-go-developer"},
			currentAgent: "backend-go-developer",
			currentModel: "",
			want:         true,
		},
		{
			name:         "matchesModelID",
			msg:          state.Message{ToAgent: "backend-go-developer:anthropic/claude-opus-4-5"},
			currentAgent: "backend-go-developer",
			currentModel: "backend-go-developer:anthropic/claude-opus-4-5",
			want:         true,
		},
		{
			name:         "matchesAgentWhenModelSet",
			msg:          state.Message{ToAgent: "backend-go-developer"},
			currentAgent: "backend-go-developer",
			currentModel: "backend-go-developer:anthropic/claude-opus-4-5",
			want:         true,
		},
		{
			name:         "noMatchDifferentAgent",
			msg:          state.Message{ToAgent: "coordinator"},
			currentAgent: "backend-go-developer",
			currentModel: "",
			want:         false,
		},
		{
			name:         "noMatchDifferentModel",
			msg:          state.Message{ToAgent: "backend-go-developer:anthropic/claude-sonnet-4-5"},
			currentAgent: "backend-go-developer",
			currentModel: "backend-go-developer:anthropic/claude-opus-4-5",
			want:         false,
		},
		{
			name:         "modelMessageWithoutCurrentModel",
			msg:          state.Message{ToAgent: "backend-go-developer:anthropic/claude-opus-4-5"},
			currentAgent: "backend-go-developer",
			currentModel: "",
			want:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := messageMatchesRecipient(tc.msg, tc.currentAgent, tc.currentModel)
			if got != tc.want {
				t.Errorf("messageMatchesRecipient() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestMessageMatchesSender(t *testing.T) {
	cases := []struct {
		name         string
		msg          state.Message
		currentAgent string
		currentModel string
		want         bool
	}{
		{
			name:         "matchesAgentName",
			msg:          state.Message{FromAgent: "backend-go-developer"},
			currentAgent: "backend-go-developer",
			currentModel: "",
			want:         true,
		},
		{
			name:         "matchesModelID",
			msg:          state.Message{FromAgent: "backend-go-developer:anthropic/claude-opus-4-5"},
			currentAgent: "backend-go-developer",
			currentModel: "backend-go-developer:anthropic/claude-opus-4-5",
			want:         true,
		},
		{
			name:         "matchesAgentWhenModelSet",
			msg:          state.Message{FromAgent: "backend-go-developer"},
			currentAgent: "backend-go-developer",
			currentModel: "backend-go-developer:anthropic/claude-opus-4-5",
			want:         true,
		},
		{
			name:         "noMatchDifferentAgent",
			msg:          state.Message{FromAgent: "coordinator"},
			currentAgent: "backend-go-developer",
			currentModel: "",
			want:         false,
		},
		{
			name:         "noMatchDifferentModel",
			msg:          state.Message{FromAgent: "backend-go-developer:anthropic/claude-sonnet-4-5"},
			currentAgent: "backend-go-developer",
			currentModel: "backend-go-developer:anthropic/claude-opus-4-5",
			want:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := messageMatchesSender(tc.msg, tc.currentAgent, tc.currentModel)
			if got != tc.want {
				t.Errorf("messageMatchesSender() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestModelStatusSymbol(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "working",
			status: "model-working",
			want:   "◐",
		},
		{
			name:   "done",
			status: "model-done",
			want:   "●",
		},
		{
			name:   "error",
			status: "model-error",
			want:   "✕",
		},
		{
			name:   "unknown",
			status: "some-other-status",
			want:   "○",
		},
		{
			name:   "empty",
			status: "",
			want:   "○",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := modelStatusSymbol(tc.status)
			if got != tc.want {
				t.Errorf("modelStatusSymbol(%q) = %q; want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestExtractModelShortName(t *testing.T) {
	cases := []struct {
		name    string
		modelID string
		want    string
	}{
		{
			name:    "standardModelID",
			modelID: "backend-go-developer:anthropic/claude-opus-4-5",
			want:    "anthropic/claude-opus-4-5",
		},
		{
			name:    "modelIDWithVariant",
			modelID: "backend-go-developer:anthropic/claude-opus-4-5 (max)",
			want:    "anthropic/claude-opus-4-5 (max)",
		},
		{
			name:    "noColon",
			modelID: "backend-go-developer",
			want:    "backend-go-developer",
		},
		{
			name:    "emptyString",
			modelID: "",
			want:    "",
		},
		{
			name:    "colonAtEnd",
			modelID: "backend-go-developer:",
			want:    "",
		},
		{
			name:    "multipleColons",
			modelID: "agent:model/with:extra",
			want:    "model/with:extra",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractModelShortName(tc.modelID)
			if got != tc.want {
				t.Errorf("extractModelShortName(%q) = %q; want %q", tc.modelID, got, tc.want)
			}
		})
	}
}

func TestBuildMultiModelSection(t *testing.T) {
	cases := []struct {
		name         string
		currentModel string
		models       map[string]any
		currentAgent string
		wantEmpty    bool
		wantContains []string
	}{
		{
			name:         "emptyCurrentModel",
			currentModel: "",
			models:       map[string]any{"agent1": []any{"model-a", "model-b"}},
			currentAgent: "agent1",
			wantEmpty:    true,
		},
		{
			name:         "singleModel",
			currentModel: "agent1:model-a",
			models:       map[string]any{"agent1": "model-a"},
			currentAgent: "agent1",
			wantEmpty:    true,
		},
		{
			name:         "multipleModels",
			currentModel: "agent1:model-a",
			models:       map[string]any{"agent1": []any{"model-a", "model-b", "model-c"}},
			currentAgent: "agent1",
			wantEmpty:    false,
			wantContains: []string{
				"Multi-Model Agent Context",
				"**Your identity:** agent1:model-a",
				"agent1:model-a  <-- YOU",
				"agent1:model-b",
				"agent1:model-c",
				"sgai_send_message",
				"sgai_check_inbox",
			},
		},
		{
			name:         "currentModelNotFirst",
			currentModel: "agent1:model-b",
			models:       map[string]any{"agent1": []any{"model-a", "model-b"}},
			currentAgent: "agent1",
			wantEmpty:    false,
			wantContains: []string{
				"**Your identity:** agent1:model-b",
				"agent1:model-b  <-- YOU",
			},
		},
		{
			name:         "agentNotInModels",
			currentModel: "agent1:model-a",
			models:       map[string]any{"agent2": []any{"model-a", "model-b"}},
			currentAgent: "agent1",
			wantEmpty:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildMultiModelSection(tc.currentModel, tc.models, tc.currentAgent)
			if tc.wantEmpty {
				if got != "" {
					t.Errorf("buildMultiModelSection() = %q; want empty string", got)
				}
				return
			}
			if got == "" {
				t.Error("buildMultiModelSection() = empty string; want non-empty")
				return
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildMultiModelSection() missing expected content: %q", want)
				}
			}
		})
	}
}
