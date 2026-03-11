package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWorkspaceService(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, createWorkspaceResult)
	}{
		{
			name:      "createStandaloneWorkspace",
			workspace: "test-workspace",
			wantErr:   false,
			validate: func(t *testing.T, rootDir string, result createWorkspaceResult) {
				assert.Equal(t, "test-workspace", result.Name)
				assert.Equal(t, filepath.Join(rootDir, "test-workspace"), result.Dir)

				assert.DirExists(t, result.Dir)
				assert.FileExists(t, filepath.Join(result.Dir, "GOAL.md"))
				assert.DirExists(t, filepath.Join(result.Dir, ".sgai"))
			},
		},
		{
			name:        "createWorkspaceWithInvalidName",
			workspace:   "",
			wantErr:     true,
			errContains: "workspace name is invalid",
		},
		{
			name:        "createWorkspaceWithSpecialChars",
			workspace:   "test@workspace#name",
			wantErr:     true,
			errContains: "workspace name is invalid",
		},
		{
			name:      "createWorkspaceThatAlreadyExists",
			workspace: "existing-workspace",
			setupFunc: func(t *testing.T, rootDir string) { //nolint:thelper
				existingPath := filepath.Join(rootDir, "existing-workspace")
				require.NoError(t, os.MkdirAll(existingPath, 0755))
			},
			wantErr:     true,
			errContains: "a directory with this name already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			if tt.setupFunc != nil {
				tt.setupFunc(t, rootDir)
			}

			result, err := server.createWorkspaceService(tt.workspace)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, rootDir, result)
			}
		})
	}
}

func TestForkWorkspaceService(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		goalContent string
		setupFunc   func(*testing.T, string) string
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, forkWorkspaceResult)
	}{
		{
			name:        "forkFromRootWorkspace",
			goalContent: "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))

				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("initial goal"), 0644))

				return workspacePath
			},
			wantErr: false,
			validate: func(t *testing.T, _ string, result forkWorkspaceResult) { //nolint:thelper
				assert.NotEmpty(t, result.Name)
				assert.DirExists(t, result.Dir)
				assert.Equal(t, "root-workspace", result.Parent)
				assert.NotEmpty(t, result.CreatedAt)

				goalPath := filepath.Join(result.Dir, "GOAL.md")
				assert.FileExists(t, goalPath)
			},
		},
		{
			name:        "forkFromForkWorkspace",
			goalContent: "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				rootPath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(rootPath, 0755))
				require.NoError(t, initializeWorkspace(rootPath))

				forkPath := filepath.Join(rootDir, "fork-workspace")
				require.NoError(t, os.MkdirAll(forkPath, 0755))
				require.NoError(t, unpackSkeleton(forkPath))
				require.NoError(t, addGitExclude(forkPath))

				require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".jj"), 0755))
				repoFile := filepath.Join(forkPath, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte(rootPath), 0644))

				return forkPath
			},
			wantErr:     true,
			errContains: "forks cannot create new forks",
		},
		{
			name:        "forkWithEmptyGoalContent",
			goalContent: "",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))

				return workspacePath
			},
			wantErr:     true,
			errContains: "GOAL.md must have content describing the goal",
		},
		{
			name:        "forkWithOnlyFrontmatter",
			goalContent: "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))

				return workspacePath
			},
			wantErr:     true,
			errContains: "GOAL.md must have content describing the goal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			var workspacePath string
			if tt.setupFunc != nil {
				workspacePath = tt.setupFunc(t, rootDir)
			}

			result, err := server.forkWorkspaceService(workspacePath, tt.goalContent)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, rootDir, result)
			}
		})
	}
}

func TestForkExternalWorkspaceSiblingPlacement(t *testing.T) {
	sgaiRoot := t.TempDir()
	externalParent := t.TempDir()
	externalRepo := filepath.Join(externalParent, "my-external-repo")
	require.NoError(t, os.MkdirAll(externalRepo, 0755))
	require.NoError(t, initializeWorkspace(externalRepo))

	server := NewServer(sgaiRoot)
	server.mu.Lock()
	server.externalDirs[resolveSymlinks(externalRepo)] = true
	server.mu.Unlock()

	result, err := server.forkWorkspaceService(externalRepo, "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test Goal")
	require.NoError(t, err)

	assert.Equal(t, externalParent, filepath.Dir(result.Dir))
	assert.DirExists(t, result.Dir)
	assert.NotEqual(t, sgaiRoot, filepath.Dir(result.Dir))
}

