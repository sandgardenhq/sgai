package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendSelection(t *testing.T) {
	t.Run("nilConfig", func(t *testing.T) {
		b := resolveBackend(nil)
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("emptyConfig", func(t *testing.T) {
		b := resolveBackend(&projectConfig{})
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("explicitOpencode", func(t *testing.T) {
		b := resolveBackend(&projectConfig{Backend: "opencode"})
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("claudeCode", func(t *testing.T) {
		b := resolveBackend(&projectConfig{Backend: "claude-code"})
		assert.Equal(t, "claude-code", b.Name())
	})
}

func TestBackendStrictValidation(t *testing.T) {
	t.Run("nilConfig", func(t *testing.T) {
		b, err := resolveBackendStrict(nil)
		require.NoError(t, err)
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("emptyBackend", func(t *testing.T) {
		b, err := resolveBackendStrict(&projectConfig{})
		require.NoError(t, err)
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("validClaudeCode", func(t *testing.T) {
		b, err := resolveBackendStrict(&projectConfig{Backend: "claude-code"})
		require.NoError(t, err)
		assert.Equal(t, "claude-code", b.Name())
	})
	t.Run("invalidBackend", func(t *testing.T) {
		_, err := resolveBackendStrict(&projectConfig{Backend: "invalid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported backend")
	})
}

func TestBothBackendsProduceValidAgentArgs(t *testing.T) {
	backends := []Backend{&opencodeBackend{}, &claudeCodeBackend{}}
	params := AgentRunParams{
		Agent:     "coordinator",
		BaseAgent: "coordinator",
		ModelSpec: "anthropic/claude-opus-4-6 (max)",
		SessionID: "session-123",
	}

	for _, b := range backends {
		t.Run(b.Name(), func(t *testing.T) {
			args := b.BuildAgentArgs(params)
			assert.NotEmpty(t, args)
			// All backends should produce a --name or --title with the agent name
			found := false
			for _, arg := range args {
				if arg == "coordinator [anthropic/claude-opus-4-6 (max)]" {
					found = true
					break
				}
			}
			assert.True(t, found, "expected agent title in args for %s backend", b.Name())
		})
	}
}

func TestBothBackendsProduceValidAdhocArgs(t *testing.T) {
	backends := []Backend{&opencodeBackend{}, &claudeCodeBackend{}}
	modelSpec := "anthropic/claude-opus-4-6 (max)"

	for _, b := range backends {
		t.Run(b.Name(), func(t *testing.T) {
			args := b.BuildAdhocArgs(modelSpec)
			assert.NotEmpty(t, args)
			// All backends should produce args with the model
			hasModel := false
			for _, arg := range args {
				if arg == "--model" || arg == "-m" {
					hasModel = true
					break
				}
			}
			assert.True(t, hasModel, "expected model flag in adhoc args for %s backend", b.Name())
		})
	}
}

func TestBothBackendsProduceValidEnv(t *testing.T) {
	backends := []Backend{&opencodeBackend{}, &claudeCodeBackend{}}
	params := AgentEnvParams{
		Dir:             "/tmp/workspace",
		McpURL:          "http://localhost:8080/mcp",
		AgentIdentity:   "coordinator",
		InteractiveMode: "yes",
	}

	for _, b := range backends {
		t.Run(b.Name(), func(t *testing.T) {
			env := b.BuildEnv(params)
			assert.NotEmpty(t, env)

			envMap := make(map[string]string)
			for _, e := range env {
				for i := 0; i < len(e); i++ {
					if e[i] == '=' {
						envMap[e[:i]] = e[i+1:]
						break
					}
				}
			}

			// Both backends should set SGAI_MCP_URL
			assert.Equal(t, "http://localhost:8080/mcp", envMap["SGAI_MCP_URL"])
			assert.Equal(t, "coordinator", envMap["SGAI_AGENT_IDENTITY"])
			assert.Equal(t, "yes", envMap["SGAI_MCP_INTERACTIVE"])
		})
	}
}

func TestBothBackendsParseOwnEvents(t *testing.T) {
	t.Run("opencode", func(t *testing.T) {
		b := &opencodeBackend{}
		line := []byte(`{"type":"text","part":{"text":"hello"},"session_id":"s1"}`)
		event, ok := b.ParseEvent(line)
		assert.True(t, ok)
		assert.Equal(t, "text", event.Type)
		assert.Equal(t, "hello", event.Part.Text)
	})
	t.Run("claudeCode", func(t *testing.T) {
		b := &claudeCodeBackend{}
		line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]},"session_id":"s1"}`)
		event, ok := b.ParseEvent(line)
		assert.True(t, ok)
		assert.Equal(t, "text", event.Type)
		assert.Equal(t, "hello", event.Part.Text)
	})
}

func TestBothBackendsHandleInvalidEvents(t *testing.T) {
	backends := []Backend{&opencodeBackend{}, &claudeCodeBackend{}}
	for _, b := range backends {
		t.Run(b.Name(), func(t *testing.T) {
			_, ok := b.ParseEvent([]byte(`not json`))
			assert.False(t, ok)
		})
	}
}

func TestBackendBinaryNames(t *testing.T) {
	assert.Equal(t, "opencode", (&opencodeBackend{}).BinaryName())
	assert.Equal(t, "claude", (&claudeCodeBackend{}).BinaryName())
}

func TestBackendStripProviderPrefix(t *testing.T) {
	oc := &opencodeBackend{}
	cc := &claudeCodeBackend{}

	// opencode keeps the prefix
	assert.Equal(t, "anthropic/claude-opus-4-6", oc.StripProviderPrefix("anthropic/claude-opus-4-6"))

	// claude-code strips the prefix
	assert.Equal(t, "claude-opus-4-6", cc.StripProviderPrefix("anthropic/claude-opus-4-6"))

	// Both handle no-prefix models
	assert.Equal(t, "claude-opus-4-6", oc.StripProviderPrefix("claude-opus-4-6"))
	assert.Equal(t, "claude-opus-4-6", cc.StripProviderPrefix("claude-opus-4-6"))
}
