# Release Notes

## 0.0.0+20260211 — Release notes update

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes the changes listed below.

### Additional Changes

```json
{
  "Additional Changes": [
    "We updated a Go tooling dependency to keep the development toolchain current and compatible with upstream changes. Specifically, the indirect module dependency `golang.org/x/tools` was bumped from v0.41.0 to v0.42.0."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
