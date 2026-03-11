# Agent aliases

Agent aliases let a workflow refer to an existing agent prompt under an alternate name.

This is useful when the workflow wants to keep the same underlying agent prompt, but use a different name in the workflow.

## What an alias is

In SGAI, an alias is a name mapping defined in `GOAL.md` frontmatter. The alias points to an existing agent.

The weekly update for 2026-03-11 summarizes the current behavior:

* Workflows can define `alias:` mappings in `GOAL.md` frontmatter.
* Alias resolution is used when SGAI reads an agent prompt from disk.
* Alias resolution is used when SGAI parses snippets for the current agent.

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

## Using an alias

Once defined, use the alias name anywhere the workflow expects an agent name.

Example (conceptual):

* Configure or reference `reviewer-lite` in the workflow.
* SGAI resolves `reviewer-lite` to `go-readability-reviewer` when loading the agent prompt and when parsing snippets for the current agent.

## Notes and limitations

* This page documents alias behavior described in the 2026-03-11 weekly update and commit summary for `cmd/sgai: fix regressions from #356`.
* If the repository’s `GOAL.md` uses a different frontmatter style (or different key names), follow that repository’s conventions.
