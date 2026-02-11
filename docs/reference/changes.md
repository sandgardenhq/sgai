# Changes

This document describes user-visible changes by release.

## 0.0.0+20260211 — React web UI and `/api/v1` API

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes a new React single-page application web interface, removes the legacy HTMX UI, and updates related documentation.

### New Features

- Added a `bun`-built React single-page application web interface using `shadcn/ui`, backed by new `/api/v1` JSON and `SSE` endpoints.

### Breaking Changes

- Removed the legacy HTMX compose/adhoc/retro web UI and updated browser interactions to use the React single-page application interface and the `/api/v1` JSON+`SSE` API.

### Additional Changes

- Updated the installation documentation to list Node.js as a required dependency and added Node.js to the Homebrew install command.
- Added a dedicated React migration goals document to clarify intended outcomes and removed obsolete migration-related agent and skill definitions.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.