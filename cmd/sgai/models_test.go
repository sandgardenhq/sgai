package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fakeModelsVerboseOutput = `openai/gpt-5.5
{
  "id": "gpt-5.5",
  "providerID": "openai",
  "name": "GPT-5.5",
  "variants": {
    "xhigh": {"reasoningEffort": "xhigh"},
    "high": {"reasoningEffort": "high"}
  }
}
anthropic/claude-sonnet-4.5
{
  "id": "claude-sonnet-4.5",
  "providerID": "anthropic",
  "name": "Claude Sonnet 4.5",
  "variants": {}
}
`

func TestValidateProjectConfigWithOpenCodeModels(t *testing.T) {
	setupFakeOpenCode(t, fakeModelsVerboseOutput, 0)

	tests := []struct {
		name        string
		model       string
		wantErr     bool
		errContains string
	}{
		{
			name:  "base model present",
			model: "openai/gpt-5.5",
		},
		{
			name:  "variant present",
			model: "openai/gpt-5.5 (xhigh)",
		},
		{
			name:        "missing base model",
			model:       "openai/gpt-missing",
			wantErr:     true,
			errContains: "model openai/gpt-missing is not available",
		},
		{
			name:        "missing variant",
			model:       "openai/gpt-5.5 (extreme)",
			wantErr:     true,
			errContains: "variant extreme is not available for model openai/gpt-5.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectConfig(&projectConfig{DefaultModel: tt.model})

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateProjectConfigReturnsOpenCodeFailure(t *testing.T) {
	setupFakeOpenCode(t, "not logged in", 7)

	err := validateProjectConfig(&projectConfig{DefaultModel: "openai/gpt-5.5"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing opencode models")
	assert.Contains(t, err.Error(), "not logged in")
}

func TestParseOpenCodeModelsVerboseReturnsVariantsForModel(t *testing.T) {
	catalog, err := parseOpenCodeModelsVerbose([]byte(fakeModelsVerboseOutput))

	require.NoError(t, err)
	require.Contains(t, catalog, "openai/gpt-5.5")
	assert.Contains(t, catalog["openai/gpt-5.5"].Variants, "xhigh")
	assert.Contains(t, catalog["openai/gpt-5.5"].Variants, "high")
	require.Contains(t, catalog, "anthropic/claude-sonnet-4.5")
	assert.Empty(t, catalog["anthropic/claude-sonnet-4.5"].Variants)
}

func TestParseOpenCodeModelsVerboseRejectsNonObjectModelJSON(t *testing.T) {
	_, err := parseOpenCodeModelsVerbose([]byte("openai/gpt-5.5\nnull\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "model openai/gpt-5.5 JSON must be an object")
}

func TestParseOpenCodeModelsVerboseRejectsMissingOrInvalidVariants(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{
			name: "missing variants",
			model: `openai/gpt-5.5
{
  "name": "GPT-5.5"
}
`,
		},
		{
			name: "null variants",
			model: `openai/gpt-5.5
{
  "name": "GPT-5.5",
  "variants": null
}
`,
		},
		{
			name: "array variants",
			model: `openai/gpt-5.5
{
  "name": "GPT-5.5",
  "variants": []
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseOpenCodeModelsVerbose([]byte(tt.model))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "model openai/gpt-5.5 JSON must contain object-valued variants")
		})
	}
}

func TestHandleAPIListModelsReturnsOpenCodeParseFailure(t *testing.T) {
	setupFakeOpenCode(t, "openai/gpt-5.5\nnull\n", 0)
	server := NewServer(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	w := httptest.NewRecorder()

	server.handleAPIListModels(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "parsing opencode models")
	assert.Contains(t, w.Body.String(), "model openai/gpt-5.5 JSON must be an object")
}

func TestHandleAPIListModelsReturnsOpenCodeFailure(t *testing.T) {
	setupFakeOpenCode(t, "not logged in", 7)
	server := NewServer(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	w := httptest.NewRecorder()

	server.handleAPIListModels(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "listing opencode models")
}

func TestListModelsServiceUsesOpenCodeModels(t *testing.T) {
	setupFakeOpenCode(t, fakeModelsVerboseOutput, 0)
	server := NewServer(t.TempDir())

	result, err := server.listModelsService("")

	require.NoError(t, err)
	assert.Equal(t, []apiModelEntry{
		{ID: "anthropic/claude-sonnet-4.5", Name: "anthropic/claude-sonnet-4.5"},
		{ID: "openai/gpt-5.5", Name: "openai/gpt-5.5"},
	}, result.Models)
}

func setupFakeOpenCode(t *testing.T, output string, exitCode int) {
	t.Helper()

	binDir := t.TempDir()
	name := "opencode"
	if runtime.GOOS == "windows" {
		name = "opencode.bat"
	}
	path := filepath.Join(binDir, name)

	if runtime.GOOS == "windows" {
		content := "@echo off\r\nif not \"%1 %2\"==\"models --verbose\" exit /b 64\r\n"
		content += "type \"" + filepath.Join(binDir, "output.txt") + "\"\r\n"
		content += "exit /b " + strconv.Itoa(exitCode) + "\r\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0755))
	} else {
		content := "#!/bin/sh\nif [ \"$1 $2\" != \"models --verbose\" ]; then exit 64; fi\ncat \"$(dirname \"$0\")/output.txt\"\nexit " + strconv.Itoa(exitCode) + "\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "output.txt"), []byte(output), 0644))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
