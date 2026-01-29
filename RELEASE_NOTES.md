# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260129 — Additional changes

- **Date**: 2026-01-29
- **Version**: 0.0.0+20260129
- **Summary**: This release includes additional improvements across the project.

### New Features

- Added a persisted workspace state indicating whether a workspace has ever started, and exposed that state to consumers.

