# Release Notes

## 0.0.0+20260217 — Additional changes

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes the latest updates across features, fixes, and maintenance.

```json
{
  "New Features": [
    "Updated the inbox indicator to a clickable button that routes users to the first workspace needing attention and added automated tests for the navigation behavior."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
