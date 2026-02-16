# Release Notes

## 0.0.0+20260216 — New features and fixes

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This release includes new capabilities and fixes across the CLI and supporting packages.

### Source (unformatted)

```json
{
  "Additional Changes": [
    "User prompts and message content are rendered more readably in the UI. This renders ask_user_question prompts as HTML from Markdown (without forcing preformatted whitespace in multi-choice templates) and renders message bodies as Markdown in the Workspace Messages tab with new tests.",
    "Interactive mode behavior is simpler and more consistent across restarts. This enforces a persistent auto-drive lock in workflow state once the work gate is approved, persists `InteractiveAutoLock` across sessions, and updates backend/MCP/server APIs/web UI/tests accordingly.",
    "Workspace initialization is more structured and consistent for new and forked workspaces. This introduces a skeleton-based initialization flow that configures jj, updates git excludes (including `.sgai`), writes `GOAL.md`, and handles missing `jj` binaries gracefully with expanded test coverage.",
    "Repository and workflow terminology is documented more clearly for users and contributors. This adds definitions for repository types and modes to `AGENTS.md`, clarifies `.sgai` modification rules, and restricts `PROJECT_MANAGEMENT.md` location guidance in the agent documentation.",
    "Coordination and council workflows have been tightened to improve consistency and validation. This makes project-critic-council a mandatory coordinator successor, aligns council ordering with `GOAL.md`, rewrites the council protocol to a FrontMan-centric template with strict messaging steps, and ensures council falls back to the coordinator model when absent from models (while removing model entries/variants from agent definitions).",
    "Message steering and workflow state management now support more targeted interventions. This adds a steering form that injects a guidance message before the oldest unread message and adds UI/HTTP support for deleting individual workflow-state messages.",
    "Agent and skill metadata has been standardized across the repository. This consolidates `when_to_use` into `description`, adjusts metadata layout/naming, updates documentation to announce skill usage via `sgai_update_workflow_state`, and removes the meta/log-analysis skill and trigger guidelines from the skeleton.",
    "Editor presets and commands have been simplified and normalized. This updates editor execution to open paths via `exec.Command(command, path)` with stdio attached and standardizes presets after several iterations of VS Code command behavior.",
    "The macOS status bar integration has been improved and then simplified for clarity. This adds a Cocoa menu bar app for monitoring/opening factories, improves dashboard base URL generation for wildcard listen addresses, and streamlines menu contents/title formatting with supporting helpers and tests.",
    "Installation and developer documentation has been updated to reflect the current toolchain and workflow. This adds an opencode-based install guide, documents Node.js as a dependency (including Homebrew instructions), updates test listen address guidance to use ephemeral ports, and refreshes docs to use a lowercase `sgai/` root directory and require an authenticated, up-to-date opencode invocation with the documented model flag.",
    "Dependencies have been updated across Go and the webapp. This updates the Go toolchain and multiple Go modules (including `golang.org/x/sys`, `x/oauth2`, `x/term`, `x/tools`, and `github.com/modelcontextprotocol/go-sdk`) and updates web dependencies such as `react-markdown`, `lucide-react`, and `happy-dom` packages with lockfile refreshes.",
    "Notification behavior on macOS has been removed after experimentation to reduce complexity. This reverses earlier notification work by dropping notification functionality and its associated documentation/goals.",
    "The UI has been refreshed to improve navigation and mobile usability. This refactors the Dashboard layout to shared Sidebar/Sheet components with mobile support, refreshes theme/styles, and adjusts skills/templates/tests to match the updated layout.",
    "Work-gate approvals now require better context to avoid accidental approvals. This requires and displays a comprehensive summary when presenting the work-gate approval dialog.",
    "Root workspaces have a clearer, more restricted set of actions when operating in forked mode. This removes GOAL-related logic/buttons for root workspaces and hides the root repository status line when in forked mode to reduce confusion."
  ],
  "Breaking Changes": [
    "The application is now web-only, and previous command-line/stdio integration paths are no longer available. This removes the old CLI/stdio MCP flow and requires using the web UI plus per-workspace long-lived HTTP MCP servers for interactive workflows.",
    "Skill discovery now behaves like a search tool rather than returning full skill payloads. This changes `sgai_find_skills` to return only skill names and descriptions, which may require updating any callers that expected richer skill data.",
    "OpenCode launch behavior no longer attempts to use unsupported model variant flags. This removes use of the unsupported `--variant` flag and updates documentation and tests to enforce the supported invocation model."
  ],
  "Bug Fixes": [
    "Opening files in an editor is more resilient when a preferred editor is missing or fails. This adds editor availability detection, improves preset selection behavior (including allowing terminal editors), and falls back to VS Code when the configured editor command fails.",
    "Workflow and message rendering issues that could cause crashes or confusing output have been corrected. This prevents negative padding panics, improves agent name resolution for pending messages by returning base agent names, and adds tests for the affected lookup behavior.",
    "Workspaces no longer get stuck in an incorrect “ever started” state after finishing a workflow. This clears the ever-started flag when workflow state is complete and validates the behavior with new tests.",
    "HTMX-triggered compose responses now use the correct header casing for compatibility. This renames `HX-Trigger` to `Hx-Trigger` in compose handlers to align with expected header conventions."
  ],
  "Deprecations": [
    "Legacy HTMX/template-based UI flows are no longer the primary interface. This deprecates the older template-driven experience by replacing it with a React SPA and corresponding JSON/SSE APIs."
  ],
  "New Features": [
    "A new reference document explains what each built-in agent does and when to use it. This adds an SGAI agents reference that documents roles, responsibilities, and usage conventions in a single place.",
    "A new guided GOAL authoring experience helps users create better project goals from the start. This introduces a GOAL.md template + interactive wizard/composer (with reference docs and a phase-based workflow), makes it the default when GOAL content is empty, and updates preview/frontmatter/redirect handling to match the new flow.",
    "Workspaces can now run ad-hoc prompts from a dedicated area with a clearer streaming experience and safer defaults. This adds a always-available Run tab that executes one-shot opencode inferences via stdin with improved streaming UX and stricter default permissions (superseding the earlier per-workspace ad-hoc prompt UI).",
    "Workspaces can now be pinned so important repos stay easy to find. This adds persistent pin/unpin state, a pin/unpin button on workspace pages (including root workspaces that have forks), and pinned indicators across the workspace tree and macOS menu bar integration.",
    "Forked workspace management now covers more of the full lifecycle directly in the product. This adds root-only fork management, kebab-case fork creation, fork rename and delete endpoints/UI, and an automated fork-merge flow with optional PR support and improved HTMX redirect behavior.",
    "The web experience has been upgraded to a modern single-page app with live updates. This replaces the previous HTMX UI with a Bun-built React SPA, adds supporting JSON APIs plus SSE-based watchers, and updates build/test/docs to use the new webapp as the primary interface.",
    "Sgai now supports per-workspace live log updates over server-sent events. This publishes an SSE `log:append` event that includes the workspace key whenever a log line is captured, enabling log live-reload driven by per-workspace events.",
    "Sgai now runs as a web-only system with a more reliable per-workspace server model. This refactors the architecture to use one long-lived HTTP MCP server per workspace, removes the older CLI/stdio MCP flow, and rewires backend/SSE/UI so each workspace is isolated and driven by per-workspace interactive state and events.",
    "A new continuous self-drive mode makes it easier to run ongoing, goal-driven work without manual restarts. This introduces a backend continuous workflow mode driven by `GOAL.md` `continuousModePrompt` and exposes it in the UI as a dedicated Continuous Self-Drive control.",
    "STPA analysis has been moved to a skill-first workflow for consistency across projects. This adds a new `stpa-overview` skill and simplifies the `stpa-analyst` agent startup so STPA behavior is driven primarily by skills instead of agent-centric wiring.",
    "The product can now open a workspace directly in the OpenCode TUI when running locally. This adds an “Open In OpenCode” action that launches OpenCode in a terminal when the factory is running, requires opencode on PATH, and always uses LLM assistance for fork-merge metadata."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
