package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalUsageDBPathUsesConfigDirectory(t *testing.T) {
	configDir := t.TempDir()

	path, err := globalUsageDBPath(func() (string, error) { return configDir, nil })

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(configDir, "sgai", "usage.sqlite"), path)
	assert.NotContains(t, path, string(filepath.Separator)+".sgai"+string(filepath.Separator))
}

func TestOpenGlobalUsageStoreCreatesConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	original := userConfigDir
	userConfigDir = func() (string, error) { return configDir, nil }
	t.Cleanup(func() { userConfigDir = original })

	store, err := openGlobalUsageStore()

	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	info, errStat := os.Stat(filepath.Join(configDir, "sgai"))
	require.NoError(t, errStat)
	assert.True(t, info.IsDir())
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestOpenUsageStoreMigratesEmptyDatabase(t *testing.T) {
	store := openTestUsageStore(t)

	var tableName string
	errQuery := store.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='usage_steps'").Scan(&tableName)

	require.NoError(t, errQuery)
	assert.Equal(t, "usage_steps", tableName)
}

func TestUsageStoreReplaceSessionIsIdempotent(t *testing.T) {
	store := openTestUsageStore(t)
	ctx := testUsageContext(t)
	first := []state.SessionUsage{{
		SessionID: "session-1",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "step-1",
			Agent:     "go",
			SessionID: "session-1",
			Cost:      1.25,
			Tokens:    state.TokenUsage{Input: 100, Output: 20},
			Timestamp: "2026-05-01T10:00:00Z",
		}},
	}}
	second := []state.SessionUsage{{
		SessionID: "session-1",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "step-1",
			Agent:     "go",
			SessionID: "session-1",
			Cost:      2.50,
			Tokens:    state.TokenUsage{Input: 200, Output: 40},
			Timestamp: "2026-05-01T10:00:00Z",
		}},
	}}

	require.NoError(t, store.replaceSessionUsage("session", ctx, first))
	require.NoError(t, store.replaceSessionUsage("session", ctx, second))
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-01"), To: mustDate(t, "2026-05-01")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 2.50, resp.Totals.Cost, 0.0001)
	assert.Equal(t, 200, resp.Totals.Tokens.Input)
	require.Len(t, resp.Rows, 1)
}

func TestUsageStoreQueryAggregatesDailyAndFiltersProjects(t *testing.T) {
	store := openTestUsageStore(t)
	ctxA := usageWorkspaceContext{WorkspacePath: "/tmp/a", WorkspaceName: "a", RootWorkspacePath: "/tmp/root-a", RootWorkspaceName: "root-a"}
	ctxB := usageWorkspaceContext{WorkspacePath: "/tmp/b", WorkspaceName: "b", RootWorkspacePath: "/tmp/root-b", RootWorkspaceName: "root-b"}
	require.NoError(t, store.replaceSessionUsage("session", ctxA, []state.SessionUsage{{
		SessionID: "a-session",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:                     "a-1",
			Agent:                      "go",
			SessionID:                  "a-session",
			Cost:                       1.00,
			MeteredReportedCost:        0.75,
			APIEquivalentCost:          1.00,
			APIEquivalentCostAvailable: true,
			Tokens:                     state.TokenUsage{Input: 10, Output: 20, Reasoning: 30, CacheRead: 40, CacheWrite: 50},
			Timestamp:                  "2026-05-01T01:00:00Z",
		}},
	}}))
	require.NoError(t, store.replaceSessionUsage("adhoc", ctxB, []state.SessionUsage{{
		SessionID: "b-session",
		Agent:     "adhoc",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "b-1",
			Agent:     "adhoc",
			SessionID: "b-session",
			Cost:      3.00,
			Tokens:    state.TokenUsage{Input: 100},
			Timestamp: "2026-05-02T01:00:00Z",
		}},
	}}))

	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-01"), To: mustDate(t, "2026-05-02"), Project: "a", RootProject: "root-a"})

	require.NoError(t, errQuery)
	assert.InDelta(t, 1.00, resp.Totals.Cost, 0.0001)
	assert.InDelta(t, 0.75, resp.Totals.MeteredReportedCost, 0.0001)
	assert.InDelta(t, 1.00, resp.Totals.APIEquivalentCost, 0.0001)
	assert.True(t, resp.Totals.APIEquivalentCostAvailable)
	assert.Equal(t, state.TokenUsage{Input: 10, Output: 20, Reasoning: 30, CacheRead: 40, CacheWrite: 50}, resp.Totals.Tokens)
	assert.Equal(t, []usageDailyPoint{{Date: "2026-05-01", Cost: 1.00}}, resp.Daily)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "a", resp.Rows[0].Project)
	assert.Equal(t, []string{"a", "b"}, resp.Filters.Projects)
	assert.Equal(t, []string{"root-a", "root-b"}, resp.Filters.RootProjects)
}

