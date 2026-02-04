# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260204 — Updates and fixes

- **Date**: 2026-02-04
- **Version**: 0.0.0+20260204
- **Summary**: This release includes the changes captured below.

### Additional Changes

```json
{
  "Additional Changes": [
    "Developer workflows and automation are now more consistent across local and CI environments. This refines `.gitignore` handling and updates CI configuration to set up JJ more reliably."
  ],
  "New Features": [
    "Users can now switch between an automatic mode and an interactive mode in the product interface, with corresponding behavior supported end-to-end. This adds UI controls and backend handling for an auto/interactive mode toggle and centralizes workspace/JJ initialization so the selected mode is applied consistently."
  ]
}

```
