# Workflow state (`.sgai/state.json`)

`sgai` persists workflow state as JSON in `.sgai/state.json` inside the project directory.

This file is read and written by the `sgai` CLI and the `sgai serve` web server.

## File location

- Path: `.sgai/state.json`

## Status values

The `status` field in `.sgai/state.json` uses these values:

- `working`
- `agent-done`
- `complete`

### Coordinator-only statuses in MCP

The MCP tool `update_workflow_state` uses a per-agent JSON schema.

- When the current agent is `coordinator`, the schema allows `complete`.
- When the current agent is not `coordinator`, the schema only allows `working` and `agent-done`.

## Human interaction

`sgai` uses structured, multi-choice questions for human input.

- The coordinator communicates with the human partner through `ask_user_question`.
- The workflow state file stores the active question in `multiChoiceQuestion`.

## Workflow object shape

`state.json` stores a JSON object with fields used by the CLI, web UI, and MCP tools.

Common fields include:

- `status` (string)
- `task` (string)
- `progress` (array of objects with `timestamp`, `agent`, `description`)
- `multiChoiceQuestion` (object with `questions`, each with `question`, `choices`, `multiSelect`)
- `messages` (array of inter-agent messages)
- `visitCounts` (object map of agent name to integer)
- `currentAgent` (string)
- `todos` (array of todo items)
- `projectTodos` (array of todo items)
- `agentSequence` (array with `agent`, `startTime`, `isCurrent`)
- `sessionId` (string)
- `cost` (object with `totalCost`, `totalTokens`, and `byAgent`)
- `modelStatuses` (object map of model ID to status string)
- `currentModel` (string, format `agentName:modelSpec`)

## Self-drive lock (`interactiveAutoLock`)

`state.json` may include an `interactiveAutoLock` boolean.

- When `interactiveAutoLock` is `true`, self-drive stays enabled across workflow resets that reinitialize the in-memory workflow state.
- The web server persists changes to this field by saving `.sgai/state.json`.

## Messages

A message entry includes:

- `id` (number)
- `fromAgent` (string)
- `toAgent` (string)
- `body` (string)
- `read` (boolean)
- `readAt` (string)
- `readBy` (string)
- `createdAt` (string)

## TODO items

A todo item includes:

- `id` (string)
- `content` (string)
- `status` (string)
- `priority` (string)

