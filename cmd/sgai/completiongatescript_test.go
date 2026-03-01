package main

import (
	"context"
	"strings"
	"testing"
	"time"
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
			output, err := runCompletionGateScript(context.Background(), t.TempDir(), tt.script)

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

func TestRunCompletionGateScriptContextCancellation(t *testing.T) {
	t.Run("cancelledContextTerminatesScript", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		_, err := runCompletionGateScript(ctx, t.TempDir(), "sleep 30")
		elapsed := time.Since(start)

		if err == nil {
			t.Error("expected error from cancelled context, got nil")
		}
		if elapsed > 5*time.Second {
			t.Errorf("script took %v to terminate, expected fast cancellation", elapsed)
		}
	})

	t.Run("timeoutContextTerminatesScript", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		t.Cleanup(cancel)

		start := time.Now()
		_, err := runCompletionGateScript(ctx, t.TempDir(), "sleep 30")
		elapsed := time.Since(start)

		if err == nil {
			t.Error("expected error from timed-out context, got nil")
		}
		if elapsed > 5*time.Second {
			t.Errorf("script took %v to terminate, expected fast timeout", elapsed)
		}
	})
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

func TestBuildUpdateWorkflowStateSchemaStatusEnum(t *testing.T) {
	tests := []struct {
		name            string
		agent           string
		expectComplete  bool
		expectWorking   bool
		expectAgentDone bool
	}{
		{
			name:            "coordinatorHasComplete",
			agent:           "coordinator",
			expectComplete:  true,
			expectWorking:   true,
			expectAgentDone: true,
		},
		{
			name:            "nonCoordinatorLacksComplete",
			agent:           "backend-go-developer",
			expectComplete:  false,
			expectWorking:   true,
			expectAgentDone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, _ := buildUpdateWorkflowStateSchema(tt.agent)
			statusProp := schema.Properties["status"]
			if statusProp == nil {
				t.Fatal("status property not found in schema")
			}

			enumSlice := statusProp.Enum
			enumMap := make(map[string]bool)
			for _, v := range enumSlice {
				if s, ok := v.(string); ok {
					enumMap[s] = true
				}
			}

			if got := enumMap["complete"]; got != tt.expectComplete {
				t.Errorf("enum contains 'complete' = %v, want %v", got, tt.expectComplete)
			}
			if enumMap["human-communication"] {
				t.Error("enum should not contain 'human-communication'")
			}
			if got := enumMap["working"]; got != tt.expectWorking {
				t.Errorf("enum contains 'working' = %v, want %v", got, tt.expectWorking)
			}
			if got := enumMap["agent-done"]; got != tt.expectAgentDone {
				t.Errorf("enum contains 'agent-done' = %v, want %v", got, tt.expectAgentDone)
			}
		})
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
