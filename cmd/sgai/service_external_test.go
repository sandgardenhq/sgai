package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadExternalDirs(t *testing.T) {
	t.Run("noFile", func(t *testing.T) {
		server, _ := setupTestServer(t)
		server.externalConfigDir = t.TempDir()
		err := server.loadExternalDirs()
		assert.NoError(t, err)
	})

	t.Run("validFile", func(t *testing.T) {
		server, _ := setupTestServer(t)
		configDir := t.TempDir()
		server.externalConfigDir = configDir

		externalDir := t.TempDir()
		data := `["` + externalDir + `"]`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "external.json"), []byte(data), 0644))

		err := server.loadExternalDirs()
		assert.NoError(t, err)
	})

	t.Run("invalidJSON", func(t *testing.T) {
		server, _ := setupTestServer(t)
		configDir := t.TempDir()
		server.externalConfigDir = configDir

		require.NoError(t, os.WriteFile(filepath.Join(configDir, "external.json"), []byte(`{invalid}`), 0644))

		err := server.loadExternalDirs()
		assert.Error(t, err)
	})
}

func TestSaveExternalDirs(t *testing.T) {
	server, _ := setupTestServer(t)
	configDir := t.TempDir()
	server.externalConfigDir = configDir

	server.mu.Lock()
	server.externalDirs["/some/path"] = true
	server.mu.Unlock()

	err := server.saveExternalDirs()
	assert.NoError(t, err)

	data, errRead := os.ReadFile(filepath.Join(configDir, "external.json"))
	require.NoError(t, errRead)
	assert.Contains(t, string(data), "/some/path")
}

func TestIsExternalWorkspaceNotExternal(t *testing.T) {
	srv, _ := setupTestServer(t)
	assert.False(t, srv.isExternalWorkspace("/some/random/path"))
}

func TestAttachExternalWorkspaceService(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setupFunc   func(*testing.T, string, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, attachExternalResult)
	}{
		{
			name: "attachValidExternalDirectory",
			setupFunc: func(t *testing.T, _, externalPath string) {
				require.NoError(t, os.MkdirAll(externalPath, 0755))
			},
			wantErr: false,
			validate: func(t *testing.T, externalPath string, result attachExternalResult) {
				assert.NotEmpty(t, result.Name)
				assert.Equal(t, externalPath, result.Dir)
			},
		},
		{
			name: "attachWithRelativePath",
			path: "relative/path",
			setupFunc: func(_ *testing.T, _, _ string) {
			},
			wantErr:     true,
			errContains: "path must be absolute",
		},
		{
			name: "attachNonExistentDirectory",
			path: "/non/existent/directory",
			setupFunc: func(_ *testing.T, _, _ string) {
			},
			wantErr:     true,
			errContains: "directory does not exist",
		},
		{
			name: "attachFileNotDirectory",
			setupFunc: func(t *testing.T, _, externalPath string) {
				require.NoError(t, os.WriteFile(externalPath, []byte("test"), 0644))
			},
			wantErr:     true,
			errContains: "path is not a directory",
		},
		{
			name: "attachDirectoryUnderRoot",
			setupFunc: func(_ *testing.T, _, _ string) {
			},
			wantErr:     true,
			errContains: "path is within the root directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			var externalPath string
			switch {
			case tt.name == "attachDirectoryUnderRoot":
				externalPath = filepath.Join(rootDir, "subdir")
				require.NoError(t, os.MkdirAll(externalPath, 0755))
			case tt.path != "":
				externalPath = tt.path
			default:
				externalPath = filepath.Join(os.TempDir(), "external-workspace")
				t.Cleanup(func() {
					_ = os.RemoveAll(externalPath)
				})
			}

			tt.setupFunc(t, rootDir, externalPath)

			result, err := server.attachExternalWorkspaceService(externalPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, externalPath, result)
			}
		})
	}
}

