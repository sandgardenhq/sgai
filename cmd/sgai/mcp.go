package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sandgardenhq/sgai/pkg/state"
)

const todoReadDescription = `Use this tool to read the current to-do list for the session. This tool should be used proactively and frequently to ensure that you are aware of the status of the current task list. You should make use of this tool as often as possible, especially in the following situations:
- At the beginning of conversations to see what's pending
- Before starting new tasks to prioritize work
- When the user asks about previous tasks or plans
- Whenever you're uncertain about what to do next
- After completing tasks to update your understanding of remaining work
- After every few messages to ensure you're on track

Usage:
- This tool takes in no parameters. So leave the input blank or empty. DO NOT include a dummy object, placeholder string or a key like "input" or "empty". LEAVE IT BLANK.
- Returns a list of todo items with their status, priority, and content
- Use this information to track progress and plan next steps
- If no todos exist yet, an empty list will be returned`

const todoWriteDescription = `Use this tool to create and manage a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool
Use this tool proactively in these scenarios:

1. Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
2. Non-trivial and complex tasks - Tasks that require careful planning or multiple operations
3. User explicitly requests todo list - When the user directly asks you to use the todo list
4. User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
5. After receiving new instructions - Immediately capture user requirements as todos. Feel free to edit the todo list based on new information.
6. After completing a task - Mark it complete and add any new follow-up tasks
7. When you start working on a new task, mark the todo as in_progress. Ideally you should only have one todo as in_progress at a time. Complete existing tasks before starting new ones.

## When NOT to Use This Tool

Skip using this tool when:
1. There is only a single, straightforward task
2. The task is trivial and tracking it provides no organizational benefit
3. The task can be completed in less than 3 trivial steps
4. The task is purely conversational or informational

NOTE that you should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Examples of When to Use the Todo List

<example>
User: I want to add a dark mode toggle to the application settings. Make sure you run the tests and build when you're done!
Assistant: I'll help add a dark mode toggle to your application settings. Let me create a todo list to track this implementation.
*Creates todo list with the following items:*
1. Create dark mode toggle component in Settings page
2. Add dark mode state management (context/store)
3. Implement CSS-in-JS styles for dark theme
4. Update existing components to support theme switching
5. Run tests and build process, addressing any failures or errors that occur
*Begins working on the first task*

<reasoning>
The assistant used the todo list because:
1. Adding dark mode is a multi-step feature requiring UI, state management, and styling changes
2. The user explicitly requested tests and build be run afterward
3. The assistant inferred that tests and build need to pass by adding "Ensure tests and build succeed" as the final task
</reasoning>
</example>

<example>
User: Help me rename the function getCwd to getCurrentWorkingDirectory across my project
Assistant: Let me first search through your codebase to find all occurrences of 'getCwd'.
*Uses grep or search tools to locate all instances of getCwd in the codebase*
Assistant: I've found 15 instances of 'getCwd' across 8 different files. Let me create a todo list to track these changes.
*Creates todo list with specific items for each file that needs updating*

<reasoning>
The assistant used the todo list because:
1. First, the assistant searched to understand the scope of the task
2. Upon finding multiple occurrences across different files, it determined this was a complex task with multiple steps
3. The todo list helps ensure every instance is tracked and updated systematically
4. This approach prevents missing any occurrences and maintains code consistency
</reasoning>
</example>


<example>
User: I need to implement these features for my e-commerce site: user registration, product catalog, shopping cart, and checkout flow.
Assistant: I'll help implement these features. First, let's add all the features to the todo list.
*Creates a todo list breaking down each feature into specific tasks based on the project architecture*
Assistant: Let's start with user registration. This will involve creating database models, API endpoints, and frontend forms that integrate with your existing authentication system.

<reasoning>
The assistant used the todo list because:
1. The user provided multiple complex features to implement in a comma separated list
2. The todo list helps organize these large features into manageable tasks
3. This approach allows for tracking progress across the entire implementation
</reasoning>
</example>

<example>
User: Can you help optimize my React application? It's rendering slowly and has performance issues.</user>
Assistant: I'll help optimize your React application. First, let me examine your codebase to identify potential performance bottlenecks.
*Reviews component structure, render patterns, state management, and data fetching*
Assistant: After analyzing your codebase, I've identified several performance issues. Let me create a todo list to track our optimization efforts.
*Creates todo list with items like: 1) Implement memoization for expensive calculations in ProductList, 2) Add virtualization for long lists in Dashboard, 3) Optimize image loading in Gallery component, 4) Fix state update loops in ShoppingCart, 5) Review bundle size and implement code splitting*
Let's start by implementing memoization for the expensive calculations in your ProductList component.</assistant>

<reasoning>
The assistant used the todo list because:
1. First, the assistant examined the codebase to identify specific performance issues
2. Based on this analysis, it identified multiple optimization opportunities
3. Performance optimization is a non-trivial task requiring multiple steps
4. The todo list helps methodically track improvements across different components
5. This systematic approach ensures all performance bottlenecks are addressed
</reasoning>
</example>

## Examples of When NOT to Use the Todo List

<example>
User: How do I print 'Hello World' in Python?
Assistant: In Python, you can print "Hello World" with this simple code:

python
print("Hello World")

This will output the text "Hello World" to the console when executed.</assistant>

<reasoning>
The assistant did not use the todo list because this is a single, trivial task that can be completed in one step. There's no need to track multiple tasks or steps for such a straightforward request.
</reasoning>
</example>

<example>
User: What does the git status command do?
Assistant: The git status command shows the current state of your working directory and staging area. It displays which changes have been staged, which haven't, and which files aren't being tracked by Git. This is a fundamental command for understanding the current state of your repository before making commits.

<reasoning>
The assistant did not use the todo list because this is an informational request with no actual coding task to complete. The user is simply asking for an explanation, not for the assistant to perform multiple steps or tasks.
</reasoning>
</example>

<example>
User: Can you add a comment to the calculateTotal function to explain what it does?
Assistant: Sure, let me add a comment to the calculateTotal function to explain what it does.
* Uses the Edit tool to add a comment to the calculateTotal function *

<reasoning>
The assistant did not use the todo list because this is a single, straightforward task confined to one location in the code. Adding a comment doesn't require tracking multiple steps or systematic organization.
</reasoning>
</example>

<example>
User: Run npm install for me and tell me what happens.
Assistant: I'll run the npm install command for you.

*Executes: npm install*

The command completed successfully. Here's the output:
[Output of npm install command]

All dependencies have been installed according to your package.json file.

<reasoning>
The assistant did not use the todo list because this is a single command execution with immediate results. There are no multiple steps to track or organize, making the todo list unnecessary for this straightforward task.
</reasoning>
</example>

## Task States and Management

1. **Task States**: Use these states to track progress:
   - pending: Task not yet started
   - in_progress: Currently working on (limit to ONE task at a time)
   - completed: Task finished successfully
   - cancelled: Task no longer needed

2. **Task Management**:
   - Update task status in real-time as you work
   - Mark tasks complete IMMEDIATELY after finishing (don't batch completions)
   - Only have ONE task in_progress at any time
   - Complete current tasks before starting new ones
   - Cancel tasks that become irrelevant

3. **Task Breakdown**:
   - Create specific, actionable items
   - Break complex tasks into smaller, manageable steps
   - Use clear, descriptive task names

When in doubt, use this tool. Being proactive with task management demonstrates attentiveness and ensures you complete all requirements successfully.
`

