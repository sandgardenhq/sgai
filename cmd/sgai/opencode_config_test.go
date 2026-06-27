package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOpenCodeConfig struct {
	Username string                             `json:"username"`
	MCP      map[string]testOpenCodeMCP         `json:"mcp"`
	Agent    map[string]testOpenCodeAgentConfig `json:"agent"`
}

type testOpenCodeAgentConfig struct {
	Mode       string                     `json:"mode"`
	Permission map[string]json.RawMessage `json:"permission"`
}

type testOpenCodeMCP struct {
	Type    string   `json:"type"`
	Enabled bool     `json:"enabled"`
	Timeout int      `json:"timeout"`
	Command []string `json:"command"`
}

func TestBuildOpenCodeConfigContentAddsLocalSGAIMCP(t *testing.T) {
	content, err := buildOpenCodeConfigContent(`{"username":"dev","mcp":{"context7":{"type":"local","command":["npx"]}}}`, "/bin/sgai", "http://127.0.0.1:1234/mcp", "builder|model|variant", nil)
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
	env := buildManagedOpenCodeEnv("/tmp/workspace", "http://127.0.0.1:1234/mcp", "agent", "auto", nil)
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

func TestBuildOpenCodeConfigContentAddsCoordinatorTaskPolicy(t *testing.T) {
	content, err := buildOpenCodeConfigContent(`{"agent":{"coordinator":{"mode":"primary","permission":{"edit":{"*":"deny"}}},"go":{"mode":"primary"}}}`, "/bin/sgai", "http://127.0.0.1:1234/mcp", "coordinator|model|variant", []string{"go", "react", "project-critic"})
	require.NoError(t, err)

	var config testOpenCodeConfig
	require.NoError(t, json.Unmarshal([]byte(content), &config))

	coordinator := config.Agent["coordinator"]
	assert.Equal(t, "primary", coordinator.Mode)
	require.Contains(t, coordinator.Permission, "edit")
	require.Contains(t, coordinator.Permission, "task")

	var taskPolicy map[string]string
	require.NoError(t, json.Unmarshal(coordinator.Permission["task"], &taskPolicy))
	assert.Equal(t, map[string]string{
		"*":              "deny",
		"go":             "allow",
		"react":          "allow",
		"project-critic": "allow",
	}, taskPolicy)
	assert.Less(t, strings.Index(content, `"*":"deny"`), strings.Index(content, `"go":"allow"`))
	assert.Contains(t, config.Agent, "go")
}

func TestCoordinatorTaskTargetsExcludesCoordinatorAndAlwaysIncludesProjectCritic(t *testing.T) {
	targets := coordinatorTaskTargets([]string{"coordinator", "go", "project-critic", "go", "stpa-analyst", "react"})

	assert.Equal(t, []string{"go", "project-critic", "react"}, targets)
}

func TestBuildAgentEnvAddsCoordinatorTaskPolicyOnlyForCoordinator(t *testing.T) {
	t.Setenv("OPENCODE_CONFIG_CONTENT", `{"username":"dev"}`)
	coordinatorEnv := envToMap(buildAgentEnv(agentRunConfig{dir: "/tmp/workspace", agent: "coordinator", mcpURL: "http://127.0.0.1:1234/mcp", goalAgents: []string{"coordinator", "go"}}, state.Workflow{}, ""))
	nonCoordinatorEnv := envToMap(buildAgentEnv(agentRunConfig{dir: "/tmp/workspace", agent: "go", mcpURL: "http://127.0.0.1:1234/mcp", goalAgents: []string{"go"}}, state.Workflow{}, ""))

	var coordinatorConfig testOpenCodeConfig
	require.NoError(t, json.Unmarshal([]byte(coordinatorEnv["OPENCODE_CONFIG_CONTENT"]), &coordinatorConfig))
	require.Contains(t, coordinatorConfig.Agent, "coordinator")
	var taskPolicy map[string]string
	require.NoError(t, json.Unmarshal(coordinatorConfig.Agent["coordinator"].Permission["task"], &taskPolicy))
	assert.Equal(t, "allow", taskPolicy["go"])
	assert.Equal(t, "allow", taskPolicy["project-critic"])
	assert.NotContains(t, taskPolicy, "coordinator")

	var nonCoordinatorConfig testOpenCodeConfig
	require.NoError(t, json.Unmarshal([]byte(nonCoordinatorEnv["OPENCODE_CONFIG_CONTENT"]), &nonCoordinatorConfig))
	assert.NotContains(t, nonCoordinatorConfig.Agent, "coordinator")
}

func TestCoordinatorSkeletonDoesNotHardcodeTaskPolicy(t *testing.T) {
	content, errRead := os.ReadFile(filepath.Join("skel", ".sgai", "agent", "coordinator.md"))
	require.NoError(t, errRead)

	assert.NotContains(t, string(content), "  task:\n")
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
