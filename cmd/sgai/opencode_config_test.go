package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOpenCodeConfig struct {
	Username string                     `json:"username"`
	MCP      map[string]testOpenCodeMCP `json:"mcp"`
}

type testOpenCodeMCP struct {
	Type    string   `json:"type"`
	Enabled bool     `json:"enabled"`
	Timeout int      `json:"timeout"`
	Command []string `json:"command"`
}

func TestBuildOpenCodeConfigContentAddsLocalSGAIMCP(t *testing.T) {
	content, err := buildOpenCodeConfigContent(`{"username":"dev","mcp":{"context7":{"type":"local","command":["npx"]}}}`, "/bin/sgai", "http://127.0.0.1:1234/mcp", "builder|model|variant")
	require.NoError(t, err)

	var config testOpenCodeConfig
	require.NoError(t, json.Unmarshal([]byte(content), &config))
	assert.Equal(t, "dev", config.Username)

	assert.Contains(t, config.MCP, "context7")
	sgai := config.MCP["sgai"]
	assert.Equal(t, "local", sgai.Type)
	assert.True(t, sgai.Enabled)
	assert.Equal(t, opencodeMCPTimeout, sgai.Timeout)
	assert.Equal(t, []string{"/bin/sgai", "internal-mcp", "http://127.0.0.1:1234/mcp", "builder|model|variant"}, sgai.Command)
}

func TestBuildManagedOpenCodeEnvIncludesConfigContent(t *testing.T) {
	t.Setenv("OPENCODE_CONFIG_CONTENT", `{"username":"dev"}`)
	env := buildManagedOpenCodeEnv("/tmp/workspace", "http://127.0.0.1:1234/mcp", "agent", "auto")
	envMap := envToMap(env)

	assert.Equal(t, filepath.Join("/tmp/workspace", ".sgai"), envMap["OPENCODE_CONFIG_DIR"])
	assert.Equal(t, "http://127.0.0.1:1234/mcp", envMap["SGAI_MCP_URL"])
	assert.Equal(t, "agent", envMap["SGAI_AGENT_IDENTITY"])
	assert.Equal(t, "auto", envMap["SGAI_MCP_INTERACTIVE"])

	var config testOpenCodeConfig
	require.NoError(t, json.Unmarshal([]byte(envMap["OPENCODE_CONFIG_CONTENT"]), &config))
	sgai := config.MCP["sgai"]
	assert.Equal(t, "local", sgai.Type)
	assert.Equal(t, "internal-mcp", sgai.Command[1])
}

func envToMap(env []string) map[string]string {
	result := make(map[string]string)
	for _, entry := range env {
		for i := 0; i < len(entry); i++ {
			if entry[i] == '=' {
				result[entry[:i]] = entry[i+1:]
				break
			}
		}
	}
	return result
}