type findSkillsArgs struct {
	Name string `json:"name,omitempty" jsonschema:"Skill name or search query. Omit to list all skills."`
}

type findSnippetsArgs struct {
	Language string `json:"language,omitempty" jsonschema:"Programming language. Omit to list available languages."`
	Query    string `json:"query,omitempty" jsonschema:"Search query for snippet name/description."`
}

type workflowStatus string

type updateWorkflowStateArgs struct {
	Status      workflowStatus `json:"status" jsonschema:"Overall workflow status: 'working' (actively working - may need iteration) or 'agent-done' (agent's work done - needs goal verification) or 'complete' (goals verified as achieved). Valid values: working, agent-done, complete"`
	Task        string         `json:"task" jsonschema:"Current task being worked on (e.g. 'Writing tests for auth endpoints'). Use empty string to clear. Be specific about what you're doing."`
	AddProgress string         `json:"addProgress" jsonschema:"Add a progress note to track what you've accomplished. This will be appended to the progress array. Use this frequently to document your steps."`
}

type sendMessageArgs struct {
	ToAgent string `json:"toAgent" jsonschema:"The agent who will receive this message. Must be one of the agents in the workflow."`
	Body    string `json:"body" jsonschema:"The content of the message to send."`
}

type projectTodoWriteArgs struct {
	Todos []state.TodoItem `json:"todos" jsonschema:"The updated todo list"`
}

type questionItem struct {
	Question    string   `json:"question" jsonschema:"The question to ask"`
	Choices     []string `json:"choices" jsonschema:"Multiple-choice options for this question"`
	MultiSelect bool     `json:"multiSelect,omitempty" jsonschema:"Allow multiple selections (default: false)"`
}

