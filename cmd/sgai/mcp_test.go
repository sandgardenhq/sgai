package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sandgardenhq/sgai/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSnippetsByFuzzyMatch(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "http-server.go"), []byte("---\ndescription: HTTP server setup\n---\npackage main\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "json-parsing.go"), []byte("---\ndescription: JSON parsing utilities\n---\npackage main\n"), 0644))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	t.Run("matchByName", func(t *testing.T) {
		result, err := findSnippetsByFuzzyMatch(dir, entries, "http")
		require.NoError(t, err)
		assert.Contains(t, result, "http-server")
	})

	t.Run("matchByDescription", func(t *testing.T) {
		result, err := findSnippetsByFuzzyMatch(dir, entries, "parsing")
		require.NoError(t, err)
		assert.Contains(t, result, "json-parsing")
	})

	t.Run("noMatch", func(t *testing.T) {
		result, err := findSnippetsByFuzzyMatch(dir, entries, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestAskUserQuestionNoCoord(t *testing.T) {
	result, err := askUserQuestion(t.Context(), nil, askUserQuestionArgs{
		Questions: []questionItem{{Question: "test?", Choices: []string{"yes", "no"}}},
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Error")
}

func TestAskUserQuestionNoQuestions(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusWorking,
		InteractionMode: state.ModeBrainstorming,
	})
	require.NoError(t, err)

	result, errQ := askUserQuestion(t.Context(), coord, askUserQuestionArgs{})
	require.NoError(t, errQ)
	assert.Contains(t, result, "Error")
}

func TestAskUserQuestionEmptyChoices(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusWorking,
		InteractionMode: state.ModeBrainstorming,
	})
	require.NoError(t, err)

	result, errQ := askUserQuestion(t.Context(), coord, askUserQuestionArgs{
		Questions: []questionItem{{Question: "test?", Choices: nil}},
	})
	require.NoError(t, errQ)
	assert.Contains(t, result, "Error")
	assert.Contains(t, result, "no choices")
}

func TestAskUserWorkGateNoCoord(t *testing.T) {
	result, err := askUserWorkGate(t.Context(), nil, "summary")
	require.NoError(t, err)
	assert.Contains(t, result, "Error")
}

func TestAskUserWorkGateEmptySummary(t *testing.T) {
	result, err := askUserWorkGate(t.Context(), nil, "")
	require.NoError(t, err)
	assert.Contains(t, result, "Error")
	assert.Contains(t, result, "summary is required")
}

func TestAskUserQuestionToolsNotAllowed(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusWorking,
		InteractionMode: state.ModeSelfDrive,
	})
	require.NoError(t, err)

	result, errQ := askUserQuestion(t.Context(), coord, askUserQuestionArgs{
		Questions: []questionItem{{Question: "test?", Choices: []string{"yes"}}},
	})
	require.NoError(t, errQ)
	assert.Contains(t, result, "Error")
	assert.Contains(t, result, "not allowed")
}

func TestAskUserWorkGateToolsNotAllowed(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:          state.StatusWorking,
		InteractionMode: state.ModeSelfDrive,
	})
	require.NoError(t, err)

	result, errQ := askUserWorkGate(t.Context(), coord, "summary of work")
	require.NoError(t, errQ)
	assert.Contains(t, result, "Error")
	assert.Contains(t, result, "not allowed")
}

func TestMCPHandlerErrorPaths(t *testing.T) {
	t.Run("findSkillsHandlerError", func(t *testing.T) {
		ctx := &mcpContext{
			workingDir: "/nonexistent/path/12345",
			coord:      nil,
			agentName:  "test",
		}
		_, _, err := ctx.findSkillsHandler(context.Background(), nil, findSkillsArgs{Name: "exact-match"})
		assert.Error(t, err)
	})

	t.Run("findSnippetsHandlerNoError", func(t *testing.T) {
		ctx := &mcpContext{
			workingDir: "/nonexistent/path/12345",
			coord:      nil,
			agentName:  "test",
		}
		result, _, err := ctx.findSnippetsHandler(context.Background(), nil, findSnippetsArgs{Language: "go"})
		assert.NoError(t, err)
		require.NotNil(t, result)
	})
}

func newTestMCPContext(t *testing.T) (*mcpContext, string) {
	t.Helper()
	dir := t.TempDir()
	sgaiDir := filepath.Join(dir, ".sgai")
	require.NoError(t, os.MkdirAll(sgaiDir, 0o755))
	skillsDir := filepath.Join(sgaiDir, "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0o755))
	snippetsDir := filepath.Join(sgaiDir, "snippets")
	require.NoError(t, os.MkdirAll(snippetsDir, 0o755))

	statePath := filepath.Join(sgaiDir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(statePath, state.Workflow{
		Status:       state.StatusWorking,
		CurrentAgent: "test-agent",
		Messages:     []state.Message{},
		Progress:     []state.ProgressEntry{},
	})
	require.NoError(t, errCoord)

	ctx := &mcpContext{
		workingDir: dir,
		coord:      coord,
		dagAgents:  []string{"coordinator", "test-agent", "reviewer"},
		agentName:  "test-agent",
	}
	return ctx, dir
}

func TestFindSkillsHandlerSuccess(t *testing.T) {
	ctx, dir := newTestMCPContext(t)

	skillDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: A test skill\n---\nSkill content"), 0o644))

	result, _, err := ctx.findSkillsHandler(context.Background(), nil, findSkillsArgs{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Content, 1)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "test-skill")
}

func TestFindSkillsHandlerExactMatch(t *testing.T) {
	ctx, dir := newTestMCPContext(t)

	skillDir := filepath.Join(dir, ".sgai", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\ndescription: My skill\n---\nDetailed skill content"), 0o644))

	result, _, err := ctx.findSkillsHandler(context.Background(), nil, findSkillsArgs{Name: "my-skill"})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "my-skill")
}

func TestFindSnippetsHandlerListLanguages(t *testing.T) {
	ctx, dir := newTestMCPContext(t)

	goDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "hello.go"), []byte("// snippet: hello world\npackage main"), 0o644))

	result, _, err := ctx.findSnippetsHandler(context.Background(), nil, findSnippetsArgs{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "go")
}

