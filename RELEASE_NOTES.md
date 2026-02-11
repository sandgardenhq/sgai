# Release Notes

## 0.0.0+20260211 — Release updates

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes the latest updates.

```json
{
  "Additional Changes": [
    "Updated the Markdown rendering dependency by upgrading `react-markdown` to `v10.1.0` and refreshing the Bun lockfile."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
