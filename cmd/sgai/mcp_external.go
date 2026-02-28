package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type externalMCPContext struct {
	srv *Server
}

func buildExternalMCPHandler(srv *Server) http.Handler {
	ctx := &externalMCPContext{srv: srv}
	return mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return buildExternalMCPServer(ctx)
	}, nil)
}

func buildExternalMCPServer(ctx *externalMCPContext) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "sgai-external"}, nil)
	registerExternalTools(server, ctx)
	return server
}

func registerExternalTools(server *mcp.Server, ctx *externalMCPContext) {
	registerStateTools(server, ctx)
	registerWorkspaceTools(server, ctx)
	registerSessionTools(server, ctx)
	registerKnowledgeTools(server, ctx)
	registerComposeTools(server, ctx)
	registerAdhocTools(server, ctx)
	registerEditorTools(server, ctx)
	registerModelTools(server, ctx)
	registerElicitationTool(server, ctx)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		return nil, fmt.Errorf("failed to encode result: %w", errMarshal)
	}
	return textResult(string(data)), nil
}

type emptyExternalResult struct{}

func (ctx *externalMCPContext) resolveWorkspacePath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("workspace name is required")
	}
	path := ctx.srv.resolveWorkspaceNameToPath(name)
	if path == "" {
		return "", fmt.Errorf("workspace not found: %s", name)
	}
	return path, nil
}

func (ctx *externalMCPContext) resolveAnyWorkspacePath(name string) (string, error) {
	if name != "" {
		path := ctx.srv.resolveWorkspaceNameToPath(name)
		if path != "" {
			return path, nil
		}
	}
	groups, errScan := ctx.srv.scanWorkspaceGroups()
	if errScan != nil || len(groups) == 0 {
		return "", fmt.Errorf("no workspaces found")
	}
	return groups[0].Root.Directory, nil
}

func registerStateTools(server *mcp.Server, ctx *externalMCPContext) {
	type listWorkspacesArgs struct{}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_workspaces",
		Description: "List all workspaces and their current status.",
		InputSchema: mustSchema[listWorkspacesArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ listWorkspacesArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		state := ctx.srv.buildFullFactoryState()
		result, err := jsonResult(state)
		return result, emptyExternalResult{}, err
	})

	type getWorkspaceStateArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name to get state for"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workspace_state",
		Description: "Get detailed state for a specific workspace.",
		InputSchema: mustSchema[getWorkspaceStateArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getWorkspaceStateArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		stateResult, err := ctx.srv.getWorkspaceStateService(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		if !stateResult.Found {
			return textResult(fmt.Sprintf("workspace not found: %s", args.Workspace)), emptyExternalResult{}, nil
		}
		result, err := jsonResult(stateResult.Workspace)
		return result, emptyExternalResult{}, err
	})

	type getWorkflowSVGArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workflow_svg",
		Description: "Get the workflow diagram as SVG for a workspace.",
		InputSchema: mustSchema[getWorkflowSVGArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getWorkflowSVGArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		svg := ctx.srv.getWorkflowSVGService(workspacePath)
		if svg == "" {
			return textResult("workflow SVG not available"), emptyExternalResult{}, nil
		}
		return textResult(svg), emptyExternalResult{}, nil
	})

	type getWorkspaceDiffArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workspace_diff",
		Description: "Get the current git diff for a workspace.",
		InputSchema: mustSchema[getWorkspaceDiffArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getWorkspaceDiffArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		diffResult := ctx.srv.workspaceDiffService(workspacePath)
		result, err := jsonResult(diffResult)
		return result, emptyExternalResult{}, err
	})
}

