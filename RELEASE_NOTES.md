# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260203 — Release updates

- **Date**: 2026-02-03
- **Version**: 0.0.0+20260203
- **Summary**: This release includes the changes listed below.

<!--
{
  "Additional Changes": [
    "JavaScript usage rules have been slightly relaxed to reduce unnecessary friction during development. In addition, Idiomorph script usage has been simplified to rely on idiomorph-ext only, removing other script variants from the supported path."
  ],
  "New Features": [
    "Users can now create and update a workspace GOAL from a guided, web-based composer when no goal content exists yet. The new GOAL.md composer flow supports LLM-assisted editing, includes reference documentation, and becomes the default entry point when the workspace goal body is empty (falling back to direct editing only when GOAL.md already has content).",
    "Operators can now run an optional one-shot, web-based prompt from the Internals tab for ad-hoc interactions without leaving the UI. This adds an HTMX-driven interface plus an auto-refresh preservation mechanism (with corresponding agent guidance) and introduces centralized ANSI-stripping for command output rendering."
  ]
}

-->
