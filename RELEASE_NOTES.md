# Release Notes

## 0.0.0+20260217 — Additional changes

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes the updates listed in the sections below.

```json
{
  "Additional Changes": [
    "The server now uses an explicitly created network listener so its logs report the real address and port that were bound, including when the port is assigned dynamically. This change switches startup to create and pass a `net.Listener` (instead of relying on implicit binding) and documents the intent in a `GOALS` file to ensure consistent logging of the bound endpoint across environments."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
