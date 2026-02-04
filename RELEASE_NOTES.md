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

{
  "Bug Fixes": [
    "Initialization no longer fails when the `jj` executable is missing, reducing setup friction on systems that have not installed it yet. The `initJJ` path now treats a missing `jj` binary as a non-fatal condition while still proceeding with the rest of the workspace setup.",
    "The workspace documentation artifact was updated to align with the new initialization workflow and make the intended setup steps clearer. The previous workspace `GOALS` document was replaced with a multi-agent workflow/config script that reflects the current orchestration and configuration model."
  ],
  "New Features": [
    "Workspace initialization now uses a reusable, standardized setup flow so new workspaces start in a consistent and fully prepared state. The initializer unpacks a `.sgai` skeleton, initializes `jj`/Git repositories, configures `git` excludes, writes `GOAL.md`, and the related handlers and tests were updated to call this shared implementation.",
    "A new GOALS specification was added to clearly define the expected behavior for workspace creation pre-population and initialization. The spec documents required outcomes and edge cases for the pre-population pipeline and initialization steps so implementations and tests can validate against a single source of truth."
  ]
}

