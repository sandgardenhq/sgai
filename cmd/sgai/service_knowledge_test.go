package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAgentsService(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, listAgentsResult)
	}{
		{
			name: "listAgentsWithAgentFiles",
			setupFunc: func(t *testing.T, workspacePath string) {
				agentsDir := filepath.Join(workspacePath, ".sgai", "agent")
				require.NoError(t, os.MkdirAll(agentsDir, 0755))
				agent1Content := `---
description: Agent 1 description
---
# Agent 1 Instructions`
				agent1Path := filepath.Join(agentsDir, "agent1.md")
				require.NoError(t, os.WriteFile(agent1Path, []byte(agent1Content), 0644))
				agent2Content := `---
description: Agent 2 description
---
# Agent 2 Instructions`
				agent2Path := filepath.Join(agentsDir, "agent2.md")
				require.NoError(t, os.WriteFile(agent2Path, []byte(agent2Content), 0644))
			},
			validate: func(t *testing.T, result listAgentsResult) {
				assert.Len(t, result.Agents, 2)
				agentNames := make(map[string]bool)
				for _, agent := range result.Agents {
					agentNames[agent.Name] = true
				}
				assert.True(t, agentNames["agent1"])
				assert.True(t, agentNames["agent2"])
			},
		},
		{
			name: "listAgentsNoAgents",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			validate: func(t *testing.T, result listAgentsResult) {
				assert.Empty(t, result.Agents)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result := server.listAgentsService(workspacePath)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func newTestServerWithWorkspace(t *testing.T) (*Server, string) {
	t.Helper()
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	return server, workspacePath
}

func TestListSkillsService(t *testing.T) {
	t.Run("listSkillsWithSkills", func(t *testing.T) {
		server, workspacePath := newTestServerWithWorkspace(t)
		skillsDir := filepath.Join(workspacePath, ".sgai", "skills", "test-skill")
		require.NoError(t, os.MkdirAll(skillsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: Test Skill\ndescription: A test skill\n---\n# Test Skill Content"), 0644))

		result := server.listSkillsService(workspacePath)
		assert.NotEmpty(t, result.Categories)
	})

	t.Run("listSkillsNoSkills", func(t *testing.T) {
		server, workspacePath := newTestServerWithWorkspace(t)
		require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

		result := server.listSkillsService(workspacePath)
		assert.Empty(t, result.Categories)
	})
}

func TestSkillDetailService(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		setupFunc func(*testing.T, string)
		wantFound bool
		validate  func(*testing.T, skillDetailResult)
	}{
		{
			name:      "skillFound",
			skillName: "test-skill",
			setupFunc: func(t *testing.T, workspacePath string) {
				skillsDir := filepath.Join(workspacePath, ".sgai", "skills", "test-skill")
				require.NoError(t, os.MkdirAll(skillsDir, 0755))
				skillContent := `---
name: Test Skill
description: A test skill
---
# Test Skill Content`
				skillPath := filepath.Join(skillsDir, "SKILL.md")
				require.NoError(t, os.WriteFile(skillPath, []byte(skillContent), 0644))
			},
			wantFound: true,
			validate: func(t *testing.T, result skillDetailResult) {
				assert.True(t, result.Found)
				assert.Equal(t, "test-skill", result.Name)
				assert.Contains(t, result.RawContent, "Test Skill Content")
			},
		},
		{
			name:      "skillNotFound",
			skillName: "non-existent-skill",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai", "skills"), 0755))
			},
			wantFound: false,
			validate: func(t *testing.T, result skillDetailResult) {
				assert.False(t, result.Found)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result := server.skillDetailService(workspacePath, tt.skillName)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestListSnippetsService(t *testing.T) {
	t.Run("listSnippetsWithSnippets", func(t *testing.T) {
		server, workspacePath := newTestServerWithWorkspace(t)
		snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets", "go")
		require.NoError(t, os.MkdirAll(snippetsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(snippetsDir, "test-snippet.go"), []byte("---\nname: Test Snippet\ndescription: A test snippet\n---\npackage main"), 0644))

		result := server.listSnippetsService(workspacePath)
		assert.NotEmpty(t, result.Languages)
	})

	t.Run("listSnippetsNoSnippets", func(t *testing.T) {
		server, workspacePath := newTestServerWithWorkspace(t)
		require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

		result := server.listSnippetsService(workspacePath)
		assert.Empty(t, result.Languages)
	})
}

func TestSnippetsByLanguageService(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		setupFunc func(*testing.T, string)
		wantFound bool
		validate  func(*testing.T, snippetsByLanguageResult)
	}{
		{
			name: "snippetsFound",
			lang: "go",
			setupFunc: func(t *testing.T, workspacePath string) {
				snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets", "go")
				require.NoError(t, os.MkdirAll(snippetsDir, 0755))
				snippetContent := `---
name: Test Snippet
description: A test snippet
---
package main`
				snippetPath := filepath.Join(snippetsDir, "test-snippet.go")
				require.NoError(t, os.WriteFile(snippetPath, []byte(snippetContent), 0644))
			},
			wantFound: true,
			validate: func(t *testing.T, result snippetsByLanguageResult) {
				assert.True(t, result.Found)
				assert.Equal(t, "go", result.Language)
			},
		},
		{
			name: "snippetsNotFound",
			lang: "python",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai", "snippets"), 0755))
			},
			wantFound: false,
			validate: func(t *testing.T, result snippetsByLanguageResult) {
				assert.False(t, result.Found)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result := server.snippetsByLanguageService(workspacePath, tt.lang)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestSnippetDetailService(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		fileName  string
		setupFunc func(*testing.T, string)
		wantFound bool
		validate  func(*testing.T, snippetDetailResult)
	}{
		{
			name:     "snippetFound",
			lang:     "go",
			fileName: "test-snippet",
			setupFunc: func(t *testing.T, workspacePath string) {
				snippetsDir := filepath.Join(workspacePath, ".sgai", "snippets", "go")
				require.NoError(t, os.MkdirAll(snippetsDir, 0755))
				snippetContent := `---
name: Test Snippet
description: A test snippet
when_to_use: When testing
---
package main`
				snippetPath := filepath.Join(snippetsDir, "test-snippet.go")
				require.NoError(t, os.WriteFile(snippetPath, []byte(snippetContent), 0644))
			},
			wantFound: true,
			validate: func(t *testing.T, result snippetDetailResult) {
				assert.True(t, result.Found)
				assert.Equal(t, "Test Snippet", result.Name)
				assert.Equal(t, "go", result.Language)
				assert.Equal(t, "A test snippet", result.Description)
			},
		},
		{
			name:     "snippetNotFound",
			lang:     "go",
			fileName: "non-existent",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai", "snippets", "go"), 0755))
			},
			wantFound: false,
			validate: func(t *testing.T, result snippetDetailResult) {
				assert.False(t, result.Found)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result := server.snippetDetailService(workspacePath, tt.lang, tt.fileName)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestListModelsService(t *testing.T) {
	t.Skip("Integration test - requires opencode CLI to be installed")
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	result := server.listModelsService("test-workspace")
	assert.NotEmpty(t, result.Models)
}

func TestListModelsServiceEmptyWorkspace(t *testing.T) {
	t.Skip("Integration test - requires opencode CLI to be installed")
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	result := server.listModelsService("non-existent-workspace")
	assert.Empty(t, result.Models)
	assert.Empty(t, result.DefaultModel)
}
