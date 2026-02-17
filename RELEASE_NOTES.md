# Release Notes

## 0.0.0+20260217 — Improved ad-hoc opencode runs

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes improved ad-hoc `opencode` run environment inheritance and output routing.

### New Features

- Added support for ad-hoc `opencode` runs to inherit configured environment variables and mirror prefixed process output to `sgai` `stdout`/`stderr`.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
