# Recent updates

This page captures recent product changes in a lightweight format for documentation readers.

```json
{
  "Additional Changes": [
    "Updated workflow event handling to be scoped per workspace to improve isolation and traceability. The frontend and retrospective processing now consume workspace-scoped events and interpret `needsInput` consistently for stateful workflow execution."
  ],
  "Breaking Changes": [
    "Removed command-line and terminal-based interaction support in favor of a web-only runtime. The release removed `CLI` execution, `stdio`-based `MCP`, and terminal interaction paths, and updated retrospective handling plus the frontend to rely on workspace-scoped events and `needsInput` flags."
  ],
  "New Features": [
    "Refactored the service into a web-only server that runs automated workflows within the server process. The server now executes workflows in-process, uses per-workspace remote `MCP` servers over `HTTP`, and publishes workspace-scoped `SSE` event streams with per-workspace logging."
  ]
}

```
