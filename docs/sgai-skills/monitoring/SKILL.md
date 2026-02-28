---
name: monitoring
description: Monitor sgai workspace status, events, progress, diffs, and workflow diagrams. Use when you need to observe what agents are doing, track progress, get the current state of all workspaces, subscribe to real-time updates via SSE, or inspect code changes.
compatibility: Requires a running sgai server. SSE streaming requires a client that supports Server-Sent Events.
---

# Monitoring

Monitor the sgai factory to understand what agents are doing and track their progress.

## Get Full Factory State

Returns all workspaces and their complete state.

**Endpoint:** `GET /api/v1/state`

```bash
curl -s $BASE_URL/api/v1/state | jq .
```

Response:
```json
{
  "workspaces": [
    {
      "name": "my-project",
      "dir": "/path/to/workspaces/my-project",
      "running": true,
      "needsInput": false,
      "inProgress": true,
      "pinned": false,
      "isRoot": false,
      "isFork": false,
      "hasSgai": true,
      "status": "working",
      "badgeClass": "badge-running",
      "badgeText": "Running",
      "currentAgent": "backend-go-developer",
      "currentModel": "anthropic/claude-sonnet-4-6",
      "task": "Writing authentication endpoints",
      "latestProgress": "Created JWT middleware",
      "cost": {
        "inputTokens": 45000,
        "outputTokens": 12000,
        "cacheCreationInputTokens": 8000,
        "cacheReadInputTokens": 32000
      },
      "events": [...],
      "messages": [...],
      "projectTodos": [...],
      "agentTodos": [...],
      "commits": [...],
      "log": [...],
      "pendingQuestion": null,
      "forks": []
    }
  ]
}
```

### Key State Fields

| Field | Description |
|-------|-------------|
| `running` | Session is active |
| `needsInput` | Blocked waiting for human response |
| `inProgress` | Work is actively happening |
| `status` | Workflow status: "working", "agent-done", "complete", etc. |
| `currentAgent` | Currently executing agent |
| `task` | Current task being worked on |
| `latestProgress` | Most recent progress note |
| `pendingQuestion` | Non-null when human input needed |
| `badgeText` | Human-readable status badge |

### Cost Tracking

```json
"cost": {
  "inputTokens": 45000,
  "outputTokens": 12000,
  "cacheCreationInputTokens": 8000,
  "cacheReadInputTokens": 32000
}
```

### Event Timeline

```json
"events": [
  {
    "timestamp": "2026-02-27T17:00:00Z",
    "formattedTime": "5:00 PM",
    "agent": "coordinator",
    "description": "Planning implementation",
    "showDateDivider": false,
    "dateDivider": ""
  }
]
```

### Agent Messages

```json
"messages": [
  {
    "id": 42,
    "fromAgent": "coordinator",
    "toAgent": "backend-go-developer",
    "body": "Please implement the authentication module",
    "subject": "Implement auth",
    "read": true,
    "readAt": "2026-02-27T17:00:01Z",
    "createdAt": "2026-02-27T17:00:00Z"
  }
]
```

## Get Single Workspace State

Equivalent to filtering state by workspace name:

```bash
curl -s $BASE_URL/api/v1/state | jq '.workspaces[] | select(.name == "my-project")'
```

Or use the MCP tool `get_workspace_state`:
```
get_workspace_state(workspace: "my-project")
```

## Real-Time Updates via SSE

Subscribe to state change notifications.

**Endpoint:** `GET /api/v1/signal`

```bash
curl -s -N $BASE_URL/api/v1/signal
```

Events emitted:
```
event: reload
data: {}

event: reload
data: {}
```

Each `reload` event means state has changed. Fetch `/api/v1/state` after receiving one.

### SSE with Reconnection

```bash
#!/bin/bash
while true; do
  curl -s -N "$BASE_URL/api/v1/signal" | while IFS= read -r line; do
    if [[ "$line" == "event: reload" ]]; then
      echo "State changed, fetching..."
      curl -s $BASE_URL/api/v1/state > /tmp/latest-state.json
    fi
  done
  echo "SSE disconnected, reconnecting..."
  sleep 2
done
```

## Get Workspace Diff

Get the current code changes (jj diff) for a workspace.

**Endpoint:** `GET /api/v1/workspaces/{name}/diff`

```bash
curl -s $BASE_URL/api/v1/workspaces/my-project/diff
```

Response:
```json
{
  "diff": "diff --git a/cmd/api/auth.go b/cmd/api/auth.go\nnew file mode 100644\n..."
}
```

## Get Workflow SVG Diagram

Get a visual diagram of the agent workflow.

**Endpoint:** `GET /api/v1/workspaces/{name}/workflow.svg`

```bash
# Save to file
curl -s $BASE_URL/api/v1/workspaces/my-project/workflow.svg > workflow.svg

# Or open directly in browser
open $BASE_URL/api/v1/workspaces/my-project/workflow.svg
```

Returns SVG with the agent flow graph. The current agent is highlighted.

## Monitor Agent Sequence

The `agentSequence` field shows the execution history:

```bash
curl -s $BASE_URL/api/v1/state | jq '.workspaces[0].agentSequence'
```

Response:
```json
[
  {
    "agent": "coordinator",
    "model": "anthropic/claude-opus-4-6",
    "elapsedTime": "2m 15s",
    "isCurrent": false
  },
  {
    "agent": "backend-go-developer",
    "model": "anthropic/claude-sonnet-4-6",
    "elapsedTime": "8m 42s",
    "isCurrent": true
  }
]
```

## Monitor Todos

Check what the agent has planned:

```bash
# Project-level todos (from TodoWrite)
curl -s $BASE_URL/api/v1/state | jq '.workspaces[0].projectTodos'

# Agent-level todos (current agent's todo list)
curl -s $BASE_URL/api/v1/state | jq '.workspaces[0].agentTodos'
```

Todo format:
```json
[
  {
    "id": "todo-1",
    "content": "Implement JWT authentication",
    "status": "in_progress",
    "priority": "high"
  }
]
```

## Monitor Commit History

```bash
curl -s $BASE_URL/api/v1/state | jq '.workspaces[0].commits'
```

Response:
```json
[
  {
    "changeId": "abc123",
    "commitId": "def456",
    "timestamp": "2026-02-27T17:00:00Z",
    "bookmarks": ["main"],
    "description": "cmd/api: add authentication endpoints",
    "graphChar": "◆"
  }
]
```

## Monitor Fork Status

For root workspaces, check their forks:

```bash
curl -s $BASE_URL/api/v1/state | jq '.workspaces[] | select(.isRoot) | .forks'
```

Fork entry:
```json
[
  {
    "name": "feature-auth",
    "dir": "/path/to/workspaces/feature-auth",
    "running": true,
    "needsInput": false,
    "inProgress": true,
    "pinned": false,
    "commitAhead": 3,
    "commits": [...],
    "summary": "Implementing auth"
  }
]
```
