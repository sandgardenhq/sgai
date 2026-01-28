// Tests for agent snippet parsing functionality.
package main

import (
	"slices"
	"testing"
)

func TestParseAgentSnippets(t *testing.T) {
	dir := "skel"

	tests := []struct {
		name         string
		agent        string
		wantSnippets []string
	}{
		{
			name:         "backendGoDeveloperNoSnippets",
			agent:        "backend-go-developer",
			wantSnippets: nil,
		},
		{
			name:         "goReadabilityReviewerNoSnippets",
			agent:        "go-readability-reviewer",
			wantSnippets: nil,
		},
		{
			name:         "htmxPicocssFrontendDeveloperNoSnippets",
			agent:        "htmx-picocss-frontend-developer",
			wantSnippets: nil,
		},
		{
			name:         "htmxPicocssFrontendReviewerNoSnippets",
			agent:        "htmx-picocss-frontend-reviewer",
			wantSnippets: nil,
		},
		{
			name:         "shellScriptCoderNoSnippets",
			agent:        "shell-script-coder",
			wantSnippets: nil,
		},
		{
			name:         "shellScriptReviewerNoSnippets",
			agent:        "shell-script-reviewer",
			wantSnippets: nil,
		},
		{
			name:         "coordinatorNoSnippets",
			agent:        "coordinator",
			wantSnippets: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAgentSnippets(dir, tt.agent)
			if !slices.Equal(got, tt.wantSnippets) {
				t.Logf("Working directory: %s", dir)
				agentPath := dir + "/.sgai/agent/" + tt.agent + ".md"
				t.Logf("Agent path: %s", agentPath)
				t.Errorf("parseAgentSnippets(%q) = %v; want %v", tt.agent, got, tt.wantSnippets)
			}
		})
	}
}
