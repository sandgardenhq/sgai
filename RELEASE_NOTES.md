# Release Notes

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.

## 0.0.0+20260201 — Product updates and fixes

- **Date**: 2026-02-01
- **Version**: 0.0.0+20260201
- **Summary**: This release includes product updates and fixes across core functionality.

{
  "Additional Changes": [
    "Simplified the brainstorming skill guidance by removing the explicit planning handoff phase. The instructions no longer include a dedicated planning-to-execution transition step, reducing directive complexity in the brainstorming flow."
  ],
  "New Features": [
    "Added an approval question that can automatically move a session into self-driving mode once a user approves it. The approval is propagated through workflow state and integrated with MCP tools and coordinator instructions so the mode switch is enforced end-to-end."
  ]
}