func TestDeleteWorkspaceService(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string) string
		wantErr     bool
		errContains string
		validate    func(*testing.T, string)
	}{
		{
			name: "deleteExistingWorkspace",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))
				return workspacePath
			},
			wantErr: false,
			validate: func(t *testing.T, workspacePath string) { //nolint:thelper
				assert.NoDirExists(t, workspacePath)
			},
		},
		{
			name: "deleteNonExistentWorkspace",
			setupFunc: func(_ *testing.T, rootDir string) string {
				return filepath.Join(rootDir, "non-existent-workspace")
			},
			wantErr: false,
			validate: func(_ *testing.T, _ string) {
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := tt.setupFunc(t, rootDir)

			result, err := server.deleteWorkspaceService(workspacePath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result.Deleted)
			if tt.validate != nil {
				tt.validate(t, workspacePath)
			}
		})
	}
}

func TestDeleteForkService(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string) (string, string)
		confirm     bool
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, string)
	}{
		{
			name: "deleteForkFromNonRootWorkspace",
			setupFunc: func(t *testing.T, rootDir string) (string, string) { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "standalone-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))

				return workspacePath, workspacePath
			},
			confirm:     true,
			wantErr:     true,
			errContains: "workspace is not a root",
		},
		{
			name: "deleteForkWithoutConfirmation",
			setupFunc: func(t *testing.T, rootDir string) (string, string) { //nolint:thelper
				rootPath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(rootPath, 0755))
				require.NoError(t, initializeWorkspace(rootPath))
				require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".jj", "repo"), 0755))
				goalPath := filepath.Join(rootPath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0644))

				forkPath := filepath.Join(rootDir, "fork-workspace")
				require.NoError(t, os.MkdirAll(forkPath, 0755))
				require.NoError(t, initializeWorkspace(forkPath))

				return rootPath, forkPath
			},
			confirm:     false,
			wantErr:     true,
			errContains: "workspace is not a root",
		},
		{
			name: "deleteNonExistentFork",
			setupFunc: func(t *testing.T, rootDir string) (string, string) { //nolint:thelper
				rootPath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(rootPath, 0755))
				require.NoError(t, initializeWorkspace(rootPath))
				require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".jj", "repo"), 0755))
				goalPath := filepath.Join(rootPath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0644))

				forkPath := filepath.Join(rootDir, "non-existent-fork")
				return rootPath, forkPath
			},
			confirm:     true,
			wantErr:     true,
			errContains: "workspace is not a root",
		},
		{
			name: "deleteForkThatIsNotAFork",
			setupFunc: func(t *testing.T, rootDir string) (string, string) { //nolint:thelper
				rootPath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(rootPath, 0755))
				require.NoError(t, initializeWorkspace(rootPath))
				require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".jj", "repo"), 0755))
				goalPath := filepath.Join(rootPath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("# Test Goal"), 0644))

				standalonePath := filepath.Join(rootDir, "standalone-workspace")
				require.NoError(t, os.MkdirAll(standalonePath, 0755))
				require.NoError(t, initializeWorkspace(standalonePath))

				return rootPath, standalonePath
			},
			confirm:     true,
			wantErr:     true,
			errContains: "workspace is not a root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath, forkPath := tt.setupFunc(t, rootDir)

			result, err := server.deleteForkService(workspacePath, forkPath, true)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result.Deleted)
			if tt.validate != nil {
				tt.validate(t, workspacePath, forkPath)
			}
		})
	}
}