func TestFindSnippetsHandlerWithLanguage(t *testing.T) {
	ctx, dir := newTestMCPContext(t)

	goDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(goDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(goDir, "example.go"), []byte("// snippet: example\npackage main"), 0o644))

	result, _, err := ctx.findSnippetsHandler(context.Background(), nil, findSnippetsArgs{Language: "go"})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "example")
}

func TestUpdateWorkflowStateHandlerSuccess(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.updateWorkflowStateHandler(context.Background(), nil, updateWorkflowStateArgs{
		Status:      "working",
		Task:        "test task",
		AddProgress: "doing something",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "State updated successfully")
}

func TestUpdateWorkflowStateHandlerInvalidStatus(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.updateWorkflowStateHandler(context.Background(), nil, updateWorkflowStateArgs{
		Status: "bogus-status",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Error")
}

func TestSendMessageHandlerSuccess(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.sendMessageHandler(context.Background(), nil, sendMessageArgs{
		ToAgent: "reviewer",
		Body:    "Please review this code",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Message sent")
}

func TestSendMessageHandlerInvalidRecipient(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.sendMessageHandler(context.Background(), nil, sendMessageArgs{
		ToAgent: "nonexistent-agent",
		Body:    "hello",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Error")
}

func TestCheckInboxHandlerEmpty(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.checkInboxHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "no messages")
}

func TestCheckInboxHandlerWithMessages(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	require.NoError(t, ctx.coord.UpdateState(func(wf *state.Workflow) {
		wf.Messages = append(wf.Messages, state.Message{
			ID:        1,
			FromAgent: "reviewer",
			ToAgent:   "test-agent",
			Body:      "Fix this bug",
			Read:      false,
		})
	}))

	result, _, err := ctx.checkInboxHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Fix this bug")
}

func TestCheckOutboxHandlerEmpty(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.checkOutboxHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "not sent any messages")
}

func TestCheckOutboxHandlerWithMessages(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	require.NoError(t, ctx.coord.UpdateState(func(wf *state.Workflow) {
		wf.Messages = append(wf.Messages, state.Message{
			ID:        1,
			FromAgent: "test-agent",
			ToAgent:   "reviewer",
			Body:      "Ready for review",
			Read:      false,
		})
	}))

	result, _, err := ctx.checkOutboxHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Ready for review")
}

func TestPeekMessageBusHandlerEmpty(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.peekMessageBusHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "No messages")
}

func TestPeekMessageBusHandlerWithMessages(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	require.NoError(t, ctx.coord.UpdateState(func(wf *state.Workflow) {
		wf.Messages = append(wf.Messages, state.Message{
			ID:        1,
			FromAgent: "reviewer",
			ToAgent:   "coordinator",
			Body:      "Code review complete",
			Read:      false,
		})
	}))

	result, _, err := ctx.peekMessageBusHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Code review complete")
}

func TestProjectTodoWriteHandlerSuccess(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	todos := []state.TodoItem{
		{Content: "Task 1", Status: "pending", Priority: "high"},
		{Content: "Task 2", Status: "completed", Priority: "low"},
	}

	result, _, err := ctx.projectTodoWriteHandler(context.Background(), nil, projectTodoWriteArgs{Todos: todos})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Task 1")
}

func TestProjectTodoReadHandlerEmpty(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.projectTodoReadHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestProjectTodoReadHandlerWithTodos(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	require.NoError(t, ctx.coord.UpdateState(func(wf *state.Workflow) {
		wf.ProjectTodos = []state.TodoItem{
			{Content: "Review PR", Status: "pending", Priority: "high"},
		}
	}))

	result, _, err := ctx.projectTodoReadHandler(context.Background(), nil, struct{}{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Review PR")
}

func TestAskUserQuestionHandlerNoCoord(t *testing.T) {
	ctx, _ := newTestMCPContext(t)
	ctx.coord = nil

	result, _, err := ctx.askUserQuestionHandler(context.Background(), nil, askUserQuestionArgs{})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Error")
}

func TestAskUserWorkGateHandlerNoCoord(t *testing.T) {
	ctx, _ := newTestMCPContext(t)
	ctx.coord = nil

	result, _, err := ctx.askUserWorkGateHandler(context.Background(), nil, askUserWorkGateArgs{Summary: "test"})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Error")
}

func TestAskUserWorkGateHandlerEmptySummary(t *testing.T) {
	ctx, _ := newTestMCPContext(t)

	result, _, err := ctx.askUserWorkGateHandler(context.Background(), nil, askUserWorkGateArgs{Summary: ""})
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "Error")
	assert.Contains(t, text, "summary is required")
}

func TestParseAgentIdentityHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "emptyHeader",
			header:   "",
			expected: "coordinator",
		},
		{
			name:     "simpleAgentName",
			header:   "backend-go-developer",
			expected: "backend-go-developer",
		},
		{
			name:     "agentWithModelAndVariant",
			header:   "backend-go-developer|anthropic/claude-opus-4-6|max",
			expected: "backend-go-developer",
		},
		{
			name:     "agentWithPipeSeparator",
			header:   "react-developer|opencode/model1|",
			expected: "react-developer",
		},
		{
			name:     "onlyPipes",
			header:   "||",
			expected: "coordinator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, "/test", nil)
			if tt.header != "" {
				r.Header.Set("X-Sgai-Agent-Identity", tt.header)
			}
			result := parseAgentIdentityHeader(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveCallerAgent(t *testing.T) {
	tests := []struct {
		name         string
		headerAgent  string
		currentAgent string
		expected     string
	}{
		{
			name:         "nonCoordinatorHeader",
			headerAgent:  "backend-go-developer",
			currentAgent: "react-developer",
			expected:     "backend-go-developer",
		},
		{
			name:         "coordinatorHeaderWithCurrentAgent",
			headerAgent:  "coordinator",
			currentAgent: "backend-go-developer",
			expected:     "backend-go-developer",
		},
		{
			name:         "coordinatorHeaderWithCoordinatorCurrent",
			headerAgent:  "coordinator",
			currentAgent: "coordinator",
			expected:     "coordinator",
		},
		{
			name:         "coordinatorHeaderWithEmptyCurrent",
			headerAgent:  "coordinator",
			currentAgent: "",
			expected:     "coordinator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
				CurrentAgent: tt.currentAgent,
			})
			require.NoError(t, err)
			result := resolveCallerAgent(tt.headerAgent, coord)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type messageMatchTest struct {
	name         string
	agentField   string
	currentAgent string
	currentModel string
	expected     bool
}

func messageMatchCases() []messageMatchTest {
	return []messageMatchTest{
		{
			name:         "matchesAgent",
			agentField:   "backend-go-developer",
			currentAgent: "backend-go-developer",
			currentModel: "",
			expected:     true,
		},
		{
			name:         "matchesModel",
			agentField:   "opencode/glm-5",
			currentAgent: "backend-go-developer",
			currentModel: "opencode/glm-5",
			expected:     true,
		},
		{
			name:         "noMatch",
			agentField:   "react-developer",
			currentAgent: "backend-go-developer",
			currentModel: "",
			expected:     false,
		},
		{
			name:         "emptyAgentField",
			agentField:   "",
			currentAgent: "backend-go-developer",
			currentModel: "",
			expected:     false,
		},
		{
			name:         "emptyCurrentAgent",
			agentField:   "backend-go-developer",
			currentAgent: "",
			currentModel: "",
			expected:     false,
		},
		{
			name:         "modelMatchWithEmptyModel",
			agentField:   "opencode/glm-5",
			currentAgent: "backend-go-developer",
			currentModel: "",
			expected:     false,
		},
	}
}

func TestMessageMatchesRecipient(t *testing.T) {
	for _, tt := range messageMatchCases() {
		t.Run(tt.name, func(t *testing.T) {
			msg := state.Message{ToAgent: tt.agentField}
			result := messageMatchesRecipient(msg, tt.currentAgent, tt.currentModel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageMatchesSender(t *testing.T) {
	for _, tt := range messageMatchCases() {
		t.Run(tt.name, func(t *testing.T) {
			msg := state.Message{FromAgent: tt.agentField}
			result := messageMatchesSender(msg, tt.currentAgent, tt.currentModel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildUpdateWorkflowStateSchema(t *testing.T) {
	t.Run("coordinatorAgent", func(t *testing.T) {
		schema, desc := buildUpdateWorkflowStateSchema("coordinator")
		assert.NotNil(t, schema)
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "workflow state")
		statusProp := schema.Properties["status"]
		assert.NotNil(t, statusProp)
		assert.Len(t, statusProp.Enum, 3)
	})

	t.Run("nonCoordinatorAgent", func(t *testing.T) {
		schema, desc := buildUpdateWorkflowStateSchema("backend-developer")
		assert.NotNil(t, schema)
		assert.NotEmpty(t, desc)
		statusProp := schema.Properties["status"]
		assert.NotNil(t, statusProp)
		assert.Len(t, statusProp.Enum, 2)
	})
}

func TestMustSchema(t *testing.T) {
	schema := mustSchema[findSkillsArgs]()
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
}

func TestSkillDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]string
		relName     string
		expected    string
	}{
		{
			name:        "withName",
			frontmatter: map[string]string{"name": "My Skill"},
			relName:     "skills/my-skill",
			expected:    "My Skill",
		},
		{
			name:        "withoutName",
			frontmatter: map[string]string{},
			relName:     "skills/my-skill",
			expected:    "my-skill",
		},
		{
			name:        "emptyName",
			frontmatter: map[string]string{"name": ""},
			relName:     "skills/my-skill",
			expected:    "my-skill",
		},
		{
			name:        "nestedPath",
			frontmatter: map[string]string{},
			relName:     "skills/category/my-skill",
			expected:    "my-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := skillDisplayName(tt.frontmatter, tt.relName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSkillSteeringMessage(t *testing.T) {
	name := "my-skill"
	desc := "A useful skill for testing"

	result := skillSteeringMessage(name, desc)

	assert.Contains(t, result, "Found skill 'my-skill'")
	assert.Contains(t, result, "A useful skill for testing")
	assert.Contains(t, result, `skill({"name":"my-skill"})`)
}

func TestSkillRelName(t *testing.T) {
	tests := []struct {
		name      string
		skillsDir string
		file      string
		expected  string
	}{
		{
			name:      "simpleSkill",
			skillsDir: "/path/to/.sgai/skills",
			file:      "/path/to/.sgai/skills/my-skill/SKILL.md",
			expected:  "my-skill",
		},
		{
			name:      "nestedSkill",
			skillsDir: "/path/to/.sgai/skills",
			file:      "/path/to/.sgai/skills/category/my-skill/SKILL.md",
			expected:  "category/my-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := skillRelName(tt.skillsDir, tt.file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSkillDesc(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]string
		expected    string
	}{
		{
			name:        "withDescription",
			frontmatter: map[string]string{"description": "A useful skill"},
			expected:    "A useful skill",
		},
		{
			name:        "withoutDescription",
			frontmatter: map[string]string{},
			expected:    "No description",
		},
		{
			name:        "emptyDescription",
			frontmatter: map[string]string{"description": ""},
			expected:    "No description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := skillDesc(tt.frontmatter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateWorkflowState(t *testing.T) {
	tests := []struct {
		name           string
		initialState   state.Workflow
		callerAgent    string
		args           updateWorkflowStateArgs
		wantErr        bool
		wantContains   string
		wantNotContain string
	}{
		{
			name:         "nilCoordinator",
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{},
			wantContains: "Error: workflow coordinator not available.",
		},
		{
			name: "setWorkingStatus",
			initialState: state.Workflow{
				Status:   state.StatusWorking,
				Progress: []state.ProgressEntry{},
			},
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{Status: "working", Task: "doing stuff", AddProgress: "started work"},
			wantContains: "State updated successfully.",
		},
		{
			name: "invalidStatus",
			initialState: state.Workflow{
				Status:   state.StatusWorking,
				Progress: []state.ProgressEntry{},
			},
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{Status: "invalid-status"},
			wantContains: "Error: Invalid status",
		},
		{
			name: "agentDoneWithPendingTodos",
			initialState: state.Workflow{
				Status:       state.StatusWorking,
				CurrentAgent: "test-agent",
				Progress:     []state.ProgressEntry{},
				Todos: []state.TodoItem{
					{Content: "pending task", Status: "pending", Priority: "high"},
				},
			},
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{Status: "agent-done"},
			wantContains: "Error: Cannot transition to 'agent-done'",
		},
		{
			name: "agentDoneClearsTask",
			initialState: state.Workflow{
				Status:       state.StatusWorking,
				CurrentAgent: "test-agent",
				Progress:     []state.ProgressEntry{},
			},
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{Status: "agent-done", Task: "some task"},
			wantContains: "State updated successfully.",
		},
		{
			name: "preserveHumanPendingStatus",
			initialState: state.Workflow{
				Status:   state.StatusWaitingForHuman,
				Progress: []state.ProgressEntry{},
			},
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{Status: "working", Task: "my task", AddProgress: "progress note"},
			wantContains: "Waiting for human response",
		},
		{
			name: "addProgressNote",
			initialState: state.Workflow{
				Status:   state.StatusWorking,
				Progress: []state.ProgressEntry{},
			},
			callerAgent:  "test-agent",
			args:         updateWorkflowStateArgs{AddProgress: "completed step 1"},
			wantContains: "Added progress note: completed step 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nilCoordinator" {
				result, err := updateWorkflowState(nil, tt.callerAgent, tt.args)
				require.NoError(t, err)
				assert.Contains(t, result, tt.wantContains)
				return
			}

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			coord, err := state.NewCoordinatorWith(statePath, tt.initialState)
			require.NoError(t, err)

			result, err := updateWorkflowState(coord, tt.callerAgent, tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

func TestSendMessage(t *testing.T) {
	dagAgents := []string{"coordinator", "backend-go-developer", "react-developer"}

	tests := []struct {
		name         string
		callerAgent  string
		toAgent      string
		body         string
		wantContains string
	}{
		{
			name:         "nilCoordinator",
			callerAgent:  "test-agent",
			toAgent:      "coordinator",
			body:         "hello",
			wantContains: "Error: Could not read state.json",
		},
		{
			name:         "invalidTargetAgent",
			callerAgent:  "coordinator",
			toAgent:      "non-existent-agent",
			body:         "hello",
			wantContains: "Error: Agent 'non-existent-agent' is not in the workflow",
		},
		{
			name:         "sendFromCoordinator",
			callerAgent:  "coordinator",
			toAgent:      "backend-go-developer",
			body:         "please review this",
			wantContains: "Message sent successfully",
		},
		{
			name:         "sendFromNonCoordinator",
			callerAgent:  "backend-go-developer",
			toAgent:      "coordinator",
			body:         "done with review",
			wantContains: "IMPORTANT: To receive a response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nilCoordinator" {
				result, err := sendMessage(nil, dagAgents, tt.callerAgent, tt.toAgent, tt.body)
				require.NoError(t, err)
				assert.Contains(t, result, tt.wantContains)
				return
			}

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
				Status:   state.StatusWorking,
				Messages: []state.Message{},
			})
			require.NoError(t, err)

			result, err := sendMessage(coord, dagAgents, tt.callerAgent, tt.toAgent, tt.body)
			require.NoError(t, err)
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

func TestCheckInbox(t *testing.T) {
	tests := []struct {
		name         string
		callerAgent  string
		messages     []state.Message
		wantContains string
	}{
		{
			name:         "nilCoordinator",
			callerAgent:  "test-agent",
			wantContains: "Error: Could not read state.json",
		},
		{
			name:        "noMessages",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "coordinator", ToAgent: "other-agent", Body: "hello", Read: false},
			},
			wantContains: "You have no messages.",
		},
		{
			name:        "hasUnreadMessages",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "coordinator", ToAgent: "test-agent", Body: "please do this", Read: false},
				{ID: 2, FromAgent: "coordinator", ToAgent: "test-agent", Body: "also this", Read: false},
			},
			wantContains: "You have 2 message(s):",
		},
		{
			name:        "onlyReadMessages",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "coordinator", ToAgent: "test-agent", Body: "old", Read: true},
			},
			wantContains: "You have no messages.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nilCoordinator" {
				result, err := checkInbox(nil, tt.callerAgent)
				require.NoError(t, err)
				assert.Contains(t, result, tt.wantContains)
				return
			}

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
				Status:   state.StatusWorking,
				Messages: tt.messages,
			})
			require.NoError(t, err)

			result, err := checkInbox(coord, tt.callerAgent)
			require.NoError(t, err)
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

func TestCheckOutbox(t *testing.T) {
	tests := []struct {
		name         string
		callerAgent  string
		messages     []state.Message
		wantContains string
	}{
		{
			name:         "nilCoordinator",
			callerAgent:  "test-agent",
			wantContains: "Error: Could not read state.json",
		},
		{
			name:        "noSentMessages",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "other-agent", ToAgent: "test-agent", Body: "hello"},
			},
			wantContains: "You have not sent any messages.",
		},
		{
			name:        "hasPendingMessages",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "test-agent", ToAgent: "coordinator", Body: "done with work", Read: false},
			},
			wantContains: "Pending messages (1):",
		},
		{
			name:        "hasDeliveredMessages",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "test-agent", ToAgent: "coordinator", Body: "done with work", Read: true, ReadAt: "2026-03-05T10:00:00Z"},
			},
			wantContains: "Delivered messages (1):",
		},
		{
			name:        "mixedPendingAndDelivered",
			callerAgent: "test-agent",
			messages: []state.Message{
				{ID: 1, FromAgent: "test-agent", ToAgent: "coordinator", Body: "first msg", Read: true, ReadAt: "2026-03-05T10:00:00Z"},
				{ID: 2, FromAgent: "test-agent", ToAgent: "reviewer", Body: "review request", Read: false},
			},
			wantContains: "Pending messages (1):",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nilCoordinator" {
				result, err := checkOutbox(nil, tt.callerAgent)
				require.NoError(t, err)
				assert.Contains(t, result, tt.wantContains)
				return
			}

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
				Status:   state.StatusWorking,
				Messages: tt.messages,
			})
			require.NoError(t, err)

			result, err := checkOutbox(coord, tt.callerAgent)
			require.NoError(t, err)
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

func TestPeekMessageBus(t *testing.T) {
	tests := []struct {
		name         string
		messages     []state.Message
		wantContains string
	}{
		{
			name:         "nilCoordinator",
			wantContains: "Error: Could not read state.json",
		},
		{
			name:         "noMessages",
			messages:     []state.Message{},
			wantContains: "No messages in the system.",
		},
		{
			name: "hasMessages",
			messages: []state.Message{
				{ID: 1, FromAgent: "coordinator", ToAgent: "backend", Body: "do work", Read: false},
				{ID: 2, FromAgent: "backend", ToAgent: "coordinator", Body: "done", Read: true, ReadAt: "2026-03-05T10:00:00Z"},
			},
			wantContains: "Total messages: 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nilCoordinator" {
				result, err := peekMessageBus(nil)
				require.NoError(t, err)
				assert.Contains(t, result, tt.wantContains)
				return
			}

			tmpDir := t.TempDir()
			statePath := filepath.Join(tmpDir, "state.json")
			coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
				Status:   state.StatusWorking,
				Messages: tt.messages,
			})
			require.NoError(t, err)

			result, err := peekMessageBus(coord)
			require.NoError(t, err)
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

func TestFormatTodoList(t *testing.T) {
	tests := []struct {
		name     string
		todos    []state.TodoItem
		contains []string
	}{
		{
			name:     "emptyList",
			todos:    []state.TodoItem{},
			contains: []string{"0 todos"},
		},
		{
			name: "mixedStatuses",
			todos: []state.TodoItem{
				{Content: "pending task", Status: "pending", Priority: "high"},
				{Content: "in progress task", Status: "in_progress", Priority: "medium"},
				{Content: "done task", Status: "completed", Priority: "low"},
				{Content: "cancelled task", Status: "cancelled", Priority: "low"},
			},
			contains: []string{"3 todos", "○ pending task", "◐ in progress task", "● done task", "✕ cancelled task"},
		},
		{
			name: "allPending",
			todos: []state.TodoItem{
				{Content: "task a", Status: "pending", Priority: "high"},
				{Content: "task b", Status: "pending", Priority: "medium"},
			},
			contains: []string{"2 todos"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTodoList(tt.todos)
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
		})
	}
}

func TestProjectTodoWrite(t *testing.T) {
	t.Run("nilCoordinator", func(t *testing.T) {
		result, err := projectTodoWrite(nil, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Error: workflow coordinator not available.")
	})

	t.Run("writeTodos", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "state.json")
		coord, err := state.NewCoordinatorWith(statePath, state.Workflow{Status: state.StatusWorking})
		require.NoError(t, err)

		todos := []state.TodoItem{
			{Content: "first task", Status: "pending", Priority: "high"},
		}

		result, err := projectTodoWrite(coord, todos)
		require.NoError(t, err)
		assert.Contains(t, result, "first task")

		snapshot := coord.State()
		assert.Len(t, snapshot.ProjectTodos, 1)
	})
}

func TestProjectTodoRead(t *testing.T) {
	t.Run("nilCoordinator", func(t *testing.T) {
		result, err := projectTodoRead(nil)
		require.NoError(t, err)
		assert.Equal(t, "0 todos", result)
	})

	t.Run("withTodos", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "state.json")
		coord, err := state.NewCoordinatorWith(statePath, state.Workflow{
			Status: state.StatusWorking,
			ProjectTodos: []state.TodoItem{
				{Content: "my task", Status: "pending", Priority: "high"},
			},
		})
		require.NoError(t, err)

		result, err := projectTodoRead(coord)
		require.NoError(t, err)
		assert.Contains(t, result, "my task")
	})
}

func TestFindSkills(t *testing.T) {
	tests := []struct {
		name       string
		skillName  string
		setup      func(t *testing.T, dir string)
		wantErr    bool
		assertFunc func(t *testing.T, result string)
	}{
		{
			name:      "listAllSkills",
			skillName: "",
			setup: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				content := "---\nname: test-skill\ndescription: A test skill\n---\n# Test Skill"
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "test-skill")
				assert.Contains(t, result, "A test skill")
			},
		},
		{
			name:      "findByExactMatch",
			skillName: "test-skill",
			setup: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				content := "---\nname: test-skill\ndescription: A test skill\n---\n# Test Skill"
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "Found skill 'test-skill'")
			},
		},
		{
			name:      "findByPrefix",
			skillName: "coding-practices",
			setup: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "coding-practices", "go-review")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				content := "---\nname: go-review\ndescription: Go code review\n---\n# Go Review"
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "go-review")
			},
		},
		{
			name:      "findByBasename",
			skillName: "go-review",
			setup: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "coding-practices", "go-review")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				content := "---\nname: go-review\ndescription: Go code review\n---\n# Go Review"
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "Found skill 'go-review'")
			},
		},
		{
			name:      "findByFuzzyMatch",
			skillName: "review",
			setup: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "my-review-skill")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				content := "---\nname: my-review-skill\ndescription: review stuff\n---\n# Review"
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "my-review-skill")
			},
		},
		{
			name:      "noMatchReturnsEmpty",
			skillName: "nonexistent-skill-xyz",
			setup: func(t *testing.T, dir string) {
				skillDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
				require.NoError(t, os.MkdirAll(skillDir, 0755))
				content := "---\nname: test-skill\ndescription: something else entirely\n---\n# Test"
				require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
		{
			name:      "noSkillsDirectory",
			skillName: "",
			setup:     func(_ *testing.T, _ string) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			result, err := findSkills(tmpDir, tt.skillName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.assertFunc != nil {
				tt.assertFunc(t, result)
			}
		})
	}
}

