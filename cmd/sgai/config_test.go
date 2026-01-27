package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectConfigMissingFile(t *testing.T) {
	dir := t.TempDir()

	config, err := loadProjectConfig(dir)
	if err != nil {
		t.Errorf("loadProjectConfig() error = %v; want nil for missing file", err)
	}
	if config != nil {
		t.Errorf("loadProjectConfig() = %v; want nil for missing file", config)
	}
}

func TestLoadProjectConfigValidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	content := `{"defaultModel": "anthropic/claude-opus-4-5"}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := loadProjectConfig(dir)
	switch {
	case err != nil:
		t.Fatalf("loadProjectConfig() error = %v; want nil", err)
	case config == nil:
		t.Fatal("loadProjectConfig() = nil; want non-nil config")
	case config.DefaultModel != "anthropic/claude-opus-4-5":
		t.Errorf("config.DefaultModel = %q; want %q", config.DefaultModel, "anthropic/claude-opus-4-5")
	}
}

func TestLoadProjectConfigEmptyJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	content := `{}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := loadProjectConfig(dir)
	switch {
	case err != nil:
		t.Fatalf("loadProjectConfig() error = %v; want nil", err)
	case config == nil:
		t.Fatal("loadProjectConfig() = nil; want non-nil config")
	case config.DefaultModel != "":
		t.Errorf("config.DefaultModel = %q; want empty string", config.DefaultModel)
	}
}

func TestLoadProjectConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	content := `{invalid json}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadProjectConfig(dir)
	if err == nil {
		t.Fatal("loadProjectConfig() error = nil; want error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON syntax") {
		t.Errorf("error = %v; want error containing 'invalid JSON syntax'", err)
	}
}

func TestLoadProjectConfigWrongType(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	content := `{"defaultModel": 123}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadProjectConfig(dir)
	if err == nil {
		t.Error("loadProjectConfig() error = nil; want error for wrong type")
	}
}

func TestApplyConfigDefaultsWithNilConfig(t *testing.T) {
	metadata := GoalMetadata{
		Models: map[string]any{"agent1": ""},
	}
	applyConfigDefaults(nil, &metadata)

	models := getModelsForAgent(metadata.Models, "agent1")
	if len(models) != 0 {
		t.Errorf("agent1 models = %v; want empty (should not change)", models)
	}
}

func TestApplyConfigDefaultsWithEmptyDefaultModel(t *testing.T) {
	config := &projectConfig{DefaultModel: ""}
	metadata := GoalMetadata{
		Models: map[string]any{"agent1": ""},
	}
	applyConfigDefaults(config, &metadata)

	models := getModelsForAgent(metadata.Models, "agent1")
	if len(models) != 0 {
		t.Errorf("agent1 models = %v; want empty", models)
	}
}

func TestApplyConfigDefaultsAppliesDefault(t *testing.T) {
	config := &projectConfig{DefaultModel: "anthropic/claude-opus-4-5"}
	metadata := GoalMetadata{
		Models: map[string]any{
			"agent1": "",
			"agent2": "openai/gpt-4",
		},
	}
	applyConfigDefaults(config, &metadata)

	models1 := getModelsForAgent(metadata.Models, "agent1")
	if len(models1) != 1 || models1[0] != "anthropic/claude-opus-4-5" {
		t.Errorf("agent1 models = %v; want [anthropic/claude-opus-4-5]", models1)
	}
	models2 := getModelsForAgent(metadata.Models, "agent2")
	if len(models2) != 1 || models2[0] != "openai/gpt-4" {
		t.Errorf("agent2 models = %v; want [openai/gpt-4] (should not change)", models2)
	}
}

func TestApplyConfigDefaultsNilModelsMap(t *testing.T) {
	config := &projectConfig{DefaultModel: "anthropic/claude-opus-4-5"}
	metadata := GoalMetadata{}

	applyConfigDefaults(config, &metadata)

	if metadata.Models == nil {
		t.Error("metadata.Models = nil; want initialized map")
	}
}

