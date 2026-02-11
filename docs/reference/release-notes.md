# Release Notes

## 0.0.0+20260211 — Dependency update

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes a Go module dependency update.

### Additional Changes

- Updated `golang.org/x/term` from `v0.39.0` to `v0.40.0` in `go.mod` and `go.sum`.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