func TestFindSnippets(t *testing.T) {
	tests := []struct {
		name       string
		language   string
		query      string
		setup      func(t *testing.T, dir string)
		assertFunc func(t *testing.T, result string)
	}{
		{
			name:     "listLanguages",
			language: "",
			query:    "",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "snippets", "go"), 0755))
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "snippets", "python"), 0755))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "go")
				assert.Contains(t, result, "python")
			},
		},
		{
			name:     "listSnippetsForLanguage",
			language: "go",
			query:    "",
			setup: func(t *testing.T, dir string) {
				langDir := filepath.Join(dir, ".sgai", "snippets", "go")
				require.NoError(t, os.MkdirAll(langDir, 0755))
				content := "---\ndescription: HTTP server snippet\n---\npackage main\n"
				require.NoError(t, os.WriteFile(filepath.Join(langDir, "http-server.go"), []byte(content), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "http-server")
				assert.Contains(t, result, "HTTP server snippet")
			},
		},
		{
			name:     "searchSnippets",
			language: "go",
			query:    "http",
			setup: func(t *testing.T, dir string) {
				langDir := filepath.Join(dir, ".sgai", "snippets", "go")
				require.NoError(t, os.MkdirAll(langDir, 0755))
				content := "---\ndescription: HTTP server pattern\n---\npackage main\n"
				require.NoError(t, os.WriteFile(filepath.Join(langDir, "http-server.go"), []byte(content), 0644))
				content2 := "---\ndescription: JSON encoding\n---\npackage main\n"
				require.NoError(t, os.WriteFile(filepath.Join(langDir, "json-encode.go"), []byte(content2), 0644))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "HTTP server pattern")
			},
		},
		{
			name:     "nonExistentLanguage",
			language: "cobol",
			query:    "",
			setup: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "snippets"), 0755))
			},
			assertFunc: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			result, err := findSnippets(tmpDir, tt.language, tt.query)
			require.NoError(t, err)
			if tt.assertFunc != nil {
				tt.assertFunc(t, result)
			}
		})
	}
}

