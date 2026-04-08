package main

import (
	"slices"
	"testing"
)

func assertContains(t *testing.T, args []string, want string) {
	t.Helper()
	if !slices.Contains(args, want) {
		t.Errorf("args %v should contain %q", args, want)
	}
}

func assertNotContains(t *testing.T, args []string, notWant string) {
	t.Helper()
	if slices.Contains(args, notWant) {
		t.Errorf("args %v should not contain %q", args, notWant)
	}
}

func TestOpencodeBackendName(t *testing.T) {
	b := &opencodeBackend{}
	if b.Name() != "opencode" {
		t.Errorf("got %q, want opencode", b.Name())
	}
	if b.BinaryName() != "opencode" {
		t.Errorf("got %q, want opencode", b.BinaryName())
	}
}

func TestOpencodeBackendBuildAgentArgs(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		ModelSpec: "anthropic/claude-opus-4-6 (max)",
		SessionID: "ses_123",
	})
	assertContains(t, args, "--format=json")
	assertContains(t, args, "--agent")
	assertContains(t, args, "--variant")
	assertContains(t, args, "--session")
	assertContains(t, args, "coordinator")
	assertContains(t, args, "anthropic/claude-opus-4-6")
	assertContains(t, args, "max")
	assertContains(t, args, "ses_123")
}

func TestOpencodeBackendBuildAgentArgsNoModel(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "builder",
		BaseAgent: "builder",
	})
	assertContains(t, args, "--format=json")
	assertContains(t, args, "--agent")
	assertNotContains(t, args, "--model")
	assertNotContains(t, args, "--variant")
	assertNotContains(t, args, "--session")
}

func TestOpencodeBackendBuildAdhocArgs(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildAdhocArgs("anthropic/claude-opus-4-6 (max)")
	assertContains(t, args, "run")
	assertContains(t, args, "-m")
	assertContains(t, args, "anthropic/claude-opus-4-6")
	assertContains(t, args, "--variant")
	assertContains(t, args, "max")
}

func TestOpencodeBackendBuildContinuousArgs(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildContinuousArgs()
	assertContains(t, args, "run")
	assertContains(t, args, "--title")
	assertContains(t, args, "continuous-mode-prompt")
}

func TestOpencodeBackendBuildEnv(t *testing.T) {
	b := &opencodeBackend{}
	env := b.BuildEnv(AgentEnvParams{
		Dir:             "/tmp/project",
		McpURL:          "http://localhost:8080/mcp",
		AgentIdentity:   "coordinator",
		InteractiveMode: "yes",
	})
	found := map[string]bool{}
	for _, e := range env {
		switch {
		case e == "OPENCODE_CONFIG_DIR=/tmp/project/.sgai":
			found["config"] = true
		case e == "SGAI_MCP_URL=http://localhost:8080/mcp":
			found["mcp"] = true
		case e == "SGAI_AGENT_IDENTITY=coordinator":
			found["identity"] = true
		case e == "SGAI_MCP_INTERACTIVE=yes":
			found["interactive"] = true
		}
	}
	for _, key := range []string{"config", "mcp", "identity", "interactive"} {
		if !found[key] {
			t.Errorf("env missing %s", key)
		}
	}
}

func TestOpencodeBackendParseEvent(t *testing.T) {
	b := &opencodeBackend{}
	line := []byte(`{"type":"text","sessionID":"ses_1","part":{"type":"text","text":"hello"}}`)
	event, ok := b.ParseEvent(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if event.Type != "text" {
		t.Errorf("got type %q, want text", event.Type)
	}
	if event.SessionID != "ses_1" {
		t.Errorf("got session %q, want ses_1", event.SessionID)
	}
}

func TestOpencodeBackendParseEventInvalid(t *testing.T) {
	b := &opencodeBackend{}
	_, ok := b.ParseEvent([]byte(`not json`))
	if ok {
		t.Error("expected not ok for invalid JSON")
	}
}

func TestOpencodeBackendStripProviderPrefix(t *testing.T) {
	b := &opencodeBackend{}
	if got := b.StripProviderPrefix("anthropic/claude-opus-4-6"); got != "anthropic/claude-opus-4-6" {
		t.Errorf("got %q, want original value preserved", got)
	}
}
