# Sandgarden AI Software Factory Reference

Reference pages for the `sgai` CLI and its workspace file formats.

## Topics

- [CLI commands](./cli.md)
- [Environment variables](./environment-variables.md)
- [Project configuration (`sgai.json`)](./project-configuration.md)
- [Workflow state (`.sgai/state.json`)](./workflow-state.md)
- [MCP server](./mcp.md)

## Examples

- [`GOAL.example.md`](../../GOAL.example.md)
- [`sgai.example.json`](../../sgai.example.json)
- [`opencode.json`](../../opencode.json)

## Recent changes

### 0.0.0+20260212 — Pending changes

- **Date**: 2026-02-12
- **Version**: 0.0.0+20260212
- **Summary**: This update covers the changes listed below.

```json
{
  "Additional Changes": [
    "Skill usage announcements now instruct users to update workflow state through an API call instead of relying on manual plain-text guidance. Specifically, the instructions were updated to use `sgai_update_workflow_state` calls when announcing skill usage to ensure state changes are machine-enforced and consistent."
  ]
}

```
