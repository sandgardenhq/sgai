# Workflow state (`.sgai/state.json`)

`sgai` persists workflow state as JSON in `.sgai/state.json` inside the project directory.

## File location

- Path: `.sgai/state.json`

## Status values

The `status` field in `.sgai/state.json` uses these values:

- `working`
- `agent-done`
- `complete`
- `waiting-for-human`

### `waiting-for-human`

`waiting-for-human` indicates the workflow is paused until a human responds.

At this point, you typically see at least one of the following in state:

- `multiChoiceQuestion` is present (structured questions)
- `humanMessage` is non-empty (a free-form prompt for the human)

While the workflow is waiting for a response, other fields (like `task` and `progress`) can still update without changing `status`.

### Coordinator-only statuses in MCP

The MCP tool `update_workflow_state` uses a per-agent JSON schema.

- When the current agent is `coordinator`, the schema allows `complete`.
- When the current agent is not `coordinator`, the schema only allows `working` and `agent-done`.

`update_workflow_state` does not set `waiting-for-human`. The workflow enters `waiting-for-human` via the coordinator's human-interaction tools (for example, `ask_user_question`).

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
- `humanMessage` (string)
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

