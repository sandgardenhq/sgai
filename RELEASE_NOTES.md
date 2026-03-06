# Release Notes

## 0.0.0+00010101 — Workspaces, markdown editor, and MCP HTTP

- **Date**: 0001-01-01
- **Version**: 0.0.0+00010101
- **Summary**: Quick update: this release beefs up workspace management, streamlines the in-browser markdown editing experience, and rolls out an MCP-compatible HTTP endpoint.

### New Features

This release focuses on making it easier to manage workspaces and collaborate on changes, while also improving the editing experience in the web UI.

- **External workspaces** - Added attach/detach support for external workspaces with non-destructive removal semantics and confirmed delete flows.
- **Workspace full diff endpoint** - Added a server-side full diff API backed by `jj diff --git`.
- **Markdown editor** - Added a Monaco-based markdown editor with toolbar and preview for a richer in-browser experience.
- **Frontmatter preview** - Added a structured YAML frontmatter preview (plus a Monaco Ctrl/Cmd+A select-all action) while editing markdown.
- **Fork flow** - Updated `NewFork` to navigate to the workspace goal editor immediately after fork creation.
- **MCP HTTP endpoint** - Added an external MCP-compatible HTTP endpoint with tool parity to sgai’s web API.

### Bug Fixes

These changes tackle a handful of state and timeout issues to make long-running and interactive workflows more predictable.

- **Workspace rename state** - Fixed stale UI state on rename by re-keying and clearing workspace-related caches.
- **Pinned workspaces and symlinks** - Fixed pinned workspace behavior by storing directories as symlink-resolved paths.
- **Human input waiting** - Fixed `AskAndWait` timeout handling and preserved all fields when updating `opencode.jsonc` MCP configs.
- **Retrospective mode transitions** - Fixed retrospective mode transitions and tightened PR-creation guidance.
- **Completion gate cancellation** - Fixed long-running completion gate scripts by making execution context-aware and killable.
- **Workbench timeouts** - Increased MCP HTTP transport timeout to reduce failures during slow interactive responses.

### Internal Updates

Under the hood, this release streamlines coordinator and workflow execution paths, tightens review/retrospective protocols, and refreshes build/test and documentation scaffolding.

- **Coordinator model** - Added a stateful `Coordinator` to manage blocking human interaction and improve recovery.
- **Workflow runner refactor** - Refactored workflow execution into a reusable runner and updated skills/snippets lookup.
- **GOAL-based workspace metadata** - Replaced ad-hoc workspace summaries with GOAL-based descriptions across creation and forking.
- **Actions configuration** - Made an explicit empty `actions` array override default actions and removed internal UI clutter.
- **Review agent standards** - Refined reviewer prompts to be read-only, disallow bash, and treat findings as blocking.
- **Retrospective health analysis** - Required a structured health-analysis section in `AGENTS.md` and clarified notes locations.
- **Critic council dissent** - Added a `MinorityReport` dissent role to the `project-critic-council` protocol.
- **UI and navigation polish** - Streamlined SGAI syncing/logging and adjusted root repo navigation and layout behaviors.
- **Retrospective wiring coverage** - Restored conditional retrospective agent wiring and expanded flow tests.
- **Build/test tightening** - Updated `make test` to depend on both `webapp-test` and `webapp-build`.
- **Dependency updates** - Updated Go dependencies including `github.com/segmentio/asm` and the modelcontextprotocol go-sdk.
- **Repo hygiene** - Improved contributor guidance and reduced noise with a global `.DS_Store` ignore.
- **Docs and goals** - Expanded goal and deployment specs for Vercel, Cloudflare Workers, and exe.dev.
- **Fork/compose refactors** - Refactored fork, compose, and messaging subsystems to reduce duplication and harden behavior.
- **Workspace rename removal** - Removed workspace rename support in favor of explicit attach/detach flows for external workspaces.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