func TestStartMCPHTTPServer(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	url, closeFn, err := startMCPHTTPServer(t.TempDir(), coord, []string{"builder"})
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.NotNil(t, closeFn)
	t.Cleanup(closeFn)
	assert.Contains(t, url, "http://127.0.0.1:")
	assert.Contains(t, url, "/mcp")
}

func TestBuildMCPHTTPHandler(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	handler := buildMCPHTTPHandler(t.TempDir(), coord, []string{"builder"})
	assert.NotNil(t, handler)
}

func TestBuildMCPServerInternal(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	r, _ := http.NewRequest("GET", "/", nil)
	server := buildMCPServer(t.TempDir(), r, coord, []string{"builder"})
	assert.NotNil(t, server)
}

func TestBuildMCPServerWithAgentHeader(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Sgai-Agent-Identity", "builder|")
	server := buildMCPServer(t.TempDir(), r, coord, []string{"builder"})
	assert.NotNil(t, server)
}

func TestRegisterCommonToolsInternal(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	mcpCtx := &mcpContext{workingDir: t.TempDir(), coord: coord, dagAgents: []string{"builder"}, agentName: "builder"}
	registerCommonTools(server, mcpCtx, "builder")
	assert.NotNil(t, server)
}

func TestRegisterCoordinatorToolsInternal(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	mcpCtx := &mcpContext{workingDir: t.TempDir(), coord: coord, dagAgents: []string{"coordinator"}, agentName: "coordinator"}
	registerCoordinatorTools(server, mcpCtx, t.TempDir())
	assert.NotNil(t, server)
}

