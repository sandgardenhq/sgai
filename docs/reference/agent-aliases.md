# Agent Aliases

Hey there! This page explains **agent aliases**: what they are, why they’re useful, and how to configure and use them.

## What are agent aliases?

An **agent alias** is an agent name that *reuses another agent’s prompt and tools*, but can run with a *different model configuration*.

In other words, an alias lets the same role run at different cost or capability tiers by pointing a new agent name at an existing “base” agent.

## Why use an alias?

Agent aliases help when:

- The same workflow role should sometimes run on a cheaper/faster model.
- A workflow should keep the same agent “job title” (role), but swap models depending on the scenario.

## How aliasing works

1. Define an alias mapping from the alias agent name to the base agent name.
2. Configure a model for the alias agent name.
3. Use the alias agent name in your workflow.

The alias agent:

- Inherits the base agent’s prompt, tools, and snippets.
- Uses its own model configuration.
- Behaves like a regular agent in workflows.

## Example: create a “lite” variant of an agent

The following example defines:

- A base agent: `backend-go-developer`
- An alias agent: `backend-go-developer-lite`
- A model override for the alias: `anthropic/claude-haiku-4-5`

```yaml
# Example configuration
alias:
  backend-go-developer-lite: backend-go-developer

models:
  backend-go-developer-lite: anthropic/claude-haiku-4-5
```

After this, `backend-go-developer-lite` can be used anywhere an agent name is accepted (for example, in a workflow), and it will run like the base agent but with its alias-specific model.

## Notes and tips

- Define the base agent normally; the alias points at it.
- Configure the model on the alias name when the alias should use a different model.

## Next steps

- Browse the agent catalog in [SGAI Agents Reference](../AGENTS.md).