# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260204 — Additional changes

- **Date**: 2026-02-04
- **Version**: 0.0.0+20260204
- **Summary**: This release includes additional updates.

### Additional Changes

```json
{
  "New Features": [
    "The ad-hoc Run interface has been moved out of the session internals area into a dedicated Run tab that is always available. The new Run tab defaults to the coordinator’s model and includes updated execution handling, scrolling behavior, and permissions checks."
  ]
}

```
