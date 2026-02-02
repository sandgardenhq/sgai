# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — Additional updates

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes updates delivered on 2026-02-02.

{
  "Breaking Changes": [
    "The interactive workflow now uses a structured multi-choice prompt instead of free-form messaging, making the interaction flow clearer and more consistent across tools. The previous `human-communication` status and free-form message handling have been replaced with an `ask_user_question` multi-choice interaction, including corresponding updates to the workflow state/schema, CLI behavior, server UI, and documentation."
  ]
}