func registerWorkspaceTools(server *mcp.Server, ctx *externalMCPContext) {
	type createWorkspaceArgs struct {
		Name string `json:"name" jsonschema:"The workspace name (lowercase letters, numbers, dashes only)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_workspace",
		Description: "Create a new workspace with the given name.",
		InputSchema: mustSchema[createWorkspaceArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args createWorkspaceArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		createResult, err := ctx.srv.createWorkspaceService(args.Name)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(createResult)
		return result, emptyExternalResult{}, err
	})

	type forkWorkspaceArgs struct {
		Workspace string `json:"workspace" jsonschema:"The parent workspace name to fork from"`
		Name      string `json:"name" jsonschema:"The fork name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fork_workspace",
		Description: "Create a fork of an existing workspace.",
		InputSchema: mustSchema[forkWorkspaceArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args forkWorkspaceArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		forkResult, err := ctx.srv.forkWorkspaceService(workspacePath, args.Name)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(forkResult)
		return result, emptyExternalResult{}, err
	})

	type deleteForkArgs struct {
		Workspace string `json:"workspace" jsonschema:"The root workspace name"`
		ForkDir   string `json:"forkDir" jsonschema:"The full path of the fork directory to delete"`
		Confirm   bool   `json:"confirm" jsonschema:"Must be true to confirm deletion"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_fork",
		Description: "Delete a fork workspace. Requires confirmation.",
		InputSchema: mustSchema[deleteForkArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args deleteForkArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		deleteResult, err := ctx.srv.deleteForkService(workspacePath, args.ForkDir, args.Confirm)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(deleteResult)
		return result, emptyExternalResult{}, err
	})

	type renameWorkspaceArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name to rename"`
		Name      string `json:"name" jsonschema:"The new workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "rename_workspace",
		Description: "Rename a fork workspace.",
		InputSchema: mustSchema[renameWorkspaceArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args renameWorkspaceArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		renameResult, err := ctx.srv.renameWorkspaceService(workspacePath, args.Name)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(renameResult)
		return result, emptyExternalResult{}, err
	})

	type getGoalArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_goal",
		Description: "Get the GOAL.md content for a workspace.",
		InputSchema: mustSchema[getGoalArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getGoalArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		goalResult, err := ctx.srv.getGoalService(workspacePath)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		result, err := jsonResult(goalResult)
		return result, emptyExternalResult{}, err
	})

	type updateGoalArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
		Content   string `json:"content" jsonschema:"The new GOAL.md content"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_goal",
		Description: "Update the GOAL.md content for a workspace.",
		InputSchema: mustSchema[updateGoalArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args updateGoalArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		updateResult, err := ctx.srv.updateGoalService(workspacePath, args.Content)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(updateResult)
		return result, emptyExternalResult{}, err
	})

	type updateSummaryArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
		Summary   string `json:"summary" jsonschema:"The summary text"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_summary",
		Description: "Update the summary for a workspace.",
		InputSchema: mustSchema[updateSummaryArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args updateSummaryArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		updateResult, err := ctx.srv.updateSummaryService(workspacePath, args.Summary)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		result, err := jsonResult(updateResult)
		return result, emptyExternalResult{}, err
	})

	type togglePinArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "toggle_pin",
		Description: "Toggle the pinned state of a workspace.",
		InputSchema: mustSchema[togglePinArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args togglePinArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		pinResult, err := ctx.srv.togglePinService(workspacePath)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		result, err := jsonResult(pinResult)
		return result, emptyExternalResult{}, err
	})

	type updateDescriptionArgs struct {
		Workspace   string `json:"workspace" jsonschema:"The workspace name"`
		Description string `json:"description" jsonschema:"The new commit description"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_description",
		Description: "Update the jj commit description for a workspace.",
		InputSchema: mustSchema[updateDescriptionArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args updateDescriptionArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		descResult, err := ctx.srv.updateDescriptionService(workspacePath, args.Description)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(descResult)
		return result, emptyExternalResult{}, err
	})

	type deleteMessageArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
		ID        int    `json:"id" jsonschema:"The message ID to delete"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_message",
		Description: "Delete a message from a workspace.",
		InputSchema: mustSchema[deleteMessageArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args deleteMessageArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		deleteResult, err := ctx.srv.deleteMessageService(workspacePath, args.ID)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(deleteResult)
		return result, emptyExternalResult{}, err
	})
}

func registerSessionTools(server *mcp.Server, ctx *externalMCPContext) {
	type startSessionArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
		Auto      bool   `json:"auto,omitempty" jsonschema:"If true, start in self-drive mode (skip brainstorming)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_session",
		Description: "Start an agentic session for a workspace.",
		InputSchema: mustSchema[startSessionArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args startSessionArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		sessionResult, err := ctx.srv.startSessionService(workspacePath, args.Auto)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(sessionResult)
		return result, emptyExternalResult{}, err
	})

	type stopSessionArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_session",
		Description: "Stop the running session for a workspace.",
		InputSchema: mustSchema[stopSessionArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args stopSessionArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		stopResult := ctx.srv.stopSessionService(workspacePath)
		result, err := jsonResult(stopResult)
		return result, emptyExternalResult{}, err
	})

	type respondToQuestionArgs struct {
		Workspace       string   `json:"workspace" jsonschema:"The workspace name"`
		QuestionID      string   `json:"questionId" jsonschema:"The question ID from the pending question"`
		Answer          string   `json:"answer,omitempty" jsonschema:"Free text answer"`
		SelectedChoices []string `json:"selectedChoices,omitempty" jsonschema:"Selected choices for multi-choice questions"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "respond_to_question",
		Description: "Respond to a pending question in a workspace session.",
		InputSchema: mustSchema[respondToQuestionArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args respondToQuestionArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		respondResult, err := ctx.srv.respondService(workspacePath, args.QuestionID, args.Answer, args.SelectedChoices)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(respondResult)
		return result, emptyExternalResult{}, err
	})

	type steerAgentArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
		Message   string `json:"message" jsonschema:"The steering instruction message"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "steer_agent",
		Description: "Send a steering instruction to the running agent.",
		InputSchema: mustSchema[steerAgentArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args steerAgentArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		steerRes, err := ctx.srv.steerService(workspacePath, args.Message)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(steerRes)
		return result, emptyExternalResult{}, err
	})
}

