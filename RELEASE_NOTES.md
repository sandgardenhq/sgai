# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260204 — Additional updates

- **Date**: 2026-02-04
- **Version**: 0.0.0+20260204
- **Summary**: This release includes the changes listed below.

### New Features

- Added a reusable workspace initialization flow that unpacks the `.sgai` skeleton, initializes `jj`/Git repositories, configures `git` excludes, and writes `GOAL.md`.
- Added a GOALS specification that defines expected behavior and edge cases for workspace pre-population and initialization.

### Bug Fixes

- Fixed `initJJ` to treat a missing `jj` executable as non-fatal and continue workspace setup.

### Additional Changes

- Updated workspace documentation to reflect the new initialization workflow and replace the workspace `GOALS` document with a multi-agent configuration script.

