# Release Notes

## 0.0.0+20260217 — Release updates

- **Date**: 2026-02-17
- **Version**: 0.0.0+20260217
- **Summary**: This release includes the changes captured in the structured notes below.

```json
{
  "Additional Changes": [
    "The project documentation was updated to better reflect the current recommended way to run the tool and to improve the layout of the README. Specifically, the README usage was changed from `sgai serve` to `sgai`, the Features section was repositioned, and a `GOALS` specification file was added to guide a broader README rewrite."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
