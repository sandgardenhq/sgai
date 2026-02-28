---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "go-readability-reviewer"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] I need a way to drive sgai from inside Claude Code, or Codex, or any other MCP / Skill powered harness
  - [x] Create a MCP interface to sgai so that other harnesses can drive it
    - [x] Add external MCP endpoint (`/mcp/external`) to `sgai serve` using streamable HTTP
    - [x] 35 MCP tools with full parity to the web UI HTTP API (workspace lifecycle, session control, human interaction, monitoring, knowledge, compose, adhoc, list_models)
    - [x] Service layer refactor: extract business logic from HTTP handlers into shared unexported functions in `cmd/sgai/`
    - [x] MCP elicitation support: proactively push pending questions to harness via `elicitation/create` (form mode)
    - [x] `make build` and `make lint` pass
  - [x] And as alternative to MCP, create a set of skills for sgai so that other harnesses can drive it
    - [x] `docs/sgai-skills/using-sgai.md` entrypoint with cyclical probing/polling loop instructions
    - [x] Sub-skill docs: workspace-management, session-control, human-interaction, monitoring, knowledge, compose, adhoc
    - [x] Each sub-skill has HTTP endpoint details, request/response examples, workflow instructions
    - [x] README.md links to the entrypoint skill
    - [x] the skills must conform to https://agentskills.io/specification
- [x] syncing with upstream may have changed lot of assumptions, please, revalidate these changes, use https://github.com/steipete/mcporter to validate the MCP server

- [x] based on sgai.json, follow "Create PR" to update  https://github.com/sandgardenhq/sgai/pull/299 correctly

- [x] Update README.md with instructions on how to use the MCP and the new skills set
  - [x] Add example configuration on how to configure the MCP in OpenCode https://opencode.ai/docs/mcp-servers/

- [x] it seems that the MCP port number is not stable, and therefore it gets hard to configure it correctly.
