# Release Notes

## 0.0.0+20260216 — Web-only architecture and new workspace flows

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes a web-only architecture, expanded workspace management, and improvements across the UI, workflows, and documentation.

### New Features

- Added an `AGENTS` reference document that describes built-in agent roles and usage conventions.
- Added a guided `GOAL.md` authoring flow with a template, interactive wizard/composer, and phase-based workflow.
- Added an always-available Run tab for one-shot `opencode` prompts via stdin with improved streaming UX and stricter default permissions.
- Added workspace pinning with persistent pin/unpin state and pinned indicators across the workspace tree and macOS menu bar.
- Added fork lifecycle management for root workspaces, including fork create (kebab-case), rename, delete, and fork-merge with optional PR support.
- Added a Bun-built React SPA with supporting JSON APIs and SSE watchers as the primary web interface.
- Added per-workspace live log streaming via SSE `log:append` events that include the workspace key.
- Added a web-only architecture that runs one long-lived HTTP MCP server per workspace and isolates interactive state per workspace.
- Added a continuous self-drive mode driven by `GOAL.md` `continuousModePrompt` and exposed as a UI control.
- Added the `stpa-overview` skill and updated STPA startup so workflows are skill-driven rather than agent-wired.
- Added an “Open In OpenCode” action that launches the OpenCode TUI locally when `opencode` is available on `PATH`.

### Breaking Changes

- Removed the CLI/stdio MCP flow in favor of the web UI and per-workspace long-lived HTTP MCP servers.
- Updated `sgai_find_skills` to return only skill names and descriptions, so callers must not rely on full skill payloads in the search response.
- Removed use of the unsupported `--variant` flag when launching OpenCode, so invocations must use the documented model flag.

### Bug Fixes

- Fixed editor launching to fall back reliably when a preferred editor is missing or fails by detecting editor availability, allowing terminal editors, and falling back to VS Code.
- Fixed workflow and message rendering crashes by preventing negative padding panics and correcting pending-message agent name resolution.
- Fixed workflow completion to clear the “ever started” flag so workspaces do not remain in an incorrect started state.
- Fixed HTMX compose responses to use `Hx-Trigger` header casing for compatibility.

### Deprecations

- Deprecated the legacy HTMX/template-driven UI in favor of the React SPA with JSON and SSE APIs.

### Additional Changes

- Updated prompt and message rendering to display `ask_user_question` prompts as HTML from Markdown and render message bodies as Markdown in the Workspace Messages tab.
- Updated interactive mode to persist an auto-drive lock in workflow state after work-gate approval, including persisting `InteractiveAutoLock` across sessions.
- Updated workspace initialization with a skeleton-based flow that configures `jj`, updates git excludes (including `.sgai`), writes `GOAL.md`, and handles missing `jj` binaries.
- Updated `AGENTS.md` terminology to define repository types and modes and clarify `.sgai` modification rules.
- Updated coordinator and council workflows to require project-critic-council as a successor, align ordering with `GOAL.md`, and fall back to a coordinator model when council is absent.
- Added message steering that can inject a guidance message before the oldest unread message and added UI/HTTP support for deleting individual workflow-state messages.
- Updated agent and skill metadata by consolidating `when_to_use` into `description` and removing the meta/log-analysis skill and trigger guidelines from the skeleton.
- Updated editor presets to normalize path opening via `exec.Command(command, path)` with stdio attached.
- Updated the macOS menu bar integration to add a Cocoa status app, improve dashboard base URL generation for wildcard listen addresses, and streamline menu formatting.
- Updated installation and developer docs to add an OpenCode-based install guide, document Node.js dependencies, and update test listen address guidance to use ephemeral ports.
- Updated dependencies across Go and the webapp, including `golang.org/x/sys`, `golang.org/x/oauth2`, `golang.org/x/term`, `golang.org/x/tools`, `github.com/modelcontextprotocol/go-sdk`, `react-markdown`, `lucide-react`, and `happy-dom`.
- Removed macOS notification functionality and associated documentation after experimentation.
- Updated the Dashboard UI to use shared Sidebar/Sheet components with mobile support and refreshed theme and styles.
- Updated work-gate approval dialogs to require and display a comprehensive summary to reduce accidental approvals.
- Updated forked-mode root workspaces to remove `GOAL` actions and hide the root repository status line.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
