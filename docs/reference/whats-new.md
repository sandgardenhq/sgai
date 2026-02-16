# What's new

## 0.0.0+20260216 — Updates

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This update addresses the changes listed below.

{
  "Additional Changes": [
    "Updated how skills and snippets are described and surfaced in documentation for consistency.",
    "Updated the work approval dialog to require a complete summary before approval is presented.",
    "Updated repository terminology and templates to improve documentation clarity.",
    "Updated agent coordination rules to reduce stalled workflows.",
    "Updated the skill discovery interface to behave as a search-only tool.",
    "Updated the VS Code editor preset multiple times and finalized it to open files at a target location.",
    "Updated dependency versions across the Go backend and the webapp toolchain.",
    "Updated installation and usage documentation to reflect current tooling and directory conventions.",
    "Updated UI structure and navigation for improved workflow visibility."
  ],
  "Breaking Changes": [
    "Replaced the previous HTMX-based interface and server-driven templates with a React SPA and new API surface for the web UI.",
    "Converted the application to a web-only architecture with one long-lived MCP HTTP server per workspace and removed the legacy CLI/stdio MCP flow.",
    "Enforced a persistent auto-drive lock once work approval is granted and removed configuration paths that previously influenced interactive mode.",
    "Removed support for an unsupported OpenCode launch flag."
  ],
  "Bug Fixes": [
    "Fixed editor launching to remain functional when a configured editor command is unavailable or fails.",
    "Fixed workspace initialization and repository detection edge cases to avoid failures when required tools are missing.",
    "Fixed rendering and formatting issues in message and prompt output.",
    "Fixed a runtime panic and improved agent naming for workflow messaging edge cases.",
    "Fixed workflow state bookkeeping so completed workspaces do not remain marked as started."
  ],
  "Deprecations": [
    "Removed native macOS notification behavior after introducing menu bar support.",
    "Removed unused guidance and legacy analysis tooling from the default project skeleton."
  ],
  "New Features": [
    "Added a guided GOAL authoring experience that creates structured project goals and makes it the default when no goal content exists.",
    "Added a dedicated, always-available ad-hoc prompt runner in the web application for one-shot inferences with streaming output and preserved form state.",
    "Added workspace forking and fork lifecycle management to support isolated work streams.",
    "Added persistent workspace pinning across the web UI and desktop menu experience to improve navigation.",
    "Added a React-based single-page dashboard application with supporting APIs for live updates and navigation.",
    "Added a macOS status bar menu for monitoring and opening running factories.",
    "Added an action to open the current workspace in the OpenCode terminal interface when a factory is running.",
    "Added a continuous workflow mode to keep work progressing without repeated manual starts.",
    "Added a skill-based STPA workflow to standardize analysis behavior across runs."
  ]
}

