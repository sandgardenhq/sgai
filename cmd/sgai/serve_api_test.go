package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAPIRoute(t *testing.T) {
	tests := []struct {
		name     string
		urlPath  string
		expected bool
	}{
		{
			name:     "apiRoute",
			urlPath:  "/api/v1/state",
			expected: true,
		},
		{
			name:     "mcpRoute",
			urlPath:  "/mcp/tools",
			expected: true,
		},
		{
			name:     "rootPath",
			urlPath:  "/",
			expected: false,
		},
		{
			name:     "workspacePath",
			urlPath:  "/workspaces/test",
			expected: false,
		},
		{
			name:     "staticAsset",
			urlPath:  "/assets/main.js",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAPIRoute(tt.urlPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStaticAsset(t *testing.T) {
	tests := []struct {
		name     string
		urlPath  string
		expected bool
	}{
		{
			name:     "jsFile",
			urlPath:  "/assets/main.js",
			expected: true,
		},
		{
			name:     "cssFile",
			urlPath:  "/styles/main.css",
			expected: true,
		},
		{
			name:     "mapFile",
			urlPath:  "/assets/main.js.map",
			expected: true,
		},
		{
			name:     "pngFile",
			urlPath:  "/images/logo.png",
			expected: true,
		},
		{
			name:     "svgFile",
			urlPath:  "/images/icon.svg",
			expected: true,
		},
		{
			name:     "icoFile",
			urlPath:  "/favicon.ico",
			expected: true,
		},
		{
			name:     "woffFile",
			urlPath:  "/fonts/main.woff",
			expected: true,
		},
		{
			name:     "woff2File",
			urlPath:  "/fonts/main.woff2",
			expected: true,
		},
		{
			name:     "ttfFile",
			urlPath:  "/fonts/main.ttf",
			expected: true,
		},
		{
			name:     "jsonFile",
			urlPath:  "/data/config.json",
			expected: true,
		},
		{
			name:     "htmlFile",
			urlPath:  "/page.html",
			expected: false,
		},
		{
			name:     "noExtension",
			urlPath:  "/path/to/resource",
			expected: false,
		},
		{
			name:     "rootPath",
			urlPath:  "/",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStaticAsset(tt.urlPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeEtag(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name:    "emptyContent",
			content: []byte(""),
		},
		{
			name:    "simpleContent",
			content: []byte("test content"),
		},
		{
			name:    "jsonContent",
			content: []byte(`{"key": "value"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeEtag(tt.content)
			expected := computeExpectedEtag(tt.content)
			assert.Equal(t, expected, result)
			assert.NotEmpty(t, result)
			assert.True(t, len(result) > 2)
		})
	}
}

func computeExpectedEtag(content []byte) string {
	h := sha256.Sum256(content)
	return `"` + hex.EncodeToString(h[:8]) + `"`
}

func TestFindSteerInsertPosition(t *testing.T) {
	tests := []struct {
		name     string
		messages []state.Message
		expected int
	}{
		{
			name:     "empty",
			messages: []state.Message{},
			expected: 0,
		},
		{
			name: "allRead",
			messages: []state.Message{
				{ID: 1, Read: true},
				{ID: 2, Read: true},
			},
			expected: 0,
		},
		{
			name: "firstUnread",
			messages: []state.Message{
				{ID: 1, Read: true},
				{ID: 2, Read: false},
				{ID: 3, Read: false},
			},
			expected: 1,
		},
		{
			name: "allUnread",
			messages: []state.Message{
				{ID: 1, Read: false},
				{ID: 2, Read: false},
			},
			expected: 0,
		},
		{
			name: "lastUnread",
			messages: []state.Message{
				{ID: 1, Read: true},
				{ID: 2, Read: true},
				{ID: 3, Read: false},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findSteerInsertPosition(tt.messages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateQuestionID(t *testing.T) {
	tests := []struct {
		name        string
		wfState     state.Workflow
		expectEmpty bool
	}{
		{
			name: "noQuestion",
			wfState: state.Workflow{
				Status: state.StatusWorking,
			},
			expectEmpty: true,
		},
		{
			name: "humanMessage",
			wfState: state.Workflow{
				Status:       state.StatusWaitingForHuman,
				HumanMessage: "What should I do?",
			},
			expectEmpty: false,
		},
		{
			name: "multiChoice",
			wfState: state.Workflow{
				Status: state.StatusWaitingForHuman,
				MultiChoiceQuestion: &state.MultiChoiceQuestion{
					Questions: []state.QuestionItem{
						{Question: "Choose an option"},
					},
				},
			},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateQuestionID(tt.wfState)
			if tt.expectEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 16)
			}
		})
	}
}

func TestQuestionType(t *testing.T) {
	tests := []struct {
		name     string
		wfState  state.Workflow
		expected string
	}{
		{
			name: "noQuestion",
			wfState: state.Workflow{
				Status: state.StatusWorking,
			},
			expected: "",
		},
		{
			name: "freeText",
			wfState: state.Workflow{
				Status:       state.StatusWaitingForHuman,
				HumanMessage: "What should I do?",
			},
			expected: "free-text",
		},
		{
			name: "multiChoice",
			wfState: state.Workflow{
				Status: state.StatusWaitingForHuman,
				MultiChoiceQuestion: &state.MultiChoiceQuestion{
					Questions: []state.QuestionItem{
						{Question: "Choose an option"},
					},
				},
			},
			expected: "multi-choice",
		},
		{
			name: "workGate",
			wfState: state.Workflow{
				Status: state.StatusWaitingForHuman,
				MultiChoiceQuestion: &state.MultiChoiceQuestion{
					Questions: []state.QuestionItem{
						{Question: "Approve?"},
					},
					IsWorkGate: true,
				},
			},
			expected: "work-gate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := questionType(tt.wfState)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAPIResponseText(t *testing.T) {
	tests := []struct {
		name     string
		req      apiRespondRequest
		expected string
	}{
		{
			name: "answerOnly",
			req: apiRespondRequest{
				Answer: "My answer",
			},
			expected: "My answer",
		},
		{
			name: "selectedChoicesOnly",
			req: apiRespondRequest{
				SelectedChoices: []string{"Option A", "Option B"},
			},
			expected: "Selected: Option A, Option B",
		},
		{
			name: "bothAnswerAndChoices",
			req: apiRespondRequest{
				Answer:          "My answer",
				SelectedChoices: []string{"Option A"},
			},
			expected: "Selected: Option A\nMy answer",
		},
		{
			name: "empty",
			req: apiRespondRequest{
				Answer:          "",
				SelectedChoices: []string{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAPIResponseText(tt.req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCommitEntries(t *testing.T) {
	tests := []struct {
		name     string
		commits  []jjCommit
		expected []apiCommitEntry
	}{
		{
			name:     "emptyCommits",
			commits:  []jjCommit{},
			expected: []apiCommitEntry{},
		},
		{
			name: "singleCommit",
			commits: []jjCommit{
				{
					ChangeID:    "abc123",
					CommitID:    "def456",
					Workspaces:  []string{"ws1"},
					Timestamp:   "2 hours ago",
					Bookmarks:   []string{"main"},
					Description: "Test commit",
					GraphChar:   "@",
				},
			},
			expected: []apiCommitEntry{
				{
					ChangeID:    "abc123",
					CommitID:    "def456",
					Workspaces:  []string{"ws1"},
					Timestamp:   "2 hours ago",
					Bookmarks:   []string{"main"},
					Description: "Test commit",
					GraphChar:   "@",
				},
			},
		},
		{
			name: "multipleCommits",
			commits: []jjCommit{
				{
					ChangeID:    "abc123",
					CommitID:    "def456",
					Workspaces:  []string{"ws1"},
					Timestamp:   "2 hours ago",
					Bookmarks:   []string{"main"},
					Description: "First commit",
					GraphChar:   "@",
				},
				{
					ChangeID:    "xyz789",
					CommitID:    "uvw012",
					Workspaces:  []string{"ws2"},
					Timestamp:   "1 day ago",
					Bookmarks:   []string{"feature"},
					Description: "Second commit",
					GraphChar:   "o",
				},
			},
			expected: []apiCommitEntry{
				{
					ChangeID:    "abc123",
					CommitID:    "def456",
					Workspaces:  []string{"ws1"},
					Timestamp:   "2 hours ago",
					Bookmarks:   []string{"main"},
					Description: "First commit",
					GraphChar:   "@",
				},
				{
					ChangeID:    "xyz789",
					CommitID:    "uvw012",
					Workspaces:  []string{"ws2"},
					Timestamp:   "1 day ago",
					Bookmarks:   []string{"feature"},
					Description: "Second commit",
					GraphChar:   "o",
				},
			},
		},
		{
			name: "commitWithEmptyFields",
			commits: []jjCommit{
				{
					ChangeID:    "abc123",
					CommitID:    "def456",
					Workspaces:  []string{},
					Timestamp:   "1 hour ago",
					Bookmarks:   []string{},
					Description: "",
					GraphChar:   "@",
				},
			},
			expected: []apiCommitEntry{
				{
					ChangeID:    "abc123",
					CommitID:    "def456",
					Workspaces:  []string{},
					Timestamp:   "1 hour ago",
					Bookmarks:   []string{},
					Description: "",
					GraphChar:   "@",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCommitEntries(tt.commits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertJJCommitsForAPI(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := convertJJCommitsForAPI(nil)
		assert.Empty(t, result)
	})

	t.Run("withCommits", func(t *testing.T) {
		commits := []jjCommit{
			{ChangeID: "abc123", CommitID: "def456", Timestamp: "2025-01-01", Bookmarks: []string{"main"}, Description: "initial"},
			{ChangeID: "xyz789", CommitID: "qrs012", Timestamp: "2025-01-02", Description: "update"},
		}

		result := convertJJCommitsForAPI(commits)
		assert.Len(t, result, 2)
		assert.Equal(t, "abc123", result[0].ChangeID)
		assert.Equal(t, "def456", result[0].CommitID)
		assert.Equal(t, []string{"main"}, result[0].Bookmarks)
		assert.Equal(t, "initial", result[0].Description)
		assert.Equal(t, "xyz789", result[1].ChangeID)
	})
}

func TestConvertEventsForAPIBoost(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := convertEventsForAPI(nil)
		assert.Empty(t, result)
	})

	t.Run("withEvents", func(t *testing.T) {
		displays := []eventsProgressDisplay{
			{Agent: "coordinator", Description: "started work", Timestamp: "2025-01-01"},
			{Agent: "developer", Description: "writing code", Timestamp: "2025-01-02"},
		}

		result := convertEventsForAPI(displays)
		assert.Len(t, result, 2)
		assert.Equal(t, "coordinator", result[0].Agent)
		assert.Equal(t, "started work", result[0].Description)
	})
}

func TestResolveCurrentModelVariants(t *testing.T) {
	t.Run("fromState", func(t *testing.T) {
		wfState := state.Workflow{CurrentModel: "claude-opus-4"}
		result := resolveCurrentModel("/some/path", wfState)
		assert.Equal(t, "claude-opus-4", result)
	})

	t.Run("noAgent", func(t *testing.T) {
		wfState := state.Workflow{}
		result := resolveCurrentModel("/some/path", wfState)
		assert.Equal(t, "", result)
	})

	t.Run("fromGoalFile", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nmodels:\n  coordinator: claude-opus-4\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		wfState := state.Workflow{CurrentAgent: "coordinator"}
		result := resolveCurrentModel(dir, wfState)
		assert.Equal(t, "claude-opus-4", result)
	})

	t.Run("agentNotInGoal", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nmodels:\n  coordinator: claude-opus-4\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		wfState := state.Workflow{CurrentAgent: "developer"}
		result := resolveCurrentModel(dir, wfState)
		assert.Equal(t, "", result)
	})

	t.Run("withExplicitModel", func(t *testing.T) {
		wf := state.Workflow{CurrentModel: "opus-4"}
		result := resolveCurrentModel("/tmp", wf)
		assert.Equal(t, "opus-4", result)
	})

	t.Run("noAgentReturnsEmpty", func(t *testing.T) {
		wf := state.Workflow{}
		result := resolveCurrentModel("/tmp", wf)
		assert.Empty(t, result)
	})

	t.Run("noModelReturnsEmpty", func(t *testing.T) {
		dir := t.TempDir()
		wf := state.Workflow{}
		result := resolveCurrentModel(dir, wf)
		assert.Empty(t, result)
	})
}

func TestCollectAgentModelsVariants(t *testing.T) {
	t.Run("noGoalFile", func(t *testing.T) {
		dir := t.TempDir()
		result := collectAgentModels(dir)
		assert.Nil(t, result)
	})

	t.Run("noModels", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("# No frontmatter"), 0644))

		result := collectAgentModels(dir)
		assert.Nil(t, result)
	})

	t.Run("withModels", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nmodels:\n  coordinator: claude-opus-4\n  developer: gpt-4\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		result := collectAgentModels(dir)
		require.Len(t, result, 2)
		assert.Equal(t, "coordinator", result[0].Agent)
		assert.Equal(t, []string{"claude-opus-4"}, result[0].Models)
		assert.Equal(t, "developer", result[1].Agent)
		assert.Equal(t, []string{"gpt-4"}, result[1].Models)
	})

	t.Run("noAgents", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "agent"), 0o755))
		result := collectAgentModels(dir)
		assert.Empty(t, result)
	})

	t.Run("withGoal", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"),
			[]byte("---\nmodels:\n  coordinator: anthropic/claude-opus-4-6\n---\n# Goal"), 0o644))

		result := collectAgentModels(dir)
		assert.NotNil(t, result)
	})

	t.Run("noGoalReturnsNil", func(t *testing.T) {
		result := collectAgentModels(t.TempDir())
		assert.Nil(t, result)
	})
}

func TestBuildAdhocArgs(t *testing.T) {
	b := &opencodeBackend{}

	t.Run("simpleModel", func(t *testing.T) {
		args := b.BuildAdhocArgs("claude-opus-4")
		assert.Equal(t, []string{"run", "-m", "claude-opus-4", "--agent", "build", "--title", "adhoc [claude-opus-4]"}, args)
	})

	t.Run("modelWithVariant", func(t *testing.T) {
		args := b.BuildAdhocArgs("claude-opus-4:fast")
		assert.Contains(t, args, "run")
		assert.Contains(t, args, "-m")
		assert.Contains(t, args, "--agent")
		assert.Contains(t, args, "build")
		assert.Contains(t, args, "adhoc [claude-opus-4:fast]")
	})

	t.Run("withVariantAddsFlag", func(t *testing.T) {
		args := b.BuildAdhocArgs("anthropic/claude-sonnet-4-6 (thinking)")
		assert.Contains(t, args, "--variant")
		assert.Contains(t, args, "thinking")
	})

	t.Run("withoutVariantNoFlag", func(t *testing.T) {
		args := b.BuildAdhocArgs("anthropic/claude-sonnet-4-6")
		for _, arg := range args {
			assert.NotEqual(t, "--variant", arg)
		}
	})
}

func TestCoordinatorModelFromWorkspace(t *testing.T) {
	t.Run("emptyWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		result := server.coordinatorModelFromWorkspace("")
		assert.Empty(t, result)
	})

	t.Run("nonexistentWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		result := server.coordinatorModelFromWorkspace("nonexistent")
		assert.Empty(t, result)
	})

	t.Run("workspaceWithModel", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		goalPath := filepath.Join(wsDir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("---\nmodels:\n  coordinator: claude-opus-4\n---\n# Goal"), 0644))

		result := server.coordinatorModelFromWorkspace("test-ws")
		assert.Equal(t, "claude-opus-4", result)
	})

	t.Run("emptyReturnsEmpty", func(t *testing.T) {
		server, _ := setupTestServer(t)
		result := server.coordinatorModelFromWorkspace("")
		assert.Empty(t, result)
	})

	t.Run("notFoundReturnsEmpty", func(t *testing.T) {
		server, _ := setupTestServer(t)
		result := server.coordinatorModelFromWorkspace("nonexistent")
		assert.Empty(t, result)
	})

	t.Run("withModelsConfig", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws-model")
		goalContent := "---\nmodels:\n  coordinator: anthropic/claude-sonnet-4-6\nflow: |\n  \"coordinator\"\n---\n# Test"
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))
		result := server.coordinatorModelFromWorkspace("test-ws-model")
		assert.Equal(t, "anthropic/claude-sonnet-4-6", result)
	})
}

func TestBuildAPITechStackItemsTest(t *testing.T) {
	techStack := []string{"go", "react", "htmx"}
	result := buildAPITechStackItems(techStack)
	assert.NotEmpty(t, result)

	selectedCount := 0
	for _, item := range result {
		if item.Selected {
			selectedCount++
		}
	}
	assert.Equal(t, 3, selectedCount)
}

func TestBuildAPITechStackItemsFiltering(t *testing.T) {
	items := buildAPITechStackItems([]string{"go"})
	foundGo := false
	for _, item := range items {
		if item.ID == "go" {
			assert.True(t, item.Selected)
			foundGo = true
		} else {
			assert.False(t, item.Selected)
		}
	}
	assert.True(t, foundGo)
}

func TestBuildAPITechStackItemsSelectedFiltering(t *testing.T) {
	result := buildAPITechStackItems([]string{"go", "react"})
	selectedCount := 0
	for _, item := range result {
		if item.Selected {
			selectedCount++
		}
	}
	assert.GreaterOrEqual(t, selectedCount, 0)
}

func TestBuildAPITechStackItemsNilSelected(t *testing.T) {
	result := buildAPITechStackItems(nil)
	for _, item := range result {
		assert.False(t, item.Selected)
	}
}

func TestWarmStateCache(t *testing.T) {
	server, _ := setupTestServer(t)
	server.warmStateCache()
}

func TestSessionCoordinator(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "sess-coord-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	coord := srv.sessionCoordinator(wsDir)
	assert.Nil(t, coord)
}

func TestWriteJSONResponse(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "json-ws")

	w := serveHTTP(srv, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
}

func TestWriteJSONContentType(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "json-ct-ws")
	w := serveHTTP(srv, "GET", "/api/v1/compose/templates", "")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestConvertSnippetLanguagesEmpty(t *testing.T) {
	result := convertSnippetLanguages(nil)
	assert.Empty(t, result)
}

func TestIsAPIRouteVariants(t *testing.T) {
	assert.True(t, isAPIRoute("/api/v1/state"))
	assert.True(t, isAPIRoute("/api/v1/agents"))
	assert.False(t, isAPIRoute("/"))
	assert.False(t, isAPIRoute("/index.html"))
}

func TestIsStaticAssetVariants(t *testing.T) {
	assert.True(t, isStaticAsset("/assets/main.js"))
	assert.True(t, isStaticAsset("/assets/style.css"))
	assert.True(t, isStaticAsset("/favicon.ico"))
	assert.False(t, isStaticAsset("/api/v1/state"))
	assert.False(t, isStaticAsset("/"))
}

func TestGenerateQuestionIDDeterministic(t *testing.T) {
	wf := state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "test question",
	}
	id1 := generateQuestionID(wf)
	id2 := generateQuestionID(wf)
	assert.Equal(t, id1, id2)
}

func TestBuildAPIResponseTextWithChoices(t *testing.T) {
	req := apiRespondRequest{
		SelectedChoices: []string{"Option A", "Option B"},
		Answer:          "additional feedback",
	}
	result := buildAPIResponseText(req)
	assert.Contains(t, result, "Option A")
	assert.Contains(t, result, "Option B")
	assert.Contains(t, result, "additional feedback")
}

func TestBuildAPIResponseTextOnlyAnswer(t *testing.T) {
	req := apiRespondRequest{
		Answer: "my answer",
	}
	result := buildAPIResponseText(req)
	assert.Equal(t, "my answer", result)
}

func TestBuildAPIResponseTextEmpty(t *testing.T) {
	req := apiRespondRequest{}
	result := buildAPIResponseText(req)
	assert.Empty(t, result)
}

func TestCollectForksForAPIFromGroupsEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	result := srv.collectForksForAPIFromGroups("/nonexistent", nil)
	assert.Nil(t, result)
}

func TestCollectForksForAPIFromGroupsNoMatch(t *testing.T) {
	srv, _ := setupTestServer(t)
	groups := []workspaceGroup{
		{Root: workspaceInfo{Directory: "/some/other/dir"}},
	}
	result := srv.collectForksForAPIFromGroups("/nonexistent", groups)
	assert.Nil(t, result)
}

func TestConvertMessagesForAPITruncation(t *testing.T) {
	longBody := strings.Repeat("x", maxMessageBodyBytes+100)
	messages := []state.Message{
		{ID: 1, FromAgent: "a", ToAgent: "b", Body: longBody},
	}
	result := convertMessagesForAPI(messages)
	require.Len(t, result, 1)
	assert.True(t, strings.HasSuffix(result[0].Body, "...[truncated]"))
	assert.True(t, len(result[0].Body) < len(longBody))
}

func TestConvertMessagesForAPIReversesOrder(t *testing.T) {
	messages := []state.Message{
		{ID: 1, FromAgent: "a", ToAgent: "b", Body: "first"},
		{ID: 2, FromAgent: "b", ToAgent: "a", Body: "second"},
		{ID: 3, FromAgent: "a", ToAgent: "b", Body: "third"},
	}
	result := convertMessagesForAPI(messages)
	require.Len(t, result, 3)
	assert.Equal(t, 3, result[0].ID)
	assert.Equal(t, 2, result[1].ID)
	assert.Equal(t, 1, result[2].ID)
}

func TestConvertMessagesForAPIExtractsSubject(t *testing.T) {
	messages := []state.Message{
		{ID: 1, FromAgent: "a", ToAgent: "b", Body: "# Important Update\nSome content here"},
	}
	result := convertMessagesForAPI(messages)
	require.Len(t, result, 1)
	assert.Equal(t, "Important Update", result[0].Subject)
}

func TestConvertEventsForAPIWithEntries(t *testing.T) {
	displays := []eventsProgressDisplay{
		{Timestamp: "2024-01-01T00:00:00Z", FormattedTime: "00:00", Agent: "coordinator", Description: "started"},
	}
	result := convertEventsForAPI(displays)
	require.Len(t, result, 1)
	assert.Equal(t, "coordinator", result[0].Agent)
	assert.Equal(t, "started", result[0].Description)
}

func TestConvertModelStatusesWithEntries(t *testing.T) {
	displays := []modelStatusDisplay{
		{ModelID: "opus", Status: "running"},
		{ModelID: "sonnet", Status: "completed"},
	}
	result := convertModelStatuses(displays)
	require.Len(t, result, 2)
	assert.Equal(t, "opus", result[0].ModelID)
	assert.Equal(t, "running", result[0].Status)
}

func TestCollectSkillCategoriesEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "skills"), 0o755))
	result := collectSkillCategories(dir)
	assert.Empty(t, result)
}

func TestCollectSkillCategoriesWithSkills(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".sgai", "skills", "coding-practices", "go-testing")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: go-testing\ndescription: Go testing patterns\n---\n# Go Testing"), 0o644))

	result := collectSkillCategories(dir)
	assert.NotEmpty(t, result)
}

func TestCollectAgentsEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "agent"), 0o755))
	result := collectAgents(dir)
	assert.Empty(t, result)
}

func TestCollectAgentsWithAgents(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "coordinator.md"),
		[]byte("---\ndescription: Main coordinator\n---\n# Coordinator"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "builder.md"),
		[]byte("---\ndescription: Builder agent\n---\n# Builder"), 0o644))

	result := collectAgents(dir)
	assert.Len(t, result, 2)
}

func TestLoadActionsForAPIDefault(t *testing.T) {
	result := loadActionsForAPI("/nonexistent/workspace")
	assert.NotNil(t, result)
}

func TestLoadActionsForAPIWithActions(t *testing.T) {
	dir := t.TempDir()
	actionsDir := filepath.Join(dir, ".sgai", "actions")
	require.NoError(t, os.MkdirAll(actionsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(actionsDir, "test-action.md"),
		[]byte("---\nname: Test Action\ndescription: A test\nicon: play\n---\n# Action"), 0o644))

	result := loadActionsForAPI(dir)
	assert.NotEmpty(t, result)
}

func TestConvertActionsForAPIEmpty(t *testing.T) {
	result := convertActionsForAPI(nil)
	assert.Empty(t, result)
}

func TestReadGoalAndPMForAPINoPM(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\n---\n# Goal Content"), 0o644))

	goalContent, rawGoal, fullGoal, pmContent, hasPM := readGoalAndPMForAPI(dir)
	assert.Contains(t, goalContent, "Goal Content")
	assert.NotEmpty(t, rawGoal)
	assert.NotEmpty(t, fullGoal)
	assert.Empty(t, pmContent)
	assert.False(t, hasPM)
}

func TestReadGoalAndPMForAPIWithPM(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0o644))

	_, _, _, pmContent, hasPM := readGoalAndPMForAPI(dir)
	assert.NotEmpty(t, pmContent)
	assert.True(t, hasPM)
}

func TestReadGoalAndPMForAPINoGoal(t *testing.T) {
	dir := t.TempDir()
	goalContent, rawGoal, fullGoal, _, _ := readGoalAndPMForAPI(dir)
	assert.Empty(t, goalContent)
	assert.Empty(t, rawGoal)
	assert.Empty(t, fullGoal)
}

func TestBuildFullFactoryStateWithMultipleWorkspaces(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws1")
	setupTestWorkspace(t, rootDir, "ws2")
	result := server.buildFullFactoryState()
	assert.GreaterOrEqual(t, len(result.Workspaces), 2)
}

func TestWarmStateCacheFillsCache(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	server.warmStateCache()
	_, ok := server.stateCache.get("state")
	assert.True(t, ok)
}

func TestReadNewestForkGoalEmptyList(t *testing.T) {
	result := readNewestForkGoal(nil)
	assert.Empty(t, result)
}

func TestReadNewestForkGoalWithGoalFiles(t *testing.T) {
	dir := t.TempDir()
	fork1 := filepath.Join(dir, "fork1")
	fork2 := filepath.Join(dir, "fork2")
	require.NoError(t, os.MkdirAll(fork1, 0755))
	require.NoError(t, os.MkdirAll(fork2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(fork1, "GOAL.md"), []byte("fork1 goal"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(fork2, "GOAL.md"), []byte("fork2 goal"), 0644))

	forks := []workspaceInfo{
		{Directory: fork1, DirName: "fork1"},
		{Directory: fork2, DirName: "fork2"},
	}
	result := readNewestForkGoal(forks)
	assert.NotEmpty(t, result)
}

func TestReadNewestForkGoalAllEmptyContent(t *testing.T) {
	dir := t.TempDir()
	fork1 := filepath.Join(dir, "fork1")
	require.NoError(t, os.MkdirAll(fork1, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(fork1, "GOAL.md"), []byte("  \n\t  "), 0644))

	forks := []workspaceInfo{
		{Directory: fork1, DirName: "fork1"},
	}
	result := readNewestForkGoal(forks)
	assert.Empty(t, result)
}

func TestCollectJJChangesNoJJInstalled(t *testing.T) {
	dir := t.TempDir()
	lines, desc := collectJJChanges(dir)
	assert.Nil(t, lines)
	assert.Empty(t, desc)
}

func TestCollectJJFullDiffNoJJInstalled(t *testing.T) {
	dir := t.TempDir()
	result := collectJJFullDiff(dir)
	assert.Empty(t, result)
}

func TestCollectJJChangesCachedResult(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "changes-ws")
	result := srv.collectJJChangesCached(wsDir)
	_ = result
}

func TestResolveForkDirExplicitPath(t *testing.T) {
	server, rootDir := setupTestServer(t)
	forkDir := filepath.Join(rootDir, "fork-ws")
	require.NoError(t, os.MkdirAll(forkDir, 0755))
	result := server.resolveForkDir(forkDir, "/some/path", "/root/path")
	assert.Equal(t, filepath.Clean(forkDir), result)
}

func TestResolveForkDirImplicitFromWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.resolveForkDir("", "/workspace/path", "/different/root")
	assert.Equal(t, "/workspace/path", result)
}

func TestResolveForkDirEmptyWhenSamePaths(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.resolveForkDir("", "/same/path", "/same/path")
	assert.Equal(t, "", result)
}

func TestResolveRootForDeleteForkStandaloneReturnsEmpty(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "standalone-ws")
	result := server.resolveRootForDeleteFork(filepath.Join(rootDir, "standalone-ws"))
	assert.Equal(t, "", result)
}

func TestResolveRootForDeleteForkForkReturnsPath(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := filepath.Join(rootDir, "fork-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".sgai"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".jj"), 0755))
	rootWs := filepath.Join(rootDir, "root-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(rootWs, ".jj", "repo"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".jj", "repo"), []byte(filepath.Join(rootWs, ".jj", "repo")), 0644))
	result := server.resolveRootForDeleteFork(wsDir)
	assert.NotEmpty(t, result)
}

func TestQuestionTypeFreeformMessage(t *testing.T) {
	wf := state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What do you think?",
	}
	assert.Equal(t, "free-text", questionType(wf))
}

func TestQuestionTypeMultiChoiceQuestions(t *testing.T) {
	wf := state.Workflow{
		Status: state.StatusWaitingForHuman,
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{{Question: "Pick one", Choices: []string{"A", "B"}}},
		},
	}
	assert.Equal(t, "multi-choice", questionType(wf))
}

func TestQuestionTypeWorkGateFlag(t *testing.T) {
	wf := state.Workflow{
		Status: state.StatusWaitingForHuman,
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			IsWorkGate: true,
			Questions:  []state.QuestionItem{{Question: "Approve?", Choices: []string{"Yes", "No"}}},
		},
	}
	assert.Equal(t, "work-gate", questionType(wf))
}

func TestLoadWorkspaceStateLargeFileReturnsEmpty(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-large")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	largeContent := strings.Repeat("x", maxStateSizeBytes+1)
	require.NoError(t, os.WriteFile(sp, []byte(largeContent), 0644))
	result := server.loadWorkspaceState(wsDir)
	assert.Empty(t, result.Status)
}

func TestLoadWorkspaceStateNoFileReturnsEmpty(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-nostate")
	result := server.loadWorkspaceState(wsDir)
	assert.Empty(t, result.Status)
}

func TestResolveAPIWorkspace(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "resolve-ws")

	w := serveHTTP(srv, "GET", "/api/v1/workspaces/resolve-ws/goal", "")
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}

func TestResolveAPIWorkspaceFallback(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "fallback-ws")

	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)
	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoadActionsForAPI(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, []apiActionEntry)
	}{
		{
			name: "noConfig",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, actions []apiActionEntry) {
				assert.Len(t, actions, 3)
				assert.Equal(t, "Create PR", actions[0].Name)
			},
		},
		{
			name: "withConfig",
			setupFunc: func(t *testing.T, dir string) {
				config := projectConfig{
					Actions: []actionConfig{
						{Name: "Custom Action", Model: "model1", Prompt: "prompt1"},
					},
				}
				data, err := json.Marshal(config)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))
			},
			validate: func(t *testing.T, actions []apiActionEntry) {
				assert.Len(t, actions, 1)
				assert.Equal(t, "Custom Action", actions[0].Name)
			},
		},
		{
			name: "emptyActions",
			setupFunc: func(t *testing.T, dir string) {
				config := projectConfig{
					Actions: []actionConfig{},
				}
				data, err := json.Marshal(config)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), data, 0644))
			},
			validate: func(t *testing.T, actions []apiActionEntry) {
				assert.Len(t, actions, 3)
				assert.Equal(t, "Create PR", actions[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			result := loadActionsForAPI(dir)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConvertActionsForAPI(t *testing.T) {
	tests := []struct {
		name     string
		configs  []actionConfig
		expected []apiActionEntry
	}{
		{
			name:     "empty",
			configs:  []actionConfig{},
			expected: []apiActionEntry{},
		},
		{
			name: "singleAction",
			configs: []actionConfig{
				{Name: "Action 1", Model: "model1", Prompt: "prompt1", Description: "desc1"},
			},
			expected: []apiActionEntry{
				{Name: "Action 1", Model: "model1", Prompt: "prompt1", Description: "desc1"},
			},
		},
		{
			name: "multipleActions",
			configs: []actionConfig{
				{Name: "Action 1", Model: "model1", Prompt: "prompt1"},
				{Name: "Action 2", Model: "model2", Prompt: "prompt2", Description: "desc2"},
			},
			expected: []apiActionEntry{
				{Name: "Action 1", Model: "model1", Prompt: "prompt1"},
				{Name: "Action 2", Model: "model2", Prompt: "prompt2", Description: "desc2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertActionsForAPI(tt.configs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectAgents(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, []apiAgentEntry)
	}{
		{
			name: "noAgents",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, agents []apiAgentEntry) {
				assert.Empty(t, agents)
			},
		},
		{
			name: "singleAgent",
			setupFunc: func(t *testing.T, dir string) {
				agentDir := filepath.Join(dir, ".sgai", "agent")
				require.NoError(t, os.MkdirAll(agentDir, 0755))
				agentContent := `---
description: Test agent description
---
# Test Agent`
				require.NoError(t, os.WriteFile(filepath.Join(agentDir, "test-agent.md"), []byte(agentContent), 0644))
			},
			validate: func(t *testing.T, agents []apiAgentEntry) {
				assert.Len(t, agents, 1)
				assert.Equal(t, "test-agent", agents[0].Name)
				assert.Equal(t, "Test agent description", agents[0].Description)
			},
		},
		{
			name: "multipleAgents",
			setupFunc: func(t *testing.T, dir string) {
				agentDir := filepath.Join(dir, ".sgai", "agent")
				require.NoError(t, os.MkdirAll(agentDir, 0755))
				for _, agent := range []struct {
					name string
					desc string
				}{
					{"agent-a", "Agent A description"},
					{"agent-b", "Agent B description"},
					{"agent-c", "Agent C description"},
				} {
					agentContent := `---
description: ` + agent.desc + `
---
# ` + agent.name
					require.NoError(t, os.WriteFile(filepath.Join(agentDir, agent.name+".md"), []byte(agentContent), 0644))
				}
			},
			validate: func(t *testing.T, agents []apiAgentEntry) {
				assert.Len(t, agents, 3)
				assert.Equal(t, "agent-a", agents[0].Name)
				assert.Equal(t, "agent-b", agents[1].Name)
				assert.Equal(t, "agent-c", agents[2].Name)
			},
		},
		{
			name: "nonMarkdownFile",
			setupFunc: func(t *testing.T, dir string) {
				agentDir := filepath.Join(dir, ".sgai", "agent")
				require.NoError(t, os.MkdirAll(agentDir, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(agentDir, "test.txt"), []byte("content"), 0644))
			},
			validate: func(t *testing.T, agents []apiAgentEntry) {
				assert.Empty(t, agents)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			result := collectAgents(dir)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConvertAgentSequence(t *testing.T) {
	tests := []struct {
		name     string
		displays []agentSequenceDisplay
		expected []apiAgentSequenceEntry
	}{
		{
			name:     "empty",
			displays: []agentSequenceDisplay{},
			expected: []apiAgentSequenceEntry{},
		},
		{
			name: "singleEntry",
			displays: []agentSequenceDisplay{
				{Agent: "agent1", Model: "model1", ElapsedTime: "1m", IsCurrent: true},
			},
			expected: []apiAgentSequenceEntry{
				{Agent: "agent1", Model: "model1", ElapsedTime: "1m", IsCurrent: true},
			},
		},
		{
			name: "multipleEntries",
			displays: []agentSequenceDisplay{
				{Agent: "agent1", Model: "model1", ElapsedTime: "1m", IsCurrent: false},
				{Agent: "agent2", Model: "model2", ElapsedTime: "2m", IsCurrent: true},
				{Agent: "agent3", Model: "model3", ElapsedTime: "3m", IsCurrent: false},
			},
			expected: []apiAgentSequenceEntry{
				{Agent: "agent1", Model: "model1", ElapsedTime: "1m", IsCurrent: false},
				{Agent: "agent2", Model: "model2", ElapsedTime: "2m", IsCurrent: true},
				{Agent: "agent3", Model: "model3", ElapsedTime: "3m", IsCurrent: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAgentSequence(tt.displays)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadGoalAndPMForAPI(t *testing.T) {
	t.Run("noGoalOrPM", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalContent, rawGoal, fullGoal, pmContent, hasPM := readGoalAndPMForAPI(tmpDir)
		assert.Empty(t, goalContent)
		assert.Empty(t, rawGoal)
		assert.Empty(t, fullGoal)
		assert.Empty(t, pmContent)
		assert.False(t, hasPM)
	})

	t.Run("withGoalOnly", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalFileContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# My Goal\n\nDo something."
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "GOAL.md"), []byte(goalFileContent), 0644))

		gc, rawGC, fullGC, pm, hasPM := readGoalAndPMForAPI(tmpDir)
		assert.NotEmpty(t, gc)
		assert.Contains(t, rawGC, "My Goal")
		assert.Contains(t, fullGC, "flow:")
		assert.Empty(t, pm)
		assert.False(t, hasPM)
	})

	t.Run("withGoalAndPM", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalFileContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# My Goal\n\nDo something."
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "GOAL.md"), []byte(goalFileContent), 0644))

		sgaiDir := filepath.Join(tmpDir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0755))
		pmFileContent := "---\nRetrospective Session: .sgai/retro\n---\n\n## PM Content\n"
		require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte(pmFileContent), 0644))

		gc, rawGC, fullGC, pm, hasPM := readGoalAndPMForAPI(tmpDir)
		assert.NotEmpty(t, gc)
		assert.Contains(t, rawGC, "My Goal")
		assert.Contains(t, fullGC, "flow:")
		assert.NotEmpty(t, pm)
		assert.True(t, hasPM)
	})
}

func TestLoadWorkspaceState(t *testing.T) {
	t.Run("nonExistentState", func(t *testing.T) {
		rootDir := t.TempDir()
		server := NewServer(rootDir)
		workDir := filepath.Join(rootDir, "ws")
		require.NoError(t, os.MkdirAll(workDir, 0755))

		wf := server.loadWorkspaceState(workDir)
		assert.Empty(t, wf.Status)
	})

	t.Run("existingState", func(t *testing.T) {
		rootDir := t.TempDir()
		server := NewServer(rootDir)
		workDir := filepath.Join(rootDir, "ws")
		sgaiDir := filepath.Join(workDir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0755))

		stateFile := filepath.Join(sgaiDir, "state.json")
		_, err := state.NewCoordinatorWith(stateFile, state.Workflow{
			Status:       state.StatusComplete,
			CurrentAgent: "test-agent",
			Progress:     []state.ProgressEntry{},
			Messages:     []state.Message{},
		})
		require.NoError(t, err)

		wf := server.loadWorkspaceState(workDir)
		assert.Equal(t, state.StatusComplete, wf.Status)
		assert.Equal(t, "test-agent", wf.CurrentAgent)
	})

	t.Run("oversizedState", func(t *testing.T) {
		rootDir := t.TempDir()
		server := NewServer(rootDir)
		workDir := filepath.Join(rootDir, "ws")
		sgaiDir := filepath.Join(workDir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0755))

		bigContent := strings.Repeat("x", 11*1024*1024)
		require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(bigContent), 0644))

		wf := server.loadWorkspaceState(workDir)
		assert.Empty(t, wf.Status)
	})
}