func registerKnowledgeTools(server *mcp.Server, ctx *externalMCPContext) {
	type listAgentsArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional, uses first workspace if omitted)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_agents",
		Description: "List all agents available in a workspace.",
		InputSchema: mustSchema[listAgentsArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args listAgentsArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		agentsResult := ctx.srv.listAgentsService(workspacePath)
		result, err := jsonResult(agentsResult)
		return result, emptyExternalResult{}, err
	})

	type listSkillsArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_skills",
		Description: "List all skills available in a workspace.",
		InputSchema: mustSchema[listSkillsArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args listSkillsArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		skillsResult := ctx.srv.listSkillsService(workspacePath)
		result, err := jsonResult(skillsResult)
		return result, emptyExternalResult{}, err
	})

	type getSkillDetailArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
		Name      string `json:"name" jsonschema:"The skill path (e.g. 'coding-practices/go-code-review')"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_skill_detail",
		Description: "Get detailed content for a specific skill.",
		InputSchema: mustSchema[getSkillDetailArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getSkillDetailArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		skillResult := ctx.srv.skillDetailService(workspacePath, args.Name)
		if !skillResult.Found {
			return textResult("skill not found: " + args.Name), emptyExternalResult{}, nil
		}
		result, err := jsonResult(skillResult)
		return result, emptyExternalResult{}, err
	})

	type listSnippetsArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_snippets",
		Description: "List all code snippets available in a workspace.",
		InputSchema: mustSchema[listSnippetsArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args listSnippetsArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		snippetsResult := ctx.srv.listSnippetsService(workspacePath)
		result, err := jsonResult(snippetsResult)
		return result, emptyExternalResult{}, err
	})

	type listSnippetsByLanguageArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
		Language  string `json:"language" jsonschema:"The programming language"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_snippets_by_language",
		Description: "List code snippets for a specific programming language.",
		InputSchema: mustSchema[listSnippetsByLanguageArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args listSnippetsByLanguageArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		snippetsResult := ctx.srv.snippetsByLanguageService(workspacePath, args.Language)
		if !snippetsResult.Found {
			return textResult("language not found: " + args.Language), emptyExternalResult{}, nil
		}
		result, err := jsonResult(snippetsResult)
		return result, emptyExternalResult{}, err
	})

	type getSnippetDetailArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
		Language  string `json:"language" jsonschema:"The programming language"`
		FileName  string `json:"fileName" jsonschema:"The snippet file name (without extension)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_snippet_detail",
		Description: "Get detailed content for a specific code snippet.",
		InputSchema: mustSchema[getSnippetDetailArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getSnippetDetailArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		snippetResult := ctx.srv.snippetDetailService(workspacePath, args.Language, args.FileName)
		if !snippetResult.Found {
			return textResult("snippet not found"), emptyExternalResult{}, nil
		}
		result, err := jsonResult(snippetResult)
		return result, emptyExternalResult{}, err
	})
}

func registerComposeTools(server *mcp.Server, ctx *externalMCPContext) {
	type getComposeStateArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_compose_state",
		Description: "Get the compose wizard state for a workspace.",
		InputSchema: mustSchema[getComposeStateArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getComposeStateArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		composeResult := ctx.srv.composeStateService(workspacePath)
		result, err := jsonResult(composeResult)
		return result, emptyExternalResult{}, err
	})

	type saveComposeArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
		IfMatch   string `json:"ifMatch,omitempty" jsonschema:"ETag for optimistic concurrency"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_compose",
		Description: "Save the current compose state to GOAL.md.",
		InputSchema: mustSchema[saveComposeArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args saveComposeArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		saveResult, err := ctx.srv.composeSaveService(workspacePath, args.IfMatch)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(saveResult)
		return result, emptyExternalResult{}, err
	})

	type getComposeTemplatesArgs struct{}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_compose_templates",
		Description: "Get available workflow templates for the compose wizard.",
		InputSchema: mustSchema[getComposeTemplatesArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ getComposeTemplatesArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		templatesResult := ctx.srv.composeTemplatesService()
		result, err := jsonResult(templatesResult)
		return result, emptyExternalResult{}, err
	})

	type getComposePreviewArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_compose_preview",
		Description: "Preview the GOAL.md that would be generated from the current compose state.",
		InputSchema: mustSchema[getComposePreviewArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getComposePreviewArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		previewResult, err := ctx.srv.composePreviewService(workspacePath)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		result, err := jsonResult(previewResult)
		return result, emptyExternalResult{}, err
	})

	type saveComposeDraftArgs struct {
		Workspace string         `json:"workspace,omitempty" jsonschema:"The workspace name (optional)"`
		State     composerState  `json:"state" jsonschema:"The compose state to save as draft"`
		Wizard    apiWizardState `json:"wizard" jsonschema:"The wizard state to save as draft"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_compose_draft",
		Description: "Save the compose state as a draft without writing to GOAL.md.",
		InputSchema: mustSchema[saveComposeDraftArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args saveComposeDraftArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveAnyWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		draftResult := ctx.srv.composeDraftService(workspacePath, args.State, wizardState(args.Wizard))
		result, err := jsonResult(draftResult)
		return result, emptyExternalResult{}, err
	})
}

func registerAdhocTools(server *mcp.Server, ctx *externalMCPContext) {
	type getAdhocStatusArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_adhoc_status",
		Description: "Get the status of the ad-hoc prompt for a workspace.",
		InputSchema: mustSchema[getAdhocStatusArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args getAdhocStatusArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		statusResult := ctx.srv.adhocStatusService(workspacePath)
		result, err := jsonResult(statusResult)
		return result, emptyExternalResult{}, err
	})

	type startAdhocArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
		Prompt    string `json:"prompt" jsonschema:"The prompt text to run"`
		Model     string `json:"model" jsonschema:"The model to use (e.g. 'anthropic/claude-opus-4-6')"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_adhoc",
		Description: "Start an ad-hoc prompt in a workspace.",
		InputSchema: mustSchema[startAdhocArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args startAdhocArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		startResult := ctx.srv.adhocStartService(workspacePath, args.Prompt, args.Model)
		if startResult.Error != nil {
			return textResult("error: " + startResult.Error.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(startResult)
		return result, emptyExternalResult{}, err
	})

	type stopAdhocArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_adhoc",
		Description: "Stop the running ad-hoc prompt in a workspace.",
		InputSchema: mustSchema[stopAdhocArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args stopAdhocArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		stopResult := ctx.srv.adhocStopService(workspacePath)
		result, err := jsonResult(stopResult)
		return result, emptyExternalResult{}, err
	})
}