func TestRegisterCoordinatorToolsBrainstormingMode(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{InteractionMode: state.ModeBrainstorming})
	require.NoError(t, errCoord)
	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	mcpCtx := &mcpContext{workingDir: t.TempDir(), coord: coord, dagAgents: []string{"coordinator"}, agentName: "coordinator"}
	registerCoordinatorTools(server, mcpCtx, t.TempDir())
	assert.NotNil(t, server)
}

func TestAskUserQuestionSelfDriveMode(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{InteractionMode: state.ModeSelfDrive})
	require.NoError(t, errCoord)
	args := askUserQuestionArgs{
		Questions: []questionItem{{Question: "test?", Choices: []string{"yes", "no"}}},
	}
	result, err := askUserQuestion(context.Background(), coord, args)
	require.NoError(t, err)
	assert.Contains(t, result, "not allowed")
}

func TestAskUserWorkGateSelfDriveMode(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{InteractionMode: state.ModeSelfDrive})
	require.NoError(t, errCoord)
	result, err := askUserWorkGate(context.Background(), coord, "test summary")
	require.NoError(t, err)
	assert.Contains(t, result, "not allowed")
}

func TestAskUserQuestionNilCoordinator(t *testing.T) {
	args := askUserQuestionArgs{
		Questions: []questionItem{{Question: "test?", Choices: []string{"yes", "no"}}},
	}
	result, err := askUserQuestion(context.Background(), nil, args)
	require.NoError(t, err)
	assert.Contains(t, result, "not allowed")
}

