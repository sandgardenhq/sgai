# Release Notes

## 0.0.0+20260216 — Additional updates

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes additional updates across features and fixes.

```json
{
  "New Features": [
    "You can now start an ad-hoc run directly from the Forks tab using an inline run box that includes model selection and displays the output in place. Under the hood, the Forks UI has been updated to use the newer Models API for model picking and the associated test coverage has been updated to match the new UI flow and API usage."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
