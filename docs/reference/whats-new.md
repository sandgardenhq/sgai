# What's new

## 0.0.0+20260216 — React web UI and workflow updates

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This update addresses web UI, workflow, and documentation changes.

### New Features

- Added a guided `GOAL.md` authoring experience that creates structured project goals and is the default when no goal content exists.
- Added a dedicated ad hoc prompt runner in the web application for one-shot inferences.
- Added workspace forking and fork lifecycle management.
- Added persistent workspace pinning across the web UI and desktop menu.
- Added a `React` single-page dashboard application with supporting APIs for live updates and navigation.
- Added a macOS status bar menu for monitoring and opening running factories.
- Added an action to open the current workspace in the `opencode` terminal interface when a factory is running.
- Added a continuous workflow mode to keep work progressing without repeated manual starts.
- Added a skill-based STPA workflow to standardize analysis behavior across runs.

### Breaking Changes

- Replaced the previous `HTMX`-based interface and server-driven templates with a `React` SPA and new API surface for the web UI.
- Converted the application to a web-only architecture with one long-lived `MCP` HTTP server per workspace and removed the legacy `CLI`/`stdio` `MCP` flow.
- Enforced a persistent auto-drive lock once work approval is granted and removed configuration paths that previously influenced interactive mode.
- Removed support for an unsupported OpenCode launch flag.

### Bug Fixes

- Fixed editor launching to remain functional when a configured editor command is unavailable or fails.
- Fixed workspace initialization and repository detection edge cases to avoid failures when required tools are missing.
- Fixed rendering and formatting issues in message and prompt output.
- Fixed a runtime panic and improved agent naming for workflow messaging edge cases.
- Fixed workflow state bookkeeping so completed workspaces do not remain marked as started.

### Deprecations

- Removed native macOS notification behavior after introducing menu bar support.
- Removed unused guidance and legacy analysis tooling from the default project skeleton.

### Additional Changes

- Updated skills and snippets documentation by consolidating `when_to_use` into `description` for consistency.
- Updated the work approval dialog to require a complete summary before approval is presented.
- Updated repository terminology in `AGENTS.md` and templates in `GOAL.example.md` for clarity.
- Updated agent coordination guidance so non-coordinator agents yield control after sending messages.
- Updated `sgai find_skills` to behave as a search-only tool.
- Updated the VS Code editor preset to open files at a target location using `code -g {path}`.
- Updated dependency versions across the Go backend and the webapp toolchain.
- Updated installation and usage documentation to require Node.js, standardize the root directory to `sgai/`, and document authenticated `opencode` usage.
- Updated UI structure and navigation for improved workflow visibility.

