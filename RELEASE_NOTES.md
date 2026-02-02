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

```json
{
  "Additional Changes": [
    "Updated the retrospective and writer agent prompts so they load shared skills and snippets from the standard overlay directories, and so they include agent improvement files during execution. Specifically, the prompts now read skills/snippets from the sgai overlay directory structure, incorporate the agent improvement artifacts, and send a coordinator notification via `sgai_send_message` before marking workflow steps as completed."
  ]
}

```
