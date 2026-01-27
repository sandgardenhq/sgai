package main

import (
	"strings"
	"testing"
)

func TestRunCompletionGateScript(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		expectError bool
		outputMatch string
	}{
		{
			name:        "exit0Success",
			script:      "exit 0",
			expectError: false,
		},
		{
			name:        "exit1Failure",
			script:      "exit 1",
			expectError: true,
		},
		{
			name:        "echoAndExit0",
			script:      "echo 'success'; exit 0",
			expectError: false,
			outputMatch: "success",
		},
		{
			name:        "echoAndExit1",
			script:      "echo 'failure'; exit 1",
			expectError: true,
			outputMatch: "failure",
		},
		{
			name:        "trueCommand",
			script:      "true",
			expectError: false,
		},
		{
			name:        "falseCommand",
			script:      "false",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runCompletionGateScript(tt.script)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.outputMatch != "" && !strings.Contains(output, tt.outputMatch) {
				t.Errorf("output %q does not contain %q", output, tt.outputMatch)
			}
		})
	}
}

func TestFormatCompletionGateScriptFailureMessage(t *testing.T) {
	script := "./check-success.sh"
	output := "Test failed: expected 5, got 3"

	msg := formatCompletionGateScriptFailureMessage(script, output)

	expectedParts := []string{
		"From: environment",
		"To: coordinator",
		"Subject: computable definition of success has failed",
		script,
		output,
		"<pre>",
		"</pre>",
	}

	for _, part := range expectedParts {
		if !strings.Contains(msg, part) {
			t.Errorf("message does not contain %q\nGot: %s", part, msg)
		}
	}
}

func TestParseYAMLFrontmatterWithCompletionGateScript(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedScript string
	}{
		{
			name: "completionGateScriptPresent",
			content: `---
completionGateScript: ./check-success.sh
---
# Goal content`,
			expectedScript: "./check-success.sh",
		},
		{
			name: "completionGateScriptWithOtherFields",
			content: `---
flow: |
  "coordinator" -> "developer"
interactive: yes
completionGateScript: make test
---
# Goal content`,
			expectedScript: "make test",
		},
		{
			name: "noCompletionGateScript",
			content: `---
flow: |
  "coordinator" -> "developer"
---
# Goal content`,
			expectedScript: "",
		},
		{
			name:           "noFrontmatter",
			content:        "# Goal content without frontmatter",
			expectedScript: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := parseYAMLFrontmatter([]byte(tt.content))
			if err != nil {
				t.Fatalf("parseYAMLFrontmatter() error = %v", err)
			}

			if metadata.CompletionGateScript != tt.expectedScript {
				t.Errorf("CompletionGateScript = %q, want %q", metadata.CompletionGateScript, tt.expectedScript)
			}
		})
	}
}
