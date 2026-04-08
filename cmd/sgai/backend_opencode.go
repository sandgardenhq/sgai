package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type opencodeBackend struct{}

func (b *opencodeBackend) Name() string       { return "opencode" }
func (b *opencodeBackend) BinaryName() string { return "opencode" }

func (b *opencodeBackend) BuildAgentArgs(p AgentRunParams) []string {
	args := []string{"run", "--format=json", "--agent", p.BaseAgent}
	if p.ModelSpec != "" {
		model, variant := parseModelAndVariant(p.ModelSpec)
		args = append(args, "--model", model)
		if variant != "" {
			args = append(args, "--variant", variant)
		}
	}
	if p.SessionID != "" {
		args = append(args, "--session", p.SessionID)
	}
	title := p.Agent
	if p.ModelSpec != "" {
		title = p.Agent + " [" + p.ModelSpec + "]"
	}
	args = append(args, "--title", title)
	return args
}

func (b *opencodeBackend) BuildAdhocArgs(modelSpec string) []string {
	baseModel, variant := parseModelAndVariant(modelSpec)
	args := []string{"run", "-m", baseModel, "--agent", "build", "--title", "adhoc [" + modelSpec + "]"}
	if variant != "" {
		args = append(args, "--variant", variant)
	}
	return args
}

func (b *opencodeBackend) BuildEnv(p AgentEnvParams) []string {
	return append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+filepath.Join(p.Dir, ".sgai"),
		"SGAI_MCP_URL="+p.McpURL,
		"SGAI_AGENT_IDENTITY="+p.AgentIdentity,
		"SGAI_MCP_INTERACTIVE="+p.InteractiveMode)
}

func (b *opencodeBackend) BuildContinuousArgs() []string {
	return []string{"run", "--title", "continuous-mode-prompt"}
}

func (b *opencodeBackend) ParseEvent(line []byte) (streamEvent, bool) {
	var event streamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return streamEvent{}, false
	}
	return event, true
}

func (b *opencodeBackend) ValidateModels(models map[string]any) error {
	return validateModels(models)
}

func (b *opencodeBackend) ExportSession(dir, sessionID, outputPath string) error {
	cmd := exec.Command("opencode", "export", sessionID)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("opencode export failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outputPath, output, 0644)
}

func (b *opencodeBackend) StripProviderPrefix(model string) string {
	return model // opencode uses the full "anthropic/claude-opus-4-6" form
}
