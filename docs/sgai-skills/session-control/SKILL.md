---
name: session-control
description: Start and stop agentic sessions in sgai workspaces. Use when you need to launch AI agent sessions or halt running sessions.
compatibility: Requires a running sgai server with workspaces that have a GOAL.md configured.
---

# Session Control

Sessions are the running instances of AI agents working on a workspace. Each workspace can have at most one running session.

## Start a Session

**Endpoint:** `POST /api/v1/workspaces/{name}/start`

```bash
# Brainstorming mode (interactive, pauses for human input)
curl -X POST $BASE_URL/api/v1/workspaces/my-project/start \
  -H "Content-Type: application/json" \
  -d '{"auto": false}'

# Self-drive mode (fully autonomous, no pausing)
curl -X POST $BASE_URL/api/v1/workspaces/my-project/start \
  -H "Content-Type: application/json" \
  -d '{"auto": true}'
```

Request:
```json
{"auto": true}
```

Response:
```json
{
  "name": "my-project",
  "status": "running",
  "running": true,
  "message": "session started"
}
```

### Session Modes

| Mode | `auto` | Description |
|------|--------|-------------|
| Brainstorming | `false` | Pauses at key decision points to ask for input |
| Self-drive | `true` | Runs fully autonomously without interruption |
| Continuous | (auto-detected) | If workspace has a `continuous.md` prompt, uses continuous mode |

Notes:
- Root workspaces cannot start agentic work (only forks/standalone can)
- If already running, returns `"message": "session already running"`

## Stop a Session

**Endpoint:** `POST /api/v1/workspaces/{name}/stop`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/stop
```

Response:
```json
{
  "name": "my-project",
  "status": "stopped",
  "running": false,
  "message": "session stopped"
}
```

If already stopped: `"message": "session already stopped"`

## Check Running Status

Use the full state endpoint to check session status:

```bash
STATE=$(curl -s $BASE_URL/api/v1/state)
RUNNING=$(echo $STATE | jq '.workspaces[] | select(.name=="my-project") | .running')
TASK=$(echo $STATE | jq -r '.workspaces[] | select(.name=="my-project") | .task')
```

Key workspace state fields for session monitoring:

| Field | Type | Description |
|-------|------|-------------|
| `running` | bool | Is session active? |
| `inProgress` | bool | Is work actively happening? |
| `task` | string | Current task description |
| `status` | string | Workflow status string |
| `latestProgress` | string | Most recent progress note |

## Open in OpenCode (Local Only)

Open the workspace in OpenCode terminal (localhost connections only).

**Endpoint:** `POST /api/v1/workspaces/{name}/open-opencode`

```bash
curl -X POST http://localhost:PORT/api/v1/workspaces/my-project/open-opencode
```

Response:
```json
{
  "opened": true,
  "message": "opened in opencode"
}
```

Errors:
- `403` — remote connections not allowed (localhost only)
- `409` — session not running

## Open in Editor

Open the workspace directory in the configured editor (VS Code, etc.).

**Endpoint:** `POST /api/v1/workspaces/{name}/open-editor`

```bash
curl -X POST $BASE_URL/api/v1/workspaces/my-project/open-editor
```

Response:
```json
{
  "opened": true,
  "editor": "code",
  "message": "opened in editor"
}
```

### Open Specific Files

```bash
# Open GOAL.md in editor
curl -X POST $BASE_URL/api/v1/workspaces/my-project/open-editor/goal

# Open PROJECT_MANAGEMENT.md in editor
curl -X POST $BASE_URL/api/v1/workspaces/my-project/open-editor/project-management
```
