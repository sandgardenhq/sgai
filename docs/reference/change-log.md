# Change log

## 0.0.0+20260211 — Updates

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes the changes listed below.

### Changes

```json
{
  "Additional Changes": [
    "Updated the Go SDK dependency to a newer release to keep the project current and compatible with upstream changes. Specifically, the module dependency `github.com/modelcontextprotocol/go-sdk` was bumped from `v1.2.0` to `v1.3.0` in the dependency set."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
