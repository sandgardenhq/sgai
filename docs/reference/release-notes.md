# Release Notes

## 0.0.0+20260216 — Release notes update

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes a React-based web UI and workspace server architecture updates, expanded workspace management capabilities, and reliability improvements.

### New Features

- Added a guided `GOAL.md` authoring experience in the web app to standardize how goals are created and edited.
- Added ad-hoc “Run” prompting as a first-class workspace capability with improved streaming output.
- Added comprehensive workspace fork management to support isolated experimentation and controlled merges.
- Added persistent workspace pinning to improve navigation and prioritization across the UI and menu integrations.
- Added a macOS status bar application to monitor and open running factories from the desktop.
- Added an “Open in OpenCode” action to open the current workspace in the OpenCode TUI when the factory is running.
- Added a continuous workflow execution mode for self-drive to support long-running goal-driven operation.
- Added live log streaming notifications to the web UI to improve observability of running workspaces.
- Added an STPA workflow entrypoint as a reusable skill to reduce agent-specific setup.

### Breaking Changes

- Updated the product to a web-only application model with a long-lived per-workspace server process.
- Replaced the previous HTMX-based web UI with a React SPA and corresponding API surface.
- Removed support for launching OpenCode with unsupported model variant flags.

### Bug Fixes

- Fixed message rendering and formatting issues in the workspace UI to improve readability and template correctness.
- Fixed workflow state and agent display edge cases that could cause runtime errors or confusing labels.
- Fixed editor selection reliability when invoking external editors from the UI.
- Fixed workspace lifecycle state to avoid stale “ever started” indicators after completion.

### Security

- Updated coordination safety rules to reduce unintended concurrent actions by non-coordinator agents.

### Deprecations

- Removed legacy HTMX UI flows and related migration-specific guidance after the React SPA became the supported interface.
- Removed obsolete skeleton skills and guidance that were no longer supported.

### Additional Changes

- Updated editor presets and open-in-editor behavior to simplify execution and improve cross-environment compatibility.
- Updated coordination and council modeling rules to make agent configuration more consistent and testable.
- Updated documentation and templates to improve onboarding and reduce ambiguity.
- Updated dependencies and toolchain versions across Go and the webapp.
- Updated workspace initialization and repository handling to standardize setup and improve reliability.
- Updated workflow interaction controls to improve visibility and governance.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.