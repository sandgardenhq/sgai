# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — Documentation updates

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes documentation updates for goal completion reporting and coordinator responsibilities.

### Additional Changes

- Updated documentation to clarify that only the coordinator updates `GOAL.md` checkboxes and that workers report completions using `GOAL COMPLETE:` messages.

