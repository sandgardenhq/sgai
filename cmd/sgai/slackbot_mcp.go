package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sandgardenhq/sgai/pkg/state"
)

type slackReplyArgs struct {
	Text string `json:"text" jsonschema:"The text message to send to the Slack thread."`
}

type slackReplyBlocksArgs struct {
	Blocks string `json:"blocks" jsonschema:"Block Kit JSON payload to send to the Slack thread."`
}

type sgaiListWorkspacesArgs struct{}

type sgaiWorkspaceStatusArgs struct {
	Workspace string `json:"workspace" jsonschema:"Name of the workspace to get status for."`
}

type sgaiStartWorkspaceArgs struct {
	Workspace string `json:"workspace" jsonschema:"Name of the workspace to start."`
	Auto      bool   `json:"auto,omitempty" jsonschema:"Whether to start in auto (self-driving) mode."`
}

type sgaiStopWorkspaceArgs struct {
	Workspace string `json:"workspace" jsonschema:"Name of the workspace to stop."`
}

type sgaiRespondToQuestionArgs struct {
	Workspace       string   `json:"workspace" jsonschema:"Name of the workspace."`
	Answer          string   `json:"answer,omitempty" jsonschema:"Free-text answer."`
	SelectedChoices []string `json:"selectedChoices,omitempty" jsonschema:"Selected choice(s) for multi-choice questions."`
}

type sgaiReadEventsArgs struct {
	Workspace string `json:"workspace" jsonschema:"Name of the workspace to read events from."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of events to return (default 20)."`
}

type sgaiReadMessagesArgs struct {
	Workspace string `json:"workspace" jsonschema:"Name of the workspace to read messages from."`
}

type sgaiConnectThreadArgs struct {
	Workspace string `json:"workspace" jsonschema:"Name of the workspace to connect to this thread."`
}

type sgaiDisconnectThreadArgs struct{}

type sgaiToggleEventUpdatesArgs struct {
	Enabled bool `json:"enabled" jsonschema:"Whether to enable or disable event updates for this thread."`
}

type slackBotMCPContext struct {
	rootDir   string
	channelID string
	threadTS  string
	replyCh   chan<- slackReplyMessage
	sessions  *sessionDB
	server    *Server
}

type slackReplyMessage struct {
	channelID string
	threadTS  string
	text      string
	blocks    string
}

var (
	schemaSlackReply         = mustSchema[slackReplyArgs]()
	schemaSlackReplyBlocks   = mustSchema[slackReplyBlocksArgs]()
	schemaListWorkspaces     = mustSchema[sgaiListWorkspacesArgs]()
	schemaWorkspaceStatus    = mustSchema[sgaiWorkspaceStatusArgs]()
	schemaStartWorkspace     = mustSchema[sgaiStartWorkspaceArgs]()
	schemaStopWorkspace      = mustSchema[sgaiStopWorkspaceArgs]()
	schemaRespondToQuestion  = mustSchema[sgaiRespondToQuestionArgs]()
	schemaReadEvents         = mustSchema[sgaiReadEventsArgs]()
	schemaReadMessages       = mustSchema[sgaiReadMessagesArgs]()
	schemaConnectThread      = mustSchema[sgaiConnectThreadArgs]()
	schemaDisconnectThread   = mustSchema[sgaiDisconnectThreadArgs]()
	schemaToggleEventUpdates = mustSchema[sgaiToggleEventUpdatesArgs]()
)

func startSlackBotMCPServer(ctx *slackBotMCPContext) (string, func(), error) {
	listener, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		return "", nil, fmt.Errorf("listening on random port: %w", errListen)
	}

	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return buildSlackBotMCPServer(ctx)
	}, nil)

	serveMux := http.NewServeMux()
	serveMux.Handle("/mcp", handler)

	httpServer := &http.Server{Handler: serveMux}
	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("slack-bot MCP HTTP server error:", err)
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://127.0.0.1:%d/mcp", addr.Port)

	closeFn := func() {
		if err := httpServer.Close(); err != nil {
			log.Println("closing slack-bot MCP HTTP server:", err)
		}
	}

	return url, closeFn, nil
}

