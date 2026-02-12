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

### 0.0.0+20260212 — Improved skill announcement state updates

- **Date**: 2026-02-12
- **Version**: 0.0.0+20260212
- **Summary**: This release includes improvements to skill usage announcements and workflow state updates.

#### Additional Changes

- Updated skill usage announcements to call `sgai_update_workflow_state` so workflow state updates are enforced consistently.
