# Agent Aliases

Agent aliases let a workflow refer to an existing agent prompt under an alternate agent name.

In SGAI, aliases are configured in `GOAL.md` frontmatter under the `alias:` key. An alias maps an alias name to a *base agent* name.

The base agent name is then used when SGAI loads the agent definition from `.sgai/agent/<agent>.md`.

## Why use an alias?

Aliases are useful when the workflow should treat two agent names as separate roles, but both roles should use the same underlying agent definition.

The repository also documents a common pattern: use an alias to run the same role with a different model configuration.

## Configure an alias in `GOAL.md`

Add an `alias:` section to the YAML frontmatter.

Example (from `GOAL.example.md`):

```yaml
---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
alias:
  # "backend-go-developer-lite": "backend-go-developer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  # "backend-go-developer-lite": "anthropic/claude-haiku-4-5"
---
```

In this example:

1. `backend-go-developer-lite` is an alias.
2. `backend-go-developer` is the base agent.
3. The `models:` map can include a model entry for the alias name.

## Use an alias in the workflow flow

Once defined, an alias behaves like a regular agent name in `flow:`.

For example:

```yaml
flow: |
  "backend-go-developer-lite" -> "go-readability-reviewer"
```

## What gets inherited

The repository’s README describes alias behavior like this:

- The alias inherits the base agent’s prompt, tools, and snippets.
- The alias uses its own model configuration.

## How alias resolution works (implementation notes)

When SGAI builds flow messages, it resolves the base agent name before reading agent files and parsing snippets.

In `cmd/sgai/dag.go`, `buildFlowMessage`:

1. Accepts an `alias map[string]string` parameter.
2. Calls `resolveBaseAgent(alias, agent)` to resolve each agent name to a base agent name.
3. Loads agent markdown from `.sgai/agent/<base-agent>.md`.
