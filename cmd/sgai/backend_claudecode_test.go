package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCodeBackendName(t *testing.T) {
	b := &claudeCodeBackend{}
	if b.Name() != "claude-code" {
		t.Errorf("got %q, want claude-code", b.Name())
	}
	if b.BinaryName() != "claude" {
		t.Errorf("got %q, want claude", b.BinaryName())
	}
}

func TestClaudeCodeBackendBuildAgentArgs(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		ModelSpec: "anthropic/claude-opus-4-6 (max)",
		SessionID: "abc-123-def",
	})
	assertContains(t, args, "-p")
	assertContains(t, args, "--output-format")
	assertContains(t, args, "stream-json")
	assertContains(t, args, "--verbose")
	assertContains(t, args, "--model")
	assertContains(t, args, "claude-opus-4-6") // provider prefix stripped
	assertContains(t, args, "--effort")
	assertContains(t, args, "max")
	assertContains(t, args, "--session-id")
	assertContains(t, args, "abc-123-def")
	assertContains(t, args, "--name")
	assertNotContains(t, args, "--format=json") // opencode flag
	// Verify no "anthropic/" in args
	for _, arg := range args {
		if strings.HasPrefix(arg, "anthropic/") {
			t.Errorf("args should not contain anthropic/ prefixed model, got %q", arg)
		}
	}
}

func TestClaudeCodeBackendBuildAgentArgsNoModel(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "builder",
		BaseAgent: "builder",
	})
	assertContains(t, args, "-p")
	assertNotContains(t, args, "--model")
	assertNotContains(t, args, "--effort")
	assertNotContains(t, args, "--session-id")
}

func TestClaudeCodeBackendBuildAdhocArgs(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAdhocArgs("anthropic/claude-opus-4-6 (max)")
	assertContains(t, args, "-p")
	assertContains(t, args, "--output-format")
	assertContains(t, args, "stream-json")
	assertContains(t, args, "--model")
	assertContains(t, args, "claude-opus-4-6")
	assertContains(t, args, "--effort")
	assertContains(t, args, "max")
	assertNotContains(t, args, "run") // opencode uses "run"
}

func TestClaudeCodeBackendBuildContinuousArgs(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildContinuousArgs()
	assertContains(t, args, "-p")
	assertContains(t, args, "--output-format")
	assertContains(t, args, "--name")
	assertContains(t, args, "continuous-mode-prompt")
}

func TestClaudeCodeBackendBuildEnv(t *testing.T) {
	b := &claudeCodeBackend{}
	env := b.BuildEnv(AgentEnvParams{
		Dir:             "/tmp/project",
		McpURL:          "http://localhost:8080/mcp",
		AgentIdentity:   "coordinator",
		InteractiveMode: "yes",
	})
	found := map[string]bool{}
	for _, e := range env {
		switch {
		case e == "SGAI_MCP_URL=http://localhost:8080/mcp":
			found["mcp"] = true
		case e == "SGAI_AGENT_IDENTITY=coordinator":
			found["identity"] = true
		case e == "SGAI_MCP_INTERACTIVE=yes":
			found["interactive"] = true
		}
	}
	for _, key := range []string{"mcp", "identity", "interactive"} {
		if !found[key] {
			t.Errorf("env missing %s", key)
		}
	}
	// Should NOT have OPENCODE_CONFIG_DIR
	for _, e := range env {
		if strings.HasPrefix(e, "OPENCODE_CONFIG_DIR=") {
			t.Error("env should not contain OPENCODE_CONFIG_DIR for claude-code backend")
		}
	}
}

