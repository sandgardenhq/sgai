# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260203 — Configurable editor links

- **Date**: 2026-02-03
- **Version**: 0.0.0+20260203
- **Summary**: This release includes configurable editor selection for `Open in Editor` links in the web UI.

### New Features

- Updated the web UI so `Open in Editor` links use a configurable editor via the `editor` project setting (with environment variable fallbacks) instead of a single hard-coded editor.
