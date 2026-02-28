---
name: using-sgai
description: Drive sgai (Software Garden AI) from any MCP-capable harness or AI agent. Covers the cyclical probe/poll/act workflow for managing AI software factory workspaces, sessions, and human interaction. Use this as the entrypoint when orchestrating sgai from Claude Code, Codex, or any AI harness.
compatibility: Requires a running sgai server (sgai serve). Works with any HTTP client or MCP-capable harness.
---

# Using sgai from an AI Harness

sgai is a software factory system that runs AI agents in workspaces. This skill teaches you to drive it via the HTTP API or MCP tools.

## Quick Start: Base URL

All examples assume `BASE_URL=http://127.0.0.1:PORT` where PORT is shown in the server startup log:
```
sgai serve listening on http://127.0.0.1:PORT
```

The MCP endpoint is at `/mcp/external` on the same server (e.g. `http://127.0.0.1:PORT/mcp/external`).

## The Cyclical Probe/Poll/Act Loop

The core pattern for driving sgai is a continuous loop:

```
LOOP:
  1. PROBE  → GET /api/v1/state          # Discover all workspaces + status
  2. CHECK  → pendingQuestion != null?   # Does any workspace need human input?
  3. ACT    → based on workspace status  # Start, steer, respond, or wait
  4. WAIT   → poll again after delay     # Repeat
```

### Step 1: Probe — Get Factory State

```bash
curl -s $BASE_URL/api/v1/state
```

Response shape:
```json
{
  "workspaces": [
    {
      "name": "my-project",
      "running": false,
      "needsInput": false,
      "inProgress": false,
      "status": "agent-done",
      "currentAgent": "coordinator",
      "task": "Planning implementation",
      "pendingQuestion": null
    }
  ]
}
```

Key fields to check per workspace:
- `running` — is a session active?
- `needsInput` — does the agent need a human response?
- `pendingQuestion` — non-null when human input is required
- `status` — current workflow status string
- `inProgress` — is work actively happening?

### Step 2: Check for Pending Questions

When `workspace.pendingQuestion != null`, the agent is blocked waiting for human input.

```json
{
  "pendingQuestion": {
    "questionId": "abc123def456",
    "type": "free-text",
    "agentName": "coordinator",
    "message": "Which approach should we take?",
    "questions": []
  }
}
```

Question types:
- `"free-text"` — respond with a text answer
- `"multi-choice"` — select from provided choices
- `"work-gate"` — approve to proceed (select approval text)

### Step 3: Act Based on Status

| Workspace State | Action |
|----------------|--------|
| `needsInput: true` | Call respond endpoint with answer |
| `running: false` and has goal | Start session |
| `running: true` | Monitor / optionally steer |
| Session complete | Check results, start next task |

### Step 4: Respond to Questions

```bash
# Free-text response
curl -s -X POST $BASE_URL/api/v1/workspaces/{name}/respond \
  -H "Content-Type: application/json" \
  -d '{"questionId": "abc123def456", "answer": "Use the microservice approach"}'

# Multi-choice response
curl -s -X POST $BASE_URL/api/v1/workspaces/{name}/respond \
  -H "Content-Type: application/json" \
  -d '{"questionId": "abc123def456", "selectedChoices": ["Option A"]}'
```

## Sub-skills

For detailed documentation on specific operations:

- [workspace-management](../workspace-management/SKILL.md) — Create, fork, delete, rename workspaces
- [session-control](../session-control/SKILL.md) — Start/stop sessions, steer agents
- [human-interaction](../human-interaction/SKILL.md) — Respond to questions and work gates
- [monitoring](../monitoring/SKILL.md) — List workspaces, get state, diffs, SVGs
- [knowledge](../knowledge/SKILL.md) — Agents, skills, snippets
- [compose](../compose/SKILL.md) — Compose wizard: state, save, preview, draft, templates
- [adhoc](../adhoc/SKILL.md) — Ad-hoc prompt start/stop/status

## MCP Interface

If using the MCP interface instead of HTTP, all tools are available at `/mcp/external`:

```bash
# List all 38 tools
npx mcporter list --http-url http://HOST:PORT/mcp/external --allow-http
```

Key MCP tools mirror the HTTP API:
- `list_workspaces` → GET /api/v1/state
- `start_session` → POST /api/v1/workspaces/{name}/start
- `respond_to_question` → POST /api/v1/workspaces/{name}/respond
- `wait_for_question` → polls + elicitation (MCP only)

## Real-Time Updates via SSE

Subscribe to state changes instead of polling:

```bash
curl -s -N $BASE_URL/api/v1/signal
# Emits: event: reload\ndata: {}\n\n
```

When you receive a `reload` event, re-fetch `/api/v1/state`.

## Common Workflow: Start a Project End-to-End

```bash
# 1. Create workspace
curl -X POST $BASE_URL/api/v1/workspaces \
  -d '{"name": "my-project"}'

# 2. Write a GOAL.md
curl -X PUT $BASE_URL/api/v1/workspaces/my-project/goal \
  -d '{"content": "# My Goal\n- [ ] Build the feature"}'

# 3. Start session in auto (self-drive) mode
curl -X POST $BASE_URL/api/v1/workspaces/my-project/start \
  -d '{"auto": true}'

# 4. Poll for completion
while true; do
  STATE=$(curl -s $BASE_URL/api/v1/state)
  NEEDS_INPUT=$(echo $STATE | jq '.workspaces[0].needsInput')
  if [ "$NEEDS_INPUT" = "true" ]; then
    # Handle question...
  fi
  sleep 5
done
```
