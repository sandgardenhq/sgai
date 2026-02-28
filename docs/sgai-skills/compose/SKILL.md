---
name: compose
description: Use the sgai compose wizard to create and manage GOAL.md files for workspaces. Supports reading compose state, updating wizard fields, saving drafts, previewing generated GOAL.md content, browsing workflow templates, and writing the final GOAL.md to the workspace.
compatibility: Requires a running sgai server. The compose wizard maintains in-memory state per workspace session.
---

# Compose Wizard

The compose wizard is a guided interface for creating GOAL.md files. It maintains state about the workflow configuration, agent assignments, tech stack, and goals.

## Get Compose State

**Endpoint:** `GET /api/v1/compose?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/compose?workspace=my-project"
```

Response:
```json
{
  "workspace": "my-project",
  "state": {
    "flow": "\"coordinator\" -> \"backend-go-developer\" -> \"go-readability-reviewer\"",
    "models": {
      "coordinator": "anthropic/claude-opus-4-6",
      "backend-go-developer": "anthropic/claude-sonnet-4-6"
    },
    "goals": "- [ ] Build a REST API for user management\n- [ ] Write tests with 80%+ coverage",
    "completionGateScript": "make test"
  },
  "wizard": {
    "currentStep": 3,
    "fromTemplate": "go-backend",
    "description": "User management REST API",
    "techStack": ["go", "postgresql"],
    "safetyAnalysis": false,
    "completionGate": "make test"
  },
  "techStackItems": [
    {"id": "go", "name": "Go", "selected": true},
    {"id": "react", "name": "React", "selected": false},
    {"id": "postgresql", "name": "PostgreSQL", "selected": true}
  ],
  "flowError": ""
}
```

### Compose State Fields

**`state`** — the raw GOAL.md configuration:
- `flow` — agent flow definition (quoted agent names with `->`)
- `models` — map of agent name to model ID
- `goals` — markdown goal checklist
- `completionGateScript` — shell command to verify completion

**`wizard`** — user-facing wizard state:
- `currentStep` — which step in the wizard (1-based)
- `fromTemplate` — template ID if started from template
- `description` — project description
- `techStack` — selected technology stack items
- `safetyAnalysis` — whether STPA analysis is included
- `completionGate` — gate script for completion

## Get Compose Templates

**Endpoint:** `GET /api/v1/compose/templates`

```bash
curl -s "$BASE_URL/api/v1/compose/templates"
```

Response:
```json
{
  "templates": [
    {
      "id": "go-backend",
      "name": "Go Backend API",
      "description": "REST API with Go, including code review and testing agents.",
      "icon": "🔧",
      "agents": [
        {"name": "coordinator", "model": "anthropic/claude-opus-4-6"},
        {"name": "backend-go-developer", "model": "anthropic/claude-sonnet-4-6"}
      ],
      "flow": "\"backend-go-developer\" -> \"go-readability-reviewer\""
    }
  ]
}
```

## Preview Generated GOAL.md

Preview what the GOAL.md would look like without saving.

**Endpoint:** `GET /api/v1/compose/preview?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/compose/preview?workspace=my-project"
```

Response:
```json
{
  "content": "---\nflow: |\n  \"backend-go-developer\" -> \"go-readability-reviewer\"\nmodels:\n  coordinator: anthropic/claude-opus-4-6\n  backend-go-developer: anthropic/claude-sonnet-4-6\ncompletionGateScript: make test\n---\n\n- [ ] Build a REST API\n",
  "flowError": "",
  "etag": "\"abc123def456\""
}
```

- `content` — the full GOAL.md content that would be saved
- `flowError` — non-empty if the flow definition has syntax errors
- `etag` — current file ETag for optimistic concurrency

## Save Draft (In-Memory)

Save the compose state in memory without writing GOAL.md.

**Endpoint:** `POST /api/v1/compose/draft?workspace={name}`

```bash
curl -X POST "$BASE_URL/api/v1/compose/draft?workspace=my-project" \
  -H "Content-Type: application/json" \
  -d '{
    "state": {
      "flow": "\"coordinator\" -> \"backend-go-developer\"",
      "models": {"coordinator": "anthropic/claude-opus-4-6"},
      "goals": "- [ ] Build authentication\n",
      "completionGateScript": ""
    },
    "wizard": {
      "currentStep": 2,
      "fromTemplate": "",
      "description": "Auth system",
      "techStack": ["go"],
      "safetyAnalysis": false,
      "completionGate": ""
    }
  }'
```

Response:
```json
{"saved": true}
```

Use this to update the wizard state incrementally as the user fills in fields.

## Save Compose (Write GOAL.md)

Write the current compose state to GOAL.md.

**Endpoint:** `POST /api/v1/compose?workspace={name}`

```bash
# Simple save
curl -X POST "$BASE_URL/api/v1/compose?workspace=my-project"

# With optimistic concurrency (prevent overwriting concurrent changes)
ETAG=$(curl -s "$BASE_URL/api/v1/compose/preview?workspace=my-project" | jq -r '.etag')
curl -X POST "$BASE_URL/api/v1/compose?workspace=my-project" \
  -H "If-Match: $ETAG"
```

Response (201 Created):
```json
{
  "saved": true,
  "workspace": "my-project"
}
```

Errors:
- `412 Precondition Failed` — GOAL.md was modified since the etag was fetched

## Complete Compose Workflow

```bash
# 1. Start from a template
curl -X POST "$BASE_URL/api/v1/compose/draft?workspace=my-project" \
  -H "Content-Type: application/json" \
  -d '{
    "state": {
      "flow": "\"backend-go-developer\" -> \"go-readability-reviewer\"",
      "models": {
        "coordinator": "anthropic/claude-opus-4-6",
        "backend-go-developer": "anthropic/claude-sonnet-4-6",
        "go-readability-reviewer": "anthropic/claude-opus-4-6"
      },
      "goals": "- [ ] Build REST API\n- [ ] Add authentication\n- [ ] Write tests\n",
      "completionGateScript": "make test"
    },
    "wizard": {
      "currentStep": 4,
      "description": "User management API",
      "techStack": ["go", "postgresql"],
      "safetyAnalysis": false,
      "completionGate": "make test"
    }
  }'

# 2. Preview to verify
curl -s "$BASE_URL/api/v1/compose/preview?workspace=my-project" | jq '.content'

# 3. Save to GOAL.md
curl -X POST "$BASE_URL/api/v1/compose?workspace=my-project"

# 4. Start the session
curl -X POST $BASE_URL/api/v1/workspaces/my-project/start -d '{"auto": true}'
```
