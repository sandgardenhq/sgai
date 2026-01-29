# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260129 — Feature and maintenance updates

- **Date**: 2026-01-29
- **Version**: 0.0.0+20260129
- **Summary**: This release includes improved routing for in-progress workspace links.

### New Features

- Updated in-progress workspace links to open the most relevant page for the workspace state by routing to `Respond` when input is required and to `Workspace Progress` when no input is required.