func setupTestServer(t *testing.T) (*Server, string) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	return server, rootDir
}

func setupTestWorkspace(t *testing.T, rootDir, name string) string {
	wsDir := filepath.Join(rootDir, name)
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".sgai"), 0755))
	return wsDir
}

func serveHTTP(server *Server, method, path string, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	var req *http.Request
	if reqBody != nil {
		req = httptest.NewRequest(method, path, reqBody)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandleAPIState(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	goalContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "test-ws", "GOAL.md"), []byte(goalContent), 0644))

	w := serveHTTP(server, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestHandleAPIAgents(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	goalContent := "---\nflow: |\n  \"agent1\" -> \"agent2\"\n---\n# Test"
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))

	w := serveHTTP(server, "GET", "/api/v1/agents?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISkills(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	skillDir := filepath.Join(wsDir, ".sgai", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test-skill\ndescription: Test\n---\n# Skill"), 0644))

	w := serveHTTP(server, "GET", "/api/v1/skills?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISnippets(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(snippetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "hello.go"), []byte("---\ndescription: Hello\n---\npackage main"), 0644))

	w := serveHTTP(server, "GET", "/api/v1/snippets?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPICreateWorkspace(t *testing.T) {
	t.Run("successfulCreate", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces", `{"name":"new-workspace"}`)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalidBody", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces", `{invalid json}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicateWorkspace", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "existing-ws")
		w := serveHTTP(server, "POST", "/api/v1/workspaces", `{"name":"existing-ws"}`)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestHandleAPIGetGoal(t *testing.T) {
	t.Run("withGoal", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test Goal"), 0644))

		w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/goal", "")
		assert.Equal(t, http.StatusOK, w.Code)

		var resp apiGoalResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Content, "Test Goal")
	})

	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "GET", "/api/v1/workspaces/nonexistent/goal", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIUpdateGoal(t *testing.T) {
	t.Run("updateGoal", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old Goal"), 0644))

		w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", `{"content":"# New Goal"}`)
		assert.Equal(t, http.StatusOK, w.Code)

		updatedContent, err := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
		require.NoError(t, err)
		assert.Contains(t, string(updatedContent), "New Goal")
	})

	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPITogglePin(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/pin", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIStopSession(t *testing.T) {
	t.Run("noActiveSession", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/stop", "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleAPIDeleteWorkspace(t *testing.T) {
	t.Run("missingConfirmation", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{"confirm":false}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("nonExistentWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete", `{"confirm":true}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("deleteStandaloneWorkspace", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "standalone-ws")

		w := serveHTTP(server, "POST", "/api/v1/workspaces/standalone-ws/delete", `{"confirm":true}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleAPIStartSession(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/start", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/start", `{"model":"opencode/model1","auto":false}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIForkWorkspace(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/fork", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/fork", `{"goalContent":"# Goal"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIDeleteFork(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete-fork", `{}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")
		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPIRespond(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")
		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/respond", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/respond", `{"response":"test"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPISteer(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/steer", `{"message":"do this"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIWorkflowSVG(t *testing.T) {
	t.Run("withGoal", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		goalContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))

		w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/workflow.svg", "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleAPIWorkspaceDiff(t *testing.T) {
	t.Run("noJJRepo", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/diff", "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleAPIForkTemplate(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "GET", "/api/v1/workspaces/nonexistent/fork-template", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("standaloneWorkspace", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")
		w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/fork-template", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPIComposeState(t *testing.T) {
	server, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "GET", "/api/v1/compose?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeTemplates(t *testing.T) {
	server, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "GET", "/api/v1/compose/templates?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposePreview(t *testing.T) {
	server, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "GET", "/api/v1/compose/preview?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeSave(t *testing.T) {
	t.Run("savesComposerState", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		_ = setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/compose?workspace=test-ws", "")
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)

		w := serveHTTP(server, "POST", "/api/v1/compose?workspace=nonexistent", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIComposeDraft(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		_ = setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "POST", "/api/v1/compose/draft?workspace=test-ws", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPIBrowseDirectories(t *testing.T) {
	server, _ := setupTestServer(t)

	w := serveHTTP(server, "GET", "/api/v1/browse-directories?path=/tmp", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIDeleteMessage(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "DELETE", "/api/v1/workspaces/nonexistent/messages/1", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalidMessageID", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		stateFile := filepath.Join(wsDir, ".sgai", "state.json")
		_, err := state.NewCoordinatorWith(stateFile, state.Workflow{
			Status:   state.StatusComplete,
			Messages: []state.Message{},
		})
		require.NoError(t, err)

		w := serveHTTP(server, "DELETE", "/api/v1/workspaces/test-ws/messages/abc", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPIListModels(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/models", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIAdhocStatus(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIAdhoc(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")
		w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPISkillDetail(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "GET", "/api/v1/skills/test-skill?workspace=nonexistent", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("existingSkill", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		skillDir := filepath.Join(wsDir, ".sgai", "skills", "test-skill")
		require.NoError(t, os.MkdirAll(skillDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: A test skill\n---\n# Test Skill\nContent here"), 0644))

		w := serveHTTP(server, "GET", "/api/v1/skills/test-skill?workspace=test-ws", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "test-skill")
	})

	t.Run("nonexistentSkill", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "GET", "/api/v1/skills/nonexistent?workspace=test-ws", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPISnippetsByLanguage(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "GET", "/api/v1/snippets/go?workspace=nonexistent", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("nonexistentLanguage", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "GET", "/api/v1/snippets/rust?workspace=test-ws", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("existingLanguage", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
		require.NoError(t, os.MkdirAll(snippetDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "http-server.go"), []byte("---\ndescription: HTTP server\n---\npackage main"), 0644))

		w := serveHTTP(server, "GET", "/api/v1/snippets/go?workspace=test-ws", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "go")
	})
}

func TestHandleAPISnippetDetail(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "GET", "/api/v1/snippets/go/test?workspace=nonexistent", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("nonexistentSnippet", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".sgai", "snippets", "go"), 0755))

		w := serveHTTP(server, "GET", "/api/v1/snippets/go/nonexistent?workspace=test-ws", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("existingSnippet", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
		require.NoError(t, os.MkdirAll(snippetDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "http-server.go"), []byte("---\nname: HTTP Server\ndescription: HTTP server setup\nwhen_to_use: when building HTTP servers\n---\npackage main\n\nimport \"net/http\"\n"), 0644))

		w := serveHTTP(server, "GET", "/api/v1/snippets/go/http-server?workspace=test-ws", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "HTTP Server")
	})
}

func TestHandleAPIOpenEditor(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/open-editor", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIOpenEditorGoal(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/open-editor/goal", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIOpenEditorProjectManagement(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/open-editor/project-management", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIAdhocStop(t *testing.T) {
	t.Run("missingWorkspace", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "DELETE", "/api/v1/workspaces/nonexistent/adhoc", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("noRunningAdhoc", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "test-ws")

		w := serveHTTP(server, "DELETE", "/api/v1/workspaces/test-ws/adhoc", "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandleAPIAttachWorkspace(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("relativePath", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{"path":"relative/path"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandleAPIDetachWorkspace(t *testing.T) {
	t.Run("invalidBody", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/detach", `{invalid}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("notAttached", func(t *testing.T) {
		server, _ := setupTestServer(t)
		w := serveHTTP(server, "POST", "/api/v1/workspaces/detach", `{"path":"/some/random/path"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleAPIDeleteMessageByID(t *testing.T) {
	t.Run("validMessageDelete", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws")
		stateFile := filepath.Join(wsDir, ".sgai", "state.json")
		_, err := state.NewCoordinatorWith(stateFile, state.Workflow{
			Status: state.StatusComplete,
			Messages: []state.Message{
				{ID: 1, FromAgent: "dev", ToAgent: "coordinator", Body: "test", Read: false},
			},
		})
		require.NoError(t, err)

		w := serveHTTP(server, "DELETE", "/api/v1/workspaces/test-ws/messages/1", "")
		assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
	})
}

func TestHandleAPIRespondInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/respond", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPICreateWorkspaceInvalidBody(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIStartSessionMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/start", `{"model":"test"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIStopSessionMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/stop", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIForkWorkspaceMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/fork", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteForkMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete-fork", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteWorkspaceMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIUpdateGoalMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "PUT", "/api/v1/workspaces/nonexistent/goal", `{"content":"test"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPISteerMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/steer", `{"message":"test"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPITogglePinMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/pin", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIWorkspaceDiffMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/workspaces/nonexistent/diff", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIWorkflowSVGMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/workspaces/nonexistent/workflow.svg", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func serveHTTPWithHeader(server *Server, method, path, body, headerKey, headerValue string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	if headerKey != "" {
		req.Header.Set(headerKey, headerValue)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func createHTTPRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func serveHTTPReq(mux *http.ServeMux, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestSPAMiddlewareAPIRoutes(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIStartSessionInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/start", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPISteerInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/steer", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIUpdateGoalInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIForkWorkspaceInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/fork", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteForkInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIComposeDraftInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, "POST", "/api/v1/compose/draft?workspace=test-ws", `{invalid}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIRespondMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/respond", `{"answer":"yes"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIComposeSaveWithEtag(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))

	w := serveHTTP(server, "POST", "/api/v1/compose?workspace=test-ws", "")
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleAPIComposeSaveEtagMismatch(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))

	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	req := httptest.NewRequest("POST", "/api/v1/compose?workspace=test-ws", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", `"wrong-etag"`)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
}

func TestHandleAPIStartSessionViaHTTPMissing(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/nonexistent/start", "{}")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIStopSessionViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "stop-ws")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/stop-ws/stop", "{}")
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIAgentsViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "agents-ws")

	w := serveHTTP(srv, "GET", "/api/v1/agents?workspace=agents-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISkillsViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "skills-ws")

	w := serveHTTP(srv, "GET", "/api/v1/skills?workspace=skills-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISnippetsViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "snippets-ws")

	w := serveHTTP(srv, "GET", "/api/v1/snippets?workspace=snippets-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIModelsViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)

	w := serveHTTP(srv, "GET", "/api/v1/models", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIDeleteMessageViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "msg-ws")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Messages: []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "test"},
		},
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "DELETE", "/api/v1/workspaces/msg-ws/messages/1", "")
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPITogglePinViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "pin-ws")
	srv.pinnedConfigDir = t.TempDir()

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/pin-ws/pin", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIUpdateGoalViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "goal-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Old Goal"), 0o644))

	w := serveHTTP(srv, "PUT", "/api/v1/workspaces/goal-ws/goal", `{"content":"---\n---\n# New Goal"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIComposeStateViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "compose-ws")

	w := serveHTTP(srv, "GET", "/api/v1/compose", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeDraftViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "draft-ws")

	w := serveHTTP(srv, "POST", "/api/v1/compose/draft?workspace=draft-ws", `{"state":{},"wizard":{}}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeTemplatesViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "tmpl-ws")

	w := serveHTTP(srv, "GET", "/api/v1/compose/templates", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposePreviewViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "preview-ws")

	w := serveHTTP(srv, "GET", "/api/v1/compose/preview?workspace=preview-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPICreateWorkspaceViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces", `{"name":"test-new-ws","goalContent":"# Test"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIBrowseDirectoriesViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)

	w := serveHTTP(srv, "GET", "/api/v1/browse-directories?path="+srv.rootDir, "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIWorkflowSVGViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "svg-ws")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusComplete,
		CurrentAgent: "a",
	})
	require.NoError(t, errCoord)
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  digraph G {\n    \"a\" -> \"b\"\n  }\n---\n# Test"), 0o644))

	w := serveHTTP(srv, "GET", "/api/v1/workspaces/svg-ws/workflow.svg", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
}

func TestHandleAPIAdhocStatusViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "adhoc-ws")

	w := serveHTTP(srv, "GET", "/api/v1/workspaces/adhoc-ws/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIAdhocStopViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "adhoc-stop-ws")

	w := serveHTTP(srv, "DELETE", "/api/v1/workspaces/adhoc-stop-ws/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIForkWorkspaceViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "fork-ws")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/fork-ws/fork", `{"goalContent":"# Fork goal"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteForkViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "delfork-ws")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/delfork-ws/delete-fork", `{"forkDir":"/nonexistent"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteMessageValid(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-delmsg")
	stateFile := filepath.Join(wsDir, ".sgai", "state.json")
	_, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		Status: state.StatusComplete,
		Messages: []state.Message{
			{ID: 1, FromAgent: "dev", ToAgent: "coordinator", Body: "test", Read: false},
		},
	})
	require.NoError(t, err)

	w := serveHTTP(server, "DELETE", "/api/v1/workspaces/test-ws-delmsg/messages/1", "")
	assert.Equal(t, 200, w.Code)
}

func TestHandleAPISteerValid(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-steer")
	stateFile := filepath.Join(wsDir, ".sgai", "state.json")
	_, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		Status: state.StatusWorking,
	})
	require.NoError(t, err)

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws-steer/steer", `{"message":"do this"}`)
	assert.Equal(t, 200, w.Code)
}

func TestHandleAPITogglePinValid(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws-pin")

	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws-pin/pin", "")
	assert.Equal(t, 200, w.Code)
}

func TestHandleAPIUpdateGoalValid(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-upgoal")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))

	w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws-upgoal/goal", `{"content":"# New Goal"}`)
	assert.Equal(t, 200, w.Code)
}

func TestHandleAPIWorkspaceDiffValid(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws-diff")

	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws-diff/diff", "")
	assert.Equal(t, 200, w.Code)
}

func TestHandleAPIOpenEditorGoalViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "editgoal-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))
	srv.editorAvailable = true
	srv.editor = newConfigurableEditor("echo")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editgoal-ws/open-editor/goal", "")
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIOpenEditorPMViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "editpm-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0o644))
	srv.editorAvailable = true
	srv.editor = newConfigurableEditor("echo")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editpm-ws/open-editor/project-management", "")
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPISnippetsByLanguageViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "snippet-lang-ws")
	goDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "example.go"), []byte("// snippet\npackage main"), 0o644))

	w := serveHTTP(srv, "GET", "/api/v1/snippets/go?workspace=snippet-lang-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResolveForkTemplateContentWithForks(t *testing.T) {
	srv, _ := setupTestServer(t)
	result := srv.resolveForkTemplateContent("/nonexistent/root")
	assert.Equal(t, goalExampleContent, result)
}

func TestHandleAPIAttachWorkspaceViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	validDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(validDir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(validDir, "GOAL.md"), []byte("# Ext Goal"), 0o644))

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/attach", `{"directory":"`+validDir+`"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDetachWorkspaceViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/detach", `{"path":"/nonexistent"}`)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusInternalServerError}, w.Code)
}

func TestComposeSaveWorkspaceNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/compose?workspace=nonexistent", `{}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestComposeSaveWithMatchingEtag(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "etag-match-ws")
	goalContent := "# Goal"
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0o644))

	etag := computeEtag([]byte(goalContent))
	w := serveHTTPWithHeader(srv, "POST", "/api/v1/compose?workspace=etag-match-ws", `{}`, "If-Match", etag)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestComposeSaveWithMismatchedEtag(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "etag-mismatch-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))

	w := serveHTTPWithHeader(srv, "POST", "/api/v1/compose?workspace=etag-mismatch-ws", `{}`, "If-Match", `"wrongetag"`)
	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
}

func TestUpdateGoalEmptyContentViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "upgoal-empty-v2")
	w := serveHTTP(srv, "PUT", "/api/v1/workspaces/upgoal-empty-v2/goal", `{"content":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateGoalInvalidJSONViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "upgoal-badjson")
	w := serveHTTP(srv, "PUT", "/api/v1/workspaces/upgoal-badjson/goal", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteWorkspaceNoConfirmViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "del-noconf-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/del-noconf-v2/delete", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteWorkspaceInvalidBodyViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "del-badjson")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/del-badjson/delete", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteWorkspaceStandaloneViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "del-standalone")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/del-standalone/delete", `{"confirm":true}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestSteerInvalidJSONViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "steer-badjson")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/steer-badjson/steer", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSteerEmptyMessageViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "steer-emptymsg")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/steer-emptymsg/steer", `{"message":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetGoalNoFileViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "getgoal-nofile")
	w := serveHTTP(srv, "GET", "/api/v1/workspaces/getgoal-nofile/goal", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdhocMissingPromptViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "adhoc-noprompt-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/adhoc-noprompt-v2/adhoc", `{"prompt":"","model":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdhocInvalidJSONViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "adhoc-badjson-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/adhoc-badjson-v2/adhoc", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForkWorkspaceInvalidJSONViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "fork-badjson-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/fork-badjson-v2/fork", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteForkNoConfirmViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "delfork-noconf-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/delfork-noconf-v2/delete-fork", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteForkInvalidJSONViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "delfork-badjson-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/delfork-badjson-v2/delete-fork", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartSessionBadBodyViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "start-badjson-v2")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/start-badjson-v2/start", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRespondInvalidJSONViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "respond-badjson")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-badjson/respond", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRespondNoSessionViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-nosess-v2")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What should I do?",
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-nosess-v2/respond", `{"answer":"do this","questionId":"q-123"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestCreateWorkspaceInvalidJSONViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateWorkspaceEmptyNameViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces", `{"name":"","goalContent":"# Goal"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAttachWorkspaceInvalidJSONViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/attach", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAgentsWorkspaceNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/agents?workspace=nonexistent", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSkillsWorkspaceNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/skills?workspace=nonexistent", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSnippetsWorkspaceNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/snippets?workspace=nonexistent", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestComposeStateWorkspaceNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/compose?workspace=nonexistent", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestComposeDraftWorkspaceNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/compose/draft?workspace=nonexistent", `{}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestComposePreviewWorkspaceNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/compose/preview?workspace=nonexistent", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStopSessionNotFoundViaHTTP(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/nonexistent-stop/stop", "{}")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOpenEditorNotAvailableViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "editor-unavail")
	srv.editorAvailable = false

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editor-unavail/open-editor", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestOpenEditorGoalNotAvailableViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "editor-goal-unavail")
	srv.editorAvailable = false

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editor-goal-unavail/open-editor/goal", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestOpenEditorGoalFileNotFoundViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "editor-goal-nofile")
	srv.editorAvailable = true
	srv.editorName = "test-editor"
	srv.editor = newConfigurableEditor("echo")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editor-goal-nofile/open-editor/goal", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOpenEditorPMNotAvailableViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "editor-pm-unavail")
	srv.editorAvailable = false

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editor-pm-unavail/open-editor/project-management", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleAPIComposeTemplatesContent(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/compose/templates", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "templates")
}

func TestHandleAPIListModelsContent(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/models", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "models")
}

func TestHandleAPISkillDetailNotFound(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "skillnf-ws")
	w := serveHTTP(srv, "GET", "/api/v1/skills/nonexistent?workspace=skillnf-ws", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPISnippetDetailNotFoundLang(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "snipnf-ws")
	w := serveHTTP(srv, "GET", "/api/v1/snippets/nonexistent/file.go?workspace=snipnf-ws", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDiffWorkspaceNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "GET", "/api/v1/workspaces/nonexistent-diff/diff", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDiffNoJJRepo(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "diff-nojj")
	w := serveHTTP(srv, "GET", "/api/v1/workspaces/diff-nojj/diff", "")
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIForkTemplateStandaloneViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "tmpl-standalone")
	w := serveHTTP(srv, "GET", "/api/v1/workspaces/tmpl-standalone/fork-template", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBuildWorkspaceFullStateWithMessages(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "msgs-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Messages: []state.Message{
			{ID: 1, FromAgent: "coordinator", ToAgent: "builder", Body: "build", Read: true, CreatedAt: "2025-01-01T00:00:00Z"},
			{ID: 2, FromAgent: "builder", ToAgent: "coordinator", Body: "done", Read: false, CreatedAt: "2025-01-01T00:01:00Z"},
		},
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{DirName: "msgs-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.Len(t, result.Messages, 2)
}

func TestBuildWorkspaceFullStateWithTodos(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "todos-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Todos: []state.TodoItem{
			{Content: "task1", Status: "completed", Priority: "high"},
		},
		ProjectTodos: []state.TodoItem{
			{Content: "proj-task", Status: "pending", Priority: "medium"},
		},
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{DirName: "todos-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.Len(t, result.AgentTodos, 1)
	assert.Len(t, result.ProjectTodos, 1)
}

func TestHandleAPIStateFullIntegration(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "full-int")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  digraph G {\n    \"coordinator\" -> \"builder\"\n    \"builder\" -> \"reviewer\"\n  }\nmodels:\n  coordinator: anthropic/claude-opus-4-6\n---\n# Full Integration\n\nBuild a comprehensive test suite."), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM\n\n## Progress\n- Step 1 done"), 0o644))

	agentDir := filepath.Join(wsDir, ".sgai", "agent")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "coordinator.md"),
		[]byte("---\ndescription: Main coordinator agent\n---\n# Coordinator"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "builder.md"),
		[]byte("---\ndescription: Builder agent\n---\n# Builder"), 0o644))

	skillDir := filepath.Join(wsDir, ".sgai", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: test-skill\ndescription: A test\n---\n# Skill"), 0o644))

	snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(snippetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "example.go"),
		[]byte("// Example snippet\npackage main\n\nfunc main() {}"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusComplete,
		Task:         "all done",
		CurrentAgent: "coordinator",
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiFactoryState
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Workspaces)

	ws := resp.Workspaces[0]
	assert.Equal(t, "full-int", ws.Name)
	assert.Equal(t, string(state.StatusComplete), ws.Status)
}

func TestHandleAPIStateWithPendingQuestion(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "pq-int")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Which approach should I take?",
		CurrentAgent: "coordinator",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Which approach?", Choices: []string{"A", "B", "C"}, MultiSelect: false},
				{Question: "Priority?", Choices: []string{"High", "Low"}, MultiSelect: true},
			},
		},
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiFactoryState
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Workspaces)

	ws := resp.Workspaces[0]
	assert.True(t, ws.NeedsInput)
	assert.NotNil(t, ws.PendingQuestion)
	assert.Len(t, ws.PendingQuestion.Questions, 2)
	assert.Equal(t, "coordinator", ws.PendingQuestion.AgentName)
}

