# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — Added self-drive interactive mode

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes improved `sgai` interactive behavior.

### New Features

- Added an interactive mode that propagates through `sgai` workflows and the MCP server to support fully automated operation, including `auto`/`auto-session` self-drive behavior and automatic answer selection for multi-choice prompts.