type askUserQuestionArgs struct {
	Questions []questionItem `json:"questions" jsonschema:"Array of questions to present to the user"`
}

// cmdMCP starts an MCP server exposing sgai custom tools via stdio transport.
// It reads SGAI_MCP_WORKING_DIRECTORY and SGAI_MCP_INTERACTIVE.
func cmdMCP(_ []string) {
	workingDir := os.Getenv("SGAI_MCP_WORKING_DIRECTORY")
	if workingDir == "" {
		workingDir = "."
	}
	interactive := os.Getenv("SGAI_MCP_INTERACTIVE")
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		log.Fatalln("failed to resolve working directory:", err)
	}

	wfState, err := state.Load(filepath.Join(absDir, ".sgai", "state.json"))
	if err != nil && !os.IsNotExist(err) {
		log.Fatalln("failed to load workflow state:", err)
	}

	currentAgent := wfState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "coordinator"
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "sgai"}, nil)

	mcpCtx := &mcpContext{workingDir: absDir}

	findSkillsSchema, err := jsonschema.For[findSkillsArgs](nil)
	if err != nil {
		log.Fatalln("failed to create find_skills schema:", err)
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_skills",
		Description: "Find skills by name. Returns content for exact matches or lists for searches.",
		InputSchema: findSkillsSchema,
	}, mcpCtx.findSkillsHandler)

	findSnippetsSchema, err := jsonschema.For[findSnippetsArgs](nil)
	if err != nil {
		log.Fatalln("failed to create find_snippets schema:", err)
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_snippets",
		Description: "Find code snippets by language and query.",
		InputSchema: findSnippetsSchema,
	}, mcpCtx.findSnippetsHandler)

	updateWorkflowStateSchema, updateWorkflowStateDescription := buildUpdateWorkflowStateSchema(currentAgent)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_workflow_state",
		Description: updateWorkflowStateDescription,
		InputSchema: updateWorkflowStateSchema,
	}, mcpCtx.updateWorkflowStateHandler)

	sendMessageSchema, err := jsonschema.For[sendMessageArgs](nil)
	if err != nil {
		log.Fatalln("failed to create send_message schema:", err)
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "send_message",
		Description: "Send a message to another agent in the workflow. The message will be stored and delivered when the target agent starts.",
		InputSchema: sendMessageSchema,
	}, mcpCtx.sendMessageHandler)

	checkInboxSchema, err := jsonschema.For[struct{}](nil)
	if err != nil {
		log.Fatalln("failed to create check_inbox schema:", err)
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_inbox",
		Description: "Check for messages sent to the current agent. Returns all unread messages from other agents.",
		InputSchema: checkInboxSchema,
	}, mcpCtx.checkInboxHandler)

	checkOutboxSchema, err := jsonschema.For[struct{}](nil)
	if err != nil {
		log.Fatalln("failed to create check_outbox schema:", err)
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_outbox",
		Description: "Check for messages you have already sent. Returns all messages sent by the current agent.",
		InputSchema: checkOutboxSchema,
	}, mcpCtx.checkOutboxHandler)

	if currentAgent == "coordinator" {
		peekMessageBusSchema, err := jsonschema.For[struct{}](nil)
		if err != nil {
			log.Fatalln("failed to create peek_message_bus schema:", err)
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "peek_message_bus",
			Description: "Check all messages in the system (both pending and read). Returns all messages in reverse chronological order (most recent first). Coordinator-only tool for monitoring inter-agent communication.",
			InputSchema: peekMessageBusSchema,
		}, mcpCtx.peekMessageBusHandler)

		projectTodoWriteSchema, err := jsonschema.For[projectTodoWriteArgs](nil)
		if err != nil {
			log.Fatalln("failed to create project_todowrite schema:", err)
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "project_todowrite",
			Description: todoWriteDescription,
			InputSchema: projectTodoWriteSchema,
		}, mcpCtx.projectTodoWriteHandler)

		projectTodoReadSchema, err := jsonschema.For[struct{}](nil)
		if err != nil {
			log.Fatalln("failed to create project_todoread schema:", err)
		}
		mcp.AddTool(server, &mcp.Tool{
			Name:        "project_todoread",
			Description: todoReadDescription,
			InputSchema: projectTodoReadSchema,
		}, mcpCtx.projectTodoReadHandler)

		if !isSelfDriveMode(interactive) {
			askUserQuestionSchema, err := jsonschema.For[askUserQuestionArgs](nil)
			if err != nil {
				log.Fatalln("failed to create ask_user_question schema:", err)
			}
			mcp.AddTool(server, &mcp.Tool{
				Name:        "ask_user_question",
				Description: "Present one or more multiple-choice questions to the human partner. Each question has its own choices and multi-select setting. Use this for gathering structured input from the human. Example: {\"questions\": [{\"question\": \"Which database?\", \"choices\": [\"PostgreSQL\", \"MySQL\"], \"multiSelect\": false}]}",
				InputSchema: askUserQuestionSchema,
			}, mcpCtx.askUserQuestionHandler)

			askUserWorkGateSchema, err := jsonschema.For[struct{}](nil)
			if err != nil {
				log.Fatalln("failed to create ask_user_work_gate schema:", err)
			}
			mcp.AddTool(server, &mcp.Tool{
				Name:        "ask_user_work_gate",
				Description: "Present the work gate approval question to the human partner. No arguments needed - the question and choices are hardcoded. When approved, the session switches to self-driving mode for the remainder of the session.",
				InputSchema: askUserWorkGateSchema,
			}, mcpCtx.askUserWorkGateHandler)
		}
	}

	transport := &mcp.StdioTransport{}
	if err := server.Run(context.Background(), transport); err != nil {
		log.Fatalln("MCP server error:", err)
	}
}

