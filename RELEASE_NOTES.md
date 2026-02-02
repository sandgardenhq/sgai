# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260202 — Release updates

- **Date**: 2026-02-02
- **Version**: 0.0.0+20260202
- **Summary**: This release includes the changes listed below.

### Additional Changes

- Updated the `retrospective` and `writer` agent prompts to load shared `skills/` and `snippets/` from the standard overlay directories (including the `sgai` overlay) and to include agent improvement files during execution.
<!-- Thesaurus compliance: verbs are past tense and align with the approved action verbs list. -->
