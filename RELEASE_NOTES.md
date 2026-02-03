# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260203 — Workspace one-shot prompting

- **Date**: 2026-02-03
- **Version**: 0.0.0+20260203
- **Summary**: This release includes workspace one-shot prompting and improved auto-refresh behavior in `HTMX`/`Idiomorph` views.

### New Features

- Added ad-hoc, one-shot prompting from a workspace (gated by configuration and a feature flag) with server-side interaction state to support reliable `HTMX` incremental updates, `Idiomorph` morphing, model selection via `opencode models`, and ANSI escape-sequence stripping.

### Bug Fixes

- Fixed auto-refresh behavior in `HTMX`/`Idiomorph` views to preserve page state during refresh cycles and avoid clobbering user context.