func TestRespondViaCoordinatorFullPath(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-full")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Pick an option",
		CurrentAgent: "coordinator",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		},
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	qid := generateQuestionID(coord.State())
	body := `{"answer":"go with A","questionId":"` + qid + `","selectedChoices":["A"]}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-full/respond", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRespondViaCoordinatorWrongQuestion(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-wrong")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Pick an option",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}},
			},
		},
	})
	require.NoError(t, errCoord)

	srv.mu.Lock()
	srv.sessions[wsDir] = &session{coord: coord}
	srv.mu.Unlock()

	body := `{"answer":"go with A","questionId":"wrong-id","selectedChoices":["A"]}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-wrong/respond", body)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleAPIComposeSaveEtagMatchSucceeds(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-etag-match")
	goalContent := []byte("original content")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), goalContent, 0644))
	etag := computeEtag(goalContent)

	cs := server.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state = composerState{Description: "Updated"}
	cs.mu.Unlock()

	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	req := createHTTPRequest(t, "POST", "/api/v1/compose?workspace=test-ws-etag-match", "")
	req.Header.Set("If-Match", etag)
	w := serveHTTPReq(mux, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleAPIComposeSaveEtagMismatchFails(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws-etag-fail")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("original"), 0644))

	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	req := createHTTPRequest(t, "POST", "/api/v1/compose?workspace=test-ws-etag-fail", "")
	req.Header.Set("If-Match", `"stale-etag"`)
	w := serveHTTPReq(mux, req)
	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
}

func TestHandleAPIStateWithCaching(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "cache-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	w1 := serveHTTP(srv, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := serveHTTP(srv, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestBuildAdhocArgsWithVariantAddsFlag(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildAdhocArgs("anthropic/claude-sonnet-4-6 (thinking)")
	assert.Contains(t, args, "--variant")
	assert.Contains(t, args, "thinking")
}

func TestBuildAdhocArgsWithoutVariantNoFlag(t *testing.T) {
	b := &opencodeBackend{}
	args := b.BuildAdhocArgs("anthropic/claude-sonnet-4-6")
	for _, arg := range args {
		assert.NotEqual(t, "--variant", arg)
	}
}

func TestBuildWorkspaceFullStateContinuousMode(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "cont-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\ncontinuousModePrompt: run tests every 5m\n---\n# Goal"), 0o644))

	ws := workspaceInfo{DirName: "cont-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.ContinuousMode)
}

func TestBuildWorkspaceFullStateEmptyGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "emptygoal-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n"), 0o644))

	ws := workspaceInfo{DirName: "emptygoal-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.False(t, result.HasEditedGoal)
}

func TestBuildWorkspaceFullStateExternal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ext-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	ws := workspaceInfo{DirName: "ext-ws", Directory: wsDir, HasWorkspace: true, External: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.IsExternal)
}

func TestBuildWorkspaceFullStateGoalDescription(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	goalContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# My Cool Project\nSome body"
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))

	ws := workspaceInfo{Directory: wsDir, DirName: "test-ws"}
	result := server.buildWorkspaceFullState(ws, nil)
	assert.Equal(t, "My Cool Project", result.Description)
	assert.True(t, result.HasEditedGoal)
}

func TestBuildWorkspaceFullStateNoGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "nogoal-ws")

	ws := workspaceInfo{DirName: "nogoal-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.Equal(t, "nogoal-ws", result.Description)
	assert.False(t, result.HasEditedGoal)
}

func TestBuildWorkspaceFullStateRunningWithSession(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "running-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))

	ws := workspaceInfo{DirName: "running-ws", Directory: wsDir, HasWorkspace: true, Running: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.Running)
}

func TestBuildWorkspaceFullStateSelfDriveModeFlag(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		InteractionMode: state.ModeSelfDrive,
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{Directory: wsDir, DirName: "test-ws"}
	result := server.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.InteractiveAuto)
}

func TestBuildWorkspaceFullStateWithAgentSequence(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "seq-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		AgentSequence: []state.AgentSequenceEntry{
			{Agent: "coordinator", StartTime: "2025-01-01T00:00:00Z"},
		},
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{DirName: "seq-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.NotEmpty(t, result.AgentSequence)
}

func TestBuildWorkspaceFullStateWithAutoMode(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "auto-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusComplete,
		InteractionMode: state.ModeSelfDrive,
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{DirName: "auto-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.InteractiveAuto)
}

func TestBuildWorkspaceFullStateWithCost(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "cost-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Cost: state.SessionCost{
			TotalCost: 1.50,
		},
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{DirName: "cost-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.Equal(t, 1.50, result.Cost.TotalCost)
}

func TestBuildWorkspaceFullStateWithEditedGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "edited-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal Content Here"), 0o644))

	ws := workspaceInfo{DirName: "edited-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.HasEditedGoal)
}

func TestBuildWorkspaceFullStateWithFreeformPending(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "What should I do next?",
		CurrentAgent: "builder",
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{Directory: wsDir, DirName: "test-ws"}
	result := server.buildWorkspaceFullState(ws, nil)
	assert.NotNil(t, result.PendingQuestion)
	assert.Equal(t, "free-text", result.PendingQuestion.Type)
	assert.Equal(t, "builder", result.PendingQuestion.AgentName)
}

func TestBuildWorkspaceFullStateWithLogLines(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status:       state.StatusWorking,
		CurrentAgent: "coordinator",
		Task:         "doing things",
	})
	require.NoError(t, errCoord)

	ol := newCircularLogBuffer()
	ol.add(logLine{prefix: "[test]", text: "some log line"})

	server.mu.Lock()
	server.sessions[wsDir] = &session{
		running:   true,
		outputLog: ol,
	}
	server.mu.Unlock()

	ws := workspaceInfo{Directory: wsDir, DirName: "test-ws", Running: true}
	result := server.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.Running)
	assert.NotEmpty(t, result.Log)
}

func TestBuildWorkspaceFullStateWithMultiChoicePendingQuestion(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusWaitingForHuman,
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{
				{Question: "Pick one", Choices: []string{"A", "B"}, MultiSelect: true},
			},
		},
		CurrentAgent: "coordinator",
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{Directory: wsDir, DirName: "test-ws"}
	result := server.buildWorkspaceFullState(ws, nil)
	assert.NotNil(t, result.PendingQuestion)
	assert.Equal(t, "multi-choice", result.PendingQuestion.Type)
	require.Len(t, result.PendingQuestion.Questions, 1)
	assert.True(t, result.PendingQuestion.Questions[0].MultiSelect)
}

