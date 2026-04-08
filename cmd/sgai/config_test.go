package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultActionConfigs(t *testing.T) {
	configs := defaultActionConfigs()
	assert.Len(t, configs, 3)
	assert.Equal(t, "Create PR", configs[0].Name)
	assert.Equal(t, "Upstream Sync", configs[1].Name)
	assert.Equal(t, "Start Application", configs[2].Name)
}

func TestLoadProjectConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *projectConfig)
	}{
		{
			name: "validConfig",
			setupFunc: func(t *testing.T, dir string) {
				config := projectConfig{
					DefaultModel: "anthropic/claude-opus-4-6",
					Editor:       "code",
				}
				data, err := json.Marshal(config)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, config *projectConfig) {
				require.NotNil(t, config)
				assert.Equal(t, "anthropic/claude-opus-4-6", config.DefaultModel)
				assert.Equal(t, "code", config.Editor)
			},
		},
		{
			name: "noConfigFile",
			setupFunc: func(_ *testing.T, _ string) {
			},
			wantErr: false,
			validate: func(t *testing.T, config *projectConfig) {
				assert.Nil(t, config)
			},
		},
		{
			name: "invalidJSON",
			setupFunc: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), []byte("not valid json"), 0644))
			},
			wantErr:     true,
			errContains: "invalid JSON syntax",
		},
		{
			name: "configWithActions",
			setupFunc: func(t *testing.T, dir string) {
				config := projectConfig{
					DefaultModel: "anthropic/claude-opus-4-6",
					Actions: []actionConfig{
						{Name: "Test Action", Model: "test-model", Prompt: "test prompt"},
					},
				}
				data, err := json.Marshal(config)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, config *projectConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.Actions, 1)
				assert.Equal(t, "Test Action", config.Actions[0].Name)
			},
		},
		{
			name: "configWithMCP",
			setupFunc: func(t *testing.T, dir string) {
				config := projectConfig{
					DefaultModel: "anthropic/claude-opus-4-6",
					MCP: map[string]json.RawMessage{
						"test-server": json.RawMessage(`{"command": "test"}`),
					},
				}
				data, err := json.Marshal(config)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, config *projectConfig) {
				require.NotNil(t, config)
				assert.NotNil(t, config.MCP)
				assert.Contains(t, config.MCP, "test-server")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)

			config, err := loadProjectConfig(dir)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestValidateProjectConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *projectConfig
		wantErr     bool
		errContains string
	}{
		{
			name:    "nilConfig",
			config:  nil,
			wantErr: false,
		},
		{
			name: "emptyDefaultModel",
			config: &projectConfig{
				DefaultModel: "",
			},
			wantErr: false,
		},
		{
			name: "validDefaultModel",
			config: &projectConfig{
				DefaultModel: "anthropic/claude-opus-4-6",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config != nil && tt.config.DefaultModel != "" {
				if _, err := exec.LookPath("opencode"); err != nil {
					t.Skip("opencode not found in PATH")
				}
			}

			err := validateProjectConfig(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *projectConfig
		metadata *GoalMetadata
		validate func(*testing.T, *GoalMetadata)
	}{
		{
			name:   "nilConfig",
			config: nil,
			metadata: &GoalMetadata{
				Models: map[string]any{"agent1": "model1"},
			},
			validate: func(t *testing.T, m *GoalMetadata) {
				assert.Equal(t, "model1", m.Models["agent1"])
			},
		},
		{
			name: "emptyDefaultModel",
			config: &projectConfig{
				DefaultModel: "",
			},
			metadata: &GoalMetadata{
				Models: map[string]any{"agent1": "model1"},
			},
			validate: func(t *testing.T, m *GoalMetadata) {
				assert.Equal(t, "model1", m.Models["agent1"])
			},
		},
		{
			name: "applyDefaultToEmptyAgent",
			config: &projectConfig{
				DefaultModel: "default-model",
			},
			metadata: &GoalMetadata{
				Models: map[string]any{
					"agent1": "model1",
					"agent2": "",
				},
			},
			validate: func(t *testing.T, m *GoalMetadata) {
				assert.Equal(t, "model1", m.Models["agent1"])
			},
		},
		{
			name: "nilModelsMap",
			config: &projectConfig{
				DefaultModel: "default-model",
			},
			metadata: &GoalMetadata{},
			validate: func(t *testing.T, m *GoalMetadata) {
				assert.NotNil(t, m.Models)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyConfigDefaults(tt.config, tt.metadata)
			if tt.validate != nil {
				tt.validate(t, tt.metadata)
			}
		})
	}
}

func TestExtractMCPSection(t *testing.T) {
	tests := []struct {
		name        string
		oc          map[string]json.RawMessage
		wantErr     bool
		errContains string
		validate    func(*testing.T, map[string]json.RawMessage)
	}{
		{
			name: "noMCPSection",
			oc:   map[string]json.RawMessage{},
			validate: func(t *testing.T, mcp map[string]json.RawMessage) {
				assert.Empty(t, mcp)
			},
		},
		{
			name: "validMCPSection",
			oc: map[string]json.RawMessage{
				"mcp": json.RawMessage(`{"server1": {"command": "test"}}`),
			},
			validate: func(t *testing.T, mcp map[string]json.RawMessage) {
				assert.Contains(t, mcp, "server1")
			},
		},
		{
			name: "invalidMCPSection",
			oc: map[string]json.RawMessage{
				"mcp": json.RawMessage(`not valid json`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcp, err := extractMCPSection(tt.oc)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, mcp)
			}
		})
	}
}

func TestApplyCustomMCPs(t *testing.T) {
	tests := []struct {
		name      string
		config    *projectConfig
		setupFunc func(*testing.T, string)
		wantErr   bool
		validate  func(*testing.T, string)
	}{
		{
			name:   "nilConfig",
			config: nil,
			setupFunc: func(_ *testing.T, _ string) {
			},
			wantErr: false,
		},
		{
			name: "emptyMCP",
			config: &projectConfig{
				MCP: map[string]json.RawMessage{},
			},
			setupFunc: func(_ *testing.T, _ string) {
			},
			wantErr: false,
		},
		{
			name: "noOpencodeFile",
			config: &projectConfig{
				MCP: map[string]json.RawMessage{
					"test-server": json.RawMessage(`{"command": "test"}`),
				},
			},
			setupFunc: func(_ *testing.T, _ string) {
			},
			wantErr: true,
		},
		{
			name: "addNewMCP",
			config: &projectConfig{
				MCP: map[string]json.RawMessage{
					"new-server": json.RawMessage(`{"command": "new"}`),
				},
			},
			setupFunc: func(t *testing.T, dir string) {
				sgaiDir := filepath.Join(dir, ".sgai")
				require.NoError(t, os.MkdirAll(sgaiDir, 0755))
				opencodeContent := `{"mcp": {}}`
				require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "opencode.jsonc"), []byte(opencodeContent), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".sgai", "opencode.jsonc"))
				require.NoError(t, err)
				var oc map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(data, &oc))
				assert.Contains(t, oc, "mcp")
			},
		},
		{
			name: "existingMCP",
			config: &projectConfig{
				MCP: map[string]json.RawMessage{
					"existing-server": json.RawMessage(`{"command": "updated"}`),
				},
			},
			setupFunc: func(t *testing.T, dir string) {
				sgaiDir := filepath.Join(dir, ".sgai")
				require.NoError(t, os.MkdirAll(sgaiDir, 0755))
				opencodeContent := `{"mcp": {"existing-server": {"command": "original"}}}`
				require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "opencode.jsonc"), []byte(opencodeContent), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".sgai", "opencode.jsonc"))
				require.NoError(t, err)
				var oc map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(data, &oc))
				mcpRaw := oc["mcp"]
				var mcp map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(mcpRaw, &mcp))
				assert.Contains(t, mcp, "existing-server")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			err := applyCustomMCPs(dir, tt.config)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, dir)
			}
		})
	}
}

