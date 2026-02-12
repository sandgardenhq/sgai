# Release Notes

## 0.0.0+20260211 — React web UI and API layer

- **Date**: 2026-02-11
- **Version**: 0.0.0+20260211
- **Summary**: This release includes a React-based web UI backed by a new `/api/v1` API layer, fork workflow improvements, and reliability fixes across workspace and session workflows.

### New Features

- Added a React 19 single-page web UI embedded in the `sgai` binary and served by `sgai serve`.
- Added a `/api/v1/*` JSON API for workspace management, session control, and UI data fetching.
- Added a Server-Sent Events stream at `GET /api/v1/events/stream` for real-time UI updates.
- Added fork lifecycle APIs (`POST /api/v1/workspaces/{name}/fork`, `/merge`, `/delete-fork`, and `/rename`) to create, merge, delete, and rename forks.
- Added an ad-hoc prompt runner at `POST /api/v1/workspaces/{name}/adhoc` that executes `opencode run` with `stdin` and streams combined output.
- Added an `Open in OpenCode` action at `POST /api/v1/workspaces/{name}/open-opencode` that launches `opencode` with the current `--session` and `--agent`.
- Added project pinning at `POST /api/v1/workspaces/{name}/pin` to keep selected workspaces visible in the in-progress list.
- Added a steering endpoint at `POST /api/v1/workspaces/{name}/steer` that inserts a human message before the oldest unread message.

### Breaking Changes

- Removed the ability to start agentic work in root workspaces, so create a fork (via the UI or `POST /api/v1/workspaces/{name}/fork`) and start sessions from the fork.

### Bug Fixes

- Fixed new-workspace initialization to unpack the `.sgai` skeleton, initialize `.jj`/`.git`, and update `.git/info/exclude` so fork and session operations work immediately.
- Fixed forked-mode detection so repositories only enter forked mode when `jj workspace list` reports multiple workspaces.
- Fixed `Open in Editor` by restoring editor resolution and invocation across supported editor presets.
- Fixed implicit `project-critic-council` injection to match the coordinator model (including variants) when the agent is present in the flow but absent from `models`.
- Fixed a `strings: negative Repeat count` panic during workflow execution.
- Updated `/respond` rendering to use the shared Markdown-to-HTML pipeline for consistent formatting.
- Updated continuation messaging to instruct agents to yield control after calling `sgai_send_message`.

### Additional Changes

- Updated `opencode run` invocations to set `--title` for improved session traceability.
- Updated the workspace skeleton `opencode.jsonc` to deny `doom_loop` and `external_directory` permissions by default.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