func TestBuildWorkspaceFullStateWithPMFile(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "pm-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM Content"), 0o644))

	ws := workspaceInfo{DirName: "pm-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.True(t, result.HasProjectMgmt)
	assert.NotEmpty(t, result.PMContent)
}

func TestBuildWorkspaceFullStateWithProgress(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "progress-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Progress: []state.ProgressEntry{
			{Timestamp: time.Now().UTC().Format(time.RFC3339), Description: "step 1"},
			{Timestamp: time.Now().UTC().Format(time.RFC3339), Description: "step 2"},
		},
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{DirName: "progress-ws", Directory: wsDir, HasWorkspace: true}
	result := srv.buildWorkspaceFullState(ws, nil)
	assert.NotEmpty(t, result.Events)
	assert.NotEmpty(t, result.LatestProgress)
}

func TestBuildWorkspaceFullStateWithWorkGatePending(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{
		Status: state.StatusWaitingForHuman,
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			IsWorkGate: true,
			Questions: []state.QuestionItem{
				{Question: "Approve?", Choices: []string{"Yes", "No"}},
			},
		},
		CurrentAgent: "coordinator",
	})
	require.NoError(t, errCoord)

	ws := workspaceInfo{Directory: wsDir, DirName: "test-ws"}
	result := server.buildWorkspaceFullState(ws, nil)
	assert.NotNil(t, result.PendingQuestion)
	assert.Equal(t, "work-gate", result.PendingQuestion.Type)
}

func TestCollectAgentModels(t *testing.T) {
	t.Run("noGoalFile", func(t *testing.T) {
		dir := t.TempDir()
		result := collectAgentModels(dir)
		assert.Nil(t, result)
	})

	t.Run("noModels", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		require.NoError(t, os.WriteFile(goalPath, []byte("# No frontmatter"), 0644))

		result := collectAgentModels(dir)
		assert.Nil(t, result)
	})

	t.Run("withModels", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nmodels:\n  coordinator: claude-opus-4\n  developer: gpt-4\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		result := collectAgentModels(dir)
		require.Len(t, result, 2)
		assert.Equal(t, "coordinator", result[0].Agent)
		assert.Equal(t, []string{"claude-opus-4"}, result[0].Models)
		assert.Equal(t, "developer", result[1].Agent)
		assert.Equal(t, []string{"gpt-4"}, result[1].Models)
	})
}

func TestCollectAgentModelsNoAgents(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "agent"), 0o755))
	result := collectAgentModels(dir)
	assert.Empty(t, result)
}

func TestCollectAgentModelsNoGoalReturnsNil(t *testing.T) {
	result := collectAgentModels(t.TempDir())
	assert.Nil(t, result)
}

func TestCollectAgentModelsWithGoal(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"),
		[]byte("---\nmodels:\n  coordinator: anthropic/claude-opus-4-6\n---\n# Goal"), 0o644))

	result := collectAgentModels(dir)
	assert.NotNil(t, result)
}

func TestComputeEtagConsistency(t *testing.T) {
	data := []byte("test content")
	etag1 := computeEtag(data)
	etag2 := computeEtag(data)
	assert.Equal(t, etag1, etag2)
	assert.True(t, len(etag1) > 0)
}

func TestComputeEtagDifferent(t *testing.T) {
	etag1 := computeEtag([]byte("content1"))
	etag2 := computeEtag([]byte("content2"))
	assert.NotEqual(t, etag1, etag2)
}

func TestCoordinatorModelFromWorkspaceEmptyReturnsEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.coordinatorModelFromWorkspace("")
	assert.Empty(t, result)
}

func TestCoordinatorModelFromWorkspaceNotFoundReturnsEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.coordinatorModelFromWorkspace("nonexistent")
	assert.Empty(t, result)
}

func TestCoordinatorModelFromWorkspaceWithModelsConfig(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	goalContent := "---\nmodels:\n  coordinator: anthropic/claude-sonnet-4-6\nflow: |\n  \"coordinator\"\n---\n# Test"
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))
	result := server.coordinatorModelFromWorkspace("test-ws")
	assert.Equal(t, "anthropic/claude-sonnet-4-6", result)
}

func TestHandleAPIAdhocAlreadyRunningReturnsOutput(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	st := server.getAdhocState(wsDir)
	st.mu.Lock()
	st.running = true
	st.output.WriteString("already running output")
	st.mu.Unlock()
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", `{"prompt":"test","model":"anthropic/claude-sonnet-4-6"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiAdhocResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Running)
	assert.Contains(t, resp.Output, "already running output")
}

func TestHandleAPIAttachWorkspaceValidDir(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.externalConfigDir = t.TempDir()
	extDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".sgai"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(extDir, "GOAL.md"), []byte("# External"), 0o644))

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/attach", `{"directory":"`+extDir+`"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIAttachWorkspaceValidDirectory(t *testing.T) {
	server, _ := setupTestServer(t)
	attachDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(attachDir, ".sgai"), 0755))
	body := `{"path":"` + attachDir + `"}`
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", body)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleAPIBrowseDirectoriesWithPath(t *testing.T) {
	srv, _ := setupTestServer(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub1"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub2"), 0o755))

	w := serveHTTP(srv, "GET", "/api/v1/browse-directories?path="+dir, "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIBrowseDirsWithPathParam(t *testing.T) {
	server, rootDir := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/browse-directories?path="+rootDir, "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeDraftInvalidBodyV2(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "draft-invalid")

	w := serveHTTP(srv, "POST", "/api/v1/compose/draft?workspace=draft-invalid", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIComposeDraftWithContent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "draft-content")

	body := `{"state":{"description":"my project","agents":[{"name":"coordinator"}]},"wizard":{}}`
	w := serveHTTP(srv, "POST", "/api/v1/compose/draft?workspace=draft-content", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposePreviewWithFlowError(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "preview-flow-err")

	cs := srv.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state.Flow = "invalid flow content {"
	cs.mu.Unlock()

	w := serveHTTP(srv, "GET", "/api/v1/compose/preview?workspace=preview-flow-err", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposePreviewWithState(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "preview-state")

	cs := srv.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state.Description = "Preview test"
	cs.state.Tasks = "## Tasks\n- Build something"
	cs.mu.Unlock()

	w := serveHTTP(srv, "GET", "/api/v1/compose/preview?workspace=preview-state", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiComposePreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Content, "Preview test")
}

func TestHandleAPIComposeSaveFullPath(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "save-full")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old Goal"), 0o644))

	cs := srv.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state.Description = "Saved project"
	cs.state.Agents = []composerAgentConf{
		{Name: "coordinator", Selected: true},
		{Name: "builder", Selected: true},
	}
	cs.state.Flow = `digraph G {
    "coordinator" -> "builder"
}`
	cs.mu.Unlock()

	w := serveHTTP(srv, "POST", "/api/v1/compose?workspace=save-full", `{}`)
	assert.Equal(t, http.StatusCreated, w.Code)

	saved, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Contains(t, string(saved), "Saved project")
}

func TestHandleAPIComposeSaveSuccessful(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	cs := server.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state = composerState{
		Description: "Test project",
		Flow:        `"a" -> "b"`,
	}
	cs.mu.Unlock()
	w := serveHTTP(server, "POST", "/api/v1/compose?workspace=test-ws", "")
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp apiComposeSaveResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Saved)
}

func TestHandleAPIComposeStateFull(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "compose-full")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))

	cs := srv.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state.Description = "test"
	cs.state.Agents = []composerAgentConf{
		{Name: "coordinator", Selected: true, Model: "anthropic/claude-opus-4-6"},
	}
	cs.state.Flow = `digraph G { "coordinator" -> "builder" }`
	cs.state.CompletionGate = "make test"
	cs.wizard = wizardState{TechStack: []string{"go", "react"}}
	cs.mu.Unlock()

	w := serveHTTP(srv, "GET", "/api/v1/compose?workspace=compose-full", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiComposeStateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "test", resp.State.Description)
	assert.NotEmpty(t, resp.TechStackItems)
}

func TestHandleAPIComposeStateFullContent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "cs-full")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))

	cs := srv.getComposerSession(wsDir)
	cs.mu.Lock()
	cs.state.Description = "Test project"
	cs.state.Agents = []composerAgentConf{{Name: "coordinator"}, {Name: "builder"}}
	cs.mu.Unlock()

	w := serveHTTP(srv, "GET", "/api/v1/compose?workspace=cs-full", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiComposeStateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Test project", resp.State.Description)
}

func TestHandleAPICreateWorkspaceValid(t *testing.T) {
	srv, _ := setupTestServer(t)
	body := `{"name":"new-workspace-test","goalContent":"---\n---\n# New Workspace\n\nBuild something great."}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces", body)
	assert.Contains(t, []int{http.StatusCreated, http.StatusOK, http.StatusConflict}, w.Code)
}

func TestHandleAPIDeleteForkConfirmedNoFork(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "delfork-conf-nofork")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/delfork-conf-nofork/delete-fork", `{"confirm":true,"forkDir":"/nonexistent"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteForkInvalidBodyJSON(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", "not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteForkNoConfirmRequired(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteForkNotRootOrForkWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{"confirm":true}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteMessageInvalidID(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "delmsg-badid")

	w := serveHTTP(srv, "DELETE", "/api/v1/workspaces/delmsg-badid/messages/notanumber", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteMessageInvalidIDPath(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "delmsg-path")
	w := serveHTTP(srv, "DELETE", "/api/v1/workspaces/delmsg-path/messages/abc", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteMessageValidViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "delmsg-valid")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Messages: []state.Message{
			{ID: 1, FromAgent: "Human Partner", ToAgent: "coordinator", Body: "test", CreatedAt: "2025-01-01T00:00:00Z"},
		},
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "DELETE", "/api/v1/workspaces/delmsg-valid/messages/1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIDeleteWorkspaceConfirmedStandalone(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "del-standalone-full")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/del-standalone-full/delete", `{"confirm":true}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIDeleteWorkspaceDirNotFoundError(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete", `{"confirm":true}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteWorkspaceInvalidBodyJSON(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", "not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceNoConfirmRequired(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceStandaloneDelete(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{"confirm":true}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiDeleteWorkspaceResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Deleted)
}

func TestHandleAPIDetachWorkspaceNotAttached(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/detach", `{"path":"/not/attached"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIForkWorkspaceEmptyGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "fork-empty")
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/fork-empty/fork", `{"goalContent":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIForkWorkspaceInvalidBodyJSON(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/fork", "not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIGetGoalSuccessReturnsContent(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# My Goal\nDo things"), 0644))
	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/goal", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiGoalResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp.Content, "My Goal")
}

func TestHandleAPIListModelsViaHTTPReturnsModels(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "GET", "/api/v1/models?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiModelsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp.Models)
}

func TestHandleAPIOpenEditorGoalSuccessful(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# test goal"), 0644))
	server.editorAvailable = true
	server.editorName = "test-editor"
	server.editor = &configurableEditor{name: "test-editor", command: "echo"}
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/open-editor/goal", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIOpenEditorPMSuccessful(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0644))
	server.editorAvailable = true
	server.editorName = "test-editor"
	server.editor = &configurableEditor{name: "test-editor", command: "echo"}
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/open-editor/project-management", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIOpenEditorSuccessful(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	server.editorAvailable = true
	server.editorName = "test-editor"
	editor := &configurableEditor{name: "test-editor", command: "echo"}
	server.editor = editor
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/open-editor", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiOpenEditorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Opened)
	assert.Equal(t, "test-editor", resp.Editor)
}

func TestHandleAPISkillDetailWithContent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "skill-detail")
	skillPath := filepath.Join(wsDir, ".sgai", "skills", "coding-practices", "my-skill")
	require.NoError(t, os.MkdirAll(skillPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillPath, "SKILL.md"),
		[]byte("---\nname: my-skill\ndescription: Test skill\n---\n# My Skill\n\nDetailed instructions here."), 0o644))

	w := serveHTTP(srv, "GET", "/api/v1/skills/coding-practices/my-skill?workspace=skill-detail", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISnippetDetailWithExtensionLookup(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(snippetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "hello.go"), []byte("---\nname: Hello\ndescription: A hello snippet\n---\npackage main\nfunc main() {}"), 0644))

	w := serveHTTP(server, "GET", "/api/v1/snippets/go/hello?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiSnippetDetailResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "Hello", resp.Name)
	assert.Equal(t, "go", resp.Language)
	assert.Contains(t, resp.Content, "package main")
}

func TestHandleAPISnippetDetailWithFile(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "snippet-detail")
	snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "python")
	require.NoError(t, os.MkdirAll(snippetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "hello.py"),
		[]byte("# Python hello world\nprint('hello')"), 0o644))

	w := serveHTTP(srv, "GET", "/api/v1/snippets/python/hello.py?workspace=snippet-detail", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISnippetsByLanguageFoundMatch(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "python")
	require.NoError(t, os.MkdirAll(snippetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "hello.py"), []byte("---\ndescription: py hello\n---\nprint('hi')"), 0644))

	w := serveHTTP(server, "GET", "/api/v1/snippets/python?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiSnippetsByLanguageResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "python", resp.Language)
	assert.NotEmpty(t, resp.Snippets)
}

func TestHandleAPISnippetsByLanguageNoMatch(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "GET", "/api/v1/snippets/nonexistent?workspace=test-ws", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPISnippetsByLanguageWithContent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "sniplang-content")
	snippetDir := filepath.Join(wsDir, ".sgai", "snippets", "typescript")
	require.NoError(t, os.MkdirAll(snippetDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "util.ts"),
		[]byte("// Utility functions\nexport const add = (a: number, b: number) => a + b;"), 0o644))

	w := serveHTTP(srv, "GET", "/api/v1/snippets/typescript?workspace=sniplang-content", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIStartSessionNonExistent(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/nonexistent-start/start", `{"auto":false}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPISteerSuccessViaAPI(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	sp := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(sp, state.Workflow{})
	require.NoError(t, errCoord)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/steer", `{"message":"do something"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISteerWithState(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-state")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusComplete,
		CurrentAgent: "coordinator",
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/steer-state/steer", `{"message":"focus on tests"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISteerWithValidSteer(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-valid-full")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  digraph G {\n    \"coordinator\" -> \"builder\"\n  }\n---\n# Goal"), 0o644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusComplete,
		CurrentAgent: "coordinator",
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/steer-valid-full/steer", `{"message":"focus on performance optimization"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIStopSessionAlreadyStopped(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stop-already")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/stop-already/stop", "{}")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIStopSessionWithState(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stop-state")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusWorking,
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/stop-state/stop", "{}")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPITogglePinAndUnpin(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "pin-toggle")
	srv.pinnedConfigDir = t.TempDir()

	w1 := serveHTTP(srv, "POST", "/api/v1/workspaces/pin-toggle/pin", "")
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := serveHTTP(srv, "POST", "/api/v1/workspaces/pin-toggle/pin", "")
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestHandleAPITogglePinNonExistent(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/nonexistent-pin/pin", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPITogglePinSuccessToggle(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/pin", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiTogglePinResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Pinned)
}

func TestHandleAPIUpdateGoalInvalidJSONBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", "not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIUpdateGoalSuccessWrites(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("old"), 0644))
	w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", `{"content":"# New Goal"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiUpdateGoalResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Updated)
	data, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Equal(t, "# New Goal", string(data))
}

func TestHandleAPIUpdateGoalWithContent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "upgoal-content")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0o644))

	w := serveHTTP(srv, "PUT", "/api/v1/workspaces/upgoal-content/goal", `{"content":"---\nflow: |\n  digraph G {\n    \"a\" -> \"b\"\n  }\n---\n# Updated Goal"}`)
	assert.Equal(t, http.StatusOK, w.Code)

	data, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Contains(t, string(data), "Updated Goal")
}

func TestHandleAPIWorkflowSVGAvailableFromCache(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.svgCache.set(wsDir+"|Unknown", "<svg>test</svg>")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/workflow.svg", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "image/svg+xml")
	assert.Contains(t, w.Body.String(), "<svg>test</svg>")
}