func buildSlackBotMCPServer(ctx *slackBotMCPContext) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "sgai-slack-bot"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_reply",
		Description: "Send a text message to the current Slack thread.",
		InputSchema: schemaSlackReply,
	}, ctx.slackReplyHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_reply_blocks",
		Description: "Send a rich Block Kit formatted message to the current Slack thread.",
		InputSchema: schemaSlackReplyBlocks,
	}, ctx.slackReplyBlocksHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_list_workspaces",
		Description: "List all available workspaces with their current status.",
		InputSchema: schemaListWorkspaces,
	}, ctx.listWorkspacesHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_workspace_status",
		Description: "Get detailed status of a workspace including state, current agent, progress, and todos.",
		InputSchema: schemaWorkspaceStatus,
	}, ctx.workspaceStatusHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_start_workspace",
		Description: "Start a workspace session.",
		InputSchema: schemaStartWorkspace,
	}, ctx.startWorkspaceHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_stop_workspace",
		Description: "Stop a running workspace session.",
		InputSchema: schemaStopWorkspace,
	}, ctx.stopWorkspaceHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_respond_to_question",
		Description: "Answer an agent's multi-choice or free-text question in a workspace.",
		InputSchema: schemaRespondToQuestion,
	}, ctx.respondToQuestionHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_read_events",
		Description: "Read workspace progress events.",
		InputSchema: schemaReadEvents,
	}, ctx.readEventsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_read_messages",
		Description: "Read inter-agent messages for a workspace.",
		InputSchema: schemaReadMessages,
	}, ctx.readMessagesHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_connect_thread",
		Description: "Map the current Slack thread to a workspace. Subsequent messages in this thread will be routed to that workspace.",
		InputSchema: schemaConnectThread,
	}, ctx.connectThreadHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_disconnect_thread",
		Description: "Unmap the current Slack thread from its workspace.",
		InputSchema: schemaDisconnectThread,
	}, ctx.disconnectThreadHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sgai_toggle_event_updates",
		Description: "Toggle receiving live event updates (progress entries) for this thread.",
		InputSchema: schemaToggleEventUpdates,
	}, ctx.toggleEventUpdatesHandler)

	return server
}

