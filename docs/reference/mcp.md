# MCP server

`sgai` provides an MCP (Model Context Protocol) server that exposes workflow-management tools over HTTP.

`sgai` starts an MCP server per workspace and passes its URL to agent processes.

## Connect

The agent process receives the MCP URL in `SGAI_MCP_URL`.

The server listens on `127.0.0.1` and serves MCP requests under the `/mcp` path.

## Agent identity header

Each MCP request can include an `X-SGAI-Agent-Identity` header.

The header format is:

```text
<name>|<model>|<variant>
```

Only `<name>` is required.

If the header is missing (or `<name>` is empty), the server treats the agent name as `coordinator`.

## Working directory

The MCP server uses a workspace directory as its working directory.

## Tools

### `find_skills`

- Input: `{ "name": "..." }` (optional)
- Behavior:
  - When `name` is empty, lists available skills.
  - When `name` matches a skill name, returns the skill content.

### `find_snippets`

- Input: `{ "language": "...", "query": "..." }` (both optional)
- Behavior:
  - When `language` and `query` are empty, lists available languages.
  - When `language` is set and `query` is empty, lists snippets for that language.
  - When `language` and `query` are set, returns the matching snippet content or matching snippet list.

### `update_workflow_state`

Update `.sgai/state.json`.

Input fields:

- `status`
- `task`
- `addProgress`

#### Status values depend on the current agent

`update_workflow_state.status` only allows:

- `working`
- `agent-done`

When the current agent is `coordinator`, it also allows:

- `complete`

#### TODO guardrails

Transitions to `agent-done` or `complete` fail if there are pending TODO items.

### `send_message`

Send a message to another agent.

Input:

```json
{
  "toAgent": "name-or-model-id",
  "body": "message body"
}
```

Notes:

- The target must be an agent in the workflow.
- When `currentModel` is set in state, `send_message` uses that as `fromAgent`.

### `check_inbox`

Return unread messages for the current agent (and current model, if set) and mark them as read.

### `check_outbox`

Return messages sent by the current agent (and current model, if set).

### `peek_message_bus` (coordinator only)

Return all messages in the system (pending and read), in reverse chronological order.

### `project_todowrite` (coordinator only)

Write the project todo list to state.

### `project_todoread` (coordinator only)

Read the project todo list from state.

### `ask_user_question` (coordinator only)

Present one or more multiple-choice questions to the human partner.

Input:

- `questions`: array of `{ "question": string, "choices": string[], "multiSelect": boolean }`