func TestUsageStoreQueryPrefersLiveSessionRowsOverDuplicateBackfillRows(t *testing.T) {
	store := openTestUsageStore(t)
	ctx := testUsageContext(t)
	sessions := []state.SessionUsage{{
		SessionID: "session-1",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "step-1",
			Agent:     "go",
			SessionID: "session-1",
			Cost:      5.00,
			Tokens:    state.TokenUsage{Input: 500},
			Timestamp: "2026-05-07T01:00:00Z",
		}},
	}}
	require.NoError(t, store.replaceSessionUsage("session", ctx, sessions))
	require.NoError(t, store.replaceWorkspaceUsage(ctx, sessions))

	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-07"), To: mustDate(t, "2026-05-07")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 5.00, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 500}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "session", resp.Rows[0].Source)
}

func TestUsageStoreQueryPrefersLiveSessionRowsOverAgentFallbackBackfillRows(t *testing.T) {
	store := openTestUsageStore(t)
	ctx := testUsageContext(t)
	fallbackSessions := agentStepSessions(ctx, []state.AgentCost{{
		Agent: "go",
		Steps: []state.StepCost{{
			StepID:    "same-underlying-step",
			Agent:     "go",
			Cost:      5.00,
			Tokens:    state.TokenUsage{Input: 500},
			Timestamp: "2026-05-10T01:00:00Z",
		}},
	}})
	liveSessions := []state.SessionUsage{{
		SessionID: "real-session-1",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "same-underlying-step",
			Agent:     "go",
			SessionID: "real-session-1",
			Cost:      5.00,
			Tokens:    state.TokenUsage{Input: 500},
			Timestamp: "2026-05-10T01:00:00Z",
		}},
	}}

	require.NoError(t, store.replaceWorkspaceUsage(ctx, fallbackSessions))
	require.NoError(t, store.replaceSessionUsage("session", ctx, liveSessions))
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-10"), To: mustDate(t, "2026-05-10")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 5.00, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 500}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "session", resp.Rows[0].Source)

	server := NewServer(t.TempDir())
	server.usageStore = store
	apiResp := serveUsageHTTP(server, "/api/v1/usage?from=2026-05-10&to=2026-05-10")
	assert.Equal(t, http.StatusOK, apiResp.Code)
	var got usageResponse
	require.NoError(t, json.Unmarshal(apiResp.Body.Bytes(), &got))
	assert.InDelta(t, 5.00, got.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 500}, got.Totals.Tokens)
}

func TestUsageStoreQueryKeepsDistinctAgentFallbackAndLiveRows(t *testing.T) {
	store := openTestUsageStore(t)
	ctx := testUsageContext(t)
	fallbackSessions := agentStepSessions(ctx, []state.AgentCost{{
		Agent: "go",
		Steps: []state.StepCost{{
			StepID:    "fallback-only-step",
			Agent:     "go",
			Cost:      2.00,
			Tokens:    state.TokenUsage{Input: 200},
			Timestamp: "2026-05-11T01:00:00Z",
		}},
	}})
	liveSessions := []state.SessionUsage{{
		SessionID: "real-session-2",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "live-only-step",
			Agent:     "go",
			SessionID: "real-session-2",
			Cost:      3.00,
			Tokens:    state.TokenUsage{Input: 300},
			Timestamp: "2026-05-11T01:00:00Z",
		}},
	}}

	require.NoError(t, store.replaceWorkspaceUsage(ctx, fallbackSessions))
	require.NoError(t, store.replaceSessionUsage("session", ctx, liveSessions))
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-11"), To: mustDate(t, "2026-05-11")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 5.00, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 500}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 1)
	assert.Contains(t, resp.Rows[0].Source, "backfill")
	assert.Contains(t, resp.Rows[0].Source, "session")
}

