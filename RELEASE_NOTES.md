# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — General updates

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes changes across multiple areas.

### Additional Changes

{
  "Additional Changes": [
    "The project documentation now makes it clear that only the coordinator is responsible for updating GOAL.md checkboxes, and workers must report completed goals in a consistent way. Specifically, workers must send completion reports using “GOAL COMPLETE:” messages, and the project-completion-verification skill documentation has been extended with coordinator-only instructions for marking and verifying completed items in GOAL.md."
  ]
}

