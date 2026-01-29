# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260129 — Release updates

- **Date**: 2026-01-29
- **Version**: 0.0.0+20260129
- **Summary**: This release includes the updates listed below.

### Breaking Changes

- Renamed SGAI environment variables to uppercase for consistent configuration across the CLI, MCP integration, and notifications, including `sgai_NTFY` → `SGAI_NTFY` and `sgai_MCP_WORKING_DIRECTORY` → `SGAI_MCP_WORKING_DIRECTORY`.
- Updated migration steps: Replace `sgai_NTFY` with `SGAI_NTFY` and `sgai_MCP_WORKING_DIRECTORY` with `SGAI_MCP_WORKING_DIRECTORY` in your shell environment and CI configuration.