func TestAskUserQuestionEmptyQuestionList(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{InteractionMode: state.ModeBrainstorming})
	require.NoError(t, errCoord)
	args := askUserQuestionArgs{Questions: nil}
	result, err := askUserQuestion(context.Background(), coord, args)
	require.NoError(t, err)
	assert.Contains(t, result, "At least one question is required")
}

func TestAskUserQuestionNoChoices(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{InteractionMode: state.ModeBrainstorming})
	require.NoError(t, errCoord)
	args := askUserQuestionArgs{
		Questions: []questionItem{{Question: "test?", Choices: nil}},
	}
	result, err := askUserQuestion(context.Background(), coord, args)
	require.NoError(t, err)
	assert.Contains(t, result, "has no choices")
}

func TestAskUserWorkGateBlankSummary(t *testing.T) {
	result, err := askUserWorkGate(context.Background(), nil, "")
	require.NoError(t, err)
	assert.Contains(t, result, "summary is required")
}

func TestAskUserWorkGateNilCoordinator(t *testing.T) {
	result, err := askUserWorkGate(context.Background(), nil, "my summary")
	require.NoError(t, err)
	assert.Contains(t, result, "not allowed")
}

func TestBuildMCPHTTPHandlerCreation(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	coord, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	handler := buildMCPHTTPHandler(dir, coord, []string{"coordinator"})
	assert.NotNil(t, handler)
}