func TestGetGoalService(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string) string
		wantErr     bool
		errContains string
		validate    func(*testing.T, getGoalResult)
	}{
		{
			name: "getExistingGoal",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))

				goalContent := "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal"
				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte(goalContent), 0644))

				return workspacePath
			},
			wantErr: false,
			validate: func(t *testing.T, result getGoalResult) {
				assert.Contains(t, result.Content, "# Test Goal")
			},
		},
		{
			name: "getGoalFromNonExistentWorkspace",
			setupFunc: func(_ *testing.T, rootDir string) string {
				return filepath.Join(rootDir, "non-existent-workspace")
			},
			wantErr:     true,
			errContains: "failed to read GOAL.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := tt.setupFunc(t, rootDir)

			result, err := server.getGoalService(workspacePath)

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

func TestUpdateGoalService(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		setupFunc   func(*testing.T, string) string
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, updateGoalResult)
	}{
		{
			name:    "updateExistingGoal",
			content: "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Updated Goal",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))

				goalPath := filepath.Join(workspacePath, "GOAL.md")
				require.NoError(t, os.WriteFile(goalPath, []byte("initial goal"), 0644))

				return workspacePath
			},
			wantErr: false,
			validate: func(t *testing.T, workspacePath string, result updateGoalResult) { //nolint:thelper
				assert.True(t, result.Updated)
				assert.Equal(t, "test-workspace", result.Workspace)

				goalPath := filepath.Join(workspacePath, "GOAL.md")
				data, err := os.ReadFile(goalPath)
				require.NoError(t, err)
				assert.Contains(t, string(data), "# Updated Goal")
			},
		},
		{
			name:    "updateGoalWithEmptyContent",
			content: "",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))
				return workspacePath
			},
			wantErr:     true,
			errContains: "content cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := tt.setupFunc(t, rootDir)

			result, err := server.updateGoalService(workspacePath, tt.content)

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

func TestTogglePinService(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string) string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *Server, string, togglePinResult)
	}{
		{
			name: "togglePinOn",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))
				return workspacePath
			},
			wantErr: false,
			validate: func(t *testing.T, server *Server, workspacePath string, result togglePinResult) {
				assert.True(t, result.Pinned)
				assert.True(t, server.isPinned(workspacePath))
			},
		},
		{
			name: "togglePinOff",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "test-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))
				return workspacePath
			},
			wantErr: false,
			validate: func(t *testing.T, server *Server, workspacePath string, result togglePinResult) {
				assert.True(t, result.Pinned)

				result2, err := server.togglePinService(workspacePath)
				require.NoError(t, err)
				assert.False(t, result2.Pinned)
				assert.False(t, server.isPinned(workspacePath))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := tt.setupFunc(t, rootDir)

			result, err := server.togglePinService(workspacePath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, server, workspacePath, result)
			}
		})
	}
}

func TestFailForkWorkspaceSetup(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		errCause    error
		wantErr     bool
		errContains string
	}{
		{
			name:        "failWithSetupError",
			message:     "failed to unpack skeleton",
			errCause:    fmt.Errorf("skeleton unpack failed"),
			wantErr:     true,
			errContains: "failed to unpack skeleton",
		},
		{
			name:        "failWithGitExcludeError",
			message:     "failed to add git exclude",
			errCause:    fmt.Errorf("git exclude failed"),
			wantErr:     true,
			errContains: "failed to add git exclude",
		},
		{
			name:        "failWithGoalWriteError",
			message:     "failed to create GOAL.md",
			errCause:    fmt.Errorf("goal write failed"),
			wantErr:     true,
			errContains: "failed to create GOAL.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()

			workspacePath := filepath.Join(rootDir, "root-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))

			forkPath := filepath.Join(rootDir, "fork-workspace")
			require.NoError(t, os.MkdirAll(forkPath, 0755))

			err := failForkWorkspaceSetup(workspacePath, forkPath, tt.message, tt.errCause)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestRollbackForkWorkspaceCreation(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, string)
	}{
		{
			name: "rollbackForkWithoutJjRepo",
			setupFunc: func(t *testing.T, _, forkPath string) { //nolint:thelper
				require.NoError(t, os.MkdirAll(forkPath, 0755))
			},
			wantErr:     true,
			errContains: "failed to forget fork workspace during rollback",
			validate: func(t *testing.T, forkPath string) { //nolint:thelper
				assert.NoDirExists(t, forkPath)
			},
		},
		{
			name: "rollbackNonExistentFork",
			setupFunc: func(_ *testing.T, _, _ string) {
			},
			wantErr:     true,
			errContains: "failed to forget fork workspace during rollback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()

			workspacePath := filepath.Join(rootDir, "root-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))

			forkPath := filepath.Join(rootDir, "fork-workspace")

			tt.setupFunc(t, workspacePath, forkPath)

			err := rollbackForkWorkspaceCreation(workspacePath, forkPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, forkPath)
			}
		})
	}
}