type mcpContext struct {
	workingDir string
}

type emptyResult struct{}

func (c *mcpContext) findSkillsHandler(_ context.Context, _ *mcp.CallToolRequest, args findSkillsArgs) (*mcp.CallToolResult, emptyResult, error) {
	result, err := findSkills(c.workingDir, args.Name)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) findSnippetsHandler(_ context.Context, _ *mcp.CallToolRequest, args findSnippetsArgs) (*mcp.CallToolResult, emptyResult, error) {
	result, err := findSnippets(c.workingDir, args.Language, args.Query)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) updateWorkflowStateHandler(_ context.Context, _ *mcp.CallToolRequest, args updateWorkflowStateArgs) (*mcp.CallToolResult, emptyResult, error) {
	result, err := updateWorkflowState(c.workingDir, args)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) sendMessageHandler(_ context.Context, _ *mcp.CallToolRequest, args sendMessageArgs) (*mcp.CallToolResult, emptyResult, error) {
	result, err := sendMessage(c.workingDir, args.ToAgent, args.Body)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) checkInboxHandler(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyResult, error) {
	result, err := checkInbox(c.workingDir)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) checkOutboxHandler(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyResult, error) {
	result, err := checkOutbox(c.workingDir)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) peekMessageBusHandler(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyResult, error) {
	result, err := peekMessageBus(c.workingDir)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) projectTodoWriteHandler(_ context.Context, _ *mcp.CallToolRequest, args projectTodoWriteArgs) (*mcp.CallToolResult, emptyResult, error) {
	result, err := projectTodoWrite(c.workingDir, args.Todos)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) projectTodoReadHandler(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyResult, error) {
	result, err := projectTodoRead(c.workingDir)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) askUserQuestionHandler(_ context.Context, _ *mcp.CallToolRequest, args askUserQuestionArgs) (*mcp.CallToolResult, emptyResult, error) {
	result, err := askUserQuestion(c.workingDir, args)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func (c *mcpContext) askUserWorkGateHandler(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyResult, error) {
	result, err := askUserWorkGate(c.workingDir)
	if err != nil {
		return nil, emptyResult{}, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, emptyResult{}, nil
}

func askUserQuestion(workingDir string, args askUserQuestionArgs) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		currentState = state.Workflow{
			Status:   state.StatusWorking,
			Progress: []state.ProgressEntry{},
		}
	}

	if len(args.Questions) == 0 {
		return "Error: At least one question is required", nil
	}

	for i, q := range args.Questions {
		if len(q.Choices) == 0 {
			return fmt.Sprintf("Error: Question %d has no choices", i+1), nil
		}
	}

	questions := make([]state.QuestionItem, len(args.Questions))
	for i, q := range args.Questions {
		questions[i] = state.QuestionItem{
			Question:    q.Question,
			Choices:     q.Choices,
			MultiSelect: q.MultiSelect,
		}
	}

	currentState.MultiChoiceQuestion = &state.MultiChoiceQuestion{
		Questions: questions,
	}
	currentState.HumanMessage = args.Questions[0].Question
	currentState.Status = state.StatusWaitingForHuman

	if err := state.Save(statePath, currentState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Presented %d question(s) to user:\n", len(args.Questions)))
	for i, q := range args.Questions {
		result.WriteString(fmt.Sprintf("\nQuestion %d: %s\n", i+1, q.Question))
		result.WriteString(fmt.Sprintf("  Choices: %v\n", q.Choices))
		result.WriteString(fmt.Sprintf("  MultiSelect: %v\n", q.MultiSelect))
	}
	return result.String(), nil
}

func askUserWorkGate(workingDir string) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		currentState = state.Workflow{
			Status:   state.StatusWorking,
			Progress: []state.ProgressEntry{},
		}
	}

	currentState.MultiChoiceQuestion = &state.MultiChoiceQuestion{
		Questions: []state.QuestionItem{
			{
				Question:    "Is the definition complete? May I begin implementation?",
				Choices:     []string{workGateApprovalText, "Not ready yet, need more clarification"},
				MultiSelect: false,
			},
		},
		IsWorkGate: true,
	}
	currentState.HumanMessage = "Is the definition complete? May I begin implementation?"
	currentState.Status = state.StatusWaitingForHuman

	if err := state.Save(statePath, currentState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return "Presented work gate question to user:\n\nQuestion: Is the definition complete? May I begin implementation?\n  Choices: [DEFINITION IS COMPLETE, BUILD MAY BEGIN, Not ready yet, need more clarification]\n  MultiSelect: false", nil
}

