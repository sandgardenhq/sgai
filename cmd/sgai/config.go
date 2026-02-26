package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "sgai.json"

func defaultActionConfigs() []actionConfig {
	return []actionConfig{
		{
			Name:        "Create PR",
			Model:       "anthropic/claude-opus-4-6 (max)",
			Prompt:      "copy GOAL.md into GOALS/ following the instructions from README.md; store the git path (by querying jj) into GIT_DIR, and using GH, make a draft PR for the commit at @ (jj); CRITICAL: commit message, the PR title and body, must adhere to the standard of previous commits - update all of these if necessary; once you are done, using bash(`open`), open the PR for me.",
			Description: "Create a draft pull request from current changes",
		},
		{
			Name:        "Upstream Sync",
			Model:       "anthropic/claude-opus-4-6 (max)",
			Prompt:      "`jj git fetch --all-remotes`; rebase against main@origin (`jj rebase -d main@origin`), fix merge conflicts, and push",
			Description: "Fetch and rebase against upstream main branch",
		},
		{
			Name:        "Start Application",
			Model:       "anthropic/claude-opus-4-6 (max)",
			Prompt:      "start the application server and ensure it is running properly; use the instructions inside `.deploy/` if available; if this is a networked application, and it starts at localhost, use 'localhost:0' to randomize the application start.",
			Description: "Start the application locally",
		},
	}
}

type actionConfig struct {
	Name        string `json:"name"`
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	Description string `json:"description,omitempty"`
}

// projectConfig represents the sgai.json configuration file.
// The configuration file must be located at the project root, as a sibling to the .sgai directory.
type projectConfig struct {
	DefaultModel string                     `json:"defaultModel,omitempty"`
	MCP          map[string]json.RawMessage `json:"mcp,omitempty"`
	Editor       string                     `json:"editor,omitempty"`
	Actions      []actionConfig             `json:"actions,omitempty"`
}

func loadProjectConfig(dir string) (*projectConfig, error) {
	configPath := filepath.Join(dir, configFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading config file: %s", configPath)
		}
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	var config projectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		if errSyntax, ok := err.(*json.SyntaxError); ok {
			return nil, fmt.Errorf("invalid JSON syntax in config file %s at offset %d: %w",
				configPath, errSyntax.Offset, err)
		}
		if errUnmarshal, ok := err.(*json.UnmarshalTypeError); ok {
			return nil, fmt.Errorf("invalid JSON type in config file %s at field %s: expected %s, got %s",
				configPath, errUnmarshal.Field, errUnmarshal.Type, errUnmarshal.Value)
		}
		return nil, fmt.Errorf("parsing config file %s: %w", configPath, err)
	}

	return &config, nil
}

func validateProjectConfig(config *projectConfig) error {
	if config == nil {
		return nil
	}

	if config.DefaultModel == "" {
		return nil
	}

	validModels, err := fetchValidModels()
	if err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	baseModel, _ := parseModelAndVariant(config.DefaultModel)
	if !validModels[baseModel] {
		return fmt.Errorf("invalid defaultModel in config file: %s", config.DefaultModel)
	}

	return nil
}

func applyConfigDefaults(config *projectConfig, metadata *GoalMetadata) {
	if config == nil || config.DefaultModel == "" {
		return
	}

	if metadata.Models == nil {
		metadata.Models = make(map[string]any)
	}

	for agent := range metadata.Models {
		models := getModelsForAgent(metadata.Models, agent)
		if len(models) == 0 {
			metadata.Models[agent] = config.DefaultModel
		}
	}
}

func applyCustomMCPs(dir string, config *projectConfig) error {
	if config == nil || len(config.MCP) == 0 {
		return nil
	}

	opencodePath := filepath.Join(dir, ".sgai", "opencode.jsonc")
	data, err := os.ReadFile(opencodePath)
	if err != nil {
		return fmt.Errorf("reading opencode.jsonc: %w", err)
	}

	var oc opencodeConfig
	if err := json.Unmarshal(data, &oc); err != nil {
		return fmt.Errorf("parsing opencode.jsonc: %w", err)
	}

	if oc.MCP == nil {
		oc.MCP = make(map[string]json.RawMessage)
	}

	var added bool
	for name, value := range config.MCP {
		if _, exists := oc.MCP[name]; exists {
			continue
		}
		oc.MCP[name] = value
		added = true
	}

	if !added {
		return nil
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "\t")
	if err := enc.Encode(oc); err != nil {
		return fmt.Errorf("encoding opencode.jsonc: %w", err)
	}

	if err := os.WriteFile(opencodePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing opencode.jsonc: %w", err)
	}

	return nil
}

type opencodeConfig struct {
	Schema string                     `json:"$schema"`
	MCP    map[string]json.RawMessage `json:"mcp"`
}
