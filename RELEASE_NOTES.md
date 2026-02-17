# Release Notes

## 0.0.0+20260217 — Draft release notes

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes the changes listed below.

### Draft (unformatted)

```json
{
  "New Features": [
    "Updated the dashboard to show the new SGAI branding in the sidebar and mobile header. This adds SGAI logo asset files, updates the dashboard sidebar and mobile header to reference them, and configures the web build to serve static assets from the `/assets/` path."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