func TestClaudeCodeStripProviderPrefix(t *testing.T) {
	b := &claudeCodeBackend{}
	tests := []struct{ input, want string }{
		{"anthropic/claude-opus-4-6", "claude-opus-4-6"},
		{"openai/gpt-4o", "gpt-4o"},
		{"claude-opus-4-6", "claude-opus-4-6"}, // no prefix
	}
	for _, tt := range tests {
		got := b.StripProviderPrefix(tt.input)
		if got != tt.want {
			t.Errorf("StripProviderPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClaudeCodeBackendValidateModels(t *testing.T) {
	b := &claudeCodeBackend{}
	// Should always return nil (no validation for Claude Code)
	if err := b.ValidateModels(map[string]any{"coordinator": "whatever"}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestClaudeCodeBackendExportSession(t *testing.T) {
	b := &claudeCodeBackend{}
	// Should always return nil (no export for Claude Code)
	if err := b.ExportSession("/tmp", "ses_1", "/tmp/out.json"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestClaudeCodeBuildAgentArgsWithSystemPrompt(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	agentContent := "---\npermissions:\n  edit: allow\n---\nYou are a coordinator agent.\nFollow these rules carefully."
	if err := os.WriteFile(filepath.Join(agentDir, "coordinator.md"), []byte(agentContent), 0644); err != nil {
		t.Fatal(err)
	}

	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		ModelSpec: "anthropic/claude-opus-4-6",
		AgentDir:  dir,
	})

	// Should have --append-system-prompt with the body (frontmatter stripped)
	assertContains(t, args, "--append-system-prompt")
	found := false
	for i, arg := range args {
		if arg == "--append-system-prompt" && i+1 < len(args) {
			if strings.Contains(args[i+1], "You are a coordinator agent.") {
				found = true
			}
			// Should NOT contain frontmatter
			if strings.Contains(args[i+1], "permissions:") {
				t.Error("system prompt should not contain frontmatter")
			}
		}
	}
	if !found {
		t.Error("expected --append-system-prompt to contain agent body")
	}
}

func TestClaudeCodeBuildAgentArgsPermissionMode(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "builder",
		BaseAgent: "builder",
	})
	assertContains(t, args, "--permission-mode")
	assertContains(t, args, "bypassPermissions")
}

func TestClaudeCodeBuildAgentArgsNoAgentFile(t *testing.T) {
	dir := t.TempDir()
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "builder",
		BaseAgent: "builder",
		AgentDir:  dir,
	})
	// No --append-system-prompt when agent file doesn't exist
	assertNotContains(t, args, "--append-system-prompt")
	// But permission mode should still be set
	assertContains(t, args, "--permission-mode")
}

func TestClaudeCodeBuildAgentArgsEmptyBody(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Agent file with only frontmatter, no body
	if err := os.WriteFile(filepath.Join(agentDir, "builder.md"), []byte("---\npermissions:\n  edit: allow\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "builder",
		BaseAgent: "builder",
		AgentDir:  dir,
	})
	assertNotContains(t, args, "--append-system-prompt")
}

func TestClaudeCodeParseEventText(t *testing.T) {
	b := &claudeCodeBackend{}
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]},"session_id":"abc-123"}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "text" {
		t.Errorf("got type %q, want text", event.Type)
	}
	if event.Part.Text != "hello" {
		t.Errorf("got text %q, want hello", event.Part.Text)
	}
	if event.SessionID != "abc-123" {
		t.Errorf("got session %q, want abc-123", event.SessionID)
	}
}

func TestClaudeCodeParseEventThinking(t *testing.T) {
	b := &claudeCodeBackend{}
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"reasoning here"}]},"session_id":"s1"}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "reasoning" {
		t.Errorf("got type %q, want reasoning", event.Type)
	}
	if event.Part.Text != "reasoning here" {
		t.Errorf("got text %q, want 'reasoning here'", event.Part.Text)
	}
}

func TestClaudeCodeParseEventToolUse(t *testing.T) {
	b := &claudeCodeBackend{}
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit","input":{"file":"main.go"}}]},"session_id":"s1"}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "tool_use" {
		t.Errorf("got type %q, want tool_use", event.Type)
	}
	if event.Part.Tool != "Edit" {
		t.Errorf("got tool %q, want Edit", event.Part.Tool)
	}
	if event.Part.State == nil {
		t.Fatal("expected non-nil state")
	}
	if event.Part.State.Status != "running" {
		t.Errorf("got status %q, want running", event.Part.State.Status)
	}
}

func TestClaudeCodeParseEventResult(t *testing.T) {
	b := &claudeCodeBackend{}
	line := []byte(`{"type":"result","session_id":"s1","total_cost_usd":0.05,"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "result" {
		t.Errorf("got type %q, want result", event.Type)
	}
	if event.Part.Cost != 0.05 {
		t.Errorf("got cost %f, want 0.05", event.Part.Cost)
	}
	if event.Part.Tokens.Input != 100 {
		t.Errorf("got input tokens %d, want 100", event.Part.Tokens.Input)
	}
	if event.Part.Tokens.Output != 50 {
		t.Errorf("got output tokens %d, want 50", event.Part.Tokens.Output)
	}
	if event.Part.Tokens.Cache.Read != 10 {
		t.Errorf("got cache read %d, want 10", event.Part.Tokens.Cache.Read)
	}
	if event.Part.Tokens.Cache.Write != 5 {
		t.Errorf("got cache write %d, want 5", event.Part.Tokens.Cache.Write)
	}
}

func TestClaudeCodeParseEventSystem(t *testing.T) {
	b := &claudeCodeBackend{}
	line := []byte(`{"type":"system","subtype":"init","session_id":"s1"}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "system" {
		t.Errorf("got type %q, want system", event.Type)
	}
	if event.SessionID != "s1" {
		t.Errorf("got session %q, want s1", event.SessionID)
	}
}

func TestClaudeCodeParseEventSystemNoSession(t *testing.T) {
	b := &claudeCodeBackend{}
	_, ok := b.ParseEvent([]byte(`{"type":"system","subtype":"other"}`))
	if ok {
		t.Error("expected not ok for system event without session_id")
	}
}

func TestClaudeCodeParseEventInvalid(t *testing.T) {
	b := &claudeCodeBackend{}
	_, ok := b.ParseEvent([]byte(`not json`))
	if ok {
		t.Error("expected not ok for invalid JSON")
	}
}

func TestClaudeCodeParseEventUnknownType(t *testing.T) {
	b := &claudeCodeBackend{}
	_, ok := b.ParseEvent([]byte(`{"type":"unknown"}`))
	if ok {
		t.Error("expected not ok for unknown type")
	}
}
