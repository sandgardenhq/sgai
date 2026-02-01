# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260201 — Added `self-driving mode` approval gating

- **Date**: 2026-02-01
- **Version**: 0.0.0+20260201
- **Summary**: This release includes `self-driving mode` approval gating and simplified `brainstorming` skill guidance.

### New Features

- Added an approval question that, once approved, automatically transitions a session into `self-driving mode` and propagates the setting through workflow state and coordinator instructions.

### Additional Changes

- Updated the `brainstorming` skill guidance by removing the explicit planning handoff phase.

