# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260203 — Improved multiple-choice markdown rendering

- **Date**: 2026-02-03
- **Version**: 0.0.0+20260203
- **Summary**: This release includes improved rendering for markdown-authored multiple-choice questions.

### Bug Fixes

- Fixed rendering for markdown-authored multiple-choice questions by converting markdown to HTML for multichoice question bodies and updating templates to output the generated HTML consistently.
