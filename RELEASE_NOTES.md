# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260204 — Updates and fixes

- **Date**: 2026-02-04
- **Version**: 0.0.0+20260204
- **Summary**: This release includes the changes captured below.

### Additional Changes

```json
{
  "Additional Changes": [
    "Updated `.gitignore` handling and CI configuration to set up `jj` more reliably across local and CI environments."
  ],
  "New Features": [
    "Added an `auto`/`interactive` mode toggle with end-to-end UI and backend support."
  ]
}

```