func TestSearchSnippetsNoEntries(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(langDir, 0755))
	entries, _ := os.ReadDir(langDir)
	result, err := searchSnippets(langDir, entries, "test")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestSearchSnippetsWithEntries(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(langDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(langDir, "hello.go"), []byte("---\ndescription: Hello World\n---\npackage main"), 0644))
	entries, _ := os.ReadDir(langDir)
	result, err := searchSnippets(langDir, entries, "hello")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestSearchSnippetsNoMatch(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(langDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(langDir, "hello.go"), []byte("---\ndescription: Hello World\n---\npackage main"), 0644))
	entries, _ := os.ReadDir(langDir)
	result, err := searchSnippets(langDir, entries, "nonexistent-xyz")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFindSnippetsByNameContainsMatch(t *testing.T) {
	dir := t.TempDir()
	snippetDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(snippetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "http-handler.go"), []byte("---\ndescription: HTTP Handler\n---\npackage main"), 0644))
	snippetFiles, err := os.ReadDir(snippetDir)
	require.NoError(t, err)
	result := findSnippetsByNameContains(snippetDir, snippetFiles, "http")
	assert.NotEmpty(t, result)
}

func TestFindSnippetsByNameContainsNoMatch(t *testing.T) {
	dir := t.TempDir()
	snippetDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(snippetDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetDir, "hello.go"), []byte("---\ndescription: Hello\n---\npackage main"), 0644))
	snippetFiles, err := os.ReadDir(snippetDir)
	require.NoError(t, err)
	result := findSnippetsByNameContains(snippetDir, snippetFiles, "nonexistent")
	assert.Empty(t, result)
}

func TestAskUserQuestionWithValidCoordinator(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		InteractionMode: state.ModeBrainstorming,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			wf := coord.State()
			if wf.NeedsHumanInput() {
				coord.Respond("my answer")
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	t.Cleanup(cancel)

	result, err := askUserQuestion(ctx, coord, askUserQuestionArgs{
		Questions: []questionItem{{Question: "What color?", Choices: []string{"red", "blue"}}},
	})
	require.NoError(t, err)
	assert.Contains(t, result, "What color?")
	assert.Contains(t, result, "my answer")
}

func TestAskUserWorkGateWithValidCoordinator(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		InteractionMode: state.ModeBrainstorming,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			wf := coord.State()
			if wf.NeedsHumanInput() {
				coord.Respond("DEFINITION IS COMPLETE, BUILD MAY BEGIN")
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	t.Cleanup(cancel)

	result, err := askUserWorkGate(ctx, coord, "here is my summary")
	require.NoError(t, err)
	assert.Contains(t, result, "here is my summary")
	assert.Contains(t, result, "DEFINITION IS COMPLETE")
}

func TestFindSnippetsNoLanguage(t *testing.T) {
	dir := t.TempDir()
	snippetsDir := filepath.Join(dir, ".sgai", "snippets")
	require.NoError(t, os.MkdirAll(filepath.Join(snippetsDir, "go"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(snippetsDir, "python"), 0755))
	result, err := findSnippets(dir, "", "")
	require.NoError(t, err)
	assert.Contains(t, result, "go")
	assert.Contains(t, result, "python")
}

func TestFindSnippetsWithLanguageNoQuery(t *testing.T) {
	dir := t.TempDir()
	snippetsDir := filepath.Join(dir, ".sgai", "snippets", "go")
	require.NoError(t, os.MkdirAll(snippetsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(snippetsDir, "http-server.go"), []byte("---\ndescription: HTTP server\n---\npackage main"), 0644))
	result, err := findSnippets(dir, "go", "")
	require.NoError(t, err)
	assert.Contains(t, result, "http-server")
}

func TestFindSnippetsNonexistentLanguage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai", "snippets"), 0755))
	result, err := findSnippets(dir, "rust", "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFindSkillsEmptyName(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: Test Skill\ndescription: A test skill\n---\n# Test Skill"), 0644))
	result, err := findSkills(dir, "")
	require.NoError(t, err)
	assert.Contains(t, result, "Test Skill")
}