func findSkills(workingDir, name string) (string, error) {
	skillsDir := filepath.Join(workingDir, ".sgai", "skills")

	getAllSkillFiles := func(dir string) ([]string, error) {
		var files []string
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == "SKILL.md" {
				files = append(files, path)
			}
			return nil
		})
		return files, err
	}

	skillFiles, err := getAllSkillFiles(skillsDir)
	if err != nil {
		return "", fmt.Errorf("failed to access skills: %w", err)
	}

	if name == "" {
		var skills []string
		for _, file := range skillFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			relName, _ := filepath.Rel(skillsDir, file)
			relName = strings.TrimSuffix(relName, "/SKILL.md")
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			skills = append(skills, fmt.Sprintf("%s: %s", relName, desc))
		}
		return strings.Join(skills, "\n"), nil
	}

	for _, file := range skillFiles {
		relName, _ := filepath.Rel(skillsDir, file)
		relName = strings.TrimSuffix(relName, "/SKILL.md")
		if relName == name {
			content, err := os.ReadFile(file)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}

	var prefixMatches []string
	for _, file := range skillFiles {
		relName, _ := filepath.Rel(skillsDir, file)
		relName = strings.TrimSuffix(relName, "/SKILL.md")
		if strings.HasPrefix(relName, name) && relName != name {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			prefixMatches = append(prefixMatches, fmt.Sprintf("%s: %s", relName, desc))
		}
	}
	if len(prefixMatches) > 0 {
		return strings.Join(prefixMatches, "\n"), nil
	}

	var basenameMatches []struct {
		path    string
		content string
		desc    string
	}
	for _, file := range skillFiles {
		relName, _ := filepath.Rel(skillsDir, file)
		relName = strings.TrimSuffix(relName, "/SKILL.md")
		basename := filepath.Base(relName)
		if basename == name {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			basenameMatches = append(basenameMatches, struct {
				path    string
				content string
				desc    string
			}{relName, string(content), desc})
		}
	}
	if len(basenameMatches) == 1 {
		return basenameMatches[0].content, nil
	}
	if len(basenameMatches) > 1 {
		var matches []string
		for _, m := range basenameMatches {
			matches = append(matches, fmt.Sprintf("%s: %s", m.path, m.desc))
		}
		return strings.Join(matches, "\n"), nil
	}

	var matches []string
	for _, file := range skillFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		frontmatter := parseFrontmatterMap(content)
		contentLower := strings.ToLower(string(content))
		nameLower := strings.ToLower(name)

		if strings.Contains(strings.ToLower(frontmatter["name"]), nameLower) ||
			strings.Contains(strings.ToLower(frontmatter["description"]), nameLower) ||
			strings.Contains(contentLower, nameLower) {
			relName, _ := filepath.Rel(skillsDir, file)
			relName = strings.TrimSuffix(relName, "/SKILL.md")
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			matches = append(matches, fmt.Sprintf("%s: %s", relName, desc))
		}
	}
	return strings.Join(matches, "\n"), nil
}

