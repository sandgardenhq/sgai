# MCP server

`sgai` provides an MCP server that exposes workflow-management tools over stdio.

## Run

```sh
sgai mcp
```

## Working directory

`sgai mcp` reads the working directory from `SGAI_MCP_WORKING_DIRECTORY`.

If `SGAI_MCP_WORKING_DIRECTORY` is not set, the default working directory is `.`.

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
- `humanMessage` (string)

#### Status values depend on the current agent

`update_workflow_state.status` only allows:

- `working`
- `agent-done`

When the current agent is `coordinator`, it also allows:

- `complete`
- `human-communication`

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

Behavior:

- Writes the questions to `.sgai/state.json` as `multiChoiceQuestion`.
- Sets `humanMessage` to the first question.
- Sets `status` to `human-communication`.
