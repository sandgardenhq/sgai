# Release Notes

## 0.0.0+20260217 — Additional changes

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes the latest updates across features, fixes, and maintenance.

```json
{
  "New Features": [
    "The inbox indicator is now a clickable control that takes users directly to the first workspace that needs attention. This update changes the inbox indicator UI to a button that routes to the first needs-input workspace and adds automated test coverage for the navigation behavior."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