// findSnippets searches for code snippets in the .sgai/snippets directory.
// When language is empty, it lists available languages. When query is empty,
// it lists all snippets for the language. Otherwise, it searches for matching snippets.
//
//nolint:unparam // error is always nil by design - errors are handled by returning empty strings
func findSnippets(workingDir, language, query string) (string, error) {
	snippetsDir := filepath.Join(workingDir, ".sgai", "snippets")

	if language == "" && query == "" {
		entries, err := os.ReadDir(snippetsDir)
		if err != nil {
			return "", nil
		}
		var languages []string
		for _, entry := range entries {
			if entry.IsDir() {
				languages = append(languages, entry.Name())
			}
		}
		return strings.Join(languages, "\n"), nil
	}

	if language != "" && query == "" {
		langDir := filepath.Join(snippetsDir, language)
		entries, err := os.ReadDir(langDir)
		if err != nil {
			return "", nil
		}
		var snippets []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			desc := frontmatter["description"]
			if desc == "" {
				desc = "No description"
			}
			snippets = append(snippets, fmt.Sprintf("%s: %s", name, desc))
		}
		return strings.Join(snippets, "\n"), nil
	}

	if language != "" && query != "" {
		langDir := filepath.Join(snippetsDir, language)
		entries, err := os.ReadDir(langDir)
		if err != nil {
			return "", nil
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if name == query {
				return string(content), nil
			}
		}

		var prefixMatches []struct {
			name    string
			content string
			desc    string
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if strings.Contains(name, query) && name != query {
				desc := frontmatter["description"]
				if desc == "" {
					desc = "No description"
				}
				prefixMatches = append(prefixMatches, struct {
					name    string
					content string
					desc    string
				}{name, string(content), desc})
			}
		}
		if len(prefixMatches) == 1 {
			return prefixMatches[0].content, nil
		}
		if len(prefixMatches) > 1 {
			var matches []string
			for _, m := range prefixMatches {
				matches = append(matches, fmt.Sprintf("%s: %s", m.name, m.desc))
			}
			return strings.Join(matches, "\n"), nil
		}

		var matches []string
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(langDir, entry.Name()))
			if err != nil {
				continue
			}
			frontmatter := parseFrontmatterMap(content)
			queryLower := strings.ToLower(query)
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

			if strings.Contains(strings.ToLower(name), queryLower) ||
				strings.Contains(strings.ToLower(frontmatter["description"]), queryLower) {
				desc := frontmatter["description"]
				if desc == "" {
					desc = "No description"
				}
				matches = append(matches, fmt.Sprintf("%s: %s", name, desc))
			}
		}
		return strings.Join(matches, "\n"), nil
	}

	return "", nil
}

func updateWorkflowState(workingDir string, args updateWorkflowStateArgs) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		currentState = state.Workflow{
			Status:   state.StatusWorking,
			Progress: []state.ProgressEntry{},
		}
	}

	if currentState.Progress == nil {
		currentState.Progress = []state.ProgressEntry{}
	}

	statusPreserved := state.IsHumanPending(currentState.Status)

	if args.Status != "" && !statusPreserved {
		status := strings.Trim(string(args.Status), "\"'")
		if !slices.Contains(state.ValidStatuses, status) {
			return fmt.Sprintf("Error: Invalid status '%s'. Must be one of: %s", status, strings.Join(state.ValidStatuses, ", ")), nil
		}
		currentState.Status = status
	}

	currentState.Task = args.Task

	if args.AddProgress != "" {
		currentAgent := currentState.CurrentAgent
		if currentAgent == "" {
			currentAgent = "coordinator"
		}
		entry := state.ProgressEntry{
			Timestamp:   time.Now().Format(time.RFC3339),
			Agent:       currentAgent,
			Description: args.AddProgress,
		}
		currentState.Progress = append(currentState.Progress, entry)
	}

	if currentState.Status == state.StatusAgentDone || currentState.Status == state.StatusComplete {
		pendingCount := countPendingTodos(currentState, currentState.CurrentAgent)
		if pendingCount > 0 {
			return fmt.Sprintf("Error: Cannot transition to '%s' with %d pending TODO items. Please complete all TODO items first.", currentState.Status, pendingCount), nil
		}
	}

	if (currentState.Status == state.StatusComplete || currentState.Status == state.StatusAgentDone) && currentState.Task != "" {
		currentState.Task = ""
	}

	if err := state.Save(statePath, currentState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	response := "State updated successfully.\n"
	if statusPreserved {
		response = fmt.Sprintf("Status is currently '%s'. Waiting for human response. Your task and progress notes were updated but status was preserved.\n", currentState.Status)
	}
	response += fmt.Sprintf("  Status: %s\n", currentState.Status)
	if currentState.Task != "" {
		response += fmt.Sprintf("  Current task: %s\n", currentState.Task)
	}
	if args.AddProgress != "" {
		response += fmt.Sprintf("  Added progress note: %s\n", args.AddProgress)
	}
	response += fmt.Sprintf("  Total progress notes: %d", len(currentState.Progress))

	return response, nil
}