func TestUsageAPIRejectsInvalidFilters(t *testing.T) {
	server := NewServer(t.TempDir())
	server.usageStore = openTestUsageStore(t)

	resp := serveUsageHTTP(server, "/api/v1/usage?from=bad-date")

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUsageAPIHandlesUnavailableStoreGracefully(t *testing.T) {
	server := NewServer(t.TempDir())
	server.usageStore = nil
	server.usageStoreErr = assert.AnError

	resp := serveUsageHTTP(server, "/api/v1/usage")

	assert.Equal(t, http.StatusOK, resp.Code)
	var got usageResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))
	assert.Equal(t, "usage storage unavailable", got.Warning)
	assert.Empty(t, got.Rows)
}

func TestUsageAPIReturnsTypedEmptyResponse(t *testing.T) {
	server := NewServer(t.TempDir())
	server.usageStore = openTestUsageStore(t)

	resp := serveUsageHTTP(server, "/api/v1/usage?from=2026-05-01&to=2026-05-02")

	assert.Equal(t, http.StatusOK, resp.Code)
	var got usageResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))
	assert.NotNil(t, got.Rows)
	assert.NotNil(t, got.Daily)
	assert.NotNil(t, got.Filters.Projects)
	assert.NotNil(t, got.Filters.RootProjects)
}

func TestUsageRefreshBackfillsBeforeQuery(t *testing.T) {
	rootDir := t.TempDir()
	workspaceDir := filepath.Join(rootDir, "refresh-workspace")
	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "refresh-session",
		Agent:     "coordinator",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "refresh-step",
			Agent:     "coordinator",
			SessionID: "refresh-session",
			Cost:      1.50,
			Tokens:    state.TokenUsage{Input: 150},
			Timestamp: "2026-05-12T01:00:00Z",
		}},
	}}}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	resp := serveUsageHTTPMethod(server, http.MethodPost, "/api/v1/usage/refresh?from=2026-05-12&to=2026-05-12")

	assert.Equal(t, http.StatusOK, resp.Code)
	var got usageResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))
	assert.InDelta(t, 1.50, got.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 150}, got.Totals.Tokens)
	require.Len(t, got.Rows, 1)
}

func TestUsageBackfillBeforeDeletePreservesHistoricalRows(t *testing.T) {
	rootDir := t.TempDir()
	workspaceDir := filepath.Join(rootDir, "delete-workspace")
	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "delete-session",
		Agent:     "coordinator",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "delete-step",
			Agent:     "coordinator",
			SessionID: "delete-session",
			Cost:      2.25,
			Tokens:    state.TokenUsage{Input: 225},
			Timestamp: "2026-05-13T01:00:00Z",
		}},
	}}}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	result, errDelete := server.deleteWorkspaceService(workspaceDir)
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-13"), To: mustDate(t, "2026-05-13")})

	require.NoError(t, errDelete)
	assert.True(t, result.Deleted)
	require.NoError(t, errQuery)
	assert.NoDirExists(t, workspaceDir)
	assert.InDelta(t, 2.25, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 225}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 1)
}

func TestUsageBackfillSkipsMalformedStateAndStaysIdempotent(t *testing.T) {
	rootDir := t.TempDir()
	goodDir := filepath.Join(rootDir, "good")
	badDir := filepath.Join(rootDir, "bad")
	require.NoError(t, os.MkdirAll(filepath.Join(goodDir, ".sgai"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(badDir, ".sgai"), 0o755))
	goodState := state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "backfill-session",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "backfill-step",
			Agent:     "go",
			SessionID: "backfill-session",
			Cost:      4.00,
			Tokens:    state.TokenUsage{Input: 400},
			Timestamp: "2026-05-03T01:00:00Z",
		}},
	}}}}
	data, errJSON := json.Marshal(goodState)
	require.NoError(t, errJSON)
	require.NoError(t, os.WriteFile(filepath.Join(goodDir, ".sgai", "state.json"), data, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, ".sgai", "state.json"), []byte(`{bad`), 0o644))
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	server.backfillGlobalUsage()
	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-03"), To: mustDate(t, "2026-05-03")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 4.00, resp.Totals.Cost, 0.0001)
	require.Len(t, resp.Rows, 1)
}

