package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"sigs.k8s.io/yaml"
)

var modelVariantPattern = regexp.MustCompile(`^(.+?)\s*\(([^)]+)\)$`)

func parseModelAndVariant(modelSpec string) (model, variant string) {
	matches := modelVariantPattern.FindStringSubmatch(modelSpec)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return modelSpec, ""
}

// GoalMetadata represents the YAML frontmatter in GOAL.md files.
// It configures available agents, model selection, and workflow options.
type GoalMetadata struct {
	Agents               []string `json:"agents,omitempty" yaml:"agents,omitempty"`
	Model                string   `json:"model,omitempty" yaml:"model,omitempty"`
	Interactive          string   `json:"interactive,omitempty" yaml:"interactive,omitempty"`
	CompletionGateScript string   `json:"completionGateScript,omitempty" yaml:"completionGateScript,omitempty"`
	ContinuousModePrompt string   `json:"continuousModePrompt,omitempty" yaml:"continuousModePrompt,omitempty"`
	ContinuousModeAuto   string   `json:"continuousModeAuto,omitempty" yaml:"continuousModeAuto,omitempty"`
	ContinuousModeCron   string   `json:"continuousModeCron,omitempty" yaml:"continuousModeCron,omitempty"`
	Retrospective        string   `json:"retrospective,omitempty" yaml:"retrospective,omitempty"`
}

type agentMetadata struct {
	Log      bool     `json:"log" yaml:"log"`
	Snippets []string `json:"snippets" yaml:"snippets"`
}

func parseYAMLFrontmatterFromFile(goalPath string) (GoalMetadata, error) {
	content, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return GoalMetadata{}, nil
		}
		return GoalMetadata{}, fmt.Errorf("failed to read GOAL.md: %w", errRead)
	}

	return parseYAMLFrontmatter(content)
}

func parseYAMLFrontmatter(content []byte) (GoalMetadata, error) {
	yamlContent, ok := splitFrontmatter(content)
	if !ok {
		if bytes.HasPrefix(content, []byte("---")) {
			return GoalMetadata{}, fmt.Errorf("no closing '---' found for frontmatter")
		}
		return GoalMetadata{}, nil
	}
	var metadata GoalMetadata
	if err := yaml.Unmarshal(yamlContent, &metadata); err != nil {
		return GoalMetadata{}, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}
	return metadata, nil
}

func parseAgentFileMetadata(dir, agentName string) (agentMetadata, bool) {
	agentPath := filepath.Join(dir, ".sgai", "agent", agentName+".md")
	content, errRead := os.ReadFile(agentPath)
	if errRead != nil {
		return agentMetadata{}, false
	}
	yamlContent, ok := splitFrontmatter(content)
	if !ok {
		return agentMetadata{}, false
	}
	var metadata agentMetadata
	metadata.Log = true
	if errUnmarshal := yaml.Unmarshal(yamlContent, &metadata); errUnmarshal != nil {
		return agentMetadata{}, false
	}
	return metadata, true
}

func shouldLogAgent(dir, agentName string) bool {
	metadata, ok := parseAgentFileMetadata(dir, agentName)
	if !ok {
		return true
	}
	return metadata.Log
}

func parseAgentSnippets(dir, agentName string) []string {
	metadata, ok := parseAgentFileMetadata(dir, agentName)
	if !ok {
		return nil
	}
	return metadata.Snippets
}

func splitFrontmatter(content []byte) (yamlContent []byte, ok bool) {
	delimiter := []byte("---")
	if !bytes.HasPrefix(content, delimiter) {
		return nil, false
	}
	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	before, _, found := bytes.Cut(rest, delimiter)
	if !found {
		return nil, false
	}
	return before, true
}

func parseFrontmatterMap(content []byte) map[string]string {
	result := make(map[string]string)
	yamlContent, ok := splitFrontmatter(content)
	if !ok {
		return result
	}
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

func extractFrontmatterDescription(content string) string {
	fm := parseFrontmatterMap([]byte(content))
	return fm["description"]
}

func computeGoalChecksum(goalPath string) (string, error) {
	data, errRead := os.ReadFile(goalPath)
	if errRead != nil {
		return "", errRead
	}

	body := extractBody(data)
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func extractBody(content []byte) []byte {
	delimiter := []byte("---")

	if !bytes.HasPrefix(content, delimiter) {
		return content
	}

	rest := content[len(delimiter):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	closingIdx := bytes.Index(rest, delimiter)
	if closingIdx == -1 {
		return content
	}

	bodyStart := len(delimiter) + 1 + closingIdx + len(delimiter)
	if bodyStart < len(content) && content[bodyStart] == '\n' {
		bodyStart++
	}
	if bodyStart >= len(content) {
		return []byte{}
	}
	return content[bodyStart:]
}
