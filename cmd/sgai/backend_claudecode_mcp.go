package main

import (
	"encoding/json"
)

// buildClaudeCodeMCPConfig generates the MCP config JSON for Claude Code's --mcp-config flag.
// It always injects the SGAI MCP server and optionally translates project MCPs from
// opencode format (sgai.json "mcp" section) to Claude Code format.
func buildClaudeCodeMCPConfig(sgaiMCPURL string, projectMCPs map[string]json.RawMessage) (string, error) {
	servers := map[string]any{
		"sgai": map[string]any{
			"type": "sse",
			"url":  sgaiMCPURL,
		},
	}

	for name, rawConfig := range projectMCPs {
		var ocMCP struct {
			Type    string   `json:"type"`
			Command []string `json:"command"`
			Enabled *bool    `json:"enabled,omitempty"`
		}
		if err := json.Unmarshal(rawConfig, &ocMCP); err != nil {
			continue
		}
		if ocMCP.Enabled != nil && !*ocMCP.Enabled {
			continue
		}
		if ocMCP.Type == "local" && len(ocMCP.Command) > 0 {
			servers[name] = map[string]any{
				"command": ocMCP.Command[0],
				"args":    ocMCP.Command[1:],
			}
		}
	}

	config := map[string]any{
		"mcpServers": servers,
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
