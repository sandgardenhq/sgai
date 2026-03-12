# Agent Aliases

Agent aliases let a workflow use an existing agent definition under an alternate agent name.

In practice, an alias is a name-to-name mapping:

* **Alias name**: the name used in your `flow:`
* **Base agent name**: the agent name that SGAI uses to load the agent definition from `.sgai/agent/<agent>.md`

In SGAI, aliases are configured in `GOAL.md` frontmatter under the `alias:` key.

An alias maps an alias name to a *base agent* name.

The base agent name is then used when SGAI loads the agent definition from `.sgai/agent/<agent>.md`.

## Why use an alias?

Aliases are useful when the workflow should treat two agent names as separate roles (for example, to make the workflow easier to read), but both roles should use the same underlying agent definition.

The repository also documents a common pattern: use an alias to run the same role with a different model configuration.

Common use cases:

* **Model tiering for the same role**: keep the same prompt/tools/snippets, but assign a different model for cost/speed.
* **Two “roles” that should behave identically**: treat two names as distinct in `flow:` (for example, to make a workflow easier to read), while reusing one agent definition.

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

### Minimal example

This is the smallest useful setup: define an alias and then reference it from `flow:`.

```yaml
---
flow: |
  "backend-go-developer-lite"
alias:
  "backend-go-developer-lite": "backend-go-developer"
---
```

## Use an alias in the workflow flow

Once defined, an alias behaves like a regular agent name in `flow:`.

For example:

```yaml
flow: |
  "backend-go-developer-lite" -> "go-readability-reviewer"
```

## Model selection for aliases

An alias name can appear in the `models:` map.

In the example above, `backend-go-developer-lite`:

1. Resolves to the `backend-go-developer` agent definition when SGAI reads `.sgai/agent/<agent>.md`.
2. Can still have its own model entry under `models:`.

## What gets inherited

The repository’s README describes alias behavior like this:

- The alias inherits the base agent’s prompt, tools, and snippets.
- The alias uses its own model configuration.

## What does *not* change

An alias does not create a new `.sgai/agent/<alias>.md` file.

Instead, SGAI resolves the alias name to the base agent name before reading agent markdown and parsing snippets.

## How alias resolution works (implementation notes)

When SGAI builds flow messages, it resolves the base agent name before reading agent files and parsing snippets.

In `cmd/sgai/dag.go`, `buildFlowMessage`:

1. Accepts an `alias map[string]string` parameter.
2. Calls `resolveBaseAgent(alias, agent)` to resolve each agent name to a base agent name.
3. Loads agent markdown from `.sgai/agent/<base-agent>.md`.

## Example: reusing an agent with a different model

The pattern shown in `GOAL.example.md` uses an alias name in both `alias:` and `models:`.

```yaml
---
alias:
  "backend-go-developer-lite": "backend-go-developer"
models:
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "backend-go-developer-lite": "anthropic/claude-haiku-4-5"
---
```

In this setup:

* The workflow can schedule `backend-go-developer-lite`.
* SGAI loads the agent definition from `.sgai/agent/backend-go-developer.md`.
* SGAI can still pick a different model because `models:` includes an entry keyed by the alias name.
