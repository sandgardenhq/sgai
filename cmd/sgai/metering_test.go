package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExportedSessionCollectsStepFinish(t *testing.T) {
	data := []byte(`[
		{"type":"step_finish","sessionID":"session-1","timestamp":1700000000000,"part":{"cost":0.02,"model":"openai/gpt-5.5","tokens":{"input":1000,"output":200,"reasoning":50,"cache":{"read":300,"write":20}}}}
	]`)

	steps, children, err := parseExportedSession(data, "session-1", "openai/gpt-5.5")

	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Empty(t, children)
	assert.Equal(t, "session-1", steps[0].SessionID)
	assert.Equal(t, 1000, steps[0].Part.Tokens.Input)
	assert.Equal(t, 300, steps[0].Part.Tokens.Cache.Read)
}

func TestParseExportedSessionCollectsTaskChildren(t *testing.T) {
	data := []byte(`{
		"type":"tool",
		"part":{"tool":"task","state":{"output":"done","metadata":{"sessionID":"child-1"}}}
	}`)

	_, children, err := parseExportedSession(data, "parent-1", "openai/gpt-5.5")

	require.NoError(t, err)
	assert.Equal(t, []string{"child-1"}, children)
}

func TestCollectExportedSessionUsageDedupesRecursiveChildren(t *testing.T) {
	originalExport := exportSessionBytes
	t.Cleanup(func() { exportSessionBytes = originalExport })
	exports := map[string][]byte{
		"parent": []byte(`{"type":"tool","part":{"tool":"task","state":{"output":{"sessionID":"child"}}}}`),
		"child":  []byte(`{"type":"tool","part":{"tool":"task","state":{"output":{"sessionID":"parent"}}}}`),
	}
	exportSessionBytes = func(_ string, sessionID string) ([]byte, error) {
		return exports[sessionID], nil
	}

	usage, err := collectExportedSessionUsage(t.TempDir(), "developer", "parent", "", "openai/gpt-5.5", map[string]bool{})

	require.NoError(t, err)
	require.Len(t, usage, 2)
	assert.Equal(t, "parent", usage[0].SessionID)
	assert.Equal(t, "child", usage[1].SessionID)
	assert.Equal(t, "parent", usage[1].ParentSessionID)
}

func TestReplaceReconciledSessionsRebuildsAggregatesWithoutDoubleCounting(t *testing.T) {
	wf := state.Workflow{
		Cost: state.SessionCost{
			BySession: []state.SessionUsage{{
				SessionID:           "session-1",
				Agent:               "developer",
				MeteredReportedCost: 0.10,
				Tokens:              state.TokenUsage{Input: 10},
			}},
		},
	}
	catalog := testPricingCatalog(t)
	usage := []exportedSessionUsage{{
		SessionID: "session-1",
		Agent:     "developer",
		Model:     "openai/gpt-5.5",
		Steps: []exportedStep{{
			SessionID: "session-1",
			Model:     "openai/gpt-5.5",
			Part: part{
				Cost: 0.20,
				Tokens: partTokens{
					Input:  1000,
					Output: 1000,
				},
			},
		}},
	}}

	replaceReconciledSessions(&wf, usage, catalog, nil)

	require.Len(t, wf.Cost.BySession, 1)
	assert.InDelta(t, 0.035, wf.Cost.TotalCost, 0.0001)
	assert.InDelta(t, 0.20, wf.Cost.MeteredReportedCost, 0.0001)
	assert.Equal(t, 1000, wf.Cost.TotalTokens.Input)
	assert.Equal(t, 1000, wf.Cost.TotalTokens.Output)
}

func TestPriceTokensUsesCacheAndReasoningRates(t *testing.T) {
	catalog := testPricingCatalog(t)

	result := priceTokens(catalog, "openai/gpt-5.5", state.TokenUsage{
		Input:      10_000,
		Output:     10_000,
		Reasoning:  10_000,
		CacheRead:  10_000,
		CacheWrite: 10_000,
	}, nil)

	require.True(t, result.Available)
	assert.InDelta(t, 0.705, result.Cost, 0.0001)
}

func TestPriceTokensUsesContextTier(t *testing.T) {
	catalog := testPricingCatalog(t)

	result := priceTokens(catalog, "openai/gpt-5.5", state.TokenUsage{Input: 200001, Output: 100000}, nil)

	require.True(t, result.Available)
	assert.InDelta(t, 6.50001, result.Cost, 0.0001)
}

func TestLoadModelsDevPricingCatalogUsesStaleCacheOnFetchFailure(t *testing.T) {
	originalFetch := fetchModelsDevCatalog
	t.Cleanup(func() { fetchModelsDevCatalog = originalFetch })
	dir := t.TempDir()
	cachePath := filepath.Join(dir, ".sgai", "models.dev.cache.json")
	cache := modelsDevCache{
		FetchedAt: time.Now().Add(-48 * time.Hour),
		Catalog:   json.RawMessage(`{"openai":{"models":{"gpt-5.5":{"cost":{"input":5,"output":30}}}}}`),
	}
	data, err := json.Marshal(cache)
	require.NoError(t, err)
	require.NoError(t, osWriteFileForTest(cachePath, data))
	fetchModelsDevCatalog = func() ([]byte, error) {
		return nil, errors.New("network down")
	}

	catalog, err := loadModelsDevPricingCatalog(dir, time.Now())

	require.NoError(t, err)
	result := priceTokens(catalog, "openai/gpt-5.5", state.TokenUsage{Input: 1_000_000, Output: 1_000_000}, nil)
	require.True(t, result.Available)
	assert.InDelta(t, 35, result.Cost, 0.0001)
}

func testPricingCatalog(t *testing.T) pricingCatalog {
	t.Helper()
	catalog, err := parsePricingCatalog([]byte(`{
		"openai": {
			"models": {
				"gpt-5.5": {
					"cost": {
						"input": 5,
						"output": 30,
						"cache_read": 0.5,
						"cache_write": 5,
						"tiers": [{"input": 10,"output": 45,"cache_read": 1,"cache_write": 10,"tier": {"type":"context","size": 200000}}]
					}
				}
			}
		}
	}`))
	require.NoError(t, err)
	return catalog
}

func osWriteFileForTest(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
