# Workflow state (`.sgai/state.json`)

`sgai` persists workflow state as JSON in `.sgai/state.json` inside the project directory.

## File location

- Path: `.sgai/state.json`

## Status values

The `status` field in `.sgai/state.json` uses these values:

- `working`
- `agent-done`
- `complete`
- `human-communication`
- `waiting-for-human`

### Coordinator-only statuses in MCP

The MCP tool `update_workflow_state` uses a per-agent JSON schema.

- When the current agent is `coordinator`, the schema allows `complete` and `human-communication`.
- When the current agent is not `coordinator`, the schema only allows `working` and `agent-done`.

## Workflow object shape

`state.json` stores a JSON object with fields used by the CLI, web UI, and MCP tools.

Common fields include:

- `status` (string)
- `task` (string)
- `progress` (array of objects with `timestamp`, `agent`, `description`)
- `humanMessage` (string)
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

## Human communication: multiple-choice questions

When a coordinator asks a multiple-choice question (via the MCP tool `ask_user_question`), it writes a `multiChoiceQuestion` object to state.

Each question item includes:

- `question` (string)
- `choices` (string array)
- `multiSelect` (boolean)

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

