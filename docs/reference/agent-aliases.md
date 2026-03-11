# Agent aliases

Agent aliases let a workflow refer to an existing agent prompt under an alternate name.

Aliases are useful when the workflow wants to keep the same underlying agent prompt and tools, while changing how the agent is configured (for example, picking a different model).

## What an alias is

In SGAI, an alias is a name mapping defined in `GOAL.md` frontmatter. The alias points to an existing (base) agent.

SGAI resolves an alias to its base agent name when it runs the agent.

The README summarizes the intent like this:

* An alias lets the workflow reuse an existing agent’s prompt and tools.
* The alias can run with a different model setting so the same role can run at different cost/capability tiers.

## How to define aliases in `GOAL.md`

Add an `alias:` mapping to the frontmatter of `GOAL.md`.

Example (YAML frontmatter):

```yaml
---
alias:
  reviewer-lite: go-readability-reviewer
---
```

In this example, the workflow can refer to `reviewer-lite`, and SGAI resolves it to the `go-readability-reviewer` agent.

## Example: reuse an agent prompt with a different model

The README shows an example where an alias points at a base agent, and the model is overridden for the alias.

```yaml
---
alias:
  backend-go-developer-lite: backend-go-developer
models:
  backend-go-developer-lite: anthropic/claude-haiku-4-5
---
```

This example keeps the same underlying agent prompt and tools as `backend-go-developer`, but runs it under the `backend-go-developer-lite` name and uses the configured model mapping for that alias.

## Using an alias

Once defined, use the alias name anywhere the workflow expects an agent name.

Example (conceptual):

* Configure or reference `reviewer-lite` in the workflow.
* SGAI resolves `reviewer-lite` to `go-readability-reviewer` when loading the agent prompt and when parsing snippets for the current agent.

## Notes and limitations

* An alias inherits the base agent’s prompt, tools, and snippets, and uses its own model configuration.
* Aliased agent names behave like regular agent names in workflows.
