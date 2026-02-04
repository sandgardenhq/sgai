# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260204 — Standardized workspace initialization

- **Date**: 2026-02-04
- **Version**: 0.0.0+20260204
- **Summary**: This release includes standardized workspace initialization and related documentation updates.

### New Features

- Added a standardized workspace initialization flow that unpacks the `.sgai` skeleton, initializes `jj`/Git repositories, configures `git` excludes, and writes `GOAL.md`.
- Added a GOALS specification that defines required behavior and edge cases for workspace pre-population and initialization.

### Bug Fixes

- Fixed `initJJ` to continue workspace setup when the `jj` executable is missing.

### Additional Changes

- Updated workspace documentation to align with the standardized initialization workflow by replacing the workspace `GOALS` document with a multi-agent workflow configuration script.