func TestUsageBackfillIncludesRootWorkspaceState(t *testing.T) {
	rootDir := t.TempDir()
	rootName := filepath.Base(rootDir)
	writeWorkflowStateForUsageTest(t, rootDir, state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "root-session",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "root-step",
			Agent:     "go",
			SessionID: "root-session",
			Cost:      3.00,
			Tokens:    state.TokenUsage{Input: 300},
			Timestamp: "2026-05-08T01:00:00Z",
		}},
	}}}})
	writeWorkflowStateForUsageTest(t, filepath.Join(rootDir, "child"), state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "child-session",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "child-step",
			Agent:     "go",
			SessionID: "child-session",
			Cost:      4.00,
			Tokens:    state.TokenUsage{Input: 400},
			Timestamp: "2026-05-08T01:00:00Z",
		}},
	}}}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-08"), To: mustDate(t, "2026-05-08")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 7.00, resp.Totals.Cost, 0.0001)
	require.Len(t, resp.Rows, 2)
	rootRow := usageRowForProject(t, resp.Rows, rootName)
	assert.Equal(t, rootName, rootRow.RootProject)
	assert.Equal(t, rootDir, rootRow.WorkspacePath)
	assert.Equal(t, rootDir, rootRow.RootWorkspacePath)
}

func TestUsageBackfillIncludesExternalWorkspace(t *testing.T) {
	rootDir := t.TempDir()
	externalDir := filepath.Join(t.TempDir(), "external-workspace")
	writeWorkflowStateForUsageTest(t, externalDir, state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "external-session",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "external-step",
			Agent:     "go",
			SessionID: "external-session",
			Cost:      7.00,
			Tokens:    state.TokenUsage{Input: 700},
			Timestamp: "2026-05-14T01:00:00Z",
		}},
	}}}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store
	server.externalDirs[resolveSymlinks(externalDir)] = true

	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-14"), To: mustDate(t, "2026-05-14")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 7.00, resp.Totals.Cost, 0.0001)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "external-workspace", resp.Rows[0].Project)
}

func TestUsageBackfillKeepsHistoricalRowsForValidEmptyState(t *testing.T) {
	rootDir := t.TempDir()
	workspaceDir := filepath.Join(rootDir, "empty-after-usage")
	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{Cost: state.SessionCost{BySession: []state.SessionUsage{{
		SessionID: "stale-session",
		Agent:     "go",
		Model:     "openai/gpt-5.5",
		Steps: []state.StepCost{{
			StepID:    "stale-step",
			Agent:     "go",
			SessionID: "stale-session",
			Cost:      6.00,
			Tokens:    state.TokenUsage{Input: 600},
			Timestamp: "2026-05-09T01:00:00Z",
		}},
	}}}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	server.backfillGlobalUsage()
	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{})
	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-09"), To: mustDate(t, "2026-05-09")})

	require.NoError(t, errQuery)
	require.Len(t, resp.Rows, 1)
	assert.InDelta(t, 6.00, resp.Totals.Cost, 0.0001)
}