func registerEditorTools(server *mcp.Server, ctx *externalMCPContext) {
	type openEditorArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_editor",
		Description: "Open the workspace in the configured editor.",
		InputSchema: mustSchema[openEditorArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args openEditorArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		openResult, err := ctx.srv.openEditorService(workspacePath)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(openResult)
		return result, emptyExternalResult{}, err
	})

	type openEditorGoalArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_editor_goal",
		Description: "Open the GOAL.md file in the configured editor.",
		InputSchema: mustSchema[openEditorGoalArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args openEditorGoalArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		openResult, err := ctx.srv.openEditorGoalService(workspacePath)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(openResult)
		return result, emptyExternalResult{}, err
	})

	type openEditorPMArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_editor_pm",
		Description: "Open the PROJECT_MANAGEMENT.md file in the configured editor.",
		InputSchema: mustSchema[openEditorPMArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args openEditorPMArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		openResult, err := ctx.srv.openEditorProjectManagementService(workspacePath)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(openResult)
		return result, emptyExternalResult{}, err
	})

	type openOpencodeArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_opencode",
		Description: "Open the workspace in OpenCode (terminal-based, localhost only).",
		InputSchema: mustSchema[openOpencodeArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args openOpencodeArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}
		openResult, err := ctx.srv.openOpencodeService(workspacePath)
		if err != nil {
			return textResult("error: " + err.Error()), emptyExternalResult{}, nil
		}
		result, err := jsonResult(openResult)
		return result, emptyExternalResult{}, err
	})
}