func (c *slackBotMCPContext) slackReplyHandler(_ context.Context, _ *mcp.CallToolRequest, args slackReplyArgs) (*mcp.CallToolResult, emptyResult, error) {
	c.replyCh <- slackReplyMessage{
		channelID: c.channelID,
		threadTS:  c.threadTS,
		text:      args.Text,
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Message sent to Slack thread."}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) slackReplyBlocksHandler(_ context.Context, _ *mcp.CallToolRequest, args slackReplyBlocksArgs) (*mcp.CallToolResult, emptyResult, error) {
	c.replyCh <- slackReplyMessage{
		channelID: c.channelID,
		threadTS:  c.threadTS,
		blocks:    args.Blocks,
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Block Kit message sent to Slack thread."}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) listWorkspacesHandler(_ context.Context, _ *mcp.CallToolRequest, _ sgaiListWorkspacesArgs) (*mcp.CallToolResult, emptyResult, error) {
	groups, err := c.server.scanWorkspaceGroups()
	if err != nil {
		return nil, emptyResult{}, fmt.Errorf("scanning workspaces: %w", err)
	}

	var result strings.Builder
	for _, grp := range groups {
		running, _ := c.server.getWorkspaceStatus(grp.Root.Directory)
		wfState, _ := state.Load(statePath(grp.Root.Directory))
		status := wfState.Status
		if status == "" {
			status = "idle"
		}
		runningStr := "stopped"
		if running {
			runningStr = "running"
		}
		result.WriteString(fmt.Sprintf("- %s [%s, %s]\n", grp.Root.DirName, status, runningStr))
		for _, fork := range grp.Forks {
			forkRunning, _ := c.server.getWorkspaceStatus(fork.Directory)
			forkState, _ := state.Load(statePath(fork.Directory))
			forkStatus := forkState.Status
			if forkStatus == "" {
				forkStatus = "idle"
			}
			forkRunningStr := "stopped"
			if forkRunning {
				forkRunningStr = "running"
			}
			result.WriteString(fmt.Sprintf("  - %s [%s, %s]\n", fork.DirName, forkStatus, forkRunningStr))
		}
	}

	text := result.String()
	if text == "" {
		text = "No workspaces found."
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) resolveWorkspacePath(name string) (string, error) {
	path := c.server.resolveWorkspaceNameToPath(name)
	if path == "" {
		return "", fmt.Errorf("workspace %q not found", name)
	}
	return path, nil
}

func (c *slackBotMCPContext) workspaceStatusHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiWorkspaceStatusArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	wfState, _ := state.Load(statePath(wsPath))
	running, needsInput := c.server.getWorkspaceStatus(wsPath)

	status := wfState.Status
	if status == "" {
		status = "idle"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Workspace: %s\n", args.Workspace))
	result.WriteString(fmt.Sprintf("Status: %s\n", status))
	result.WriteString(fmt.Sprintf("Running: %v\n", running))
	result.WriteString(fmt.Sprintf("Needs Input: %v\n", needsInput))

	if wfState.CurrentAgent != "" {
		result.WriteString(fmt.Sprintf("Current Agent: %s\n", wfState.CurrentAgent))
	}
	if wfState.Task != "" {
		result.WriteString(fmt.Sprintf("Task: %s\n", wfState.Task))
	}
	if needsInput && wfState.HumanMessage != "" {
		result.WriteString(fmt.Sprintf("Pending Question: %s\n", wfState.HumanMessage))
		if wfState.MultiChoiceQuestion != nil {
			for _, q := range wfState.MultiChoiceQuestion.Questions {
				result.WriteString(fmt.Sprintf("  Choices: %s\n", strings.Join(q.Choices, ", ")))
			}
		}
	}

	if len(wfState.Progress) > 0 {
		last := wfState.Progress[len(wfState.Progress)-1]
		result.WriteString(fmt.Sprintf("Latest Progress: [%s] %s\n", last.Agent, last.Description))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result.String()}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) startWorkspaceHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiStartWorkspaceArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	result := c.server.startSession(wsPath)
	if result.alreadyRunning {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Workspace %q is already running.", args.Workspace)}},
		}, emptyResult{}, nil
	}
	if result.startError != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to start workspace %q: %s", args.Workspace, result.startError)}},
		}, emptyResult{}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Workspace %q started.", args.Workspace)}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) stopWorkspaceHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiStopWorkspaceArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	c.server.stopSession(wsPath)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Workspace %q stopped.", args.Workspace)}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) respondToQuestionHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiRespondToQuestionArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	wfState, errLoad := state.Load(statePath(wsPath))
	if errLoad != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "failed to load workspace state"}},
		}, emptyResult{}, nil
	}

	if !wfState.NeedsHumanInput() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "no pending question for this workspace"}},
		}, emptyResult{}, nil
	}

	var parts []string
	if len(args.SelectedChoices) > 0 {
		parts = append(parts, "Selected: "+strings.Join(args.SelectedChoices, ", "))
	}
	if args.Answer != "" {
		parts = append(parts, args.Answer)
	}
	responseText := strings.Join(parts, "\n")
	if responseText == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "response cannot be empty"}},
		}, emptyResult{}, nil
	}

	responsePath := filepath.Join(wsPath, ".sgai", "response.txt")
	if errWrite := os.WriteFile(responsePath, []byte(responseText), 0644); errWrite != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "failed to write response file"}},
		}, emptyResult{}, nil
	}

	wfState.Status = state.StatusWorking
	wfState.HumanMessage = ""
	wfState.MultiChoiceQuestion = nil
	if errSave := state.Save(statePath(wsPath), wfState); errSave != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "failed to save workspace state"}},
		}, emptyResult{}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Response submitted to workspace %q.", args.Workspace)}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) readEventsHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiReadEventsArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	wfState, _ := state.Load(statePath(wsPath))

	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}

	progress := wfState.Progress
	if len(progress) > limit {
		progress = progress[len(progress)-limit:]
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Events for %s (showing last %d of %d):\n\n", args.Workspace, len(progress), len(wfState.Progress)))
	for _, entry := range progress {
		result.WriteString(fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp, entry.Agent, entry.Description))
	}

	text := result.String()
	if len(progress) == 0 {
		text = fmt.Sprintf("No events for workspace %q.", args.Workspace)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) readMessagesHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiReadMessagesArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	wfState, _ := state.Load(statePath(wsPath))

	if len(wfState.Messages) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No messages for workspace %q.", args.Workspace)}},
		}, emptyResult{}, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Messages for %s (%d total):\n\n", args.Workspace, len(wfState.Messages)))
	for _, msg := range wfState.Messages {
		readStatus := "unread"
		if msg.Read {
			readStatus = "read"
		}
		subject := extractSubject(msg.Body)
		result.WriteString(fmt.Sprintf("[%s] %s -> %s: %s (%s)\n", msg.CreatedAt, msg.FromAgent, msg.ToAgent, subject, readStatus))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result.String()}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) connectThreadHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiConnectThreadArgs) (*mcp.CallToolResult, emptyResult, error) {
	wsPath, err := c.resolveWorkspacePath(args.Workspace)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}, emptyResult{}, nil
	}

	sess := slackSession{
		WorkspaceDir:        wsPath,
		ChannelID:           c.channelID,
		ThreadTS:            c.threadTS,
		EventUpdatesEnabled: false,
	}

	if errPut := c.sessions.put(c.channelID, c.threadTS, sess); errPut != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to save session: %s", errPut)}},
		}, emptyResult{}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Thread connected to workspace %q.", args.Workspace)}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) disconnectThreadHandler(_ context.Context, _ *mcp.CallToolRequest, _ sgaiDisconnectThreadArgs) (*mcp.CallToolResult, emptyResult, error) {
	if errDel := c.sessions.delete(c.channelID, c.threadTS); errDel != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to disconnect: %s", errDel)}},
		}, emptyResult{}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Thread disconnected from workspace."}},
	}, emptyResult{}, nil
}

func (c *slackBotMCPContext) toggleEventUpdatesHandler(_ context.Context, _ *mcp.CallToolRequest, args sgaiToggleEventUpdatesArgs) (*mcp.CallToolResult, emptyResult, error) {
	if errUpdate := c.sessions.updateEventUpdates(c.channelID, c.threadTS, args.Enabled); errUpdate != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to update event toggle: %s", errUpdate)}},
		}, emptyResult{}, nil
	}

	status := "disabled"
	if args.Enabled {
		status = "enabled"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Event updates %s for this thread.", status)}},
	}, emptyResult{}, nil
}
