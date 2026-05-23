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
    "description": "User management REST API",
    "completionGate": "make test",
    "safetyAnalysis": false,
    "retrospective": false,
    "agents": [
      {"name": "coordinator", "selected": true, "model": "openai/gpt-5.5 (xhigh)"},
      {"name": "go", "selected": true, "model": "openai/gpt-5.5 (low)"}
    ],
    "flow": "\"go\"",
    "tasks": "- [ ] Build a REST API for user management\n- [ ] Write tests with 80%+ coverage"
  },
  "wizard": {
    "currentStep": 3,
    "fromTemplate": "backend",
    "description": "User management REST API",
    "techStack": ["go"],
    "safetyAnalysis": false,
    "completionGate": "make test"
  },
  "techStackItems": [
    {"id": "go", "name": "Go", "selected": true},
    {"id": "react", "name": "React", "selected": false},
    {"id": "shell", "name": "Shell/Bash", "selected": false}
  ],
  "flowError": ""
}
```

### Compose State Fields

**`state`** — the raw GOAL.md configuration:
- `description` — markdown description that becomes the GOAL.md body before tasks
- `completionGate` — shell command to verify completion
- `safetyAnalysis` — whether to include Safety Analysis guidance in the GOAL.md body
- `retrospective` — whether to enable retrospective flow metadata
- `agents` — array of `{name, selected, model}` entries used to generate the `models:` frontmatter
- `flow` — agent flow definition (quoted agent names with `->`)
- `tasks` — markdown goal checklist written under `## Tasks`

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
      "id": "backend",
      "name": "Go Development",
      "description": "Go implementation and review wrapper",
      "icon": "🔧",
      "agents": [
        {"name": "coordinator", "selected": true, "model": "openai/gpt-5.5 (xhigh)"},
        {"name": "go", "selected": true, "model": "openai/gpt-5.5 (low)"}
      ],
      "flow": "\"go\""
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
  "content": "---\nflow: |\n  \"go\"\nmodels:\n  \"coordinator\": \"openai/gpt-5.5 (xhigh)\"\n  \"go\": \"openai/gpt-5.5 (low)\"\ncompletionGateScript: make test\n---\n\nUser management REST API\n\n## Tasks\n\n- [ ] Build a REST API\n",
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
      "description": "Auth system",
      "completionGate": "",
      "safetyAnalysis": false,
      "retrospective": false,
      "agents": [
        {"name": "coordinator", "selected": true, "model": "openai/gpt-5.5 (xhigh)"},
        {"name": "go", "selected": true, "model": "openai/gpt-5.5 (low)"}
      ],
      "flow": "\"go\"",
      "tasks": "- [ ] Build authentication\n"
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
      "description": "User management API",
      "completionGate": "make test",
      "safetyAnalysis": false,
      "retrospective": false,
      "agents": [
        {"name": "coordinator", "selected": true, "model": "openai/gpt-5.5 (xhigh)"},
        {"name": "go", "selected": true, "model": "openai/gpt-5.5 (low)"}
      ],
      "flow": "\"go\"",
      "tasks": "- [ ] Build REST API\n- [ ] Add authentication\n- [ ] Write tests\n"
    },
    "wizard": {
      "currentStep": 4,
      "description": "User management API",
      "techStack": ["go"],
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
