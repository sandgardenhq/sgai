# Release Notes

## 0.0.0+20260217 — Removed fork merge support

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes removal of `fork merge` support to align with the updated merge policy.

### Breaking Changes

- Removed `fork merge` support, including the `fork merge` API and merge button in the UI, to align with the updated merge policy.


## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