func TestDeleteForkByPathService(t *testing.T) {
	t.Skip("Integration test - requires real jj repository with multiple workspaces")
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string) string
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, deleteForkResult)
	}{
		{
			name: "deleteForkByPath",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				rootPath := filepath.Join(rootDir, "root-workspace")
				require.NoError(t, os.MkdirAll(rootPath, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".sgai"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(rootPath, ".jj"), 0755))

				forkPath := filepath.Join(rootDir, "fork-workspace")
				require.NoError(t, os.MkdirAll(forkPath, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".sgai"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".jj"), 0755))
				repoFile := filepath.Join(forkPath, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte(rootPath), 0644))

				return forkPath
			},
			wantErr: false,
			validate: func(t *testing.T, _ string, result deleteForkResult) { //nolint:thelper
				assert.True(t, result.Deleted)
				assert.Contains(t, result.Message, "fork deleted successfully")
			},
		},
		{
			name: "deleteForkByPathWithInvalidFork",
			setupFunc: func(_ *testing.T, rootDir string) string { //nolint:thelper
				return filepath.Join(rootDir, "non-existent-fork")
			},
			wantErr:     true,
			errContains: "could not determine root workspace for fork",
		},
		{
			name: "deleteForkByPathWithStandaloneWorkspace",
			setupFunc: func(t *testing.T, rootDir string) string { //nolint:thelper
				workspacePath := filepath.Join(rootDir, "standalone-workspace")
				require.NoError(t, os.MkdirAll(workspacePath, 0755))
				require.NoError(t, initializeWorkspace(workspacePath))
				return workspacePath
			},
			wantErr:     true,
			errContains: "could not determine root workspace for fork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			forkPath := tt.setupFunc(t, rootDir)

			result, err := server.deleteForkByPathService(forkPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, forkPath, result)
			}
		})
	}
}

