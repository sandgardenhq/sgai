# Updates

This page summarizes notable changes by build date.

## 0.0.0+20260212 — February 2026 update

- **Date**: 2026-02-12
- **Version**: 0.0.0+20260212
- **Summary**: This update covers the items listed below.

```json
{
  "Additional Changes": [
    "The documentation was cleaned up to remove outdated migration instructions and UI switcher details that are no longer applicable. Specifically, we removed HTMX-to-React migration guidance and cookie-based UI switcher documentation, deleted the shadcn mapping skill, narrowed the React best practices skill to generic Vercel rules, and added a GOAL entry to capture and track this cleanup work."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This update includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
