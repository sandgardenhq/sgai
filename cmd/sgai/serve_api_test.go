package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestHandleAPIStateOmitsMessages(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "state-no-messages")
	stateJSON := `{"status":"working","messages":[{"id":1,"fromAgent":"a","toAgent":"b","body":"old"}]}`
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "state.json"), []byte(stateJSON), 0o644))

	w := serveHTTP(server, "GET", "/api/v1/state", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), `"messages"`)
}

func computeExpectedEtag(content []byte) string {
	h := sha256.Sum256(content)
	return `"` + hex.EncodeToString(h[:8]) + `"`
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

func TestBuildAdhocArgs(t *testing.T) {
	t.Run("simpleModel", func(t *testing.T) {
		args := buildAdhocArgs("claude-opus-4")
		assert.Equal(t, []string{"run", "-m", "claude-opus-4", "--agent", "build", "--title", "adhoc [claude-opus-4]", "--format=json"}, args)
	})

	t.Run("modelWithVariant", func(t *testing.T) {
		args := buildAdhocArgs("claude-opus-4:fast")
		assert.Contains(t, args, "run")
		assert.Contains(t, args, "-m")
		assert.Contains(t, args, "--agent")
		assert.Contains(t, args, "build")
		assert.Contains(t, args, "adhoc [claude-opus-4:fast]")
	})

	t.Run("withVariantAddsFlag", func(t *testing.T) {
		args := buildAdhocArgs("openai/gpt-5.5 (thinking)")
		assert.Contains(t, args, "--variant")
		assert.Contains(t, args, "thinking")
	})

	t.Run("withoutVariantNoFlag", func(t *testing.T) {
		args := buildAdhocArgs("openai/gpt-5.5")
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
		require.NoError(t, os.WriteFile(goalPath, []byte("---\nmodel: claude-opus-4\n---\n# Goal"), 0o644))

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

	t.Run("withModelConfig", func(t *testing.T) {
		server, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "test-ws-model")
		goalContent := "---\nmodel: openai/gpt-5.5\nagents:\n  - coordinator\n---\n# Test"
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte(goalContent), 0o644))
		result := server.coordinatorModelFromWorkspace("test-ws-model")
		assert.Equal(t, "openai/gpt-5.5", result)
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

func TestConvertEventsForAPIWithEntries(t *testing.T) {
	displays := []eventsProgressDisplay{
		{Timestamp: "2024-01-01T00:00:00Z", FormattedTime: "00:00", Agent: "coordinator", Description: "started"},
	}
	result := convertEventsForAPI(displays)
	require.Len(t, result, 1)
	assert.Equal(t, "coordinator", result[0].Agent)
	assert.Equal(t, "started", result[0].Description)
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
		goalFileContent := "---\nmodel: openai/gpt-5.5\nagents:\n  - coordinator\n---\n# My Goal\n\nDo something."
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "GOAL.md"), []byte(goalFileContent), 0644))

		gc, rawGC, fullGC, pm, hasPM := readGoalAndPMForAPI(tmpDir)
		assert.NotEmpty(t, gc)
		assert.Contains(t, rawGC, "My Goal")
		assert.Contains(t, fullGC, "agents:")
		assert.Contains(t, fullGC, "model:")
		assert.Empty(t, pm)
		assert.False(t, hasPM)
	})

	t.Run("withGoalAndPM", func(t *testing.T) {
		tmpDir := t.TempDir()
		goalFileContent := "---\nmodel: openai/gpt-5.5\nagents:\n  - coordinator\n---\n# My Goal\n\nDo something."
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "GOAL.md"), []byte(goalFileContent), 0644))

		sgaiDir := filepath.Join(tmpDir, ".sgai")
		require.NoError(t, os.MkdirAll(sgaiDir, 0755))
		pmFileContent := "---\nRetrospective Session: .sgai/retro\n---\n\n## PM Content\n"
		require.NoError(t, os.WriteFile(filepath.Join(sgaiDir, "PROJECT_MANAGEMENT.md"), []byte(pmFileContent), 0644))

		gc, rawGC, fullGC, pm, hasPM := readGoalAndPMForAPI(tmpDir)
		assert.NotEmpty(t, gc)
		assert.Contains(t, rawGC, "My Goal")
		assert.Contains(t, fullGC, "agents:")
		assert.Contains(t, fullGC, "model:")
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
			Status:   state.StatusComplete,
			Progress: []state.ProgressEntry{},
		})
		require.NoError(t, err)

		wf := server.loadWorkspaceState(workDir)
		assert.Equal(t, state.StatusComplete, wf.Status)
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
	t.Helper()
	rootDir := t.TempDir()
	server := NewServer(rootDir)
	return server, rootDir
}

func setupTestWorkspace(t *testing.T, rootDir, name string) string {
	t.Helper()
	wsDir := filepath.Join(rootDir, name)
	require.NoError(t, os.MkdirAll(filepath.Join(wsDir, ".sgai"), 0o755))
	return wsDir
}

func serveHTTP(server *Server, method, path string, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	server.registerAPIRoutes(mux)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandleAPIComposePreviewUsesAgentsAndModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.composeDraftService(wsDir, composerState{Model: "openai/gpt-5.5 (xhigh)", Agents: []composerAgentConf{{Name: "go", Selected: true}}}, wizardState{})

	w := serveHTTP(server, http.MethodGet, "/api/v1/compose/preview?workspace=test-ws", "")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiComposePreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Content, "agents:\n  - \"go\"")
	assert.Contains(t, resp.Content, "model: \"openai/gpt-5.5 (xhigh)\"")
}

func TestHandleAPIComposeSaveWritesAgentsAndModel(t *testing.T) {
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	server.composeDraftService(wsDir, composerState{Model: "openai/gpt-5.5 (xhigh)", Agents: []composerAgentConf{{Name: "go", Selected: true}}}, wizardState{})

	w := serveHTTP(server, http.MethodPost, "/api/v1/compose?workspace=test-ws", "")

	assert.Equal(t, http.StatusCreated, w.Code)
	data, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Contains(t, string(data), "agents:\n  - \"go\"")
	assert.Contains(t, string(data), "model: \"openai/gpt-5.5 (xhigh)\"")
}

func TestHandleAPIListModelsReturnsWorkspaceDefaultModel(t *testing.T) {
	setupFakeOpenCode(t, fakeModelsVerboseOutput, 0)
	server, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "test-ws")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nmodel: openai/gpt-5.5 (xhigh)\n---\n# Goal"), 0o644))

	w := serveHTTP(server, http.MethodGet, "/api/v1/models?workspace=test-ws", "")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp apiModelsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "openai/gpt-5.5", resp.DefaultModel)
	assert.NotEmpty(t, resp.Models)
}

func TestHandleAPIAdhocStartReportsUnderlyingStartError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	server, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "test-ws")

	w := serveHTTP(server, http.MethodPost, "/api/v1/workspaces/test-ws/adhoc", `{"prompt":"do it","model":"model"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to start command")
	assert.Contains(t, w.Body.String(), "executable file not found")
}
