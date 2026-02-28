---
name: knowledge
description: Access agents, skills, and code snippets available in sgai workspaces. Use when you need to discover what agents are defined in a workspace, browse available skills, get skill instructions, find code snippets by language, or retrieve snippet content for a specific task.
compatibility: Requires a running sgai server. Knowledge is workspace-scoped (agents/skills/snippets from the workspace .sgai/ directory).
---

# Knowledge

sgai workspaces contain agent definitions, skills, and code snippets in their `.sgai/` directory. Use these endpoints to discover and retrieve that knowledge.

## List Available Agents

**Endpoint:** `GET /api/v1/agents?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/agents?workspace=my-project"
```

Response:
```json
{
  "agents": [
    {
      "name": "coordinator",
      "description": "Coordinates the work flow between specialized agents."
    },
    {
      "name": "backend-go-developer",
      "description": "Expert Go backend developer for building production-quality APIs."
    },
    {
      "name": "react-developer",
      "description": "Frontend developer specializing in React components."
    }
  ]
}
```

Agents are loaded from `.sgai/agent/*.md` files in the workspace.

## List Available Skills

Skills are organized into categories based on their directory structure.

**Endpoint:** `GET /api/v1/skills?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/skills?workspace=my-project"
```

Response:
```json
{
  "categories": [
    {
      "name": "coding-practices",
      "skills": [
        {
          "name": "go-code-review",
          "fullPath": "coding-practices/go-code-review",
          "description": "Go code review checklist based on official Go style guides."
        },
        {
          "name": "react-best-practices",
          "fullPath": "coding-practices/react-best-practices",
          "description": "React and Next.js performance optimization guidelines."
        }
      ]
    },
    {
      "name": "General",
      "skills": [
        {
          "name": "test-driven-development",
          "fullPath": "test-driven-development",
          "description": "Write tests first, watch them fail, then implement."
        }
      ]
    }
  ]
}
```

Skills are loaded from `.sgai/skills/*/SKILL.md` files.

## Get Skill Detail

**Endpoint:** `GET /api/v1/skills/{path...}?workspace={name}`

```bash
# Get a skill at a specific path
curl -s "$BASE_URL/api/v1/skills/coding-practices/go-code-review?workspace=my-project"

# Get a top-level skill
curl -s "$BASE_URL/api/v1/skills/test-driven-development?workspace=my-project"
```

Response:
```json
{
  "name": "go-code-review",
  "fullPath": "coding-practices/go-code-review",
  "content": "<rendered HTML of skill content>",
  "rawContent": "# Go Code Review\n\n## When to Use\n..."
}
```

- `content` — HTML-rendered markdown
- `rawContent` — raw markdown without frontmatter

## List Code Snippets

**Endpoint:** `GET /api/v1/snippets?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/snippets?workspace=my-project"
```

Response:
```json
{
  "languages": [
    {
      "name": "go",
      "snippets": [
        {
          "name": "HTTP Server with Routes",
          "fileName": "http-server-routes",
          "fullPath": "go/http-server-routes",
          "description": "Set up an HTTP server with Go 1.22+ enhanced routing.",
          "language": "go"
        }
      ]
    },
    {
      "name": "typescript",
      "snippets": [
        {
          "name": "SSE Client Hook",
          "fileName": "sse-client",
          "fullPath": "typescript/sse-client",
          "description": "React hook for Server-Sent Events with reconnection.",
          "language": "typescript"
        }
      ]
    }
  ]
}
```

## List Snippets by Language

**Endpoint:** `GET /api/v1/snippets/{lang}?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/snippets/go?workspace=my-project"
```

Response:
```json
{
  "language": "go",
  "snippets": [
    {
      "name": "HTTP Server with Routes",
      "fileName": "http-server-routes",
      "fullPath": "go/http-server-routes",
      "description": "Set up an HTTP server with Go 1.22+ enhanced routing.",
      "language": "go"
    }
  ]
}
```

## Get Snippet Detail

**Endpoint:** `GET /api/v1/snippets/{lang}/{fileName}?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/snippets/go/http-server-routes?workspace=my-project"
```

Response:
```json
{
  "name": "HTTP Server with Routes",
  "fileName": "http-server-routes",
  "language": "go",
  "description": "Set up an HTTP server with Go 1.22+ enhanced routing.",
  "whenToUse": "When defining HTTP routes using net/http ServeMux in Go 1.22+",
  "content": "package main\n\nimport \"net/http\"\n\nfunc main() {\n..."
}
```

## List Available Models

**Endpoint:** `GET /api/v1/models?workspace={name}`

```bash
curl -s "$BASE_URL/api/v1/models?workspace=my-project"
```

Response:
```json
{
  "models": [
    {"id": "anthropic/claude-opus-4-6", "name": "anthropic/claude-opus-4-6"},
    {"id": "anthropic/claude-sonnet-4-6", "name": "anthropic/claude-sonnet-4-6"},
    {"id": "openai/gpt-4o", "name": "openai/gpt-4o"}
  ],
  "defaultModel": "anthropic/claude-opus-4-6"
}
```

The `defaultModel` is the coordinator's model as defined in GOAL.md frontmatter.

## Workspace Query Parameter

All knowledge endpoints accept an optional `?workspace=name` query parameter:
- If omitted, uses the first workspace found
- If provided, scopes results to that workspace's `.sgai/` directory

```bash
# Explicit workspace
curl -s "$BASE_URL/api/v1/agents?workspace=my-project"

# First workspace (implicit)
curl -s "$BASE_URL/api/v1/agents"
```

## Knowledge Directory Structure

The workspace's `.sgai/` directory layout:
```
.sgai/
  agent/
    coordinator.md       # agent definition files
    backend-go-developer.md
  skills/
    test-driven-development/
      SKILL.md           # skill definition
    coding-practices/
      go-code-review/
        SKILL.md
  snippets/
    go/
      http-server-routes.go  # code snippets
    typescript/
      sse-client.ts
```
