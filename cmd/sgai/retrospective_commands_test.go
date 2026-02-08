package main

import (
	"strings"
	"testing"
)

func TestBuildRetrospectiveGoalContent(t *testing.T) {
	cases := []struct {
		name             string
		absSessionPath   string
		coordinatorModel string
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "withCoordinatorModel",
			absSessionPath:   "/tmp/retro-session",
			coordinatorModel: "anthropic/claude-opus-4-6 (max)",
			wantContains: []string{
				"models:",
				`"coordinator": "anthropic/claude-opus-4-6 (max)"`,
				`"retrospective-session-analyzer": "anthropic/claude-opus-4-6 (max)"`,
				`"retrospective-code-analyzer": "anthropic/claude-opus-4-6 (max)"`,
				`"retrospective-refiner": "anthropic/claude-opus-4-6 (max)"`,
				"Analyze session: /tmp/retro-session",
				"interactive: auto",
			},
		},
		{
			name:             "withoutCoordinatorModel",
			absSessionPath:   "/tmp/retro-session",
			coordinatorModel: "",
			wantContains: []string{
				"Analyze session: /tmp/retro-session",
				"interactive: auto",
			},
			wantNotContains: []string{
				"models:",
			},
		},
		{
			name:             "withSimpleModel",
			absSessionPath:   "/tmp/other-session",
			coordinatorModel: "openai/gpt-5.2",
			wantContains: []string{
				"models:",
				`"coordinator": "openai/gpt-5.2"`,
				`"retrospective-session-analyzer": "openai/gpt-5.2"`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRetrospectiveGoalContent(tc.absSessionPath, tc.coordinatorModel)

			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("result missing %q\ngot:\n%s", want, got)
				}
			}

			for _, notWant := range tc.wantNotContains {
				if strings.Contains(got, notWant) {
					t.Errorf("result should not contain %q\ngot:\n%s", notWant, got)
				}
			}
		})
	}
}
