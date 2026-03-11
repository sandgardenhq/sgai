# Agent aliases

Agent aliases let a workflow refer to an agent by an alternate name that resolves to a “base” agent at runtime.

This is useful when the same role (prompt/tools/snippets) should run under a different model configuration.

## What an alias is

An alias is a name that maps to an existing agent name (the base agent).

When an alias is used:

- The workflow resolves the alias name to a base agent name.
- The agent run uses the base agent name for the `--agent` invocation.
- The alias can still have its own model configuration.

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

In this example:

- `backend-go-developer-lite` is an alias for the base agent `backend-go-developer`.
- The base agent and the alias each have a `models:` entry.

## How to use aliases

Once defined, aliased agent names behave like regular agents in workflows.

For example, a workflow step can reference `backend-go-developer-lite`, and the runtime resolves that to `backend-go-developer` when it builds the agent invocation.

## Notes

- If an agent name is not in the alias map, it resolves to itself.