func sendMessage(workingDir, toAgent, body string) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		return "Error: Could not read state.json. Has the workflow been initialized?", nil
	}

	if currentState.Messages == nil {
		currentState.Messages = []state.Message{}
	}

	currentAgent := currentState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "coordinator"
	}

	fromAgent := currentAgent
	if currentState.CurrentModel != "" {
		fromAgent = currentState.CurrentModel
	}

	if currentState.VisitCounts == nil {
		currentState.VisitCounts = make(map[string]int)
	}

	knownAgents := make([]string, 0, len(currentState.VisitCounts))
	for agent := range currentState.VisitCounts {
		knownAgents = append(knownAgents, agent)
	}

	targetAgentName := extractAgentNameFromTarget(toAgent)
	if !slices.Contains(knownAgents, targetAgentName) {
		return fmt.Sprintf("Error: Agent '%s' is not in the workflow. Valid agents are: %s", toAgent, strings.Join(knownAgents, ", ")), nil
	}

	nextID := 1
	for _, msg := range currentState.Messages {
		if msg.ID >= nextID {
			nextID = msg.ID + 1
		}
	}

	message := state.Message{
		ID:        nextID,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		Body:      body,
		Read:      false,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	currentState.Messages = append(currentState.Messages, message)

	if err := state.Save(statePath, currentState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	result := fmt.Sprintf("Message sent successfully to %s.\nFrom: %s\nTo: %s\nBody: %s", toAgent, fromAgent, toAgent, body)
	if currentAgent != "coordinator" {
		result += "\n\nIMPORTANT: Since you are not the coordinator, consider yielding control back to the main loop using sgai_update_workflow_state({status: 'agent-done'}) after completing your message-related tasks."
	}
	return result, nil
}

func checkInbox(workingDir string) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		return "Error: Could not read state.json. Has the workflow been initialized?", nil
	}

	currentAgent := currentState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "coordinator"
	}

	currentModel := currentState.CurrentModel

	readBy := currentAgent
	if currentModel != "" {
		readBy = currentModel
	}

	var unreadMessages []state.Message
	for _, msg := range currentState.Messages {
		if messageMatchesRecipient(msg, currentAgent, currentModel) && !msg.Read {
			unreadMessages = append(unreadMessages, msg)
		}
	}

	if len(unreadMessages) == 0 {
		return "You have no messages.", nil
	}

	timestamp := time.Now().Format(time.RFC3339)
	for i := range currentState.Messages {
		if messageMatchesRecipient(currentState.Messages[i], currentAgent, currentModel) && !currentState.Messages[i].Read {
			currentState.Messages[i].Read = true
			currentState.Messages[i].ReadAt = timestamp
			currentState.Messages[i].ReadBy = readBy
		}
	}

	if err := state.Save(statePath, currentState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("You have %d message(s):\n\n", len(unreadMessages)))
	for i := 0; i < len(unreadMessages); i++ {
		msg := unreadMessages[i]
		result.WriteString(fmt.Sprintf("Message %d:\n  From: %s\n  Body: %s\n\n", i+1, msg.FromAgent, msg.Body))
	}

	return strings.TrimSpace(result.String()), nil
}

//nolint:unparam // error is always nil by design - errors are handled by returning user-friendly messages
func checkOutbox(workingDir string) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		return "Error: Could not read state.json. Has the workflow been initialized?", nil
	}

	currentAgent := currentState.CurrentAgent
	if currentAgent == "" {
		currentAgent = "coordinator"
	}

	currentModel := currentState.CurrentModel

	var unreadMessages []state.Message
	var readMessages []state.Message
	for _, msg := range currentState.Messages {
		if messageMatchesSender(msg, currentAgent, currentModel) {
			if msg.Read {
				readMessages = append(readMessages, msg)
			} else {
				unreadMessages = append(unreadMessages, msg)
			}
		}
	}

	if len(unreadMessages) == 0 && len(readMessages) == 0 {
		return "You have not sent any messages.", nil
	}

	var result strings.Builder

	if len(unreadMessages) > 0 {
		result.WriteString(fmt.Sprintf("Pending messages (%d):\n", len(unreadMessages)))
		for i, msg := range unreadMessages {
			subject := strings.Split(msg.Body, "\n")[0]
			result.WriteString(fmt.Sprintf("  %d. To: %s | Subject: %s\n", i+1, msg.ToAgent, subject))
		}
		result.WriteString("\n")
	}

	if len(readMessages) > 0 {
		result.WriteString(fmt.Sprintf("Delivered messages (%d):\n", len(readMessages)))
		for i, msg := range readMessages {
			subject := strings.Split(msg.Body, "\n")[0]
			readStatus := "Unread"
			if msg.ReadAt != "" {
				readStatus = fmt.Sprintf("Read at %s", msg.ReadAt)
			}
			result.WriteString(fmt.Sprintf("  %d. To: %s | Subject: %s | %s\n", i+1, msg.ToAgent, subject, readStatus))
		}
	}

	return strings.TrimSpace(result.String()), nil
}

