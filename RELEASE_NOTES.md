# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — Updated Go tooling dependencies

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes updated Go tooling dependencies.

### Additional Changes

- Updated Go tooling dependencies by updating `golang.org/x/tools` to `v0.41.0` and refreshing `go.sum`.

