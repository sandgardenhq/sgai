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
- `visitCounts` (object map of agent name to integer)
- `todos` (array of todo items)
- `projectTodos` (array of todo items)
- `agentSequence` (array with `agent`, `startTime`, `isCurrent`)
- `sessionId` (string)
- `cost` (object with `totalCost`, `totalTokens`, and `byAgent`)

## Handoffs

Agents return control by setting `status: "agent-done"` through the MCP `update_workflow_state` tool. Shared work notes and handoffs belong in `.sgai/PROJECT_MANAGEMENT.md`, not `.sgai/state.json`.

## TODO items

A todo item includes:

- `id` (string)
- `content` (string)
- `status` (string)
- `priority` (string)
