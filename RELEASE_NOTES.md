# Release Notes

## 0.0.0+20260217 — Additional updates

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes additional updates across the CLI and core packages.

{
  "Breaking Changes": [
    "Removed the fork merge capability from the product and eliminated the merge button from the interface to match the updated merge policy. The fork merge API and its supporting logic, tests, and type definitions were removed, and the change was documented in a `GOALS` specification for the merge-button removal."
  ]
}


## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