func TestDeleteMessageService(t *testing.T) {
	tests := []struct {
		name        string
		messageID   int
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, deleteMessageResult)
	}{
		{
			name:      "deleteNonExistentMessage",
			messageID: 999,
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "message not found",
		},
		{
			name:      "deleteExistingMessage",
			messageID: 1,
			setupFunc: func(t *testing.T, workspacePath string) {
				sgaiDir := filepath.Join(workspacePath, ".sgai")
				require.NoError(t, os.MkdirAll(sgaiDir, 0755))

				stateData := `{
					"status": "working",
					"messages": [
						{
							"id": 1,
							"fromAgent": "agent1",
							"toAgent": "agent2",
							"body": "test message",
							"read": false,
							"createdAt": "2026-03-05T10:00:00Z"
						}
					]
				}`
				statePath := filepath.Join(sgaiDir, "state.json")
				require.NoError(t, os.WriteFile(statePath, []byte(stateData), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, result deleteMessageResult) {
				assert.True(t, result.Deleted)
				assert.Equal(t, 1, result.ID)
			},
		},
		{
			name:      "deleteReadMessage",
			messageID: 2,
			setupFunc: func(t *testing.T, workspacePath string) {
				sgaiDir := filepath.Join(workspacePath, ".sgai")
				require.NoError(t, os.MkdirAll(sgaiDir, 0755))

				stateData := `{
					"status": "working",
					"messages": [
						{
							"id": 2,
							"fromAgent": "agent1",
							"toAgent": "agent2",
							"body": "read message",
							"read": true,
							"readAt": "2026-03-05T11:00:00Z",
							"readBy": "agent2",
							"createdAt": "2026-03-05T10:00:00Z"
						}
					]
				}`
				statePath := filepath.Join(sgaiDir, "state.json")
				require.NoError(t, os.WriteFile(statePath, []byte(stateData), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, result deleteMessageResult) {
				assert.True(t, result.Deleted)
				assert.Equal(t, 2, result.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			require.NoError(t, initializeWorkspace(workspacePath))

			tt.setupFunc(t, workspacePath)

			result, err := server.deleteMessageService(workspacePath, tt.messageID)

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

func TestDeleteForkByPathServiceNonExistent(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	forkPath := filepath.Join(rootDir, "non-existent-fork")
	_, err := server.deleteForkByPathService(forkPath)
	require.Error(t, err)
}

func TestDeleteForkByPathServiceStandalone(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)

	workspacePath := filepath.Join(rootDir, "standalone-workspace")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, initializeWorkspace(workspacePath))

	_, err := server.deleteForkByPathService(workspacePath)
	require.Error(t, err)
}

func TestGoalContentBodyIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "emptyContent",
			content:  "",
			expected: true,
		},
		{
			name:     "onlyFrontmatter",
			content:  "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n",
			expected: true,
		},
		{
			name:     "frontmatterWithBody",
			content:  "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal",
			expected: false,
		},
		{
			name:     "frontmatterWithWhitespaceBody",
			content:  "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n   \n\t\n",
			expected: true,
		},
		{
			name:     "noFrontmatter",
			content:  "# Test Goal",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := goalContentBodyIsEmpty(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteGoalContent(t *testing.T) {
	rootDir := t.TempDir()

	goalPath := filepath.Join(rootDir, "GOAL.md")
	content := "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test Goal"

	err := writeGoalContent(rootDir, content)
	require.NoError(t, err)

	data, err := os.ReadFile(goalPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestDeleteMessageServiceNotFoundError(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-delmsg-nf")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{})
	require.NoError(t, errCoord)
	_, errDelete := server.deleteMessageService(wsDir, 999)
	assert.Error(t, errDelete)
}

func TestGenerateRandomForkName(t *testing.T) {
	for i := 0; i < 100; i++ {
		name := generateRandomForkName()
		assert.NotEmpty(t, name)
		assert.True(t, len(name) > 5, "name should be longer than 5 characters")
		assert.True(t, strings.Contains(name, "-"), "name should contain a hyphen")
		parts := strings.Split(name, "-")
		assert.Len(t, parts, 3, "name should have 3 parts separated by hyphens")
	}
}

func TestGenerateRandomForkNameUniqueness(t *testing.T) {
	names := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		name := generateRandomForkName()
		names[name] = true
	}
	assert.True(t, len(names) > 100, "should generate many unique names")
}

func TestUpdateGoalServiceInvalidatesSVGCache(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	workspacePath := filepath.Join(rootDir, "cache-ws")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workspacePath, "GOAL.md"), []byte("# Old"), 0644))
	server.svgCache.set(workspacePath+"|coordinator", "<svg>old</svg>")
	server.svgCache.set(workspacePath+"|agent1", "<svg>old2</svg>")
	server.svgCache.set("/other/path|coordinator", "<svg>other</svg>")

	result, err := server.updateGoalService(workspacePath, "# New Goal")
	require.NoError(t, err)
	assert.True(t, result.Updated)

	_, wsOK := server.svgCache.get(workspacePath + "|coordinator")
	assert.False(t, wsOK)
	_, otherOK := server.svgCache.get("/other/path|coordinator")
	assert.True(t, otherOK)
}

func TestTogglePinServiceSuccess(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	workspacePath := filepath.Join(rootDir, "pin-ws")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))

	result, err := server.togglePinService(workspacePath)
	require.NoError(t, err)
	assert.True(t, result.Pinned)

	result2, err2 := server.togglePinService(workspacePath)
	require.NoError(t, err2)
	assert.False(t, result2.Pinned)
}

func TestDeleteWorkspaceServiceSuccess(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	workspacePath := filepath.Join(rootDir, "delete-ws")
	require.NoError(t, os.MkdirAll(workspacePath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workspacePath, "GOAL.md"), []byte("# Goal"), 0644))
	server.mu.Lock()
	server.pinnedDirs[resolveSymlinks(workspacePath)] = true
	server.mu.Unlock()

	result, err := server.deleteWorkspaceService(workspacePath)
	require.NoError(t, err)
	assert.True(t, result.Deleted)

	_, errStat := os.Stat(workspacePath)
	assert.True(t, os.IsNotExist(errStat))
}
