# Workflow state (`.sgai/state.json`)

`sgai` persists workflow state as JSON in `.sgai/state.json` inside the project directory.

## File location

- Path: `.sgai/state.json`

## Status values

The `status` field can contain these values:

- `working`
- `agent-done`
- `complete`
- `human-communication`
- `waiting-for-human`

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