//nolint:unparam // error is always nil by design - errors are handled by returning user-friendly messages
func peekMessageBus(workingDir string) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		return "Error: Could not read state.json. Has the workflow been initialized?", nil
	}

	if len(currentState.Messages) == 0 {
		return "No messages in the system.", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Total messages: %d\n\n", len(currentState.Messages)))

	for i := 0; i < len(currentState.Messages); i++ {
		msg := currentState.Messages[i]
		result.WriteString(fmt.Sprintf("Message %d (ID: %d):\n", i+1, msg.ID))
		result.WriteString(fmt.Sprintf("  From: %s\n", msg.FromAgent))
		result.WriteString(fmt.Sprintf("  To: %s\n", msg.ToAgent))
		if msg.Read {
			result.WriteString("  Status: read\n")
			if msg.ReadAt != "" {
				result.WriteString(fmt.Sprintf("  Read At: %s\n", msg.ReadAt))
			}
		} else {
			result.WriteString("  Status: pending\n")
		}
		result.WriteString(fmt.Sprintf("  Body: %s\n\n", msg.Body))
	}

	return strings.TrimSpace(result.String()), nil
}

func projectTodoWrite(workingDir string, todos []state.TodoItem) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		currentState = state.Workflow{
			Status:   state.StatusWorking,
			Progress: []state.ProgressEntry{},
		}
	}

	currentState.ProjectTodos = todos

	if err := state.Save(statePath, currentState); err != nil {
		return "", fmt.Errorf("failed to save state: %w", err)
	}

	return formatTodoList(todos), nil
}

func formatTodoList(todos []state.TodoItem) string {
	nonCompletedCount := 0
	for _, todo := range todos {
		if todo.Status != "completed" {
			nonCompletedCount++
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%d todos\n", nonCompletedCount))
	for _, todo := range todos {
		symbol := todoStatusSymbol(todo.Status)
		result.WriteString(fmt.Sprintf("→ %s %s (%s)\n", symbol, todo.Content, todo.Priority))
	}

	return strings.TrimSuffix(result.String(), "\n")
}

//nolint:unparam // error is always nil by design - errors are handled by returning "0 todos"
func projectTodoRead(workingDir string) (string, error) {
	statePath := filepath.Join(workingDir, ".sgai", "state.json")

	currentState, err := state.Load(statePath)
	if err != nil {
		return "0 todos", nil
	}

	return formatTodoList(currentState.ProjectTodos), nil
}

func extractAgentNameFromTarget(target string) string {
	if agentName, _, found := strings.Cut(target, ":"); found {
		return agentName
	}
	return target
}

func messageMatchesRecipient(msg state.Message, currentAgent, currentModel string) bool {
	if msg.ToAgent == currentAgent {
		return true
	}
	if currentModel != "" && msg.ToAgent == currentModel {
		return true
	}
	return false
}

func messageMatchesSender(msg state.Message, currentAgent, currentModel string) bool {
	if msg.FromAgent == currentAgent {
		return true
	}
	if currentModel != "" && msg.FromAgent == currentModel {
		return true
	}
	return false
}

func isSelfDriveMode(interactive string) bool {
	return interactive == "auto"
}

func buildUpdateWorkflowStateSchema(currentAgent string) (*jsonschema.Schema, string) {
	statusEnum := []any{"working", "agent-done"}
	description := "Update the workflow state file (.sgai/state.json). Use this tool to track your progress throughout your work. Update regularly after each major step. Examples: Set task when starting work, add progress notes as you complete steps, mark complete when done."

	if currentAgent == "coordinator" {
		statusEnum = append(statusEnum, "complete")
	}

	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"status": {
				Type:        "string",
				Enum:        statusEnum,
				Description: "Overall workflow status: 'working' (actively working - may need iteration) or 'agent-done' (agent's work done - needs goal verification) or 'complete' (goals verified as achieved). Valid values: working, agent-done, complete",
			},
			"task": {
				Type:        "string",
				Description: "Current task being worked on (e.g. 'Writing tests for auth endpoints'). Use empty string to clear. Be specific about what you're doing.",
			},
			"addProgress": {
				Type:        "string",
				Description: "Add a progress note to track what you've accomplished. This will be appended to the progress array. Use this frequently to document your steps.",
			},
		},
		Required: []string{"status", "task", "addProgress"},
	}

	return schema, description
}
