package main

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"
)

// GoalMetadata represents the YAML frontmatter in GOAL.md files.
// It configures workflow flow, per-agent models, and interactive mode.
// Models can be either a single string or an array of strings per agent
// (for multi-model support).
type GoalMetadata struct {
	Flow                 string         `json:"flow,omitempty" yaml:"flow,omitempty"`
	Models               map[string]any `json:"models,omitempty" yaml:"models,omitempty"`
	Interactive          string         `json:"interactive,omitempty" yaml:"interactive,omitempty"`
	CompletionGateScript string         `json:"completionGateScript,omitempty" yaml:"completionGateScript,omitempty"`
}

type agentMetadata struct {
	Log      bool     `json:"log" yaml:"log"`
	Snippets []string `json:"snippets" yaml:"snippets"`
}

func shouldLogAgent(dir, agentName string) bool {
	agentPath := filepath.Join(dir, ".sgai", "agent", agentName+".md")
	content, err := os.ReadFile(agentPath)
	if err != nil {
		return true
	}

	delimiter := []byte("---")
	if !bytes.HasPrefix(content, delimiter) {
		return true
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return true
	}

	yamlContent := before
	var metadata agentMetadata
	metadata.Log = true
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return true
	}

	return metadata.Log
}

func parseAgentSnippets(dir, agentName string) []string {
	agentPath := filepath.Join(dir, ".sgai", "agent", agentName+".md")
	content, err := os.ReadFile(agentPath)
	if err != nil {
		return nil
	}

	delimiter := []byte("---")
	if !bytes.HasPrefix(content, delimiter) {
		return nil
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return nil
	}

	yamlContent := before
	var metadata agentMetadata
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return nil
	}

	return metadata.Snippets
}

func parseFrontmatterMap(content []byte) map[string]string {
	result := make(map[string]string)
	delimiter := []byte("---")

	if !bytes.HasPrefix(content, delimiter) {
		return result
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return result
	}

	yamlContent := before

	for line := range bytes.SplitSeq(yamlContent, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if colonIdx := bytes.IndexByte(trimmed, ':'); colonIdx > 0 {
			key := string(bytes.TrimSpace(trimmed[:colonIdx]))
			value := string(bytes.TrimSpace(trimmed[colonIdx+1:]))
			result[key] = value
		}
	}

	return result
}

// validateModels checks that all agent models in the map are valid according to `opencode models`.
// Returns an error listing invalid models and agents if any are found.
// When model specs include variants (e.g., "model (variant)"), only the base model is validated.
// Supports both single string models and arrays of model strings.
func validateModels(models map[string]any) error {
	if len(models) == 0 {
		return nil
	}

	validModels, err := fetchValidModels()
	if err != nil {
		return fmt.Errorf("failed to fetch valid models: %w", err)
	}

	var invalidAgents []string
	var invalidModelNames []string
	seen := make(map[string]bool)

	for agent := range models {
		modelSpecs := getModelsForAgent(models, agent)
		for _, modelSpec := range modelSpecs {
			if modelSpec == "" {
				continue
			}
			baseModel, _ := parseModelAndVariant(modelSpec)
			if !validModels[baseModel] {
				invalidAgents = append(invalidAgents, agent)
				if !seen[baseModel] {
					invalidModelNames = append(invalidModelNames, baseModel)
					seen[baseModel] = true
				}
			}
		}
	}

	if len(invalidAgents) > 0 {
		slices.Sort(invalidAgents)
		slices.Sort(invalidModelNames)

		validModelList := slices.Sorted(maps.Keys(validModels))

		return fmt.Errorf("invalid model(s) specified:\n  agents: %s\n  invalid models: %s\n  valid models: %s",
			strings.Join(invalidAgents, ", "),
			strings.Join(invalidModelNames, ", "),
			strings.Join(validModelList, ", "))
	}

	return nil
}

func fetchValidModels() (map[string]bool, error) {
	cmd := exec.Command("opencode", "models")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("opencode models command failed: %w", err)
	}

	validModels := make(map[string]bool)
	for line := range strings.SplitSeq(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			validModels[trimmed] = true
		}
	}

	return validModels, nil
}

// tryReloadGoalMetadata attempts to reload GOAL.md frontmatter from disk.
// If reload succeeds, returns new metadata; if it fails, returns current unchanged.
func tryReloadGoalMetadata(goalPath string, current GoalMetadata) GoalMetadata {
	content, err := os.ReadFile(goalPath)
	if err != nil {
		return current
	}

	newMetadata, err := parseYAMLFrontmatter(content)
	if err != nil {
		return current
	}

	return newMetadata
}

// parseYAMLFrontmatter extracts YAML frontmatter from content delimited by "---".
// If no frontmatter is found, returns default metadata.
func parseYAMLFrontmatter(content []byte) (GoalMetadata, error) {
	delimiter := []byte("---")
	defaultMetadata := GoalMetadata{}

	if !bytes.HasPrefix(content, delimiter) {
		return defaultMetadata, nil
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, _, ok := bytes.Cut(rest, delimiter)
	if !ok {
		return GoalMetadata{}, fmt.Errorf("no closing '---' found for frontmatter")
	}

	yamlContent := before

	var metadata GoalMetadata
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return GoalMetadata{}, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return metadata, nil
}

func extractFrontmatterDescription(content string) string {
	fm := parseFrontmatterMap([]byte(content))
	return fm["description"]
}
