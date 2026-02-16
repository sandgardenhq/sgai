# Release Notes

## 0.0.0+20260216 — Ad hoc runs from Forks tab

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes improvements for starting ad hoc runs from the `Forks` tab.

### New Features

- Added an inline run box on the `Forks` tab to start ad hoc runs with model selection and in place output, and updated the UI to use the `Models API` with corresponding test coverage.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
