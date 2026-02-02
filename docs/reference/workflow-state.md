# Workflow state (`.sgai/state.json`)

`sgai` persists workflow state as JSON in `.sgai/state.json` inside the project directory.

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

`sgai` uses structured multi-choice questions for human input.

- The skeleton coordinator instructions describe the coordinator as the only agent that can communicate with the human partner via `ask_user_question`.
- The `set-workflow-state` skill documentation removes the `human-communication` status and removes the `humanMessage` field; it adds a section stating to use `ask_user_question` for human communication.

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

