package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAdhocSessionIDFromRealisticJSONOutput(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "newlineDelimitedStreamEvent",
			data: `{"type":"session","sessionID":"adhoc-stream-session"}` + "\n" +
				`{"type":"text","sessionID":"adhoc-stream-session","part":{"text":"done"}}` + "\n",
			want: "adhoc-stream-session",
		},
		{
			name: "nestedSessionMetadata",
			data: `{"type":"message","metadata":{"session":{"id":"adhoc-nested-session"}}}` + "\n",
			want: "adhoc-nested-session",
		},
		{
			name: "stringEncodedNestedMetadata",
			data: `{"type":"event","message":"{\"session\":{\"id\":\"adhoc-string-session\"}}"}` + "\n",
			want: "adhoc-string-session",
		},
		{
			name: "topLevelSessionIDWinsOverNestedChildSession",
			data: `{"type":"event","sessionID":"adhoc-origin-session","session":{"id":"child-session"},"metadata":{"session":{"id":"metadata-session"}}}` + "\n",
			want: "adhoc-origin-session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findAdhocSessionID([]byte(tt.data))

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindAdhocSessionIDPrefersTopLevelSessionIDDeterministically(t *testing.T) {
	data := []byte(`{"type":"event","sessionID":"adhoc-origin-session","session":{"id":"child-session"},"metadata":{"session":{"id":"metadata-session"}}}`)

	for range 200 {
		got := findAdhocSessionID(data)

		assert.Equal(t, "adhoc-origin-session", got)
	}
}

func TestReconcileAdhocUsageWritesGlobalUsageForOriginWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	workspacePath := setupTestWorkspace(t, rootDir, "adhoc-origin")
	configDir := t.TempDir()
	originalConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return configDir, nil }
	t.Cleanup(func() { userConfigDir = originalConfigDir })

	originalExportSessionBytes := exportSessionBytes
	originalFetchModelsDevCatalog := fetchModelsDevCatalog
	t.Cleanup(func() {
		exportSessionBytes = originalExportSessionBytes
		fetchModelsDevCatalog = originalFetchModelsDevCatalog
	})

	timestamp := time.Date(2026, 5, 2, 10, 30, 0, 0, time.UTC)
	exportSessionBytes = func(dir, sessionID string) ([]byte, error) {
		assert.Equal(t, workspacePath, dir)
		assert.Equal(t, "adhoc-full-chain-session", sessionID)
		return []byte(fmt.Sprintf(`{"messages":[{"info":{"metadata":{"time":{"completed":%d}}},"parts":[{"type":"step-finish","sessionID":"adhoc-full-chain-session","model":"openai/gpt-5.5","cost":0.042,"tokens":{"input":1000,"output":200,"reasoning":50,"cache":{"read":20,"write":10}}}]}]}`, timestamp.UnixMilli())), nil
	}
	fetchModelsDevCatalog = func() ([]byte, error) {
		return nil, errors.New("pricing unavailable in test")
	}

	output := `{"type":"session","metadata":{"session":{"id":"adhoc-full-chain-session"}}}` + "\n" +
		`{"type":"text","sessionID":"adhoc-full-chain-session","part":{"text":"human readable result"}}` + "\n"

	server.reconcileAdhocUsage(workspacePath, output, "openai/gpt-5.5")

	storePath := filepath.Join(configDir, "sgai", "usage.sqlite")
	store, errOpen := openUsageStore(storePath)
	require.NoError(t, errOpen)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-02"), To: mustDate(t, "2026-05-02")})

	require.NoError(t, errQuery)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "adhoc", resp.Rows[0].Source)
	assert.Equal(t, workspacePath, resp.Rows[0].WorkspacePath)
	assert.Equal(t, filepath.Base(workspacePath), resp.Rows[0].Project)
	assert.Equal(t, workspacePath, resp.Rows[0].RootWorkspacePath)
	assert.Equal(t, filepath.Base(workspacePath), resp.Rows[0].RootProject)
	assert.InDelta(t, 0.042, resp.Totals.MeteredReportedCost, 0.0001)
	assert.Equal(t, 1000, resp.Totals.Tokens.Input)
	assert.Equal(t, 200, resp.Totals.Tokens.Output)
}

func TestReconcileAdhocUsageAttributesForkWorkspaceToRootWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	rootWorkspacePath := setupTestWorkspace(t, rootDir, "root-workspace")
	forkWorkspacePath := setupTestWorkspace(t, rootDir, "fork-workspace")
	repoDir := filepath.Join(rootWorkspacePath, ".jj", "repo")
	require.NoError(t, os.MkdirAll(repoDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(forkWorkspacePath, ".jj"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(forkWorkspacePath, ".jj", "repo"), []byte(repoDir), 0o644))

	configDir := t.TempDir()
	originalConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return configDir, nil }
	t.Cleanup(func() { userConfigDir = originalConfigDir })

	originalExportSessionBytes := exportSessionBytes
	originalFetchModelsDevCatalog := fetchModelsDevCatalog
	t.Cleanup(func() {
		exportSessionBytes = originalExportSessionBytes
		fetchModelsDevCatalog = originalFetchModelsDevCatalog
	})

	timestamp := time.Date(2026, 5, 3, 11, 30, 0, 0, time.UTC)
	exportSessionBytes = func(dir, sessionID string) ([]byte, error) {
		assert.Equal(t, forkWorkspacePath, dir)
		assert.Equal(t, "adhoc-fork-session", sessionID)
		return []byte(fmt.Sprintf(`{"messages":[{"info":{"metadata":{"time":{"completed":%d}}},"parts":[{"type":"step-finish","sessionID":"adhoc-fork-session","model":"openai/gpt-5.5","cost":0.084,"tokens":{"input":2000,"output":400,"cache":{}}}]}]}`, timestamp.UnixMilli())), nil
	}
	fetchModelsDevCatalog = func() ([]byte, error) {
		return nil, errors.New("pricing unavailable in test")
	}

	server.reconcileAdhocUsage(forkWorkspacePath, `{"type":"session","sessionID":"adhoc-fork-session"}`, "openai/gpt-5.5")

	storePath := filepath.Join(configDir, "sgai", "usage.sqlite")
	store, errOpen := openUsageStore(storePath)
	require.NoError(t, errOpen)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-03"), To: mustDate(t, "2026-05-03")})

	require.NoError(t, errQuery)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "adhoc", resp.Rows[0].Source)
	assert.Equal(t, forkWorkspacePath, resp.Rows[0].WorkspacePath)
	assert.Equal(t, filepath.Base(forkWorkspacePath), resp.Rows[0].Project)
	assert.Equal(t, rootWorkspacePath, resp.Rows[0].RootWorkspacePath)
	assert.Equal(t, filepath.Base(rootWorkspacePath), resp.Rows[0].RootProject)
	assert.InDelta(t, 0.084, resp.Totals.MeteredReportedCost, 0.0001)
	assert.Equal(t, 2000, resp.Totals.Tokens.Input)
	assert.Equal(t, 400, resp.Totals.Tokens.Output)
}
