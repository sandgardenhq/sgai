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
- **Summary**: This release includes additional updates across the project.

```json
{
  "Additional Changes": [
    "Improved the visibility of the flow by logging a message before the completion gate script runs in the sgai flow agent. Specifically, the sgai flow agent now emits a console log immediately prior to invoking the completion gate script to aid debugging and trace execution order."
  ]
}

```
