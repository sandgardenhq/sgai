package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func findSkills(workingDir, name string) (string, error) {
	skillsDir := filepath.Join(workingDir, ".sgai", "skills")

	getAllSkillFiles := func(dir string) ([]string, error) {
		var files []string
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == "SKILL.md" {
				files = append(files, path)
			}
			return nil
		})
		return files, err
	}

	skillFiles, err := getAllSkillFiles(skillsDir)
	if err != nil {
		return "", fmt.Errorf("failed to access skills: %w", err)
	}

	if name == "" {
		var skills []string
		for _, file := range skillFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			relName, _ := filepath.Rel(skillsDir, file)
			relName = strings.TrimSuffix(relName, "/SKILL.md")
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			skills = append(skills, fmt.Sprintf("%s: %s", relName, desc))
		}
		return strings.Join(skills, "\n"), nil
	}

	for _, file := range skillFiles {
		relName, _ := filepath.Rel(skillsDir, file)
		relName = strings.TrimSuffix(relName, "/SKILL.md")
		if relName == name {
			content, err := os.ReadFile(file)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}

	var prefixMatches []string
	for _, file := range skillFiles {
		relName, _ := filepath.Rel(skillsDir, file)
		relName = strings.TrimSuffix(relName, "/SKILL.md")
		if strings.HasPrefix(relName, name) && relName != name {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			prefixMatches = append(prefixMatches, fmt.Sprintf("%s: %s", relName, desc))
		}
	}
	if len(prefixMatches) > 0 {
		return strings.Join(prefixMatches, "\n"), nil
	}

	var basenameMatches []struct {
		path    string
		content string
		desc    string
	}
	for _, file := range skillFiles {
		relName, _ := filepath.Rel(skillsDir, file)
		relName = strings.TrimSuffix(relName, "/SKILL.md")
		basename := filepath.Base(relName)
		if basename == name {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			basenameMatches = append(basenameMatches, struct {
				path    string
				content string
				desc    string
			}{relName, string(content), desc})
		}
	}
	if len(basenameMatches) == 1 {
		return basenameMatches[0].content, nil
	}
	if len(basenameMatches) > 1 {
		var matches []string
		for _, m := range basenameMatches {
			matches = append(matches, fmt.Sprintf("%s: %s", m.path, m.desc))
		}
		return strings.Join(matches, "\n"), nil
	}

	var matches []string
	for _, file := range skillFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		frontmatter := parseFrontmatterMap(content)
		contentLower := strings.ToLower(string(content))
		nameLower := strings.ToLower(name)

		if strings.Contains(strings.ToLower(frontmatter["name"]), nameLower) ||
			strings.Contains(strings.ToLower(frontmatter["description"]), nameLower) ||
			strings.Contains(contentLower, nameLower) {
			relName, _ := filepath.Rel(skillsDir, file)
			relName = strings.TrimSuffix(relName, "/SKILL.md")
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			matches = append(matches, fmt.Sprintf("%s: %s", relName, desc))
		}
	}
	return strings.Join(matches, "\n"), nil
}
