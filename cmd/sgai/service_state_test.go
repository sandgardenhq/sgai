package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspaceStateService(t *testing.T) {
	tests := []struct {
		name          string
		workspaceName string
		setupFunc     func(*testing.T, string)
		wantErr       bool
		errContains   string
		validate      func(*testing.T, workspaceStateResult)
	}{
		{
			name:          "getExistingWorkspaceState",
			workspaceName: "test-workspace",
			setupFunc: func(t *testing.T, rootDir string) {
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr: false,
			validate: func(t *testing.T, result workspaceStateResult) {
				assert.True(t, result.Found)
				assert.Equal(t, "test-workspace", result.Workspace.Name)
			},
		},
		{
			name:          "getNonExistentWorkspaceState",
			workspaceName: "non-existent-workspace",
			setupFunc:     func(_ *testing.T, _ string) {},
			wantErr:       false,
			validate: func(t *testing.T, result workspaceStateResult) {
				assert.False(t, result.Found)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			tt.setupFunc(t, rootDir)

			result, err := server.getWorkspaceStateService(tt.workspaceName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestGetAgentDelegationSVGService(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, string)
	}{
		{
			name: "getSVGForWorkspaceWithGoal",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
				goalContent := "---\nagents:\n  - coordinator\n  - agent1\n  - agent2\nmodel: openai/gpt-5.5\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, svg string) {
				assert.NotEmpty(t, svg)
				assert.Contains(t, svg, "svg")
				assert.Contains(t, svg, "agent1")
				assert.Contains(t, svg, "agent2")
				assert.NotContains(t, svg, "coordinator")
			},
		},
		{
			name: "getSVGForWorkspaceWithoutGoal",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			validate: func(t *testing.T, svg string) {
				assert.Empty(t, svg)
			},
		},
		{
			name: "getSVGForWorkspaceWithOnlyCoordinator",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
				goalContent := "---\nagents:\n  - coordinator\nmodel: openai/gpt-5.5\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))
			},
			validate: func(t *testing.T, svg string) {
				assert.Empty(t, svg)
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

			svg := server.getAgentDelegationSVGService(workspacePath)

			if tt.validate != nil {
				tt.validate(t, svg)
			}
		})
	}
}

func TestUpdateDescriptionService(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, updateDescriptionResult)
	}{
		{
			name:        "updateDescription",
			description: "New commit description",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, initializeWorkspace(workspacePath))
			},
			wantErr: false,
			validate: func(t *testing.T, _ string, result updateDescriptionResult) {
				assert.True(t, result.Updated)
				assert.Equal(t, "New commit description", result.Description)
			},
		},
		{
			name:        "updateDescriptionWithEmptyString",
			description: "",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, initializeWorkspace(workspacePath))
			},
			wantErr: false,
			validate: func(t *testing.T, _ string, result updateDescriptionResult) {
				assert.True(t, result.Updated)
				assert.Equal(t, "", result.Description)
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

			result, err := server.updateDescriptionService(workspacePath, tt.description)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, workspacePath, result)
			}
		})
	}
}

func TestGetWorkspaceStateServiceWithMultipleWorkspaces(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	rootPath := filepath.Join(rootDir, "root-workspace")
	require.NoError(t, os.MkdirAll(rootPath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".sgai"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".jj", "repo"), 0755))

	forkPath := filepath.Join(rootDir, "fork-workspace")
	require.NoError(t, os.MkdirAll(forkPath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".sgai"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".jj"), 0755))
	repoFile := filepath.Join(forkPath, ".jj", "repo")
	require.NoError(t, os.WriteFile(repoFile, []byte(rootPath), 0644))

	standalonePath := filepath.Join(rootDir, "standalone-workspace")
	require.NoError(t, os.MkdirAll(standalonePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(standalonePath, ".sgai"), 0755))

	result, err := server.getWorkspaceStateService("root-workspace")
	require.NoError(t, err)
	assert.True(t, result.Found)
	assert.Equal(t, "root-workspace", result.Workspace.Name)

	result, err = server.getWorkspaceStateService("fork-workspace")
	require.NoError(t, err)
	assert.True(t, result.Found)
	assert.Equal(t, "fork-workspace", result.Workspace.Name)

	result, err = server.getWorkspaceStateService("standalone-workspace")
	require.NoError(t, err)
	assert.True(t, result.Found)
	assert.Equal(t, "standalone-workspace", result.Workspace.Name)

	result, err = server.getWorkspaceStateService("non-existent-workspace")
	require.NoError(t, err)
	assert.False(t, result.Found)
}

func TestGetAgentDelegationSVGServiceWithDifferentAgentLists(t *testing.T) {
	tests := []struct {
		name        string
		goalContent string
		validate    func(*testing.T, string)
	}{
		{
			name:        "goalWithSimpleAgents",
			goalContent: "---\nagents:\n  - agent1\n  - agent2\nmodel: openai/gpt-5.5\n---\n# Test Goal",
			validate: func(t *testing.T, svg string) {
				assert.NotEmpty(t, svg)
				assert.Contains(t, svg, "svg")
				assert.Contains(t, svg, "agent1")
				assert.Contains(t, svg, "agent2")
				assert.NotContains(t, svg, "coordinator")
			},
		},
		{
			name:        "goalWithMultipleAgents",
			goalContent: "---\nagents:\n  - agent1\n  - agent2\n  - agent3\nmodel: openai/gpt-5.5\n---\n# Complex Goal",
			validate: func(t *testing.T, svg string) {
				assert.NotEmpty(t, svg)
				assert.Contains(t, svg, "svg")
				assert.Contains(t, svg, "agent1")
				assert.Contains(t, svg, "agent2")
				assert.Contains(t, svg, "agent3")
				assert.NotContains(t, svg, "coordinator")
			},
		},
		{
			name:        "goalWithXMLSignificantAgentName",
			goalContent: "---\nagents:\n  - 'research <review> & verify'\nmodel: openai/gpt-5.5\n---\n# Escaped Goal",
			validate: func(t *testing.T, svg string) {
				assert.NotEmpty(t, svg)
				assert.Contains(t, svg, "research &lt;review&gt; &amp; verify")
				assert.NotContains(t, svg, "research <review> & verify")
			},
		},
		{
			name:        "goalWithNoAgents",
			goalContent: "---\nmodel: openai/gpt-5.5\n---\n# Test Goal",
			validate: func(t *testing.T, svg string) {
				assert.Empty(t, svg)
				assert.NotContains(t, svg, "coordinator")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

			goalPath := filepath.Join(workspacePath, "GOAL.md")
			require.NoError(t, os.WriteFile(goalPath, []byte(tt.goalContent), 0644))

			svg := server.getAgentDelegationSVGService(workspacePath)

			if tt.validate != nil {
				tt.validate(t, svg)
			}
		})
	}
}

func TestWorkspaceDiffServiceWithoutJJRepo(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "test-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))

	result := server.workspaceDiffService(workspacePath)
	assert.Empty(t, result.Diff)
}
