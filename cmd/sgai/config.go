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
			Model:       "openai/gpt-5.5 (xhigh)",
			Prompt:      "copy GOAL.md into GOALS/ following the instructions from README.md; store the git path (by querying jj) into GIT_DIR, and using GH, make a draft PR for the commit at @ (jj); CRITICAL: commit message, the PR title and body, must adhere to the standard of previous commits - update all of these if necessary; once you are done, using bash(`open`), open the PR for me.",
			Description: "Create a draft pull request from current changes",
		},
		{
			Name:        "Upstream Sync",
			Model:       "openai/gpt-5.5 (xhigh)",
			Prompt:      "`jj git fetch --all-remotes`; rebase against main@origin (`jj rebase -d main@origin`), fix merge conflicts, and push",
			Description: "Fetch and rebase against upstream main branch",
		},
		{
			Name:        "Start Application",
			Model:       "openai/gpt-5.5 (xhigh)",
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

	catalog, errModels := fetchValidModels()
	if errModels != nil {
		return fmt.Errorf("validating defaultModel in config file: %w", errModels)
	}

	if errValidate := validateModelSpec(catalog, config.DefaultModel); errValidate != nil {
		return fmt.Errorf("invalid defaultModel in config file: %w", errValidate)
	}

	return nil
}

func applyConfigDefaults(config *projectConfig, metadata *GoalMetadata) {
	if config == nil || config.DefaultModel == "" {
		return
	}

	if metadata.Model == "" {
		metadata.Model = config.DefaultModel
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

	var oc map[string]json.RawMessage
	if err := json.Unmarshal(data, &oc); err != nil {
		return fmt.Errorf("parsing opencode.jsonc: %w", err)
	}

	mcpSection, err := extractMCPSection(oc)
	if err != nil {
		return fmt.Errorf("extracting mcp section: %w", err)
	}

	var added bool
	for name, value := range config.MCP {
		if _, exists := mcpSection[name]; exists {
			continue
		}
		mcpSection[name] = value
		added = true
	}

	if !added {
		return nil
	}

	mcpRaw, err := json.Marshal(mcpSection)
	if err != nil {
		return fmt.Errorf("encoding mcp section: %w", err)
	}
	oc["mcp"] = mcpRaw

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

func extractMCPSection(oc map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	raw, exists := oc["mcp"]
	if !exists {
		return make(map[string]json.RawMessage), nil
	}
	var mcpSection map[string]json.RawMessage
	if err := json.Unmarshal(raw, &mcpSection); err != nil {
		return nil, err
	}
	return mcpSection, nil
}
