package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

type modelCatalog map[string]modelCatalogEntry

type modelCatalogEntry struct {
	Variants map[string]json.RawMessage
}

func fetchValidModels() (modelCatalog, error) {
	cmd := exec.Command("opencode", "models", "--verbose")
	output, errCommand := cmd.CombinedOutput()
	if errCommand != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, fmt.Errorf("listing opencode models: %w", errCommand)
		}
		return nil, fmt.Errorf("listing opencode models: %w: %s", errCommand, message)
	}

	catalog, errParse := parseOpenCodeModelsVerbose(output)
	if errParse != nil {
		return nil, fmt.Errorf("parsing opencode models: %w", errParse)
	}
	if len(catalog) == 0 {
		return nil, fmt.Errorf("parsing opencode models: no models returned")
	}
	return catalog, nil
}

func parseOpenCodeModelsVerbose(output []byte) (modelCatalog, error) {
	lines := bytes.Split(output, []byte("\n"))
	catalog := make(modelCatalog)
	var currentID string
	var jsonBuf bytes.Buffer

	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}

		if currentID == "" {
			if strings.HasPrefix(trimmed, "{") {
				return nil, fmt.Errorf("model JSON appeared before model id")
			}
			currentID = trimmed
			jsonBuf.Reset()
			continue
		}

		jsonBuf.Write(line)
		jsonBuf.WriteByte('\n')
		if !json.Valid(jsonBuf.Bytes()) {
			continue
		}

		entry, errEntry := parseModelEntry(currentID, jsonBuf.Bytes())
		if errEntry != nil {
			return nil, errEntry
		}
		catalog[currentID] = entry
		currentID = ""
		jsonBuf.Reset()
	}

	if currentID != "" {
		return nil, fmt.Errorf("incomplete JSON for model %s", currentID)
	}

	return catalog, nil
}

func parseModelEntry(id string, data []byte) (modelCatalogEntry, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return modelCatalogEntry{}, fmt.Errorf("model %s JSON must be an object", id)
	}

	var raw map[string]json.RawMessage
	if errUnmarshal := json.Unmarshal(data, &raw); errUnmarshal != nil {
		return modelCatalogEntry{}, fmt.Errorf("parsing model %s: %w", id, errUnmarshal)
	}
	variantsData, exists := raw["variants"]
	if !exists {
		return modelCatalogEntry{}, fmt.Errorf("model %s JSON must contain object-valued variants", id)
	}
	if !isJSONObject(variantsData) {
		return modelCatalogEntry{}, fmt.Errorf("model %s JSON must contain object-valued variants", id)
	}

	var variants map[string]json.RawMessage
	if errUnmarshal := json.Unmarshal(variantsData, &variants); errUnmarshal != nil {
		return modelCatalogEntry{}, fmt.Errorf("parsing model %s variants: %w", id, errUnmarshal)
	}
	return modelCatalogEntry{Variants: variants}, nil
}

func isJSONObject(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func validateModelSpec(catalog modelCatalog, modelSpec string) error {
	baseModel, variant := parseModelAndVariant(modelSpec)
	entry, exists := catalog[baseModel]
	if !exists {
		return fmt.Errorf("model %s is not available from opencode", baseModel)
	}
	if variant == "" {
		return nil
	}
	if _, exists := entry.Variants[variant]; !exists {
		return fmt.Errorf("variant %s is not available for model %s", variant, baseModel)
	}
	return nil
}

func modelEntries(catalog modelCatalog) []apiModelEntry {
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	entries := make([]apiModelEntry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, apiModelEntry{ID: id, Name: id})
	}
	return entries
}