func TestLoadProjectConfigWithMCP(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, configFileName)

	content := `{
		"mcp": {
			"my-mcp": {"type": "local", "command": ["npx", "my-tool"], "enabled": true}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := loadProjectConfig(dir)
	switch {
	case err != nil:
		t.Fatalf("loadProjectConfig() error = %v; want nil", err)
	case config == nil:
		t.Fatal("loadProjectConfig() = nil; want non-nil config")
	case len(config.MCP) != 1:
		t.Fatalf("len(config.MCP) = %d; want 1", len(config.MCP))
	}

	if _, exists := config.MCP["my-mcp"]; !exists {
		t.Error("config.MCP[\"my-mcp\"] not found; want present")
	}
}

func TestApplyCustomMCPsNilConfig(t *testing.T) {
	dir := t.TempDir()
	if err := applyCustomMCPs(dir, nil); err != nil {
		t.Errorf("applyCustomMCPs(nil) = %v; want nil", err)
	}
}

func TestApplyCustomMCPsEmptyMCP(t *testing.T) {
	dir := t.TempDir()
	config := &projectConfig{}
	if err := applyCustomMCPs(dir, config); err != nil {
		t.Errorf("applyCustomMCPs(empty) = %v; want nil", err)
	}
}

func TestApplyCustomMCPsAddsNew(t *testing.T) {
	dir := t.TempDir()
	setupOpencodeConfig(t, dir)

	config := &projectConfig{
		MCP: map[string]json.RawMessage{
			"my-custom": json.RawMessage(`{"type":"local","command":["npx","my-tool"],"enabled":true}`),
		},
	}

	if err := applyCustomMCPs(dir, config); err != nil {
		t.Fatalf("applyCustomMCPs() = %v; want nil", err)
	}

	oc := readOpencodeConfig(t, dir)
	if _, exists := oc.MCP["my-custom"]; !exists {
		t.Error("MCP[\"my-custom\"] not found after apply")
	}
	if _, exists := oc.MCP["playwright"]; !exists {
		t.Error("MCP[\"playwright\"] missing after apply")
	}
	if _, exists := oc.MCP["context7"]; !exists {
		t.Error("MCP[\"context7\"] missing after apply")
	}
}

func TestApplyCustomMCPsDoesNotOverrideDefaults(t *testing.T) {
	dir := t.TempDir()
	setupOpencodeConfig(t, dir)

	config := &projectConfig{
		MCP: map[string]json.RawMessage{
			"playwright": json.RawMessage(`{"type":"local","command":["custom-playwright"],"enabled":false}`),
		},
	}

	if err := applyCustomMCPs(dir, config); err != nil {
		t.Fatalf("applyCustomMCPs() = %v; want nil", err)
	}

	oc := readOpencodeConfig(t, dir)

	var playwrightConfig struct {
		Command []string `json:"command"`
	}
	if err := json.Unmarshal(oc.MCP["playwright"], &playwrightConfig); err != nil {
		t.Fatalf("unmarshal playwright config: %v", err)
	}

	if len(playwrightConfig.Command) == 0 {
		t.Fatal("playwright command is empty")
	}
	if playwrightConfig.Command[0] == "custom-playwright" {
		t.Error("playwright was overridden; want default preserved")
	}
}

func TestApplyCustomMCPsMultiple(t *testing.T) {
	dir := t.TempDir()
	setupOpencodeConfig(t, dir)

	config := &projectConfig{
		MCP: map[string]json.RawMessage{
			"mcp-a": json.RawMessage(`{"type":"local","command":["tool-a"]}`),
			"mcp-b": json.RawMessage(`{"type":"local","command":["tool-b"]}`),
			"mcp-c": json.RawMessage(`{"type":"local","command":["tool-c"]}`),
		},
	}

	if err := applyCustomMCPs(dir, config); err != nil {
		t.Fatalf("applyCustomMCPs() = %v; want nil", err)
	}

	oc := readOpencodeConfig(t, dir)

	for _, name := range []string{"mcp-a", "mcp-b", "mcp-c", "playwright", "context7"} {
		if _, exists := oc.MCP[name]; !exists {
			t.Errorf("MCP[%q] not found after apply", name)
		}
	}
}

func TestApplyCustomMCPsNoWriteWhenAllExist(t *testing.T) {
	dir := t.TempDir()
	setupOpencodeConfig(t, dir)

	opencodePath := filepath.Join(dir, ".sgai", "opencode.jsonc")
	infoBefore, err := os.Stat(opencodePath)
	if err != nil {
		t.Fatal(err)
	}

	config := &projectConfig{
		MCP: map[string]json.RawMessage{
			"playwright": json.RawMessage(`{"type":"local","command":["custom"]}`),
			"context7":   json.RawMessage(`{"type":"local","command":["custom"]}`),
		},
	}

	if err := applyCustomMCPs(dir, config); err != nil {
		t.Fatalf("applyCustomMCPs() = %v; want nil", err)
	}

	infoAfter, err := os.Stat(opencodePath)
	if err != nil {
		t.Fatal(err)
	}

	if infoAfter.ModTime() != infoBefore.ModTime() {
		t.Error("file was modified; want no modification when all MCPs already exist")
	}
}

func setupOpencodeConfig(t *testing.T, dir string) {
	t.Helper()
	factoraDir := filepath.Join(dir, ".sgai")
	if err := os.MkdirAll(factoraDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `{
	"$schema": "https://opencode.ai/config.json",
	"mcp": {
		"playwright": {
			"type": "local",
			"command": ["npx", "@playwright/mcp@latest"]
		},
		"context7": {
			"type": "local",
			"command": ["npx", "-y", "@upstash/context7-mcp"],
			"enabled": true
		}
	}
}`
	opencodePath := filepath.Join(factoraDir, "opencode.jsonc")
	if err := os.WriteFile(opencodePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readOpencodeConfig(t *testing.T, dir string) opencodeConfig {
	t.Helper()
	opencodePath := filepath.Join(dir, ".sgai", "opencode.jsonc")
	data, err := os.ReadFile(opencodePath)
	if err != nil {
		t.Fatalf("reading opencode.jsonc: %v", err)
	}
	var oc opencodeConfig
	if err := json.Unmarshal(data, &oc); err != nil {
		t.Fatalf("parsing opencode.jsonc: %v", err)
	}
	return oc
}
