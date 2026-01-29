# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260129 — Feature and maintenance updates

- **Date**: 2026-01-29
- **Version**: 0.0.0+20260129
- **Summary**: This release includes additional updates captured in the changelog below.

### Additional Changes

```json
{
  "New Features": [
    "In-progress workspace links now open the most relevant page for the current state of the workspace so users land where they can take the next action. The routing logic conditionally sends these links to either the Respond page (when the workspace needs input) or the Workspace Progress page (when no input is required)."
  ]
}

```
