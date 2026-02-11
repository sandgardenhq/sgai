# Updates

## 0.0.0+20260211 — TBD

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This update covers the changes included in the raw payload below.

### Raw payload

```json
{
  "Bug Fixes": [
    "Completed workspaces no longer retain an \"ever-started\" state after a session ends. This change clears the persisted ever-started tracking flag for completed workspaces on session termination to prevent stale status from carrying into subsequent sessions."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This update covers improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
