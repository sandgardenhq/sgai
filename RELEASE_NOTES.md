# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260129 — Standardized configuration environment variables

- **Date**: 2026-01-29
- **Version**: 0.0.0+20260129
- **Summary**: This release includes standardized environment variables for consistent configuration across the CLI, MCP integration, and notifications.

### Breaking Changes

- Updated SGAI configuration environment variables to uppercase for consistent behavior across the CLI, MCP integration, and notifications (migrate `sgai_NTFY` → `SGAI_NTFY` and `sgai_MCP_WORKING_DIRECTORY` → `SGAI_MCP_WORKING_DIRECTORY`).

