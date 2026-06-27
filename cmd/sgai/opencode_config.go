package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const opencodeMCPTimeout = 43200000

type openCodeConfigContent struct {
	MCP   map[string]json.RawMessage     `json:"mcp,omitempty"`
	Agent map[string]openCodeAgentConfig `json:"agent,omitempty"`
	extra map[string]json.RawMessage
}

type openCodeAgentConfig struct {
	Permission map[string]json.RawMessage `json:"permission,omitempty"`
	extra      map[string]json.RawMessage
}

type openCodeLocalMCP struct {
	Type    string   `json:"type"`
	Enabled bool     `json:"enabled"`
	Timeout int      `json:"timeout"`
	Command []string `json:"command"`
}

func (c *openCodeConfigContent) UnmarshalJSON(data []byte) error {
	fields := map[string]json.RawMessage{}
	if errUnmarshal := json.Unmarshal(data, &fields); errUnmarshal != nil {
		return errUnmarshal
	}
	c.extra = fields
	if rawMCP, ok := fields["mcp"]; ok {
		if errUnmarshal := json.Unmarshal(rawMCP, &c.MCP); errUnmarshal != nil {
			return errUnmarshal
		}
		delete(c.extra, "mcp")
	}
	if rawAgent, ok := fields["agent"]; ok {
		if errUnmarshal := json.Unmarshal(rawAgent, &c.Agent); errUnmarshal != nil {
			return errUnmarshal
		}
		delete(c.extra, "agent")
	}
	return nil
}

func (c openCodeConfigContent) MarshalJSON() ([]byte, error) {
	fields := map[string]json.RawMessage{}
	for key, value := range c.extra {
		fields[key] = value
	}
	if len(c.MCP) > 0 {
		mcpData, errMarshal := json.Marshal(c.MCP)
		if errMarshal != nil {
			return nil, errMarshal
		}
		fields["mcp"] = mcpData
	}
	if len(c.Agent) > 0 {
		agentData, errMarshal := json.Marshal(c.Agent)
		if errMarshal != nil {
			return nil, errMarshal
		}
		fields["agent"] = agentData
	}
	return json.Marshal(fields)
}

func (c *openCodeAgentConfig) UnmarshalJSON(data []byte) error {
	fields := map[string]json.RawMessage{}
	if errUnmarshal := json.Unmarshal(data, &fields); errUnmarshal != nil {
		return errUnmarshal
	}
	c.extra = fields
	if rawPermission, ok := fields["permission"]; ok {
		if errUnmarshal := json.Unmarshal(rawPermission, &c.Permission); errUnmarshal != nil {
			return errUnmarshal
		}
		delete(c.extra, "permission")
	}
	return nil
}

func (c openCodeAgentConfig) MarshalJSON() ([]byte, error) {
	fields := map[string]json.RawMessage{}
	for key, value := range c.extra {
		fields[key] = value
	}
	if len(c.Permission) > 0 {
		permissionData, errMarshal := json.Marshal(c.Permission)
		if errMarshal != nil {
			return nil, errMarshal
		}
		fields["permission"] = permissionData
	}
	return json.Marshal(fields)
}

func buildOpenCodeConfigContent(baseContent, sgaiBinPath, mcpURL, agentIdentity string) (string, error) {
	config := openCodeConfigContent{}
	if baseContent != "" {
		if errUnmarshal := json.Unmarshal([]byte(baseContent), &config); errUnmarshal != nil {
			return "", fmt.Errorf("parsing existing OPENCODE_CONFIG_CONTENT: %w", errUnmarshal)
		}
	}

	if config.MCP == nil {
		config.MCP = map[string]json.RawMessage{}
	}
	sgaiMCP := openCodeLocalMCP{
		Type:    "local",
		Enabled: true,
		Timeout: opencodeMCPTimeout,
		Command: []string{sgaiBinPath, "internal-mcp", mcpURL, agentIdentity},
	}
	sgaiData, errMarshalSGAI := json.Marshal(sgaiMCP)
	if errMarshalSGAI != nil {
		return "", fmt.Errorf("encoding sgai mcp config: %w", errMarshalSGAI)
	}
	config.MCP["sgai"] = sgaiData

	data, errMarshal := json.Marshal(config)
	if errMarshal != nil {
		return "", fmt.Errorf("encoding OPENCODE_CONFIG_CONTENT: %w", errMarshal)
	}
	return string(data), nil
}

func sgaiExecutablePath() string {
	path, errExecutable := os.Executable()
	if errExecutable == nil && path != "" {
		return path
	}
	if len(os.Args) > 0 && os.Args[0] != "" {
		if abs, errAbs := filepath.Abs(os.Args[0]); errAbs == nil {
			return abs
		}
		return os.Args[0]
	}
	return "sgai"
}

func buildBaseOpenCodeEnv(dir string) []string {
	env := slices.DeleteFunc(os.Environ(), func(e string) bool {
		return len(e) >= 4 && e[:4] == "PWD="
	})
	return append(env,
		"PWD="+dir,
		"OPENCODE_CONFIG_DIR="+filepath.Join(dir, ".sgai"))
}

func buildManagedOpenCodeEnv(dir, mcpURL, agentIdentity, interactiveEnv string) []string {
	configContent, errConfig := buildOpenCodeConfigContent(os.Getenv("OPENCODE_CONFIG_CONTENT"), sgaiExecutablePath(), mcpURL, agentIdentity)
	if errConfig != nil {
		logFatalConfigContent(errConfig)
	}

	return append(buildBaseOpenCodeEnv(dir),
		"OPENCODE_CONFIG_CONTENT="+configContent,
		"SGAI_MCP_URL="+mcpURL,
		"SGAI_AGENT_IDENTITY="+agentIdentity,
		"SGAI_MCP_INTERACTIVE="+interactiveEnv)
}

func logFatalConfigContent(err error) {
	fmt.Fprintln(os.Stderr, "failed to build OPENCODE_CONFIG_CONTENT:", err)
	os.Exit(1)
}
