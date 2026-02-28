# sgai Skills

Skills for driving [sgai](https://github.com/sandgardenhq/sgai) (Software Garden AI) from any MCP-capable harness or AI agent.

## Entrypoint

Start here: **[using-sgai/SKILL.md](using-sgai/SKILL.md)**

This skill explains the cyclical probe/poll/act workflow for orchestrating sgai from Claude Code, Codex, or any AI harness that supports skills or HTTP.

## Sub-skills

| Skill | Description |
|-------|-------------|
| [workspace-management](workspace-management/SKILL.md) | Create, fork, delete, rename workspaces |
| [session-control](session-control/SKILL.md) | Start/stop sessions, steer agents |
| [human-interaction](human-interaction/SKILL.md) | Respond to questions and work gates |
| [monitoring](monitoring/SKILL.md) | List workspaces, get state, diffs, SVGs |
| [knowledge](knowledge/SKILL.md) | Agents, skills, snippets, models |
| [compose](compose/SKILL.md) | Compose wizard: state, save, preview, draft, templates |
| [adhoc](adhoc/SKILL.md) | Ad-hoc prompt start/stop/status |

## MCP Interface

sgai also exposes a full MCP server at `/mcp/external` with 38 tools that mirror the HTTP API:

```bash
npx mcporter list --http-url http://127.0.0.1:PORT/mcp/external --allow-http
```

## Quick Start

```bash
# 1. Start sgai server
sgai serve -listen-addr 127.0.0.1:PORT ./workspaces

# 2. Create a workspace
curl -X POST http://localhost:PORT/api/v1/workspaces -d '{"name": "my-project"}'

# 3. Write a goal
curl -X PUT http://localhost:PORT/api/v1/workspaces/my-project/goal \
  -d '{"content": "- [ ] Build something great"}'

# 4. Start in auto mode
curl -X POST http://localhost:PORT/api/v1/workspaces/my-project/start -d '{"auto": true}'

# 5. Monitor progress
curl -s http://localhost:PORT/api/v1/state | jq '.workspaces[0].latestProgress'
```

## Specification

These skills conform to the [agentskills.io specification](https://agentskills.io/specification).
