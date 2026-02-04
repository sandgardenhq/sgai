# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260204 — Improved editor selection reliability

- **Date**: 2026-02-04
- **Version**: 0.0.0+20260204
- **Summary**: This release includes improved editor selection fallback behavior.

### Bug Fixes

- Fixed editor selection to fall back to a working default when the preferred editor was unavailable.
