# Release Notes

## 0.0.0+20260216 — Release notes update

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes the changes listed below.

```json
{
  "Additional Changes": [
    "Updated editor presets and open-in-editor behavior to simplify execution and improve cross-environment compatibility. The release simplified editor presets to execute commands as `exec.Command(command, path)` with stdio attached and refined VS Code preset behavior to use `code -g {path}` as the final configuration.",
    "Updated coordination and council modeling rules to make agent configuration more consistent and testable. The release made `project-critic-council` a mandatory coordinator successor, enforced ordering driven by `GOAL.md`, removed model entries and variants from agent definitions, and added tests ensuring implicit model inheritance from the coordinator when not explicitly listed.",
    "Updated documentation and templates to improve onboarding and reduce ambiguity. The release added terminology definitions for repository types and modes in `AGENTS.md`, expanded `GOAL.example.md` with guidance and nested checklist examples, clarified `.sgai` modification rules and restricted `PROJECT_MANAGEMENT.md` location, and updated installation/opencode usage docs (including the lowercase `sgai/` root and authentication/model requirements).",
    "Updated dependencies and toolchain versions across Go and the webapp. The release bumped the Go toolchain and multiple Go modules (including `golang.org/x/sys`, `golang.org/x/term`, `golang.org/x/tools`, `golang.org/x/oauth2`, and `github.com/modelcontextprotocol/go-sdk`) and updated webapp dependencies such as `react-markdown`, `lucide-react`, and `happy-dom` with corresponding lockfile refreshes.",
    "Updated workspace initialization and repository handling to standardize setup and improve reliability. The release introduced structured workspace initialization (skeleton unpack, `.sgai` git exclude, `GOAL.md` creation), improved `jj` detection/classification via filesystem inspection, handled missing `jj` gracefully, and extended CI to install `jj` with comprehensive tests.",
    "Updated workflow interaction controls to improve visibility and governance. The release added a steering form that injects guidance before the oldest unread message, added UI/HTTP support to delete individual workflow messages, required a comprehensive summary in the work-gate approval dialog, and persisted `InteractiveAutoLock` so approved self-drive mode remains locked across restarts."
  ],
  "Breaking Changes": [
    "Refactored the product into a web-only application model with a long-lived per-workspace server process. The release removed the legacy CLI/stdio MCP flow, introduced one long-lived HTTP MCP server per workspace, and rewired SSE and the UI so workspace interactivity and logs are driven through per-workspace events.",
    "Replaced the previous HTMX-based web UI with a React SPA and corresponding API surface. The release removed template-based UI flows, added JSON APIs plus an SSE watcher for the new webapp, and updated build/test/docs pipelines to assume the SPA architecture.",
    "Removed support for launching OpenCode with unsupported model variant flags. The release eliminated use of the `--variant` flag, documented the incompatibility, and adjusted related agent/model configuration expectations accordingly."
  ],
  "Bug Fixes": [
    "Fixed message rendering and formatting issues in the workspace UI to improve readability and template correctness. The release rendered message bodies as Markdown in the Workspace Messages tab with tests and corrected earlier prompt rendering to convert Markdown to HTML without forcing preformatted whitespace for multi-choice templates.",
    "Fixed workflow state and agent display edge cases that could cause runtime errors or confusing labels. The release prevented negative padding panics, returned base agent names for pending messages, and added tests validating agent lookup behavior.",
    "Fixed editor selection reliability when invoking external editors from the UI. The release added editor availability fallback logic, stopped disabling terminal editors, and later added a VS Code fallback when the configured editor command fails to open a file.",
    "Fixed workspace lifecycle state to avoid stale “ever started” indicators after completion. The release cleared the per-workspace ever-started flag once workflow state was complete and added tests covering the reset behavior."
  ],
  "Deprecations": [
    "Removed legacy HTMX UI flows and related migration-specific guidance after the React SPA became the supported interface. The release deleted old template-based web flows and removed HTMX/React migration skills and guidance, retaining only the written migration plan where applicable.",
    "Removed obsolete skeleton skills and guidance that were no longer supported. The release removed the `meta/log-analysis` skill and its trigger guidelines from the skeleton to keep the default configuration minimal and current."
  ],
  "New Features": [
    "Added a guided GOAL authoring experience in the web app to standardize how goals are created and edited. The release implemented a `GOAL.md` template+wizard flow (replacing the earlier composer), introduced a dedicated editor UI when `GOAL.md` content is empty, and added UI state to toggle auto vs interactive composition modes.",
    "Added ad-hoc “Run” prompting as a first-class workspace capability with improved streaming output. The release moved the Run UI into a dedicated workspace tab, executed one-shot `opencode` runs via stdin with stricter default permissions, and later replaced the HTMX flow with a Bun-built React SPA backed by JSON APIs and an SSE watcher.",
    "Added comprehensive workspace fork management to support isolated experimentation and controlled merges. The release introduced root-only fork management mode, kebab-case fork creation, automated fork merge (optionally via PR), fork rename and delete endpoints with UI and tests, and stricter gating to prevent forks from creating further forks.",
    "Added persistent workspace pinning to improve navigation and prioritization across the UI and menu integrations. The release introduced pin/unpin state persisted per workspace, exposed controls on the workspace page and for root workspaces with forks, and displayed pinned status in the tree view and Mac menu bar integrations.",
    "Added a macOS status bar application to monitor and open running factories from the desktop. The release implemented a Cocoa menu bar app, improved base URL generation for wildcard listen addresses, and later simplified menu contents and title formatting with supporting helpers and tests.",
    "Added an “Open in OpenCode” action to open the current workspace in the OpenCode TUI when the factory is running. The release added a terminal-based launch path, required `opencode` to be available in `PATH`, and ensured fork-merge metadata capture always used LLM assistance.",
    "Added a continuous workflow execution mode for self-drive to support long-running goal-driven operation. The release introduced backend continuous mode driven by `GOAL.md` `continuousModePrompt` and added a dedicated “Continuous Self-Drive” control in the UI.",
    "Added live log streaming notifications to the web UI to improve observability of running workspaces. The release published an SSE `log:append` event including the workspace key whenever a log line was captured and documented the live-reload behavior.",
    "Added an STPA workflow entrypoint as a reusable skill to reduce agent-specific setup. The release wired a new `stpa-overview` skill and simplified the `stpa-analyst` agent startup to delegate behavior to the skill-based workflow."
  ],
  "Security": [
    "Strengthened coordination safety rules to reduce unintended concurrent actions by non-coordinator agents. The release required non-coordinator agents to yield control after sending messages and introduced a continuation nudge when unread outbound messages are present."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.