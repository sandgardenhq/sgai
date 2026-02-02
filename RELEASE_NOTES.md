# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — Release updates

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes the following updates.

```json
{
  "Additional Changes": [
    "The system now uses a single, consistent set of human-wait states so workflows behave the same way when waiting for a person. Specifically, the `waiting-for-human` state is unified via shared helpers, Auto mode is constrained to the `auto` value, and workflow update handling preserves `waiting-for-human` status across updates, with corresponding CLI/server/MCP logic and tests updated to match."
  ]
}

```