func registerModelTools(server *mcp.Server, ctx *externalMCPContext) {
	type listModelsArgs struct {
		Workspace string `json:"workspace,omitempty" jsonschema:"The workspace name for default model lookup (optional)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_models",
		Description: "List all available AI models.",
		InputSchema: mustSchema[listModelsArgs](),
	}, func(_ context.Context, _ *mcp.CallToolRequest, args listModelsArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		modelsResult := ctx.srv.listModelsService(args.Workspace)
		result, err := jsonResult(modelsResult)
		return result, emptyExternalResult{}, err
	})
}

func registerElicitationTool(server *mcp.Server, ctx *externalMCPContext) {
	type waitForQuestionArgs struct {
		Workspace string `json:"workspace" jsonschema:"The workspace name to watch for pending questions"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "wait_for_question",
		Description: "Wait for a pending question in a workspace and respond to it using MCP elicitation. Polls until a question is pending, then presents it to the harness for a response.",
		InputSchema: mustSchema[waitForQuestionArgs](),
	}, func(toolCtx context.Context, req *mcp.CallToolRequest, args waitForQuestionArgs) (*mcp.CallToolResult, emptyExternalResult, error) {
		workspacePath, err := ctx.resolveWorkspacePath(args.Workspace)
		if err != nil {
			return nil, emptyExternalResult{}, err
		}

		questionID, answer, err := elicitPendingQuestion(toolCtx, req.Session, ctx.srv, workspacePath)
		if err != nil {
			return textResult("elicitation error: " + err.Error()), emptyExternalResult{}, nil
		}
		if answer == "" {
			return textResult("no response provided"), emptyExternalResult{}, nil
		}

		respondResult, errRespond := ctx.srv.respondService(workspacePath, questionID, answer, nil)
		if errRespond != nil {
			return textResult("respond error: " + errRespond.Error()), emptyExternalResult{}, nil
		}

		result, err := jsonResult(respondResult)
		return result, emptyExternalResult{}, err
	})
}

func elicitPendingQuestion(ctx context.Context, session *mcp.ServerSession, srv *Server, workspacePath string) (questionID, answer string, err error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-ticker.C:
			wfState := srv.workspaceCoordinator(workspacePath).State()
			if !wfState.NeedsHumanInput() {
				continue
			}

			qID := generateQuestionID(wfState)
			message := wfState.HumanMessage
			if message == "" {
				message = "A workspace agent is waiting for your input."
			}

			schema := buildPendingQuestionSchema(wfState)

			elicitResult, errElicit := session.Elicit(ctx, &mcp.ElicitParams{
				Message:         message,
				RequestedSchema: schema,
			})
			if errElicit != nil {
				return "", "", fmt.Errorf("elicitation failed: %w", errElicit)
			}

			if elicitResult.Action != "accept" {
				return qID, "", nil
			}

			ans := ""
			if elicitResult.Content != nil {
				if a, ok := elicitResult.Content["answer"].(string); ok {
					ans = strings.TrimSpace(a)
				}
				if len(elicitResult.Content) > 0 && ans == "" {
					var parts []string
					for k, v := range elicitResult.Content {
						parts = append(parts, fmt.Sprintf("%s=%v", k, v))
					}
					ans = strings.Join(parts, ", ")
				}
			}

			return qID, ans, nil
		}
	}
}

func buildPendingQuestionSchema(_ any) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"answer": {
				Type:        "string",
				Description: "Your response to the agent's question",
			},
		},
		Required: []string{"answer"},
	}
}
