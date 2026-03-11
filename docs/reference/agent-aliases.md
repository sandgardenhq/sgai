# Agent aliases

Agent aliases let a workflow refer to an agent by one name while running another “base” agent behind the scenes.

In `sgai`, an alias maps an **alias name** to a **base agent name**. When a workflow step tries to run an agent, `sgai` resolves the alias to the base agent name and then executes that base agent.

## Why use aliases

Aliases are useful when you want a stable role name in your workflow, but you want the underlying agent implementation to be reused.

For example, an alias can keep the workflow role name constant while selecting a different model for that role.

## How alias resolution works

When an alias is configured:

1. The workflow references the agent using the alias name.
2. Before running the step, `sgai` resolves the alias name to the base agent name.
3. `sgai` runs `opencode` with `--agent` set to the base agent name.

This alias resolution is also used when:

- Building flow messages (agent files are accessed using the resolved base agent name).
- Parsing agent snippets (snippets are parsed using the resolved base agent name).

## Related references

- [`GOAL.example.md`](../../GOAL.example.md)
- [CLI commands](./cli.md)