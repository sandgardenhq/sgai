package main

import (
	"encoding/json"
	"testing"
)

func TestBuildClaudeCodeMCPConfigBasic(t *testing.T) {
	result, err := buildClaudeCodeMCPConfig("http://localhost:8080/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatal("invalid JSON:", err)
	}

	servers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("missing mcpServers")
	}

	sgai, ok := servers["sgai"].(map[string]any)
	if !ok {
		t.Fatal("missing sgai server")
	}

	if sgai["type"] != "sse" {
		t.Errorf("got type %q, want sse", sgai["type"])
	}
	if sgai["url"] != "http://localhost:8080/mcp" {
		t.Errorf("got url %q, want http://localhost:8080/mcp", sgai["url"])
	}
}

func TestBuildClaudeCodeMCPConfigWithProjectMCPs(t *testing.T) {
	projectMCPs := map[string]json.RawMessage{
		"playwright": json.RawMessage(`{"type": "local", "command": ["npx", "@playwright/mcp"]}`),
		"context7":   json.RawMessage(`{"type": "local", "command": ["npx", "-y", "@upstash/context7-mcp"]}`),
	}

	result, err := buildClaudeCodeMCPConfig("http://localhost:8080/mcp", projectMCPs)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatal("invalid JSON:", err)
	}

	servers := config["mcpServers"].(map[string]any)

	// SGAI server should be present
	if _, ok := servers["sgai"]; !ok {
		t.Error("missing sgai server")
	}

	// Playwright should be translated
	pw, ok := servers["playwright"].(map[string]any)
	if !ok {
		t.Fatal("missing playwright server")
	}
	if pw["command"] != "npx" {
		t.Errorf("got command %q, want npx", pw["command"])
	}
	args, ok := pw["args"].([]any)
	if !ok {
		t.Fatal("missing args")
	}
	if len(args) != 1 || args[0] != "@playwright/mcp" {
		t.Errorf("got args %v, want [@playwright/mcp]", args)
	}

	// Context7 should be translated
	c7, ok := servers["context7"].(map[string]any)
	if !ok {
		t.Fatal("missing context7 server")
	}
	if c7["command"] != "npx" {
		t.Errorf("got command %q, want npx", c7["command"])
	}
}

func TestBuildClaudeCodeMCPConfigDisabledServer(t *testing.T) {
	projectMCPs := map[string]json.RawMessage{
		"disabled": json.RawMessage(`{"type": "local", "command": ["test"], "enabled": false}`),
		"enabled":  json.RawMessage(`{"type": "local", "command": ["test2"]}`),
	}

	result, err := buildClaudeCodeMCPConfig("http://localhost:8080/mcp", projectMCPs)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatal("invalid JSON:", err)
	}

	servers := config["mcpServers"].(map[string]any)

	if _, ok := servers["disabled"]; ok {
		t.Error("disabled server should not be included")
	}
	if _, ok := servers["enabled"]; !ok {
		t.Error("enabled server should be included")
	}
}

func TestBuildClaudeCodeMCPConfigNonLocalSkipped(t *testing.T) {
	projectMCPs := map[string]json.RawMessage{
		"remote": json.RawMessage(`{"type": "remote", "url": "http://example.com"}`),
	}

	result, err := buildClaudeCodeMCPConfig("http://localhost:8080/mcp", projectMCPs)
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatal("invalid JSON:", err)
	}

	servers := config["mcpServers"].(map[string]any)
	if _, ok := servers["remote"]; ok {
		t.Error("non-local server should not be included")
	}
}

func TestBuildClaudeCodeMCPConfigInvalidJSON(t *testing.T) {
	projectMCPs := map[string]json.RawMessage{
		"bad": json.RawMessage(`not json`),
	}

	result, err := buildClaudeCodeMCPConfig("http://localhost:8080/mcp", projectMCPs)
	if err != nil {
		t.Fatal(err)
	}

	// Should still have valid JSON with just sgai server
	var config map[string]any
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatal("invalid JSON:", err)
	}
	servers := config["mcpServers"].(map[string]any)
	if _, ok := servers["sgai"]; !ok {
		t.Error("sgai server should still be present")
	}
	if _, ok := servers["bad"]; ok {
		t.Error("bad server should not be included")
	}
}

func TestClaudeCodeBuildAgentArgsMCPConfig(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		McpURL:    "http://localhost:8080/mcp",
	})

	assertContains(t, args, "--mcp-config")

	// Find the --mcp-config value and verify it's valid JSON with sgai server
	for i, arg := range args {
		if arg == "--mcp-config" && i+1 < len(args) {
			var config map[string]any
			if err := json.Unmarshal([]byte(args[i+1]), &config); err != nil {
				t.Fatal("--mcp-config value is not valid JSON:", err)
			}
			servers := config["mcpServers"].(map[string]any)
			sgai, ok := servers["sgai"].(map[string]any)
			if !ok {
				t.Fatal("missing sgai server in --mcp-config")
			}
			if sgai["url"] != "http://localhost:8080/mcp" {
				t.Errorf("sgai url = %q, want http://localhost:8080/mcp", sgai["url"])
			}
			return
		}
	}
	t.Error("--mcp-config flag not found in args")
}

func TestClaudeCodeBuildAgentArgsNoMCPWithoutURL(t *testing.T) {
	b := &claudeCodeBackend{}
	args := b.BuildAgentArgs(AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
	})
	assertNotContains(t, args, "--mcp-config")
}