func TestResolveBackend(t *testing.T) {
	t.Run("defaultsToOpencode", func(t *testing.T) {
		b := resolveBackend(nil)
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("emptyBackendDefaultsToOpencode", func(t *testing.T) {
		config := &projectConfig{}
		b := resolveBackend(config)
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("explicitOpencode", func(t *testing.T) {
		config := &projectConfig{Backend: "opencode"}
		b := resolveBackend(config)
		assert.Equal(t, "opencode", b.Name())
	})
	t.Run("claudeCode", func(t *testing.T) {
		config := &projectConfig{Backend: "claude-code"}
		b := resolveBackend(config)
		assert.Equal(t, "claude-code", b.Name())
	})
}

func TestResolveBackendStrict(t *testing.T) {
	t.Run("nilConfig", func(t *testing.T) {
		b, err := resolveBackendStrict(nil)
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

func TestLoadProjectConfigBackendField(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"backend": "claude-code", "defaultModel": "anthropic/claude-opus-4-6"}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))

	config, err := loadProjectConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "claude-code", config.Backend)

	b := resolveBackend(config)
	assert.Equal(t, "claude-code", b.Name())
}

func TestLoadProjectConfigTypeError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), []byte(`{"editor": 12345}`), 0644))
	_, err := loadProjectConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON type")
}

func TestLoadProjectConfigPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)
	require.NoError(t, os.WriteFile(configPath, []byte(`{}`), 0000))
	t.Cleanup(func() { _ = os.Chmod(configPath, 0644) })
	_, err := loadProjectConfig(dir)
	if err != nil {
		assert.Contains(t, err.Error(), "permission denied")
	}
}

func TestApplyCustomMCPsInvalidOpencodeJSON(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "opencode.jsonc"), []byte("not valid json"), 0644))
	cfg := &projectConfig{
		MCP: map[string]json.RawMessage{
			"test-server": json.RawMessage(`{"command": "test"}`),
		},
	}
	err := applyCustomMCPs(dir, cfg)
	assert.Error(t, err)
}

func TestApplyCustomMCPsInvalidMCPSection(t *testing.T) {
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "opencode.jsonc"), []byte(`{"mcp": "not a map"}`), 0644))
	cfg := &projectConfig{
		MCP: map[string]json.RawMessage{
			"test-server": json.RawMessage(`{"command": "test"}`),
		},
	}
	err := applyCustomMCPs(dir, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting mcp section")
}
