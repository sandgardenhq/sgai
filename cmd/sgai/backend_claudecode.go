package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type claudeCodeBackend struct{}

func (b *claudeCodeBackend) Name() string       { return "claude-code" }
func (b *claudeCodeBackend) BinaryName() string { return "claude" }

func (b *claudeCodeBackend) BuildAgentArgs(p AgentRunParams) []string {
	args := []string{"-p", "--output-format", "stream-json", "--verbose"}

	if p.ModelSpec != "" {
		model, variant := parseModelAndVariant(p.ModelSpec)
		args = append(args, "--model", b.StripProviderPrefix(model))
		if variant != "" {
			args = append(args, "--effort", variant)
		}
	}
	if p.SessionID != "" {
		args = append(args, "--session-id", p.SessionID)
	}
	title := p.Agent
	if p.ModelSpec != "" {
		title = p.Agent + " [" + p.ModelSpec + "]"
	}
	args = append(args, "--name", title)

	// Inject agent system prompt from .sgai/agent/<name>.md
	if p.AgentDir != "" && p.BaseAgent != "" {
		agentPath := filepath.Join(p.AgentDir, ".sgai", "agent", p.BaseAgent+".md")
		if content, err := os.ReadFile(agentPath); err == nil {
			body := strings.TrimSpace(string(extractBody(content)))
			if body != "" {
				args = append(args, "--append-system-prompt", body)
			}
		}
	}

	// Skip permissions — SGAI controls permissions via its own MCP layer
	args = append(args, "--permission-mode", "bypassPermissions")

	// Inject MCP config with SGAI MCP server
	if p.McpURL != "" {
		mcpConfig, err := buildClaudeCodeMCPConfig(p.McpURL, nil)
		if err == nil {
			args = append(args, "--mcp-config", mcpConfig)
		}
	}

	return args
}

func (b *claudeCodeBackend) BuildAdhocArgs(modelSpec string) []string {
	baseModel, variant := parseModelAndVariant(modelSpec)
	args := []string{"-p", "--output-format", "stream-json", "--verbose",
		"--model", b.StripProviderPrefix(baseModel),
		"--name", "adhoc [" + modelSpec + "]"}
	if variant != "" {
		args = append(args, "--effort", variant)
	}
	return args
}

func (b *claudeCodeBackend) BuildEnv(p AgentEnvParams) []string {
	return append(os.Environ(),
		"SGAI_MCP_URL="+p.McpURL,
		"SGAI_AGENT_IDENTITY="+p.AgentIdentity,
		"SGAI_MCP_INTERACTIVE="+p.InteractiveMode)
}

func (b *claudeCodeBackend) BuildContinuousArgs() []string {
	return []string{"-p", "--output-format", "stream-json", "--verbose",
		"--name", "continuous-mode-prompt"}
}

func (b *claudeCodeBackend) StripProviderPrefix(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
}

func (b *claudeCodeBackend) ValidateModels(models map[string]any) error {
	return nil // Claude Code has no `models` command; skip validation
}

func (b *claudeCodeBackend) ExportSession(dir, sessionID, outputPath string) error {
	return nil // Claude Code sessions are stored in ~/.claude/; no export needed
}

// claudeCodeRawEvent represents the raw JSON from Claude Code's stream-json output.
type claudeCodeRawEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Message   *struct {
		Content []struct {
			Type     string         `json:"type"`
			Text     string         `json:"text,omitempty"`
			Thinking string         `json:"thinking,omitempty"`
			Name     string         `json:"name,omitempty"`
			ID       string         `json:"id,omitempty"`
			Input    map[string]any `json:"input,omitempty"`
		} `json:"content,omitempty"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage,omitempty"`
	} `json:"message,omitempty"`
	// Result event fields (type=="result")
	Result       string  `json:"result,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	Usage        *struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage,omitempty"`
}

func (b *claudeCodeBackend) ParseEvent(line []byte) (streamEvent, bool) {
	var raw claudeCodeRawEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return streamEvent{}, false
	}

	base := streamEvent{SessionID: raw.SessionID}

	switch raw.Type {
	case "system":
		if raw.SessionID != "" {
			base.Type = "system"
			return base, true
		}
		return streamEvent{}, false

	case "assistant":
		if raw.Message == nil || len(raw.Message.Content) == 0 {
			return streamEvent{}, false
		}
		content := raw.Message.Content[0]
		switch content.Type {
		case "text":
			base.Type = "text"
			base.Part.Text = content.Text
		case "thinking":
			base.Type = "reasoning"
			base.Part.Text = content.Thinking
		case "tool_use":
			base.Type = "tool_use"
			base.Part.Tool = content.Name
			base.Part.State = &toolState{
				Status: "running",
				Input:  content.Input,
			}
		default:
			return streamEvent{}, false
		}
		return base, true

	case "result":
		base.Type = "result"
		if raw.Usage != nil {
			base.Part.Cost = raw.TotalCostUSD
			base.Part.Tokens.Input = raw.Usage.InputTokens
			base.Part.Tokens.Output = raw.Usage.OutputTokens
			base.Part.Tokens.Cache.Read = raw.Usage.CacheReadInputTokens
			base.Part.Tokens.Cache.Write = raw.Usage.CacheCreationInputTokens
		}
		return base, true

	default:
		return streamEvent{}, false
	}
}
