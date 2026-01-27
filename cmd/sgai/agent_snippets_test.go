// Tests for agent snippet parsing functionality.
package main

import (
	"slices"
	"testing"
)

func TestParseAgentSnippets(t *testing.T) {
	dir := "../.."

	tests := []struct {
		name         string
		agent        string
		wantSnippets []string
	}{
		{
			name:         "backend-go-developer has go snippets",
			agent:        "backend-go-developer",
			wantSnippets: []string{"go"},
		},
		{
			name:         "go-readability-reviewer has go snippets",
			agent:        "go-readability-reviewer",
			wantSnippets: []string{"go"},
		},
		{
			name:         "htmx-picocss-frontend-developer has htmx html css snippets",
			agent:        "htmx-picocss-frontend-developer",
			wantSnippets: []string{"htmx", "html", "css"},
		},
		{
			name:         "htmx-picocss-frontend-reviewer has htmx html css snippets",
			agent:        "htmx-picocss-frontend-reviewer",
			wantSnippets: []string{"htmx", "html", "css"},
		},
		{
			name:         "shell-script-coder has bash snippets",
			agent:        "shell-script-coder",
			wantSnippets: []string{"bash"},
		},
		{
			name:         "shell-script-reviewer has bash snippets",
			agent:        "shell-script-reviewer",
			wantSnippets: []string{"bash"},
		},
		{
			name:         "coordinator has no snippets",
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