func TestUsageStoreWritesSessionSummaryWhenStepsAreMissing(t *testing.T) {
	store := openTestUsageStore(t)
	ctx := testUsageContext(t)
	require.NoError(t, store.replaceSessionUsage("session", ctx, []state.SessionUsage{{
		SessionID:                    "summary-session",
		Agent:                        "coordinator",
		Model:                        "openai/gpt-5.5",
		Tokens:                       state.TokenUsage{Input: 120, Output: 30, Reasoning: 10, CacheRead: 500},
		MeteredReportedCost:          0.50,
		APIEquivalentCost:            0.75,
		APIEquivalentCostAvailable:   true,
		APIEquivalentCostUnavailable: "",
	}}))

	today := dateOnly(time.Now().UTC())
	resp, errQuery := store.query(usageQuery{From: today, To: today})

	require.NoError(t, errQuery)
	require.Len(t, resp.Rows, 1)
	assert.InDelta(t, 0.75, resp.Totals.Cost, 0.0001)
	assert.InDelta(t, 0.50, resp.Totals.MeteredReportedCost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 120, Output: 30, Reasoning: 10, CacheRead: 500}, resp.Totals.Tokens)
}

func TestUsageBackfillImportsAgentStepsWhenSessionsHaveNoUsableSteps(t *testing.T) {
	rootDir := t.TempDir()
	workspaceDir := filepath.Join(rootDir, "agent-only")
	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{Cost: state.SessionCost{
		TotalTokens: state.TokenUsage{Input: 150, Output: 25, Reasoning: 10, CacheRead: 500},
		ByAgent: []state.AgentCost{{
			Agent:  "coordinator",
			Cost:   1.75,
			Tokens: state.TokenUsage{Input: 150, Output: 25, Reasoning: 10, CacheRead: 500},
			Steps: []state.StepCost{{
				StepID:    "coordinator-step-1",
				Agent:     "coordinator",
				Cost:      1.75,
				Tokens:    state.TokenUsage{Input: 150, Output: 25, Reasoning: 10, CacheRead: 500},
				Timestamp: "2026-05-04T01:00:00Z",
			}},
		}},
		BySession: []state.SessionUsage{{
			SessionID: "empty-session",
			Agent:     "coordinator",
			Model:     "openai/gpt-5.5",
		}},
	}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	server.backfillGlobalUsage()
	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-04"), To: mustDate(t, "2026-05-04")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 1.75, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 150, Output: 25, Reasoning: 10, CacheRead: 500}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, "agent-only", resp.Rows[0].Project)
	assert.Equal(t, "agent-only", resp.Rows[0].RootProject)
}

func TestUsageBackfillPrefersSessionStepsOverAgentAggregateSteps(t *testing.T) {
	rootDir := t.TempDir()
	workspaceDir := filepath.Join(rootDir, "mixed")
	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{Cost: state.SessionCost{
		TotalTokens: state.TokenUsage{Input: 999},
		ByAgent: []state.AgentCost{{
			Agent:  "go",
			Cost:   9.99,
			Tokens: state.TokenUsage{Input: 999},
			Steps: []state.StepCost{{
				StepID:    "agent-step-1",
				Agent:     "go",
				Cost:      9.99,
				Tokens:    state.TokenUsage{Input: 999},
				Timestamp: "2026-05-05T01:00:00Z",
			}},
		}},
	}})
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store
	server.backfillGlobalUsage()

	writeWorkflowStateForUsageTest(t, workspaceDir, state.Workflow{Cost: state.SessionCost{
		TotalTokens: state.TokenUsage{Input: 999},
		ByAgent: []state.AgentCost{{
			Agent:  "go",
			Cost:   9.99,
			Tokens: state.TokenUsage{Input: 999},
			Steps: []state.StepCost{{
				StepID:    "agent-step-1",
				Agent:     "go",
				Cost:      9.99,
				Tokens:    state.TokenUsage{Input: 999},
				Timestamp: "2026-05-05T01:00:00Z",
			}},
		}},
		BySession: []state.SessionUsage{{
			SessionID: "session-1",
			Agent:     "go",
			Model:     "openai/gpt-5.5",
			Steps: []state.StepCost{{
				StepID:    "session-step-1",
				Agent:     "go",
				SessionID: "session-1",
				Cost:      2.00,
				Tokens:    state.TokenUsage{Input: 200},
				Timestamp: "2026-05-05T01:00:00Z",
			}},
		}},
	}})

	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-05"), To: mustDate(t, "2026-05-05")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 2.00, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 200}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 1)
}