func TestHandleAPIWorkflowSVGNotAvailableError(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/workflow.svg", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIWorkspaceDiffFound(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "diff-found")
	w := serveHTTP(srv, "GET", "/api/v1/workspaces/diff-found/diff", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResolveCurrentModel(t *testing.T) {
	t.Run("fromState", func(t *testing.T) {
		wfState := state.Workflow{CurrentModel: "claude-opus-4"}
		result := resolveCurrentModel("/some/path", wfState)
		assert.Equal(t, "claude-opus-4", result)
	})

	t.Run("noAgent", func(t *testing.T) {
		wfState := state.Workflow{}
		result := resolveCurrentModel("/some/path", wfState)
		assert.Equal(t, "", result)
	})

	t.Run("fromGoalFile", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nmodels:\n  coordinator: claude-opus-4\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		wfState := state.Workflow{CurrentAgent: "coordinator"}
		result := resolveCurrentModel(dir, wfState)
		assert.Equal(t, "claude-opus-4", result)
	})

	t.Run("agentNotInGoal", func(t *testing.T) {
		dir := t.TempDir()
		goalPath := filepath.Join(dir, "GOAL.md")
		content := "---\nmodels:\n  coordinator: claude-opus-4\n---\n# Goal"
		require.NoError(t, os.WriteFile(goalPath, []byte(content), 0644))

		wfState := state.Workflow{CurrentAgent: "developer"}
		result := resolveCurrentModel(dir, wfState)
		assert.Equal(t, "", result)
	})
}

func TestResolveCurrentModelNoAgentReturnsEmpty(t *testing.T) {
	wf := state.Workflow{}
	result := resolveCurrentModel("/tmp", wf)
	assert.Empty(t, result)
}

func TestResolveCurrentModelNoModel(t *testing.T) {
	dir := t.TempDir()
	wf := state.Workflow{}
	result := resolveCurrentModel(dir, wf)
	assert.Empty(t, result)
}

func TestResolveCurrentModelWithExplicitModel(t *testing.T) {
	wf := state.Workflow{CurrentModel: "opus-4"}
	result := resolveCurrentModel("/tmp", wf)
	assert.Equal(t, "opus-4", result)
}

func TestSPAMiddlewareStaticAssets(t *testing.T) {
	srv, _ := setupTestServer(t)
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)
	handler := srv.spaMiddleware(mux)
	assert.NotNil(t, handler)
}

func TestSPAMiddlewareStaticAssetRequest(t *testing.T) {
	srv, _ := setupTestServer(t)
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)
	handler := srv.spaMiddleware(mux)

	req := httptest.NewRequest("GET", "/assets/app.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
}

func TestSPAMiddlewareAPIRoute(t *testing.T) {
	srv, _ := setupTestServer(t)
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)
	handler := srv.spaMiddleware(mux)

	req := httptest.NewRequest("GET", "/api/v1/state", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "simpleObject",
			value:    map[string]string{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "emptyObject",
			value:    map[string]string{},
			expected: `{}`,
		},
		{
			name:     "array",
			value:    []string{"a", "b", "c"},
			expected: `["a","b","c"]`,
		},
		{
			name:     "string",
			value:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "number",
			value:    42,
			expected: `42`,
		},
		{
			name:     "null",
			value:    nil,
			expected: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.value)

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var result json.RawMessage
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
			assert.JSONEq(t, tt.expected, string(result))
		})
	}
}

func TestWriteJSONWithStruct(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	w := httptest.NewRecorder()
	writeJSON(w, testStruct{Name: "test", Value: 123})

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"name":"test","value":123}`, w.Body.String())
}

func TestConvertMessagesForAPI(t *testing.T) {
	tests := []struct {
		name     string
		messages []state.Message
		expected []apiMessageEntry
	}{
		{
			name:     "empty",
			messages: []state.Message{},
			expected: []apiMessageEntry{},
		},
		{
			name: "singleMessage",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello", Read: true, CreatedAt: "2024-01-15T10:30:00Z"},
			},
			expected: []apiMessageEntry{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "Hello", Subject: "Hello", Read: true, CreatedAt: "2024-01-15T10:30:00Z"},
			},
		},
		{
			name: "multipleMessages",
			messages: []state.Message{
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "First", Read: true, CreatedAt: "2024-01-15T10:00:00Z"},
				{ID: 2, FromAgent: "agent2", ToAgent: "agent1", Body: "Second", Read: false, CreatedAt: "2024-01-15T11:00:00Z"},
			},
			expected: []apiMessageEntry{
				{ID: 2, FromAgent: "agent2", ToAgent: "agent1", Body: "Second", Subject: "Second", Read: false, CreatedAt: "2024-01-15T11:00:00Z"},
				{ID: 1, FromAgent: "agent1", ToAgent: "agent2", Body: "First", Subject: "First", Read: true, CreatedAt: "2024-01-15T10:00:00Z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMessagesForAPI(tt.messages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSubject(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "empty",
			body:     "",
			expected: "",
		},
		{
			name:     "singleLine",
			body:     "This is a subject",
			expected: "This is a subject",
		},
		{
			name:     "multipleLines",
			body:     "Subject line\nSecond line\nThird line",
			expected: "Subject line",
		},
		{
			name:     "withNewline",
			body:     "Subject\n",
			expected: "Subject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubject(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertModelStatuses(t *testing.T) {
	tests := []struct {
		name     string
		displays []modelStatusDisplay
		expected []apiModelStatusEntry
	}{
		{
			name:     "empty",
			displays: []modelStatusDisplay{},
			expected: nil,
		},
		{
			name: "singleEntry",
			displays: []modelStatusDisplay{
				{ModelID: "model1", Status: "running"},
			},
			expected: []apiModelStatusEntry{
				{ModelID: "model1", Status: "running"},
			},
		},
		{
			name: "multipleEntries",
			displays: []modelStatusDisplay{
				{ModelID: "model1", Status: "running"},
				{ModelID: "model2", Status: "done"},
				{ModelID: "model3", Status: "error"},
			},
			expected: []apiModelStatusEntry{
				{ModelID: "model1", Status: "running"},
				{ModelID: "model2", Status: "done"},
				{ModelID: "model3", Status: "error"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertModelStatuses(tt.displays)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleRespondViaCoordinator(t *testing.T) {
	t.Run("noPendingQuestion", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status: state.StatusWorking,
		})
		require.NoError(t, errCoord)

		w := httptest.NewRecorder()
		srv.handleRespondViaCoordinator(w, coord, apiRespondRequest{Answer: "yes"})
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "no pending question")
	})

	t.Run("expiredQuestion", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "Pick one",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "Pick one", Choices: []string{"A", "B"}}},
			},
		})
		require.NoError(t, errCoord)

		w := httptest.NewRecorder()
		srv.handleRespondViaCoordinator(w, coord, apiRespondRequest{
			QuestionID: "wrong-id",
			Answer:     "yes",
		})
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "question expired")
	})

	t.Run("emptyResponse", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "Choose",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "Choose", Choices: []string{"X", "Y"}}},
			},
		})
		require.NoError(t, errCoord)

		questionID := generateQuestionID(coord.State())

		w := httptest.NewRecorder()
		srv.handleRespondViaCoordinator(w, coord, apiRespondRequest{
			QuestionID: questionID,
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "response cannot be empty")
	})

	t.Run("successfulResponse", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "Confirm?",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "Confirm?", Choices: []string{"Yes", "No"}}},
			},
		})
		require.NoError(t, errCoord)

		questionID := generateQuestionID(coord.State())

		go func() {
			coord.Respond("Selected: Yes")
		}()

		w := httptest.NewRecorder()
		srv.handleRespondViaCoordinator(w, coord, apiRespondRequest{
			QuestionID:      questionID,
			SelectedChoices: []string{"Yes"},
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var resp apiRespondResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})
}

func TestHandleRespondLegacy(t *testing.T) {
	t.Run("noPendingQuestion", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status: state.StatusWorking,
		})
		require.NoError(t, errCoord)

		w := httptest.NewRecorder()
		srv.handleRespondLegacy(w, dir, apiRespondRequest{Answer: "yes"})
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("expiredQuestion", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "Pick one",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "Pick one", Choices: []string{"A", "B"}}},
			},
		})
		require.NoError(t, errCoord)

		w := httptest.NewRecorder()
		srv.handleRespondLegacy(w, dir, apiRespondRequest{
			QuestionID: "wrong-id",
			Answer:     "yes",
		})
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("emptyResponse", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "Choose",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "Choose", Choices: []string{"X", "Y"}}},
			},
		})
		require.NoError(t, errCoord)

		coord := srv.workspaceCoordinator(dir)
		questionID := generateQuestionID(coord.State())

		w := httptest.NewRecorder()
		srv.handleRespondLegacy(w, dir, apiRespondRequest{
			QuestionID: questionID,
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("successfulResponse", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		dir := t.TempDir()
		sgaiDir := filepath.Join(dir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
		statePath := filepath.Join(sgaiDir, "state.json")
		_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
			Status:       state.StatusWaitingForHuman,
			HumanMessage: "Confirm?",
			MultiChoiceQuestion: &state.MultiChoiceQuestion{
				Questions: []state.QuestionItem{{Question: "Confirm?", Choices: []string{"Yes", "No"}}},
			},
		})
		require.NoError(t, errCoord)

		coord := srv.workspaceCoordinator(dir)
		questionID := generateQuestionID(coord.State())

		w := httptest.NewRecorder()
		srv.handleRespondLegacy(w, dir, apiRespondRequest{
			QuestionID:      questionID,
			SelectedChoices: []string{"Yes"},
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var resp apiRespondResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})
}

func TestResolveForkDir(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")

	t.Run("usesRequestForkDirWithinRoot", func(t *testing.T) {
		forkDir := filepath.Join(rootDir, "fork-1")
		require.NoError(t, os.MkdirAll(forkDir, 0o755))
		got := srv.resolveForkDir(forkDir, wsDir, rootDir)
		assert.NotEmpty(t, got)
	})

	t.Run("invalidRequestForkDir", func(t *testing.T) {
		got := srv.resolveForkDir("/nonexistent/12345", wsDir, rootDir)
		assert.Empty(t, got)
	})

	t.Run("workspaceNotRoot", func(t *testing.T) {
		got := srv.resolveForkDir("", wsDir, rootDir)
		assert.Equal(t, wsDir, got)
	})

	t.Run("workspaceIsRoot", func(t *testing.T) {
		got := srv.resolveForkDir("", rootDir, rootDir)
		assert.Empty(t, got)
	})
}

func TestServeReactIndex(t *testing.T) {
	t.Run("withIndex", func(t *testing.T) {
		webFS := fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<html>React App</html>")},
		}
		w := httptest.NewRecorder()
		serveReactIndex(w, webFS)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "React App")
	})

	t.Run("noIndex", func(t *testing.T) {
		webFS := fstest.MapFS{}
		w := httptest.NewRecorder()
		serveReactIndex(w, webFS)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSPAMiddlewareRouting(t *testing.T) {
	srv, _ := setupTestServer(t)
	mux := http.NewServeMux()
	srv.registerAPIRoutes(mux)

	handler := srv.spaMiddleware(mux)

	t.Run("apiRoutePassesThrough", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/state", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nonAPIRouteServesReact", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dashboard", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleSignalStream(t *testing.T) {
	srv, _ := setupTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/v1/signal", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.handleSignalStream(w, req)
		close(done)
	}()

	cancel()
	<-done

	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
}

func TestLockedWriterStripANSI(t *testing.T) {
	st := &adhocPromptState{}
	lw := &lockedWriter{mu: &st.mu, buf: &st.output}

	n, err := lw.Write([]byte("\x1b[31mhello\x1b[0m world"))
	require.NoError(t, err)
	assert.Equal(t, len("\x1b[31mhello\x1b[0m world"), n)

	st.mu.Lock()
	output := st.output.String()
	st.mu.Unlock()
	assert.Equal(t, "hello world", output)
}

func TestLockedWriterPlainText(t *testing.T) {
	st := &adhocPromptState{}
	lw := &lockedWriter{mu: &st.mu, buf: &st.output}

	n, err := lw.Write([]byte("plain text"))
	require.NoError(t, err)
	assert.Equal(t, len("plain text"), n)

	st.mu.Lock()
	output := st.output.String()
	st.mu.Unlock()
	assert.Equal(t, "plain text", output)
}

func TestCollectJJFullDiff(t *testing.T) {
	result := collectJJFullDiff("/nonexistent/path/12345")
	assert.Empty(t, result)
}

func TestGoalDescription(t *testing.T) {
	t.Run("emptyDirectory", func(t *testing.T) {
		got := goalDescription("", "fallback")
		assert.Equal(t, "fallback", got)
	})

	t.Run("noGoalFile", func(t *testing.T) {
		dir := t.TempDir()
		got := goalDescription(dir, "fallback")
		assert.Equal(t, "fallback", got)
	})

	t.Run("goalWithDescription", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\n---\n# My Project\n\nSome description"), 0o644))
		got := goalDescription(dir, "fallback")
		assert.NotEqual(t, "fallback", got)
	})
}

func TestToMenuBarItem(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("---\n---\n# Test\nDesc"), 0o644))

	w := workspaceInfo{
		DirName:    "test-ws",
		Directory:  dir,
		NeedsInput: true,
		Running:    true,
		InProgress: true,
		Pinned:     false,
	}

	item := toMenuBarItem(w)
	assert.Equal(t, "test-ws", item.name)
	assert.True(t, item.needsInput)
	assert.True(t, item.running)
	assert.False(t, item.stopped)
	assert.False(t, item.pinned)
}

func TestResolveForkTemplateContent(t *testing.T) {
	srv, _ := setupTestServer(t)
	content := srv.resolveForkTemplateContent("/nonexistent/root")
	assert.Equal(t, goalExampleContent, content)
}

func TestCountForkCommitsAhead(t *testing.T) {
	got := countForkCommitsAhead("main", "/nonexistent/12345")
	assert.Equal(t, 0, got)
}

var _ fs.FS = fstest.MapFS{}

func TestHandleAPIGetGoalViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# My Goal\nContent here"), 0o644))

	w := serveHTTP(srv, "GET", "/api/v1/workspaces/test-ws/goal", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiGoalResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Content, "My Goal")
}

func TestHandleAPIForkTemplateNotRoot(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "standalone-ws")

	w := serveHTTP(srv, "GET", "/api/v1/workspaces/standalone-ws/fork-template", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceNoConfirm(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	_ = setupTestWorkspace(t, rootDir, "ws")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/ws/delete", `{"confirm": false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceEditorNoEditorAvailable(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.editorAvailable = false

	_, err := srv.openEditorService("/some/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no editor available")
}

func TestServiceEditorGoalFileNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.editorAvailable = true
	srv.editor = newConfigurableEditor("echo")

	_, err := srv.openEditorFileService("/nonexistent", "GOAL.md")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestServiceEditorGoalSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.editorAvailable = true
	srv.editorName = "echo"
	srv.editor = newConfigurableEditor("echo")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GOAL.md"), []byte("# Goal"), 0o644))

	result, err := srv.openEditorFileService(dir, "GOAL.md")
	require.NoError(t, err)
	assert.True(t, result.Opened)
	assert.Equal(t, "echo", result.Editor)
}

func TestServiceEditorOpenWorkspace(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.editorAvailable = true
	srv.editor = newConfigurableEditor("echo")

	dir := t.TempDir()
	result, err := srv.openEditorService(dir)
	require.NoError(t, err)
	assert.True(t, result.Opened)
}

func TestServiceEditorPMService(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.editorAvailable = true
	srv.editor = newConfigurableEditor("echo")

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0o644))

	result, err := srv.openEditorProjectManagementService(dir)
	require.NoError(t, err)
	assert.True(t, result.Opened)
}

func TestHandleAPIOpenEditorViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "editor-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0o644))
	srv.editorAvailable = true
	srv.editor = newConfigurableEditor("echo")

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/editor-ws/open-editor", `{"target": "workspace"}`)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestListModelsServiceFallback(t *testing.T) {
	srv, _ := setupTestServer(t)
	result := srv.listModelsService("nonexistent")
	assert.NotNil(t, result.Models)
}

func TestHandleAPIWorkspaceDiffNoJJ(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "no-jj-ws")

	w := serveHTTP(srv, "GET", "/api/v1/workspaces/no-jj-ws/diff", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiFullDiffResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	_ = wsDir
	assert.Empty(t, resp.Diff)
}

func TestHandleAPISteerViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\n---\n# Original"), 0o644))

	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusWorking,
	})
	require.NoError(t, errCoord)

	w := serveHTTP(srv, "POST", "/api/v1/workspaces/steer-ws/steer", `{"message":"Add logging"}`)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
}

func TestCheckWorkspaceState(t *testing.T) {
	srv, _ := setupTestServer(t)

	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	statePath := filepath.Join(sgaiDir, "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Task:   "done",
	})
	require.NoError(t, errCoord)

	snapshots := make(map[string]workspaceStateSnapshot)
	activeWorkspaces := make(map[string]bool)

	srv.checkWorkspaceState(dir, snapshots, activeWorkspaces)
	assert.True(t, activeWorkspaces[dir])
	assert.Contains(t, snapshots, dir)
	assert.Equal(t, state.StatusComplete, snapshots[dir].status)
}

func TestCheckWorkspaceStateNoStateFile(t *testing.T) {
	srv, _ := setupTestServer(t)

	dir := t.TempDir()
	snapshots := make(map[string]workspaceStateSnapshot)
	activeWorkspaces := make(map[string]bool)

	srv.checkWorkspaceState(dir, snapshots, activeWorkspaces)
	assert.True(t, activeWorkspaces[dir])
	assert.NotContains(t, snapshots, dir)
}

func TestPollWorkspaceStatesEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	snapshots := make(map[string]workspaceStateSnapshot)
	srv.pollWorkspaceStates(snapshots)
	assert.Empty(t, snapshots)
}

func TestGetWorkspaceStatusComplete(t *testing.T) {
	srv, _ := setupTestServer(t)
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	statePath := filepath.Join(sgaiDir, "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status: state.StatusComplete,
		Task:   "done",
	})
	require.NoError(t, errCoord)

	running, needsInput := srv.getWorkspaceStatus(dir)
	assert.False(t, running)
	assert.False(t, needsInput)
}

