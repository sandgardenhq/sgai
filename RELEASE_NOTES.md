# Release Notes

## 0.0.0+20260217 — Additional changes

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes the updates listed in the sections below.

```json
{
  "Additional Changes": [
    "Updated the server startup to create and pass an explicit `net.Listener` so logs report the actual bound address and port, including when the port is assigned dynamically."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