func TestDetachExternalWorkspaceService(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string, string, *Server)
		wantErr     bool
		errContains string
		validate    func(*testing.T, detachExternalResult)
	}{
		{
			name: "detachAttachedWorkspace",
			setupFunc: func(t *testing.T, _ string, externalPath string, server *Server) {
				require.NoError(t, os.MkdirAll(externalPath, 0755))
				_, err := server.attachExternalWorkspaceService(externalPath)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, result detachExternalResult) {
				assert.True(t, result.Detached)
				assert.Equal(t, "external workspace detached", result.Message)
			},
		},
		{
			name: "detachNonAttachedWorkspace",
			setupFunc: func(t *testing.T, _ string, externalPath string, _ *Server) {
				require.NoError(t, os.MkdirAll(externalPath, 0755))
			},
			wantErr:     true,
			errContains: "directory is not attached as an external workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			externalPath := filepath.Join(os.TempDir(), "external-workspace")
			t.Cleanup(func() {
				_ = os.RemoveAll(externalPath)
			})

			tt.setupFunc(t, rootDir, externalPath, server)

			result, err := server.detachExternalWorkspaceService(externalPath)

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

func TestIsExternalWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string, string, *Server)
		expected  bool
	}{
		{
			name: "isExternalTrue",
			setupFunc: func(t *testing.T, _ string, externalPath string, server *Server) {
				require.NoError(t, os.MkdirAll(externalPath, 0755))
				_, err := server.attachExternalWorkspaceService(externalPath)
				require.NoError(t, err)
			},
			expected: true,
		},
		{
			name: "isExternalFalse",
			setupFunc: func(t *testing.T, _ string, externalPath string, _ *Server) {
				require.NoError(t, os.MkdirAll(externalPath, 0755))
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			externalPath := filepath.Join(os.TempDir(), "external-workspace")
			t.Cleanup(func() {
				_ = os.RemoveAll(externalPath)
			})

			tt.setupFunc(t, rootDir, externalPath, server)

			result := server.isExternalWorkspace(externalPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeleteExternalForkService(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not found in PATH")
	}

	tests := []struct {
		name        string
		setupFunc   func(*testing.T, string, string, *Server)
		wantErr     bool
		errContains string
		validate    func(*testing.T, string, deleteExternalForkResult)
	}{
		{
			name: "deleteExternalFork",
			setupFunc: func(t *testing.T, _ string, forkPath string, _ *Server) {
				require.NoError(t, os.MkdirAll(forkPath, 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(forkPath, ".sgai"), 0755))
			},
			wantErr:     true,
			errContains: "could not determine root workspace for fork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			forkPath := filepath.Join(os.TempDir(), "external-fork")
			t.Cleanup(func() {
				_ = os.RemoveAll(forkPath)
			})

			tt.setupFunc(t, rootDir, forkPath, server)

			result, err := server.deleteExternalForkService(forkPath)

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

func TestBrowseDirectoriesService(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		setupFunc   func(*testing.T, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, []directoryEntry)
	}{
		{
			name: "browseValidDirectory",
			setupFunc: func(t *testing.T, path string) {
				require.NoError(t, os.MkdirAll(filepath.Join(path, "dir1"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(path, "dir2"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(path, "file1.txt"), []byte("test"), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, entries []directoryEntry) {
				assert.GreaterOrEqual(t, len(entries), 2)
				for _, entry := range entries {
					assert.True(t, entry.IsDir)
					assert.NotEmpty(t, entry.Name)
					assert.NotEmpty(t, entry.Path)
				}
			},
		},
		{
			name:        "browseNonExistentDirectory",
			path:        "/non/existent/directory",
			setupFunc:   func(_ *testing.T, _ string) {},
			wantErr:     true,
			errContains: "directory does not exist",
		},
		{
			name:      "browseEmptyPath",
			path:      "",
			setupFunc: func(_ *testing.T, _ string) {},
			wantErr:   false,
			validate: func(t *testing.T, entries []directoryEntry) {
				assert.NotNil(t, entries)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := t.TempDir()
			if tt.path != "" && tt.path != "/non/existent/directory" {
				testDir = tt.path
			}

			tt.setupFunc(t, testDir)

			result, err := browseDirectoriesService(tt.path)

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

func TestBrowseDirectoriesServicePermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	dir := t.TempDir()
	restrictedDir := filepath.Join(dir, "restricted")
	require.NoError(t, os.MkdirAll(restrictedDir, 0755))
	require.NoError(t, os.Chmod(restrictedDir, 0000))
	t.Cleanup(func() { _ = os.Chmod(restrictedDir, 0755) })

	_, err := browseDirectoriesService(restrictedDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading directory")
}

func TestBrowseDirectoriesServiceHiddenDirsExcluded(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".hidden"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "visible"), 0755))

	entries, err := browseDirectoriesService(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasPrefix(entry.Name, "."))
	}
}

func TestClassifyWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		expected  workspaceKind
	}{
		{
			name: "classifyStandaloneWorkspace",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			expected: workspaceStandalone,
		},
		{
			name: "classifyRootWorkspace",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".jj", "repo"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			expected: workspaceStandalone,
		},
		{
			name: "classifyForkWorkspace",
			setupFunc: func(t *testing.T, workspacePath string) {
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".jj"), 0755))
				repoFile := filepath.Join(workspacePath, ".jj", "repo")
				require.NoError(t, os.WriteFile(repoFile, []byte("/path/to/parent"), 0644))
				require.NoError(t, os.MkdirAll(filepath.Join(workspacePath, ".sgai"), 0755))
			},
			expected: workspaceFork,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir := t.TempDir()
			server := NewServer(rootDir)

			workspacePath := filepath.Join(rootDir, "test-workspace")
			require.NoError(t, os.MkdirAll(workspacePath, 0755))
			tt.setupFunc(t, workspacePath)

			result := server.classifyWorkspaceCached(workspacePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
