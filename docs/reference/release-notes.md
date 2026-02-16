# Release Notes

## 0.0.0+20260216 — Updated web UI and workspace lifecycle

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes web UI and workspace lifecycle updates.

### New Features

- Added a guided `GOAL.md` authoring experience in the web app.
- Added ad-hoc “Run” prompting as a workspace capability with improved streaming output.
- Added workspace fork management to support isolated experimentation and controlled merges.
- Added persistent workspace pinning.
- Added a macOS status bar application.
- Added an “Open in OpenCode” action.
- Added a continuous workflow execution mode for self-drive.
- Added live log streaming notifications to the web UI.
- Added an STPA workflow entrypoint as a reusable skill.

### Breaking Changes

- Updated the product to a web-only application model with a long-lived per-workspace server process.
- Replaced the previous HTMX-based web UI with a React SPA and corresponding API surface.
- Removed support for launching OpenCode with unsupported model variant flags.

### Bug Fixes

- Fixed message rendering and formatting issues in the workspace UI.
- Fixed workflow state and agent display edge cases.
- Fixed editor selection reliability when invoking external editors from the UI.
- Fixed workspace lifecycle state to avoid stale “ever started” indicators after completion.

### Security

- Updated coordination safety rules to reduce unintended concurrent actions by non-coordinator agents.

### Deprecations

- Removed legacy HTMX UI flows and related migration-specific guidance.
- Removed obsolete skeleton skills and guidance.

### Additional Changes

- Updated editor presets and open-in-editor behavior.
- Updated coordination and council modeling rules.
- Updated documentation and templates.
- Updated dependencies and toolchain versions across Go and the webapp.
- Updated workspace initialization and repository handling.
- Updated workflow interaction controls.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.