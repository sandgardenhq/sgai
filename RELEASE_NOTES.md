# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260129 — Release updates

- **Date**: 2026-01-29
- **Version**: 0.0.0+20260129
- **Summary**: This release includes the updates listed below.

<!-- Generated release notes payload (to be formatted into sections and bullets in subsequent commits):

{
  "Breaking Changes": [
    "Environment variable names for SGAI configuration were standardized so they are consistent across the command-line tooling, the MCP integration, and notification handling. Specifically, several sgai-related environment variables were renamed to uppercase to align with conventional ENV naming and to ensure the CLI, MCP, and notification code paths read the same keys."
  ]
}


-->