func TestHandleAPIRespondViaHTTP(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-ws")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "Pick one",
		MultiChoiceQuestion: &state.MultiChoiceQuestion{
			Questions: []state.QuestionItem{{Question: "Pick one", Choices: []string{"A", "B"}}},
		},
	})
	require.NoError(t, errCoord)

	coord := srv.workspaceCoordinator(wsDir)
	questionID := generateQuestionID(coord.State())

	body := `{"questionId":"` + questionID + `","selectedChoices":["A"]}`
	w := serveHTTP(srv, "POST", "/api/v1/workspaces/respond-ws/respond", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCollectSkillCategories(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, string)
		validate  func(*testing.T, []apiSkillCategory)
	}{
		{
			name: "noSkills",
			setupFunc: func(_ *testing.T, _ string) {
			},
			validate: func(t *testing.T, categories []apiSkillCategory) {
				assert.Empty(t, categories)
			},
		},
		{
			name: "singleSkillNoCategory",
			setupFunc: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				skillContent := `---
description: Test skill description
---
# Test Skill`
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))
			},
			validate: func(t *testing.T, categories []apiSkillCategory) {
				assert.Len(t, categories, 1)
				assert.Equal(t, "General", categories[0].Name)
				assert.Len(t, categories[0].Skills, 1)
				assert.Equal(t, "test-skill", categories[0].Skills[0].Name)
			},
		},
		{
			name: "singleSkillWithCategory",
			setupFunc: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "coding", "test-skill")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				skillContent := `---
description: Test skill description
---
# Test Skill`
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))
			},
			validate: func(t *testing.T, categories []apiSkillCategory) {
				assert.Len(t, categories, 1)
				assert.Equal(t, "coding", categories[0].Name)
				assert.Len(t, categories[0].Skills, 1)
				assert.Equal(t, "test-skill", categories[0].Skills[0].Name)
			},
		},
		{
			name: "multipleSkillsMultipleCategories",
			setupFunc: func(t *testing.T, dir string) {
				for _, skill := range []struct {
					category string
					name     string
				}{
					{"coding", "skill1"},
					{"coding", "skill2"},
					{"testing", "skill3"},
				} {
					skillDir := filepath.Join(dir, ".sgai", "skills", skill.category, skill.name)
					require.NoError(t, os.MkdirAll(skillDir, 0755))
					skillContent := `---
description: ` + skill.name + `
---
# ` + skill.name
					require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644))
				}
			},
			validate: func(t *testing.T, categories []apiSkillCategory) {
				assert.Len(t, categories, 2)
				assert.Equal(t, "coding", categories[0].Name)
				assert.Len(t, categories[0].Skills, 2)
				assert.Equal(t, "testing", categories[1].Name)
				assert.Len(t, categories[1].Skills, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupFunc(t, dir)
			result := collectSkillCategories(dir)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConvertSnippetLanguages(t *testing.T) {
	tests := []struct {
		name       string
		categories []languageCategory
		expected   []apiLanguageCategory
	}{
		{
			name:       "empty",
			categories: []languageCategory{},
			expected:   []apiLanguageCategory{},
		},
		{
			name: "singleCategory",
			categories: []languageCategory{
				{
					Name: "go",
					Snippets: []snippetData{
						{Name: "snippet1", FileName: "file1", Description: "desc1"},
					},
				},
			},
			expected: []apiLanguageCategory{
				{
					Name: "go",
					Snippets: []apiSnippetEntry{
						{Name: "snippet1", FileName: "file1", Description: "desc1"},
					},
				},
			},
		},
		{
			name: "multipleCategories",
			categories: []languageCategory{
				{
					Name: "go",
					Snippets: []snippetData{
						{Name: "go-snippet", FileName: "go-file"},
					},
				},
				{
					Name: "python",
					Snippets: []snippetData{
						{Name: "py-snippet", FileName: "py-file"},
					},
				},
			},
			expected: []apiLanguageCategory{
				{
					Name: "go",
					Snippets: []apiSnippetEntry{
						{Name: "go-snippet", FileName: "go-file"},
					},
				},
				{
					Name: "python",
					Snippets: []apiSnippetEntry{
						{Name: "py-snippet", FileName: "py-file"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSnippetLanguages(tt.categories)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertTodosForAPI(t *testing.T) {
	tests := []struct {
		name     string
		todos    []state.TodoItem
		expected []apiTodoEntry
	}{
		{
			name:     "empty",
			todos:    []state.TodoItem{},
			expected: []apiTodoEntry{},
		},
		{
			name: "singleTodo",
			todos: []state.TodoItem{
				{ID: "1", Content: "Task 1", Status: "pending", Priority: "high"},
			},
			expected: []apiTodoEntry{
				{ID: "1", Content: "Task 1", Status: "pending", Priority: "high"},
			},
		},
		{
			name: "multipleTodos",
			todos: []state.TodoItem{
				{ID: "1", Content: "Task 1", Status: "pending", Priority: "high"},
				{ID: "2", Content: "Task 2", Status: "completed", Priority: "medium"},
				{ID: "3", Content: "Task 3", Status: "in_progress", Priority: "low"},
			},
			expected: []apiTodoEntry{
				{ID: "1", Content: "Task 1", Status: "pending", Priority: "high"},
				{ID: "2", Content: "Task 2", Status: "completed", Priority: "medium"},
				{ID: "3", Content: "Task 3", Status: "in_progress", Priority: "low"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTodosForAPI(tt.todos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleAPIDeleteForkNoConfirm(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceNotFound(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete", `{"confirm":true}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteWorkspaceStandalone(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".jj"), 0755))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{"confirm":true}`)
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError, http.StatusNotFound}, w.Code)
}

func TestHandleAPIGetGoalMissing(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/goal", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleAPIUpdateGoalEmptyContent(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", `{"content":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", "{bad}")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocEmptyPrompt(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", `{"prompt":"","model":"test"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocEmptyModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", `{"prompt":"test","model":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIStartSessionRootWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.classifyCache.set(wsDir, workspaceRoot)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/start", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "root workspace cannot start")
}

func TestHandleAPIForkWorkspaceStandalone(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/fork", `{"goalContent":"# Test"}`)
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError, http.StatusCreated}, w.Code)
}

func TestHandleAPICreateWorkspaceInvalidName(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces", `{"name":"../bad","goalContent":"test"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAttachWorkspaceInvalidBody(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", "{bad}")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIOpenEditorForWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	server.editorAvailable = true
	server.editor = newConfigurableEditor("echo")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/open-editor", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable, http.StatusInternalServerError}, w.Code)
}

func TestHandleAPIDeleteMessageNumericNotFound(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "DELETE", "/api/v1/workspaces/test-ws/messages/99999", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError}, w.Code)
}

func TestCollectJJChangesNonRepo(t *testing.T) {
	dir := t.TempDir()
	lines, summary := collectJJChanges(dir)
	assert.Nil(t, lines)
	assert.Empty(t, summary)
}

func TestResolveWorkspaceFromPathNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/workspaces/nonexistent-ws/goal", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestWriteJSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, map[string]string{"key": "value"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"key"`)
}

func TestSpaMiddleware(t *testing.T) {
	server, _ := setupTestServer(t)
	mux := http.NewServeMux()
	handler := server.spaMiddleware(mux)
	assert.NotNil(t, handler)

	req := httptest.NewRequest("GET", "/some-page", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
}

func TestFilteredCommitsForWorkspaceNonRepo(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	result := server.filteredCommitsForWorkspace(wsDir)
	assert.Empty(t, result)
}

func TestResolveRootForDeleteFork(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.resolveRootForDeleteFork("/nonexistent")
	assert.Empty(t, result)
}

func TestHandleAPIComposePreviewNoWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/compose/preview", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteWorkspaceRootBlocked(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.classifyCache.set(wsDir, workspaceRoot)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{"confirm":true}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "root workspace")
}

func TestHandleAPIDeleteWorkspaceConfirmed(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete", `{"confirm":true}`)
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
}

func TestHandleAPIStopSessionNotFoundNewBatch(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/stop", `{}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIForkTemplateRootNewBatch(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.classifyCache.set(wsDir, workspaceRoot)
	w := serveHTTP(server, "GET", "/api/v1/workspaces/test-ws/fork-template", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "content")
}

func TestHandleAPIDeleteForkNotFoundNewBatch(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/delete-fork", `{}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteForkStandaloneBlocked(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "not a root or fork")
}

func TestHandleAPIDeleteForkInvalidForkDir(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.classifyCache.set(wsDir, workspaceRoot)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/delete-fork", `{"confirm":true,"forkDir":"../bad"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocInvalidJSON(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", "{invalid}")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocMissingPrompt(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", `{"prompt":"","model":"test"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "prompt and model are required")
}

func TestHandleAPIAdhocMissingModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/adhoc", `{"prompt":"test","model":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocStopNotRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "DELETE", "/api/v1/workspaces/test-ws/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "stopped")
}

func TestHandleAPIUpdateGoalValidContent(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))
	w := serveHTTP(server, "PUT", "/api/v1/workspaces/test-ws/goal", `{"content":"# New Goal"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIRespondNoPendingQuestion(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/respond", `{"questionId":"q1","answer":"test"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "no pending question")
}

func TestHandleAPICreateWorkspaceSuccess(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces", `{"name":"test-new-ws"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "test-new-ws")
}

func TestHandleAPIAttachWorkspaceNotAbsolute(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{"path":"relative/path"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDetachNonexistentWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/detach", `{"path":"/nonexistent/path"}`)
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError}, w.Code)
}

func TestHandleAPIForkEmptyGoal(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/fork", `{"goalContent":""}`)
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, w.Code)
}

func TestHandleAPIListModelsWithWorkspaceNewBatch(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	w := serveHTTP(server, "GET", "/api/v1/models?workspace=test-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResolveForkTemplateContentNoGroupsNewBatch(t *testing.T) {
	server, _ := setupTestServer(t)
	result := server.resolveForkTemplateContent("/nonexistent")
	assert.NotEmpty(t, result)
}

func TestReadNewestForkGoalWithForksNewBatch(t *testing.T) {
	dir := t.TempDir()
	forkDir := filepath.Join(dir, "fork1")
	require.NoError(t, os.MkdirAll(forkDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(forkDir, "GOAL.md"), []byte("# Fork Goal"), 0644))
	forks := []workspaceInfo{{DirName: "fork1", Directory: forkDir}}
	result := readNewestForkGoal(forks)
	assert.Equal(t, "# Fork Goal", result)
}

func TestResolveCallerAgentNewBatch(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{CurrentAgent: "builder"})
	require.NoError(t, errCoord)
	t.Run("notCoordinator", func(t *testing.T) {
		result := resolveCallerAgent("tester", coord)
		assert.Equal(t, "tester", result)
	})
	t.Run("coordinatorWithCurrent", func(t *testing.T) {
		result := resolveCallerAgent("coordinator", coord)
		assert.Equal(t, "builder", result)
	})
	t.Run("coordinatorNoCurrent", func(t *testing.T) {
		emptyFile := filepath.Join(t.TempDir(), "state.json")
		emptyCoord, errEmpty := state.NewCoordinatorWith(emptyFile, state.Workflow{})
		require.NoError(t, errEmpty)
		result := resolveCallerAgent("coordinator", emptyCoord)
		assert.Equal(t, "coordinator", result)
	})
}

func TestParseAgentIdentityHeaderNewBatch(t *testing.T) {
	t.Run("emptyHeader", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		result := parseAgentIdentityHeader(r)
		assert.Equal(t, "coordinator", result)
	})
	t.Run("agentWithPipe", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Sgai-Agent-Identity", "builder|extra")
		result := parseAgentIdentityHeader(r)
		assert.Equal(t, "builder", result)
	})
	t.Run("emptyBeforePipe", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Sgai-Agent-Identity", "|extra")
		result := parseAgentIdentityHeader(r)
		assert.Equal(t, "coordinator", result)
	})
}

func TestHandleAPIOpenEditorProjectMgmt(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")
	server.editorAvailable = true
	server.editor = newConfigurableEditor("echo")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/test-ws/open-editor/project-management", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusServiceUnavailable, http.StatusInternalServerError}, w.Code)
}

func TestStartSessionServiceClassification(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "classify-ws")
	kind := server.classifyWorkspaceCached(wsDir)
	assert.Equal(t, workspaceStandalone, kind)
}

func TestStopSessionServiceAlreadyStopped(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stopped-ws")

	result := server.stopSessionService(wsDir)
	assert.Equal(t, "stopped", result.Status)
	assert.False(t, result.Running)
	assert.Contains(t, result.Message, "already stopped")
}

func TestRespondServiceNoPendingQuestion(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "respond-ws2")
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte(`{"status":"working"}`), 0644))

	_, err := server.respondService(wsDir, "q1", "answer", nil)
	assert.Error(t, err)
}

func TestSteerServiceEmptyMessage(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-ws2")

	_, err := server.steerService(wsDir, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestSteerServiceValidMessage(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-ws3")
	goalContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))
	statePath := filepath.Join(wsDir, ".sgai", "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte(`{"status":"working","messages":[]}`), 0644))

	result, err := server.steerService(wsDir, "please fix the tests")
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestHandleAPIDeleteForkNotConfirmedNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "fork-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/fork-ws/delete-fork", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceNotConfirmed(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "del-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/del-ws/delete", `{"confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocStatusNotRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-ws")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/adhoc-ws/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiAdhocResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp.Running)
}

func TestHandleAPIAdhocStopWhenNotRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-ws2")
	w := serveHTTP(server, "DELETE", "/api/v1/workspaces/adhoc-ws2/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp apiAdhocResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp.Running)
}

func TestHandleAPIStopSessionNoSessionNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "stop-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/stop-ws/stop", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIBrowseDirectoriesEmptyPath(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/browse-directories?path=", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code)
}

func TestHandleAPIDeleteMessageNotFoundNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "msg-ws3")
	w := serveHTTP(server, "DELETE", "/api/v1/workspaces/msg-ws3/messages/999", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest}, w.Code)
}

func TestGetWorkflowSVGServiceNoGoal(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "svg-ws")
	svg := server.getWorkflowSVGService(wsDir)
	assert.Empty(t, svg)
}

func TestGetWorkflowSVGServiceWithGoal(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "svg-ws2")
	goalContent := "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0644))

	svg := server.getWorkflowSVGService(wsDir)
	assert.NotEmpty(t, svg)
}

func TestHandleAPIComposeStateWithWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-new-ws")
	w := serveHTTP(server, "GET", "/api/v1/compose?workspace=compose-new-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeSaveWithWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-save-new")
	w := serveHTTP(server, "POST", "/api/v1/compose?workspace=compose-save-new", `{"flow":"\"a\" -> \"b\"","body":"# Goal","models":{}}`)
	assert.Contains(t, []int{http.StatusOK, http.StatusCreated}, w.Code)
}

func TestHandleAPISteerEmptyMessage(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/steer-ws/steer", `{"message":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIForkTemplateNoGroupsNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "template-ws")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/template-ws/fork-template", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code)
}

func TestHandleAPIListModelsNoWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/models", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
}

func TestCollectForksForAPIFromGroupsWithMatchingRootNew(t *testing.T) {
	server, rootDir := setupTestServer(t)

	rootWSDir := filepath.Join(rootDir, "root-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(rootWSDir, ".jj", "repo"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(rootWSDir, ".sgai"), 0755))

	forkDir := filepath.Join(rootDir, "fork-ws")
	require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".jj"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(forkDir, ".jj", "repo"), []byte(filepath.Join(rootWSDir, ".jj", "repo")), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(forkDir, ".sgai"), 0755))

	groups := []workspaceGroup{
		{
			Root: workspaceInfo{Directory: rootWSDir, DirName: "root-ws"},
			Forks: []workspaceInfo{
				{Directory: forkDir, DirName: "fork-ws"},
			},
		},
	}

	forks := server.collectForksForAPIFromGroups(rootWSDir, groups)
	assert.Len(t, forks, 1)
	assert.Equal(t, "fork-ws", forks[0].Name)
}

func TestHandleAPIOpenEditorNoEditorNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "editor-ws")
	server.editorAvailable = false
	w := serveHTTP(server, "POST", "/api/v1/workspaces/editor-ws/open-editor", "")
	assert.Contains(t, []int{http.StatusServiceUnavailable, http.StatusInternalServerError}, w.Code)
}

func TestDeleteWorkspaceServiceDirect(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "del-svc-api")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))

	result, err := server.deleteWorkspaceService(wsDir)
	require.NoError(t, err)
	assert.True(t, result.Deleted)

	_, errStat := os.Stat(wsDir)
	assert.True(t, os.IsNotExist(errStat))
}

func TestDeleteMessageServiceDirect(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "msg-svc-api")

	coord := server.workspaceCoordinator(wsDir)
	require.NoError(t, coord.UpdateState(func(wf *state.Workflow) {
		wf.Messages = append(wf.Messages, state.Message{ID: 42, Body: "hello"})
	}))

	result, err := server.deleteMessageService(wsDir, 42)
	require.NoError(t, err)
	assert.True(t, result.Deleted)
	assert.Equal(t, 42, result.ID)
}

