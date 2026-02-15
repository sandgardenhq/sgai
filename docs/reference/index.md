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

## Recent updates

### 0.0.0+20260215

- **Date**: 2026-02-15
- **Version**: 0.0.0+20260215
- **Summary**: This update addresses the changes listed below.

```json
{
  "Breaking Changes": [
    "Native notification support has been removed, so applications will no longer be able to send or receive notifications through this product. The notifications package has been deleted, and a goals document has been added to document the decision and expected migration away from native notifications."
  ]
}

```
