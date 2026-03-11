# Agent aliases

Agent aliases let a workflow refer to an agent by an alternate name that resolves to a “base” agent at runtime.

This is useful when the same role (prompt/tools/snippets) should run under a different model configuration.

This repository supports aliases through `GOAL.md` frontmatter and uses alias resolution when reading agent prompt files and snippets.

## What an alias is

An alias is a name that maps to an existing agent name (the base agent).

When an alias is used:

- The workflow resolves the alias name to a base agent name.
- The workflow uses the base agent name when it needs to read the agent prompt from disk (for example, `.sgai/agent/<agent>.md`) and when it needs to parse snippets for the current agent.
- The alias can still have its own model configuration.

In other words: **use the alias name in your flow**, and SGAI resolves it to the base agent name when it needs to look up the base agent’s on-disk content.

## How to define aliases

Aliases are configured in `GOAL.md` frontmatter.

Example (from the repository README):

```md
---
alias:
  backend-go-developer-lite: backend-go-developer
models:
  backend-go-developer: anthropic/claude-opus-4-6
  backend-go-developer-lite: anthropic/claude-haiku-4-5
---
```

An additional example appears in `cmd/sgai/GOAL.example.md` (commented out in that file):

```md
---
# alias:
#   backend-go-developer-lite: backend-go-developer

# alias:
#   backend-go-developer-lite: anthropic/claude-haiku-4-5
---
```

Note: the `GOAL.example.md` snippet shows two alternative ways to write a commented example block. Refer to the repository README example for a working `alias:` + `models:` pairing.

In this example:

- `backend-go-developer-lite` is an alias for the base agent `backend-go-developer`.
- The base agent and the alias each have a `models:` entry.

## How to use aliases

Once defined, aliased agent names behave like regular agents in workflows.

For example, a workflow step can reference `backend-go-developer-lite`, and the runtime resolves that to `backend-go-developer` when it builds the agent invocation.

### Example: use an alias in a flow

```md
---
flow: |
  "backend-go-developer-lite" -> "go-readability-reviewer"
alias:
  backend-go-developer-lite: backend-go-developer
models:
  backend-go-developer: anthropic/claude-opus-4-6
  backend-go-developer-lite: anthropic/claude-haiku-4-5
---
```

In this setup:

- The flow refers to `backend-go-developer-lite`.
- SGAI resolves `backend-go-developer-lite` to `backend-go-developer` when reading agent content from `.sgai/agent/backend-go-developer.md`.
- The alias can still have its own `models:` entry.

## Notes

- If an agent name is not in the alias map, it resolves to itself.
