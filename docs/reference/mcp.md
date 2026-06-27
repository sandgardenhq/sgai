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

### `skills`

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

#### Handoff

Agents return control with `status` set to `agent-done`.

Input:

```json
{
  "status": "agent-done",
  "task": "",
  "addProgress": "Implementation complete"
}
```

Notes:

- Shared context and handoff notes belong in `.sgai/PROJECT_MANAGEMENT.md`.

### `project_todowrite` (coordinator only)

Write the project todo list to state.

### `project_todoread` (coordinator only)

Read the project todo list from state.

### `ask_user_question` (coordinator only)

Present one or more multiple-choice questions to the human partner.

Input:

- `questions`: array of `{ "question": string, "choices": string[], "multiSelect": boolean }`
