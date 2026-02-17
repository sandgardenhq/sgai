# Release Notes

## 0.0.0+20260217 — Project updates

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes updates across the project.

### Draft notes (source)

```json
{
  "New Features": [
    "Ad-hoc opencode runs now inherit the configured environment and present their output in a consistent, readable format. Specifically, the runner injects the configured env vars into ad-hoc executions and prefixes and mirrors process output to the sgai stdout/stderr streams."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