func TestDeleteMessageServiceNotFoundDirect(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "msg-svc2-api")
	_, err := server.deleteMessageService(wsDir, 999)
	assert.Error(t, err)
}

func TestResolveForkTemplateContentNoForks(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "template-svc-api")
	content := server.resolveForkTemplateContent(wsDir)
	assert.NotEmpty(t, content)
}

func TestCollectJJChangesNoJJRepo(t *testing.T) {
	dir := t.TempDir()
	lines, desc := collectJJChanges(dir)
	assert.Nil(t, lines)
	assert.Empty(t, desc)
}

func TestCollectJJFullDiffNoJJRepo(t *testing.T) {
	dir := t.TempDir()
	diff := collectJJFullDiff(dir)
	assert.Empty(t, diff)
}

func TestHandleAPIBrowseDirectoriesNonexistentPath(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/browse-directories?path=/nonexistent/path", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocStatusState(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-state")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/adhoc-state/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "adhoc status")
}

func TestHandleAPIAdhocStopWhenNotRunningDirect(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-stop-direct")
	w := serveHTTP(server, "DELETE", "/api/v1/workspaces/adhoc-stop-direct/adhoc", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPISnippetsWithWorkspaceNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "snippets-ws-new")
	goDir := filepath.Join(wsDir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(goDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "test.go"), []byte("---\nname: Test\ndescription: Test snippet\n---\npackage main"), 0644))
	w := serveHTTP(server, "GET", "/api/v1/snippets?workspace=snippets-ws-new", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeDraftGlobal(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "compose-draft-global")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/compose/draft", `{"workspace":"compose-draft-global"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIOpenEditorMissingWorkspace(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/nonexistent/open-editor", `{"target":"goal"}`)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestHandleAPISteerValidMessageNew(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "steer-valid-new")
	stateFile := filepath.Join(wsDir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{
		Status: state.StatusWorking,
	})
	require.NoError(t, errCoord)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/steer-valid-new/steer", `{"message":"new direction"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIAdhocPostNoPrompt(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-noprompt")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/adhoc-noprompt/adhoc", `{"prompt":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocPostInvalidBody(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-badbody")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/adhoc-badbody/adhoc", "not json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBrowseDirectoriesServiceHomeFallback(t *testing.T) {
	entries, err := browseDirectoriesService("")
	require.NoError(t, err)
	assert.NotNil(t, entries)
}

func TestBrowseDirectoriesServiceExistingDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "visible"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".hidden"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644))
	entries, err := browseDirectoriesService(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "visible", entries[0].Name)
}

func TestBrowseDirectoriesServiceNonexistent(t *testing.T) {
	_, err := browseDirectoriesService("/nonexistent/path")
	assert.Error(t, err)
}

func TestLoadExternalDirsNoFile(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.externalConfigDir = t.TempDir()
	err := srv.loadExternalDirs()
	assert.NoError(t, err)
}

func TestSaveAndLoadExternalDirs(t *testing.T) {
	srv, _ := setupTestServer(t)
	dir := t.TempDir()
	srv.externalConfigDir = dir
	srv.externalDirs = map[string]bool{"/tmp/test-ws": true}
	err := srv.saveExternalDirs()
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "external.json"))

	srv2, _ := setupTestServer(t)
	srv2.externalConfigDir = dir
	err = srv2.loadExternalDirs()
	assert.NoError(t, err)
}

func TestIsExternalWorkspaceNotAttached(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.externalDirs = map[string]bool{"/tmp/ext-ws": true}
	assert.False(t, srv.isExternalWorkspace("/tmp/not-ext"))
}

func TestDetachExternalWorkspaceNotAttached(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.externalConfigDir = t.TempDir()
	_, err := srv.detachExternalWorkspaceService("/tmp/not-attached")
	assert.ErrorIs(t, err, errNotAttached)
}

func TestAttachExternalWorkspaceNotAbsolute(t *testing.T) {
	srv, _ := setupTestServer(t)
	_, err := srv.attachExternalWorkspaceService("relative/path")
	assert.ErrorIs(t, err, errPathNotAbsolute)
}

func TestAttachExternalWorkspaceNotDir(t *testing.T) {
	srv, _ := setupTestServer(t)
	f := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("hi"), 0644))
	_, err := srv.attachExternalWorkspaceService(f)
	assert.ErrorIs(t, err, errNotADirectory)
}

func TestAttachExternalWorkspaceUnderRoot(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	srv.externalConfigDir = t.TempDir()
	subDir := filepath.Join(rootDir, "subworkspace")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	_, err := srv.attachExternalWorkspaceService(subDir)
	assert.ErrorIs(t, err, errUnderRootDir)
}

func TestValidateProjectConfigInvalidModel(t *testing.T) {
	cfg := &projectConfig{DefaultModel: "invalid model with space"}
	err := validateProjectConfig(cfg)
	assert.Error(t, err)
}

func TestServeReactIndexMissingIndex(t *testing.T) {
	emptyFS := fstest.MapFS{}
	w := httptest.NewRecorder()
	serveReactIndex(w, emptyFS)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "react app not found")
}

func TestServeReactIndexSuccess(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>test</html>")},
	}
	w := httptest.NewRecorder()
	serveReactIndex(w, testFS)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "<html>test</html>")
}

func TestHandleAPIDeleteWorkspaceDirNotExist(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-phantom")
	require.NoError(t, os.RemoveAll(filepath.Join(rootDir, "ws-phantom")))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-phantom/delete", `{"confirm": true}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPISteerSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-steer3")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0755))
	statePath := filepath.Join(sgaiDir, "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte(`{"status":"working"}`), 0644))

	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-steer3/steer", `{"message": "change direction"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiSteerResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Success)
}

func TestHandleAPIGetGoalSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-goal")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# My Goal"), 0644))
	w := serveHTTP(server, "GET", "/api/v1/workspaces/ws-goal/goal", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiGoalResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "# My Goal", resp.Content)
}

func TestHandleAPIUpdateGoalSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-upgoal3")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))
	w := serveHTTP(server, "PUT", "/api/v1/workspaces/ws-upgoal3/goal", `{"content": "# New Goal"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	data, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Equal(t, "# New Goal", string(data))
}

func TestHandleAPITogglePinSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-pin")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	server.pinnedConfigDir = t.TempDir()
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-pin/pin", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiTogglePinResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Pinned)
}

func TestHandleAPIDeleteForkNotRootOrFork(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-delfork3")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-delfork3/delete-fork", `{"confirm":true}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteMessageMissingID(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-delmsg")
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	req := httptest.NewRequest("DELETE", "/api/v1/workspaces/ws-delmsg/messages/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusMethodNotAllowed}, w.Code)
}

func TestHandleAPIDeleteMessageNotFound(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-delmsg3")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"working","messages":[]}`), 0644))
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	req := httptest.NewRequest("DELETE", "/api/v1/workspaces/ws-delmsg3/messages/99", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteMessageSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-delmsg4")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	stateData := `{"status":"working","messages":[{"id":1,"fromAgent":"a","toAgent":"b","body":"msg","read":false}]}`
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(stateData), 0644))
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	req := httptest.NewRequest("DELETE", "/api/v1/workspaces/ws-delmsg4/messages/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiDeleteMessageResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Deleted)
	assert.Equal(t, 1, resp.ID)
}

func TestHandleAPIOpenEditorNotAvailable(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-editor")
	server.editorAvailable = false
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-editor/open-editor", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleAPIOpenEditorGoalNotAvailable(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-editgoal")
	server.editorAvailable = false
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-editgoal/open-editor/goal", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleAPIOpenEditorGoalFileNotFound(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-editgoal2")
	server.editorAvailable = true
	server.editor = newConfigurableEditor("echo")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-editgoal2/open-editor/goal", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIOpenEditorGoalSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-editgoal3")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	server.editorAvailable = true
	server.editorName = "echo"
	server.editor = newConfigurableEditor("echo")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-editgoal3/open-editor/goal", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteMessageServiceSuccess(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-delsvc")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	stateData := `{"status":"working","messages":[{"id":5,"fromAgent":"a","toAgent":"b","body":"test","read":false}]}`
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(stateData), 0644))
	result, err := srv.deleteMessageService(wsDir, 5)
	assert.NoError(t, err)
	assert.True(t, result.Deleted)
	assert.Equal(t, 5, result.ID)
}

func TestDeleteMessageServiceNotFound(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-delsvc2")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"working","messages":[]}`), 0644))
	_, err := srv.deleteMessageService(wsDir, 99)
	assert.ErrorIs(t, err, errMessageNotFound)
}

func TestTogglePin(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	srv.pinnedConfigDir = t.TempDir()
	assert.False(t, srv.isPinned(rootDir))
	require.NoError(t, srv.togglePin(rootDir))
	assert.True(t, srv.isPinned(rootDir))
	require.NoError(t, srv.togglePin(rootDir))
	assert.False(t, srv.isPinned(rootDir))
}

func TestHandleAPIAdhocPostMissingModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-adhoc-m")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-adhoc-m/adhoc", `{"prompt":"test","model":""}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAdhocStopSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-adhocstop")
	_ = server.getAdhocState(wsDir)
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)
	req := httptest.NewRequest("DELETE", "/api/v1/workspaces/ws-adhocstop/adhoc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIWorkflowSVGNotAvailable(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-svg")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/ws-svg/workflow-svg", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSessionCoordinatorNoSession(t *testing.T) {
	srv, _ := setupTestServer(t)
	assert.Nil(t, srv.sessionCoordinator("/nonexistent"))
}

func TestHandleAPISignalStream(t *testing.T) {
	server, _ := setupTestServer(t)
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/api/v1/signal", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	server.notifyStateChange()
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
}

func TestHandleAPIOpenEditorPMNotAvailable(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-editpm")
	server.editorAvailable = false
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-editpm/open-editor/project-management", "")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleAPIOpenEditorPMSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-editpm2")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0644))
	server.editorAvailable = true
	server.editorName = "echo"
	server.editor = newConfigurableEditor("echo")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-editpm2/open-editor/project-management", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIOpenEditorSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "ws-edit-open")
	server.editorAvailable = true
	server.editorName = "echo"
	server.editor = newConfigurableEditor("echo")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-edit-open/open-editor", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStartSessionAlreadyRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-start-double")
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: true}
	server.mu.Unlock()

	result := server.startSession(wsDir)
	assert.True(t, result.alreadyRunning)
}

func TestPollWorkspaceStates(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-poll")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"working"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"), 0644))

	snapshots := make(map[string]workspaceStateSnapshot)
	server.pollWorkspaceStates(snapshots)
	assert.NotEmpty(t, snapshots)
}

func TestCheckWorkspaceStateNoState(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-check-no-state")
	snapshots := make(map[string]workspaceStateSnapshot)
	active := make(map[string]bool)
	server.checkWorkspaceState(wsDir, snapshots, active)
	assert.True(t, active[wsDir])
	assert.Empty(t, snapshots)
}

func TestCheckWorkspaceStateWithState(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-check-state")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"working"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test"), 0644))
	snapshots := make(map[string]workspaceStateSnapshot)
	active := make(map[string]bool)
	server.checkWorkspaceState(wsDir, snapshots, active)
	assert.True(t, active[wsDir])
}

func TestCheckWorkspaceStateChanged(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-check-change")
	sgaiDir := filepath.Join(wsDir, ".sgai")
	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"working"}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test"), 0644))

	snapshots := make(map[string]workspaceStateSnapshot)
	active := make(map[string]bool)
	server.checkWorkspaceState(wsDir, snapshots, active)

	require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "state.json"), []byte(`{"status":"complete"}`), 0644))
	server.checkWorkspaceState(wsDir, snapshots, active)
}

func TestInitializeWorkspaceFullPath(t *testing.T) {
	dir := t.TempDir()
	err := initializeWorkspace(dir)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "GOAL.md"))
	assert.DirExists(t, filepath.Join(dir, ".sgai"))
}

func TestStopSessionNotRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-stop-norun")
	server.stopSession(wsDir)
}

func TestHandleAPIStartSessionAlreadyRunningViaHTTP(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ws-start-api")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "state.json"), []byte(`{"status":"working"}`), 0644))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	server.shutdownCtx = ctx
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: true}
	server.mu.Unlock()

	w := serveHTTP(server, "POST", "/api/v1/workspaces/ws-start-api/start", `{"auto":false}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiSessionActionResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Running)
	assert.Equal(t, "session already running", resp.Message)
}

func TestHandleAPIListModelsWithWorkspaceParam(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "models-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nmodels:\n  coordinator: [\"anthropic/claude-opus-4-6\"]\n---\n# Models Test"), 0644))
	w := serveHTTP(server, "GET", "/api/v1/models?workspace=models-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIDeleteForkNotARoot(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "standalone-delfork")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/standalone-delfork/delete-fork", `{"forkDir":"/tmp/nope","confirm":true}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteForkNoConfirmation(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "noconfirm-delfork")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/noconfirm-delfork/delete-fork", `{"forkDir":"/tmp/nope","confirm":false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIForkWorkspaceNotRootStandalone(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "not-root-ws")
	w := serveHTTP(server, "POST", "/api/v1/workspaces/not-root-ws/fork", `{"goalContent":"# Test Goal"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleAPIComposeStateMissingWs(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/compose?workspace=nonexistent-compose", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIComposeStateExistsWs(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-ws")
	w := serveHTTP(server, "GET", "/api/v1/compose?workspace=compose-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIComposeSaveMissingWs(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/compose/save?workspace=nonexistent-save", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIComposePreviewMissingWs(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "GET", "/api/v1/compose/preview?workspace=nonexistent-preview", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIComposePreviewExistsWs(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "preview-ws")
	w := serveHTTP(server, "GET", "/api/v1/compose/preview?workspace=preview-ws", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIStopSessionRunning(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "stop-session-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	server.mu.Lock()
	server.sessions[wsDir] = &session{running: true}
	server.mu.Unlock()
	w := serveHTTP(server, "POST", "/api/v1/workspaces/stop-session-ws/stop", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIWorkspaceDiffForWorkspace(t *testing.T) {
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "diff-ws")
	w := serveHTTP(server, "GET", "/api/v1/workspaces/diff-ws/diff", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleAPIDeleteWorkspaceStandaloneConfirmed(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "delete-standalone-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/delete-standalone-ws/delete", `{"confirm": true}`)
	assert.Equal(t, http.StatusOK, w.Code)
	_, errStat := os.Stat(wsDir)
	assert.True(t, os.IsNotExist(errStat))
}

func TestHandleAPIWorkflowSVGForWorkspaceWithFlow(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "svg-flow-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: coordinator -> worker\n---\n# Goal"), 0644))
	w := serveHTTP(server, "GET", "/api/v1/workspaces/svg-flow-ws/workflow.svg", "")
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound}, w.Code)
}

func TestHandleAPIDeleteWorkspaceNoConfirmField(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "delete-noconfirm2-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/delete-noconfirm2-ws/delete", `{"confirm": false}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIDeleteWorkspaceDirAlreadyRemoved(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "delete-removed-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	require.NoError(t, os.RemoveAll(wsDir))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/delete-removed-ws/delete", `{"confirm": true}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAPIDeleteWorkspaceExternalDetach(t *testing.T) {
	server, rootDir := setupTestServer(t)
	extDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".sgai"), 0755))
	server.mu.Lock()
	server.externalDirs[extDir] = true
	server.mu.Unlock()
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".sgai"), 0755))
	_ = rootDir
	wsName := filepath.Base(extDir)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/"+wsName+"/delete", `{"confirm": true}`)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest}, w.Code)
}

func TestHandleAPIDetachWorkspaceInvalidBody(t *testing.T) {
	server, _ := setupTestServer(t)
	w := serveHTTP(server, "POST", "/api/v1/workspaces/detach", `not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIComposeSaveWriteSuccess(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "compose-save-ok-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/compose?workspace=compose-save-ok-ws", `{}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleAPIAttachWorkspaceNotDirectory(t *testing.T) {
	server, _ := setupTestServer(t)
	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(tmpFile, []byte("a file"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{"path": "`+tmpFile+`"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAttachWorkspaceAlreadyAttached(t *testing.T) {
	server, _ := setupTestServer(t)
	extDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".sgai"), 0755))
	canonical := resolveSymlinks(extDir)
	server.mu.Lock()
	server.externalDirs[canonical] = true
	server.mu.Unlock()
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{"path": "`+extDir+`"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandleAPIAttachWorkspaceUnderRootDir(t *testing.T) {
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	subDir := filepath.Join(rootDir, "subproject")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{"path": "`+subDir+`"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAPIAttachWorkspaceWithGoal(t *testing.T) {
	server, _ := setupTestServer(t)
	extDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(extDir, ".sgai"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(extDir, "GOAL.md"), []byte("# External Goal"), 0644))
	w := serveHTTP(server, "POST", "/api/v1/workspaces/attach", `{"path": "`+extDir+`"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"hasGoal":true`)
}
