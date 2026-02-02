package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findSnippets searches for code snippets in the .sgai/snippets directory.
// When language is empty, it lists available languages. When query is empty,
// it lists all snippets for the language. Otherwise, it searches for matching snippets.
//
//nolint:unparam // error is always nil by design - errors are handled by returning empty strings
func findSnippets(workingDir, language, query string) (string, error) {
	snippetsDir := filepath.Join(workingDir, ".sgai", "snippets")

	if language == "" && query == "" {
		entries, err := os.ReadDir(snippetsDir)
		if err != nil {
			return "", nil
		}
		var languages []string
		for _, entry := range entries {
			if entry.IsDir() {
				languages = append(languages, entry.Name())
			}
		}
		return strings.Join(languages, "\n"), nil
	}

	if language != "" && query == "" {
		langDir := filepath.Join(snippetsDir, language)
		entries, err := os.ReadDir(langDir)
		if err != nil {
			return "", nil
		}
		var snippets []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			snippets = append(snippets, fmt.Sprintf("%s: %s", name, desc))
		}
		return strings.Join(snippets, "\n"), nil
	}

	if language != "" && query != "" {
		langDir := filepath.Join(snippetsDir, language)
		entries, err := os.ReadDir(langDir)
		if err != nil {
			return "", nil
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if name == query {
				return string(content), nil
			}
		}

		var prefixMatches []struct {
			name    string
			content string
			desc    string
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if strings.Contains(name, query) && name != query {
				desc := frontmatter["description"]
				if desc == "" {
					desc = "No description"
				}
				prefixMatches = append(prefixMatches, struct {
					name    string
					content string
					desc    string
				}{name, string(content), desc})
			}
		}
		if len(prefixMatches) == 1 {
			return prefixMatches[0].content, nil
		}
		if len(prefixMatches) > 1 {
			var matches []string
			for _, m := range prefixMatches {
				matches = append(matches, fmt.Sprintf("%s: %s", m.name, m.desc))
			}
			return strings.Join(matches, "\n"), nil
		}

		var matches []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			queryLower := strings.ToLower(query)
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

			if strings.Contains(strings.ToLower(name), queryLower) ||
				strings.Contains(strings.ToLower(frontmatter["description"]), queryLower) {
				desc := frontmatter["description"]
				if desc == "" {
					desc = "No description"
				}
				matches = append(matches, fmt.Sprintf("%s: %s", name, desc))
			}
		}
		return strings.Join(matches, "\n"), nil
	}

	return "", nil
}
