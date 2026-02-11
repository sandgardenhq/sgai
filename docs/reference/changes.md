# Changes

This document describes user-visible changes by release.

## 0.0.0+20260211 — Additional changes

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes the changes listed below.

```json
{
  "Additional Changes": [
    "The installation documentation now lists Node.js as a required dependency to avoid setup mismatches. The Homebrew install command was updated to include Node.js explicitly.",
    "A dedicated migration goals document was added to clarify the intended React migration outcomes and scope. Obsolete migration-related agent and skill definitions were removed to prevent referencing outdated React migration components."
  ],
  "Breaking Changes": [
    "The legacy HTMX compose/adhoc/retro web UI has been removed in favor of the new React single-page application interface. This change deletes the corresponding HTMX handlers and templates and standardizes browser interactions on the new `/api/v1` JSON+SSE API surface."
  ],
  "New Features": [
    "A new web interface is now available, providing a modern single-page experience for using SGAI through the browser. The HTMX-based compose/adhoc/retro UI handlers and templates were removed and replaced with a Bun-built React SPA using shadcn/ui, backed by new `/api/v1` JSON + Server-Sent Events (SSE) endpoints."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.