# Release Notes

## 0.0.0+20260211 — React web UI and API layer

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes a React-based web UI backed by a new `/api/v1` API layer.

### New Features

- Added a React 19 single-page web UI embedded in the `sgai` binary.
- Added a `/api/v1/*` JSON API for workspace and session workflows.
- Added a Server-Sent Events stream at `GET /api/v1/events/stream`.
- Added fork lifecycle APIs (`POST /api/v1/workspaces/{name}/fork`, `/merge`, `/delete-fork`, and `/rename`).
- Added an ad-hoc prompt runner at `POST /api/v1/workspaces/{name}/adhoc`.
- Added an `Open in OpenCode` action at `POST /api/v1/workspaces/{name}/open-opencode`.
- Added project pinning at `POST /api/v1/workspaces/{name}/pin`.
- Added a steering endpoint at `POST /api/v1/workspaces/{name}/steer`.

### Breaking Changes

- Removed the ability to start agentic work in root workspaces.

### Bug Fixes

- Fixed new-workspace initialization so fork and session operations work immediately.
- Fixed forked-mode detection so repositories only enter forked mode when `jj workspace list` reports multiple workspaces.
- Fixed `Open in Editor` by restoring editor resolution across supported presets.
- Fixed implicit `project-critic-council` injection to match the coordinator model.
- Fixed a `strings: negative Repeat count` panic during workflow execution.
- Updated `/respond` rendering to use the shared Markdown-to-HTML pipeline.
- Updated continuation messaging to instruct agents to yield control after calling `sgai_send_message`.

### Additional Changes

- Updated `opencode run` invocations to set `--title`.
- Updated the workspace skeleton `opencode.jsonc` to deny `doom_loop` and `external_directory` by default.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
