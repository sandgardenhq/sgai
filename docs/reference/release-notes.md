# Release notes

## 0.0.0+20260216 — Release notes updates

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes the changes listed below.

```json
{
  "New Features": [
    "Message content in the Messages tab is now displayed using rich text formatting so that Markdown is rendered for easier reading. Specifically, the Messages tab renders message bodies as Markdown and includes automated tests that validate the generated HTML output."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
