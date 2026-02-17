# Release Notes

{
  "New Features": [
    "Users can now receive browser notifications when a workspace newly requires approval, including a visible permission prompt to enable notifications. This adds notification wiring for the “workspace needs approval” condition, a browser permission prompt bar, and accompanying automated tests to validate the end-to-end behavior."
  ]
}


## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
