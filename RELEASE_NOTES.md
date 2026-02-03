# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260203 — Maintenance updates

- **Date**: 2026-02-03
- **Version**: 0.0.0+20260203
- **Summary**: This release includes the changes listed below.

### Additional Changes

```json
{
  "Bug Fixes": [
    "Auto-refresh behavior in HTMX/Idiomorph views now preserves page state so dynamic updates do not clobber user context during refresh cycles. A dedicated auto-refresh preservation skill was added and adopted by HTMX PicoCSS agents to remove redundant inline scripting and centralize refresh guidance."
  ],
  "New Features": [
    "You can now run an ad-hoc, one-shot prompt directly from a workspace when the feature is enabled by a flag and configuration, with the server keeping the interaction state so updates render reliably. The workspace UI uses HTMX incremental updates with improved Idiomorph handling, strips ANSI escape sequences from output, supports model selection loaded from `opencode models`, and includes an auto-refresh behavior that preserves page state."
  ]
}

```
