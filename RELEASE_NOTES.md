# Release Notes

## 0.0.0+00010101 — Release notes update

- **Date**: 0001-01-01
- **Version**: 0.0.0+00010101
- **Summary**: Quick update: see the items below.

### Notes (raw)

```json
{
  "🚀 New Features": [
    "You can now connect external workspaces and manage them without losing data.",
    "You can now view a full workspace diff from the server to better understand changes across a project.",
    "You can now edit markdown content with a richer in-browser editor experience.",
    "You can now see YAML frontmatter as a structured preview while editing markdown.",
    "You can now create forks and immediately continue configuring their goals without extra navigation.",
    "You can now use an MCP-compatible HTTP endpoint that matches the existing web API tooling behavior."
  ],
  "🚧 Bug Fixes": [
    "Renaming a workspace no longer leaves behind stale or mismatched state that can make the UI behave incorrectly.",
    "Pinned workspaces now behave consistently even when directories are referenced through symlinks.",
    "Waiting for human input is now more reliable when a question times out or an MCP config file is updated.",
    "Retrospective flows now transition modes more predictably and follow stricter PR-creation guidance.",
    "Long-running “completion gate” scripts are less likely to hang a workflow and can be cancelled safely.",
    "The workbench plugin is less likely to fail when an interactive user response takes longer than expected."
  ],
  "🛠 Internal Updates": [
    "Workflows now have a more consistent and recoverable coordination model for human-in-the-loop operation.",
    "Workflow execution is now easier to reuse and test across different entry points.",
    "Workspace metadata is now structured around goals rather than ad-hoc summaries to unify behavior across creation and forking.",
    "The system’s “actions” configuration is now more explicit and the internal UI is less cluttered.",
    "Review agents now apply stricter, more consistent standards when evaluating changes.",
    "Retrospectives now include a required health-analysis section with clearer documentation about where information should live.",
    "The critic council protocol now supports explicit dissent to surface alternative viewpoints during review.",
    "The UI and workflow experience for syncing, logging, and navigation has been streamlined and modernized.",
    "Retrospective agent wiring is now easier to reason about and better covered by tests.",
    "Build and test automation has been tightened to reduce accidental gaps in webapp validation.",
    "Dependency and SDK updates have been applied to keep the Go stack current.",
    "Repository hygiene and contributor guidance have been improved to reduce noise and make contributions more spec-driven.",
    "Documentation and goal specifications were expanded to support new deployment and coordination workflows.",
    "Fork, compose, and messaging subsystems were refactored to reduce duplication and harden behavior without changing user-facing semantics.",
    "Workspace rename support was removed in favor of explicit attach/detach flows for external workspaces."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