func TestFindSkillsExactMatch(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".sgai", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: Test Skill\ndescription: A test skill\n---\n# Test Skill"), 0644))
	result, err := findSkills(dir, "test-skill")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestFindSkillsByBasenameSingleMatch(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".sgai", "skills", "category", "my-skill")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("---\nname: My Skill\ndescription: A great skill\n---\n# My Skill content"), 0644))
	skillFiles, err := collectSkillFiles(filepath.Join(dir, ".sgai", "skills"))
	require.NoError(t, err)
	result := findSkillsByBasename(filepath.Join(dir, ".sgai", "skills"), skillFiles, "my-skill")
	assert.NotEmpty(t, result)
}

func TestFindSkillsByBasenameMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	skillsDir1 := filepath.Join(dir, ".sgai", "skills", "cat1", "shared-name")
	skillsDir2 := filepath.Join(dir, ".sgai", "skills", "cat2", "shared-name")
	require.NoError(t, os.MkdirAll(skillsDir1, 0755))
	require.NoError(t, os.MkdirAll(skillsDir2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir1, "SKILL.md"), []byte("---\nname: Skill A\ndescription: First\n---\n# A"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir2, "SKILL.md"), []byte("---\nname: Skill B\ndescription: Second\n---\n# B"), 0644))
	skillFiles, err := collectSkillFiles(filepath.Join(dir, ".sgai", "skills"))
	require.NoError(t, err)
	result := findSkillsByBasename(filepath.Join(dir, ".sgai", "skills"), skillFiles, "shared-name")
	assert.Contains(t, result, "Skill A")
	assert.Contains(t, result, "Skill B")
}

func TestFindSnippetsByNameContainsMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "http-server.go"), []byte("---\ndescription: HTTP server\n---\npackage main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "http-client.go"), []byte("---\ndescription: HTTP client\n---\npackage main"), 0644))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	result := findSnippetsByNameContains(dir, entries, "http")
	assert.Contains(t, result, "http-server")
	assert.Contains(t, result, "http-client")
}

func TestFindSnippetsByNameContainsSingleMatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unique-snippet.go"), []byte("---\ndescription: Unique\n---\npackage main\nfunc unique() {}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.go"), []byte("---\ndescription: Other\n---\npackage main"), 0644))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	result := findSnippetsByNameContains(dir, entries, "unique")
	assert.Contains(t, result, "func unique()")
}

func TestFindSnippetsByNameContainsNoDescription(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "http-handler.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "http-router.go"), []byte("package main"), 0644))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	result := findSnippetsByNameContains(dir, entries, "http")
	assert.Contains(t, result, "No description")
}

func TestUpdateWorkflowStateInvalidStatus(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, err)
	result, err := updateWorkflowState(coord, "builder", updateWorkflowStateArgs{
		Status: "invalid-status",
		Task:   "test task",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Invalid status")
}

func TestUpdateWorkflowStateWithPendingTodos(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		CurrentAgent: "builder",
		Todos: []state.TodoItem{
			{Content: "unfinished task", Status: "pending", Priority: "high"},
		},
	})
	require.NoError(t, err)
	result, err := updateWorkflowState(coord, "builder", updateWorkflowStateArgs{
		Status: "agent-done",
		Task:   "",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "pending TODO")
}

func TestUpdateWorkflowStatePreservesHumanPendingStatus(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		Status:       state.StatusWaitingForHuman,
		HumanMessage: "waiting",
	})
	require.NoError(t, err)
	result, err := updateWorkflowState(coord, "builder", updateWorkflowStateArgs{
		Status:      "working",
		Task:        "new task",
		AddProgress: "doing stuff",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "preserved")
}

func TestUpdateWorkflowStateClearsTaskOnComplete(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{
		Status: state.StatusWorking,
	})
	require.NoError(t, err)
	result, err := updateWorkflowState(coord, "builder", updateWorkflowStateArgs{
		Status: "agent-done",
		Task:   "should be cleared",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "updated successfully")
	wf := coord.State()
	assert.Empty(t, wf.Task)
}

func TestSendMessageInvalidAgent(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, err)
	result, err := sendMessage(coord, []string{"coordinator", "builder"}, "builder", "nonexistent-agent", "hello")
	require.NoError(t, err)
	assert.Contains(t, result, "not in the workflow")
}

func TestSendMessageValidAgent(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	coord, err := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, err)
	result, err := sendMessage(coord, []string{"coordinator", "builder"}, "builder", "coordinator", "hello from builder")
	require.NoError(t, err)
	assert.Contains(t, result, "sent")
}

func TestListSnippetLanguagesNoDir(t *testing.T) {
	result, err := listSnippetLanguages(filepath.Join(t.TempDir(), "nonexistent"))
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListSnippetsForLanguageWithDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.go"), []byte("---\ndescription: Test snippet\n---\npackage main"), 0644))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	result, err := listSnippetsForLanguage(dir, entries)
	require.NoError(t, err)
	assert.Contains(t, result, "test")
	assert.Contains(t, result, "Test snippet")
}

func TestBuildMCPHTTPHandlerWithHandler(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sgai"), 0755))
	stateFile := filepath.Join(dir, ".sgai", "state.json")
	_, errCoord := state.NewCoordinatorWith(stateFile, state.Workflow{})
	require.NoError(t, errCoord)
	srv, _ := setupTestServer(t)
	srv.rootDir = dir
	handler := buildMCPHTTPHandler(dir, srv.workspaceCoordinator(dir), []string{"coordinator"})
	assert.NotNil(t, handler)
	assert.Implements(t, (*http.Handler)(nil), handler)
}
