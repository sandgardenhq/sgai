package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextResult(t *testing.T) {
	result := textResult("hello world")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	assert.Equal(t, "hello world", result.Content[0].(*mcp.TextContent).Text)
}

func TestJSONResult(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "number",
			input:    42,
			expected: `42`,
		},
		{
			name:     "bool",
			input:    true,
			expected: `true`,
		},
		{
			name:     "slice",
			input:    []string{"a", "b"},
			expected: `["a","b"]`,
		},
		{
			name:     "map",
			input:    map[string]string{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "struct",
			input:    struct{ Name string }{Name: "test"},
			expected: `{"Name":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonResult(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Content, 1)
			assert.JSONEq(t, tt.expected, result.Content[0].(*mcp.TextContent).Text)
		})
	}
}

func TestJSONResultError(t *testing.T) {
	type unmarshalable struct {
		Ch chan int
	}

	_, err := jsonResult(&unmarshalable{})
	assert.Error(t, err)
}

func TestBuildPendingQuestionSchema(t *testing.T) {
	schema := buildPendingQuestionSchema(nil)
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	assert.Contains(t, schema.Properties, "answer")
	assert.Equal(t, "string", schema.Properties["answer"].Type)
	assert.Equal(t, []string{"answer"}, schema.Required)
}

func TestResolveWorkspacePath(t *testing.T) {
	t.Run("emptyName", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		ctx := &externalMCPContext{srv: srv}
		_, err := ctx.resolveWorkspacePath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace name is required")
	})

	t.Run("notFound", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		ctx := &externalMCPContext{srv: srv}
		_, err := ctx.resolveWorkspacePath("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})
}

func TestResolveAnyWorkspacePath(t *testing.T) {
	t.Run("noWorkspaces", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		ctx := &externalMCPContext{srv: srv}
		_, err := ctx.resolveAnyWorkspacePath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no workspaces found")
	})
}

func TestBuildExternalMCPServer(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := buildExternalMCPServer(ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterExternalTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerExternalTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterStateTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerStateTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterWorkspaceTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerWorkspaceTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterSessionTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerSessionTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterKnowledgeTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerKnowledgeTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterComposeTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerComposeTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterAdhocTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerAdhocTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterEditorTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerEditorTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterModelTools(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerModelTools(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestRegisterElicitationTool(t *testing.T) {
	srv, _ := setupTestServer(t)
	ctx := &externalMCPContext{srv: srv}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "sgai-test"}, nil)
	registerElicitationTool(mcpServer, ctx)
	assert.NotNil(t, mcpServer)
}

func TestBuildExternalMCPHandler(t *testing.T) {
	srv, _ := setupTestServer(t)
	handler := buildExternalMCPHandler(srv)
	assert.NotNil(t, handler)
}

func connectMCPClient(t *testing.T, srv *Server) *mcp.ClientSession {
	t.Helper()
	ctx := &externalMCPContext{srv: srv}
	mcpServer := buildExternalMCPServer(ctx)
	ct, st := mcp.NewInMemoryTransports()
	_, errConnect := mcpServer.Connect(context.Background(), st, nil)
	require.NoError(t, errConnect)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, errClient := client.Connect(context.Background(), ct, nil)
	require.NoError(t, errClient)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestMCPToolListWorkspaces(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "mcp-ws")
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "mcp-ws", "GOAL.md"), []byte("---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"), 0644))

	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "list_workspaces",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
}

func TestMCPToolGetWorkspaceState(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "state-ws")
		require.NoError(t, os.WriteFile(filepath.Join(rootDir, "state-ws", "GOAL.md"), []byte("---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"), 0644))

		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "get_workspace_state",
			Arguments: map[string]any{"workspace": "state-ws"},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("notFound", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "get_workspace_state",
			Arguments: map[string]any{"workspace": "nonexistent"},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		tc := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, tc.Text, "workspace not found")
	})
}

func TestMCPToolGetWorkflowSVG(t *testing.T) {
	t.Run("emptyWorkspace", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "svg-mcp")
		cs := connectMCPClient(t, srv)
		_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "get_workflow_svg",
			Arguments: map[string]any{"workspace": "svg-mcp"},
		})
		require.NoError(t, err)
	})

	t.Run("workspaceNotFound", func(t *testing.T) {
		srv, _ := setupTestServer(t)
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "get_workflow_svg",
			Arguments: map[string]any{"workspace": "nonexistent"},
		})
		if err != nil {
			return
		}
		assert.True(t, result.IsError)
	})
}

func TestMCPToolGetWorkspaceDiff(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "diff-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_workspace_diff",
		Arguments: map[string]any{"workspace": "diff-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolCreateWorkspace(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_workspace",
		Arguments: map[string]any{"name": "**invalid**"},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolDeleteFork(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "delfork-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_fork",
		Arguments: map[string]any{"workspace": "delfork-mcp", "forkDir": "/tmp/nope", "confirm": true},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolGetGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "goal-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# My Goal"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_goal",
		Arguments: map[string]any{"workspace": "goal-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolUpdateGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "ugoal-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_goal",
		Arguments: map[string]any{"workspace": "ugoal-mcp", "content": "# New Goal"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolTogglePin(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "pin-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toggle_pin",
		Arguments: map[string]any{"workspace": "pin-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolUpdateDescription(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "desc-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_description",
		Arguments: map[string]any{"workspace": "desc-mcp", "description": "test desc"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolDeleteMessage(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "delmsg-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_message",
		Arguments: map[string]any{"workspace": "delmsg-mcp", "id": float64(999)},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolStartSessionWorkspaceNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "start_session",
		Arguments: map[string]any{"workspace": ""},
	})
	if err != nil {
		return
	}
	assert.True(t, result.IsError)
}

func TestMCPToolStopSession(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "stop-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "stop_session",
		Arguments: map[string]any{"workspace": "stop-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolRespondToQuestion(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "respond-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "respond_to_question",
		Arguments: map[string]any{"workspace": "respond-mcp", "questionId": "q1", "answer": "yes"},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolSteerAgent(t *testing.T) {
	t.Run("noGoal", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "steer-mcp")
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "steer_agent",
			Arguments: map[string]any{"workspace": "steer-mcp", "message": "do something"},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("withGoal", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		wsDir := setupTestWorkspace(t, rootDir, "steer-mcp2")
		require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("---\nflow: |\n  \"a\" -> \"b\"\n---\n# Test"), 0644))
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "steer_agent",
			Arguments: map[string]any{"workspace": "steer-mcp2", "message": "do something"},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestMCPToolListAgents(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "agents-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_agents",
		Arguments: map[string]any{"workspace": "agents-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolListSkills(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "skills-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_skills",
		Arguments: map[string]any{"workspace": "skills-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetSkillDetail(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "skill-detail-mcp")
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "get_skill_detail",
			Arguments: map[string]any{"workspace": "skill-detail-mcp", "name": "nonexistent"},
		})
		require.NoError(t, err)
		tc := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, tc.Text, "skill not found")
	})
}

func TestMCPToolListSnippets(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "snippets-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_snippets",
		Arguments: map[string]any{"workspace": "snippets-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolListSnippetsByLanguage(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "snippets-lang-mcp")
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "list_snippets_by_language",
			Arguments: map[string]any{"workspace": "snippets-lang-mcp", "language": "cobol"},
		})
		require.NoError(t, err)
		tc := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, tc.Text, "language not found")
	})
}

func TestMCPToolGetSnippetDetail(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		srv, rootDir := setupTestServer(t)
		setupTestWorkspace(t, rootDir, "snippet-detail-mcp")
		cs := connectMCPClient(t, srv)
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "get_snippet_detail",
			Arguments: map[string]any{"workspace": "snippet-detail-mcp", "language": "go", "fileName": "nonexistent"},
		})
		require.NoError(t, err)
		tc := result.Content[0].(*mcp.TextContent)
		assert.Contains(t, tc.Text, "snippet not found")
	})
}

func TestMCPToolGetComposeState(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_compose_state",
		Arguments: map[string]any{"workspace": "compose-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolSaveCompose(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "save-compose-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "save_compose",
		Arguments: map[string]any{"workspace": "save-compose-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetComposeTemplates(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_compose_templates",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetComposePreview(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "preview-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_compose_preview",
		Arguments: map[string]any{"workspace": "preview-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetAdhocStatus(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_adhoc_status",
		Arguments: map[string]any{"workspace": "adhoc-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolStartAdhoc(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-start-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "start_adhoc",
		Arguments: map[string]any{"workspace": "adhoc-start-mcp", "prompt": "", "model": ""},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolStopAdhoc(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-stop-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "stop_adhoc",
		Arguments: map[string]any{"workspace": "adhoc-stop-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolOpenEditor(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "editor-mcp")
	srv.editor = newConfigurableEditor("echo")
	srv.editorAvailable = true
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "open_editor",
		Arguments: map[string]any{"workspace": "editor-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolOpenEditorGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "editor-goal-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	srv.editor = newConfigurableEditor("echo")
	srv.editorAvailable = true
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "open_editor_goal",
		Arguments: map[string]any{"workspace": "editor-goal-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolOpenEditorPM(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "editor-pm-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".sgai", "PROJECT_MANAGEMENT.md"), []byte("# PM"), 0644))
	srv.editor = newConfigurableEditor("echo")
	srv.editorAvailable = true
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "open_editor_pm",
		Arguments: map[string]any{"workspace": "editor-pm-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolListModels(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "list_models",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolForkWorkspaceError(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "fork-src-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "fork_workspace",
		Arguments: map[string]any{"workspace": "fork-src-mcp", "name": ""},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolResolveAnyWorkspacePathWithWorkspace(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "any-ws")
	ctx := &externalMCPContext{srv: srv}
	path, err := ctx.resolveAnyWorkspacePath("any-ws")
	require.NoError(t, err)
	assert.Contains(t, path, "any-ws")
}

func TestMCPToolResolveAnyWorkspacePathEmpty(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "fallback-ws")
	ctx := &externalMCPContext{srv: srv}
	path, err := ctx.resolveAnyWorkspacePath("")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
}

func TestMCPToolWaitForQuestionTimeout(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "wait-mcp")
	cs := connectMCPClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wait_for_question",
		Arguments: map[string]any{"workspace": "wait-mcp"},
	})
	assert.Error(t, err)
}

func TestMCPToolCreateWorkspaceSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "create_workspace",
		Arguments: map[string]any{"name": "test-create-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestMCPToolGetGoalNoGoal(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "nogoal-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_goal",
		Arguments: map[string]any{"workspace": "nogoal-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolUpdateGoalSuccess(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "upgoal2-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_goal",
		Arguments: map[string]any{"workspace": "upgoal2-mcp", "content": "# Updated"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolUpdateGoalEmpty(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "upgoal-empty-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_goal",
		Arguments: map[string]any{"workspace": "upgoal-empty-mcp", "content": ""},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolTogglePinInvalid(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toggle_pin",
		Arguments: map[string]any{"workspace": "nonexistent-pin"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolDeleteForkSuccess(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "delws-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_fork",
		Arguments: map[string]any{"workspace": "delws-mcp", "forkDir": "/tmp/nonexistent-fork", "confirm": true},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolDeleteForkNoConfirm(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "delws-noconfirm-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_fork",
		Arguments: map[string]any{"workspace": "delws-noconfirm-mcp", "forkDir": "/tmp/nope", "confirm": false},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolSteerAgentEmpty(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "steer-empty-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "steer_agent",
		Arguments: map[string]any{"workspace": "steer-empty-mcp", "message": ""},
	})
	require.NoError(t, err)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "error:")
}

func TestMCPToolRespondToQuestionInvalid(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "respond_to_question",
		Arguments: map[string]any{"workspace": "nonexistent-resp-mcp", "questionId": "q1", "answer": "yes"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetAdhocStatusNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_adhoc_status",
		Arguments: map[string]any{"workspace": "nonexistent-adhoc-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolStopAdhocNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "stop_adhoc",
		Arguments: map[string]any{"workspace": "nonexistent-adhocstop-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolOpenEditorNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "open_editor",
		Arguments: map[string]any{"workspace": "nonexistent-editor-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolOpenEditorGoalNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "open_editor_goal",
		Arguments: map[string]any{"workspace": "nonexistent-editgoal-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolOpenEditorPMNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "open_editor_pm",
		Arguments: map[string]any{"workspace": "nonexistent-editpm-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetWorkspaceDiffNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_workspace_diff",
		Arguments: map[string]any{"workspace": "nonexistent-diff-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolDeleteMessageNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_message",
		Arguments: map[string]any{"workspace": "nonexistent-delmsg-mcp", "id": float64(1)},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolListAgentsNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_agents",
		Arguments: map[string]any{"workspace": "nonexistent-agents-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolListSkillsNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_skills",
		Arguments: map[string]any{"workspace": "nonexistent-skills-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetSkillDetailNotFoundWorkspace(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_skill_detail",
		Arguments: map[string]any{"workspace": "nonexistent-skill-mcp", "name": "test"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolListSnippetsNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_snippets",
		Arguments: map[string]any{"workspace": "nonexistent-snippets-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolListSnippetsByLangNotFoundWorkspace(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_snippets_by_language",
		Arguments: map[string]any{"workspace": "nonexistent-sniplang-mcp", "language": "go"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetSnippetDetailNotFoundWorkspace(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_snippet_detail",
		Arguments: map[string]any{"workspace": "nonexistent-snipdetail-mcp", "language": "go", "fileName": "test"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetComposeStateNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_compose_state",
		Arguments: map[string]any{"workspace": "nonexistent-compose-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolSaveComposeNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "save_compose",
		Arguments: map[string]any{"workspace": "nonexistent-savecompose-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetComposePreviewNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_compose_preview",
		Arguments: map[string]any{"workspace": "nonexistent-preview-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolWaitForQuestionNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "wait_for_question",
		Arguments: map[string]any{"workspace": "nonexistent-wait-mcp"},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMCPToolGetComposeStateExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-state-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_compose_state",
		Arguments: map[string]any{"workspace": "compose-state-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestMCPToolGetComposePreviewExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-preview-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_compose_preview",
		Arguments: map[string]any{"workspace": "compose-preview-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolSaveComposeDraft(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "compose-draft-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "save_compose_draft",
		Arguments: map[string]any{
			"workspace": "compose-draft-mcp",
			"state": map[string]any{
				"description":    "test desc",
				"completionGate": "",
				"agents":         []any{},
				"flow":           "",
				"tasks":          "",
			},
			"wizard": map[string]any{
				"currentStep":    0,
				"techStack":      []any{},
				"safetyAnalysis": false,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetAdhocStatusExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-status-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_adhoc_status",
		Arguments: map[string]any{"workspace": "adhoc-status-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestMCPToolStopAdhocExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "adhoc-stop-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "stop_adhoc",
		Arguments: map[string]any{"workspace": "adhoc-stop-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetWorkspaceDiffExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "wsdiff-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_workspace_diff",
		Arguments: map[string]any{"workspace": "wsdiff-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolSteerAgentWithMessage(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "steer-msg-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "steer_agent",
		Arguments: map[string]any{"workspace": "steer-msg-mcp", "message": "do something different"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolStopSessionExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "stopsess-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "stop_session",
		Arguments: map[string]any{"workspace": "stopsess-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolDeleteMessageExists(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "delmsg2-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_message",
		Arguments: map[string]any{"workspace": "delmsg2-mcp", "id": float64(999)},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMCPToolGetGoalSuccess(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "getgoal-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Test Goal Content"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_goal",
		Arguments: map[string]any{"workspace": "getgoal-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestMCPToolUpdateGoalVerifyContent(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "updgoal-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Old"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_goal",
		Arguments: map[string]any{"workspace": "updgoal-mcp", "content": "# Updated Goal via MCP"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	data, errRead := os.ReadFile(filepath.Join(wsDir, "GOAL.md"))
	require.NoError(t, errRead)
	assert.Contains(t, string(data), "Updated Goal via MCP")
}

func TestMCPToolTogglePinSuccess(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	setupTestWorkspace(t, rootDir, "pin-mcp")
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "toggle_pin",
		Arguments: map[string]any{"workspace": "pin-mcp"},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestMCPToolUpdateGoalEmptyContentReturnsError(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "updgoal-empty-mcp")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "GOAL.md"), []byte("# Goal"), 0644))
	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "update_goal",
		Arguments: map[string]any{"workspace": "updgoal-empty-mcp", "content": ""},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, "error:")
}

func TestMCPToolDeleteMessageWithExistingMessage(t *testing.T) {
	srv, rootDir := setupTestServer(t)
	wsDir := setupTestWorkspace(t, rootDir, "delmsg-exist-mcp")

	coord := srv.workspaceCoordinator(wsDir)
	require.NoError(t, coord.UpdateState(func(wf *state.Workflow) {
		wf.Messages = []state.Message{
			{ID: 42, FromAgent: "agent1", ToAgent: "agent2", Body: "test message"},
		}
	}))

	cs := connectMCPClient(t, srv)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "delete_message",
		Arguments: map[string]any{"workspace": "delmsg-exist-mcp", "id": float64(42)},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	tc := result.Content[0].(*mcp.TextContent)
	assert.Contains(t, tc.Text, "deleted")
}