func TestUsageBackfillKeepsAgentFallbackRowsDistinctAcrossWorkspaces(t *testing.T) {
	rootDir := t.TempDir()
	for _, workspaceName := range []string{"fair-teal-8356", "vast-sage-oe7o"} {
		writeWorkflowStateForUsageTest(t, filepath.Join(rootDir, workspaceName), state.Workflow{Cost: state.SessionCost{
			TotalTokens: state.TokenUsage{Input: 100},
			ByAgent: []state.AgentCost{{
				Agent:  "coordinator",
				Cost:   1.00,
				Tokens: state.TokenUsage{Input: 100},
				Steps: []state.StepCost{{
					StepID:    "coordinator-step-1",
					Agent:     "coordinator",
					Cost:      1.00,
					Tokens:    state.TokenUsage{Input: 100},
					Timestamp: "2026-05-06T01:00:00Z",
				}},
			}},
		}})
	}
	store := openTestUsageStore(t)
	server := NewServer(rootDir)
	server.usageStore = store

	server.backfillGlobalUsage()
	resp, errQuery := store.query(usageQuery{From: mustDate(t, "2026-05-06"), To: mustDate(t, "2026-05-06")})

	require.NoError(t, errQuery)
	assert.InDelta(t, 2.00, resp.Totals.Cost, 0.0001)
	assert.Equal(t, state.TokenUsage{Input: 200}, resp.Totals.Tokens)
	require.Len(t, resp.Rows, 2)
}

func openTestUsageStore(t *testing.T) *usageStore {
	t.Helper()
	store, err := openUsageStore(filepath.Join(t.TempDir(), "usage.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	return store
}

func testUsageContext(t *testing.T) usageWorkspaceContext {
	t.Helper()
	dir := t.TempDir()
	return usageWorkspaceContext{
		WorkspacePath:     dir,
		WorkspaceName:     filepath.Base(dir),
		RootWorkspacePath: dir,
		RootWorkspaceName: filepath.Base(dir),
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.DateOnly, value)
	require.NoError(t, err)
	return parsed
}

func serveUsageHTTP(server *Server, target string) *httptest.ResponseRecorder {
	return serveUsageHTTPMethod(server, http.MethodGet, target)
}

func serveUsageHTTPMethod(server *Server, method, target string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func writeWorkflowStateForUsageTest(t *testing.T, workspaceDir string, wf state.Workflow) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(workspaceDir, ".sgai"), 0o755))
	data, errJSON := json.Marshal(wf)
	require.NoError(t, errJSON)
	require.NoError(t, os.WriteFile(filepath.Join(workspaceDir, ".sgai", "state.json"), data, 0o644))
}

func TestUsageStoreOpenMigrationFailureIsReturned(t *testing.T) {
	dir := t.TempDir()

	_, err := openUsageStore(dir)

	require.Error(t, err)
	assert.ErrorContains(t, err, "opening usage database")
}

func TestUsageStoreUsesRealSQLDB(t *testing.T) {
	store := openTestUsageStore(t)

	assert.IsType(t, &sql.DB{}, store.db)
}

func TestEnsureUsageStoreConcurrentAccess(t *testing.T) {
	configDir := t.TempDir()
	original := userConfigDir
	userConfigDir = func() (string, error) { return configDir, nil }
	t.Cleanup(func() { userConfigDir = original })
	server := NewServer(t.TempDir())

	var wg sync.WaitGroup
	stores := make(chan *usageStore, 32)
	errs := make(chan error, 32)
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store, errStore := server.ensureUsageStore()
			stores <- store
			errs <- errStore
		}()
	}
	wg.Wait()
	close(stores)
	close(errs)

	for errStore := range errs {
		require.NoError(t, errStore)
	}
	var first *usageStore
	for store := range stores {
		require.NotNil(t, store)
		if first == nil {
			first = store
			continue
		}
		assert.Same(t, first, store)
	}
	require.NoError(t, first.close())
}

func usageRowForProject(t *testing.T, rows []usageRow, project string) usageRow {
	t.Helper()
	for _, row := range rows {
		if row.Project == project {
			return row
		}
	}
	t.Fatalf("usage row for project %q not found in %#v", project, rows)
	return usageRow{}
}
