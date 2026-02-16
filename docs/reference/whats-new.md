# What's new

## 0.0.0+20260216 — Updates

- **Date**: 2026-02-16
- **Version**: 0.0.0+20260216
- **Summary**: This update addresses the changes listed below.

{
  "Additional Changes": [
    "Updated how skills and snippets are described and surfaced in documentation for consistency. The release consolidated `when_to_use` content into `description`, adjusted metadata layout/naming, and updated documentation to announce skill usage via `sgai_update_workflow_state` rather than plain text.",
    "Updated the work approval dialog to require a complete summary before approval is presented. The release enforced summary presence in the work-gate UI/flow so approvals are always accompanied by comprehensive context.",
    "Updated repository terminology and templates to improve documentation clarity. The release added terminology definitions for repository types and modes to `AGENTS.md` and extended `GOAL.example.md` with additional guidance and a sample nested checklist.",
    "Updated agent coordination rules to reduce stalled workflows. The release strengthened instructions for non-coordinator agents to yield control after sending messages and added a continuation nudge when unread outbound messages exist.",
    "Updated the skill discovery interface to behave as a search-only tool. The release adjusted `sgai find_skills` to return only skill names and descriptions and documented the behavior change in goals documentation.",
    "Updated the VS Code editor preset multiple times and finalized it to open files at a target location. The release converged the preset to use `code -g {path}` and simplified editor presets to execute commands as `exec.Command(command, path)` with stdio attached.",
    "Updated dependency versions across the Go backend and the webapp toolchain. The release bumped the Go toolchain and multiple Go modules (including `golang.org/x/sys`, `golang.org/x/oauth2`, `golang.org/x/term`, `golang.org/x/tools`, and `github.com/modelcontextprotocol/go-sdk`) and updated webapp dependencies (including `react-markdown`, `lucide-react`, and Happy DOM packages) with lockfile refreshes.",
    "Updated installation and usage documentation to reflect current tooling and directory conventions. The release added an OpenCode-based installation guide, required Node.js as a documented dependency, clarified serving/workspace behavior, standardized the root directory to lowercase `sgai/`, and required an authenticated, up-to-date `opencode` with a specific model flag.",
    "Updated UI structure and navigation for improved workflow visibility. The release refreshed the theme, added workflow visibility and navigation improvements, introduced deployment/agent-model metadata support, and refactored the dashboard layout to shared `Sidebar` and `Sheet` components with mobile support."
  ],
  "Breaking Changes": [
    "Replaced the previous HTMX-based interface and server-driven templates with a React SPA and new API surface for the web UI. The release removed old HTMX/template flows, introduced new JSON APIs and SSE-based update paths, and required webapp build/test/documentation updates to align with the new frontend architecture.",
    "Converted the application to a web-only architecture with one long-lived MCP HTTP server per workspace and removed the legacy CLI/stdio MCP flow. The release rewired backend routing, SSE, and UI eventing so workspace interactivity and logs are driven by per-workspace events and each workspace remains isolated.",
    "Enforced a persistent auto-drive lock once work approval is granted and removed configuration paths that previously influenced interactive mode. The release dropped frontmatter/CLI interactive-mode configuration and persisted the `InteractiveAutoLock` state so the lock survives sessions and restarts across backend, API, MCP, and UI layers.",
    "Removed support for an unsupported OpenCode launch flag. The release stopped using the `--variant` option when starting `opencode` and documented the constraint so tooling and automation avoid invalid invocations."
  ],
  "Bug Fixes": [
    "Fixed editor launching to remain functional when a configured editor command is unavailable or fails. The release added editor availability fallback logic, re-enabled terminal editors for “Open in Editor,” and added a VS Code fallback when the configured command cannot open the target path, with tests covering selection behavior.",
    "Fixed workspace initialization and repository detection edge cases to avoid failures when required tools are missing. The release handled a missing `jj` binary gracefully, improved workspace classification by inspecting `.jj/repo`, and centralized fork/root gating so forks cannot create further forks, with updated tests.",
    "Fixed rendering and formatting issues in message and prompt output. The release rendered message bodies as Markdown in the Workspace Messages tab, rendered `ask_user_question` prompts as HTML from Markdown, and removed forced preformatted whitespace in multi-choice templates with output tests.",
    "Fixed a runtime panic and improved agent naming for workflow messaging edge cases. The release prevented negative padding panics and ensured pending messages return base agent names, with tests covering the agent lookup behavior.",
    "Fixed workflow state bookkeeping so completed workspaces do not remain marked as started. The release cleared the ever-started flag when workflow state reaches completion and added tests to validate the reset behavior."
  ],
  "Deprecations": [
    "Removed native macOS notification behavior after introducing menu bar support. The release dropped the notification functionality and its usage paths and documented the removal in the associated goals documentation.",
    "Removed unused guidance and legacy analysis tooling from the default project skeleton. The release removed the meta/log-analysis skill, removed HTMX/React migration-specific guidance and skills, and simplified React best-practices guidance to generic Vercel-oriented rules."
  ],
  "New Features": [
    "Added a guided GOAL authoring experience that creates structured project goals and makes it the default when no goal content exists. The release introduced a template-and-wizard `GOAL.md` composer (replacing the earlier composer), added reference documentation/specs, and updated redirects and frontmatter/preview handling to match the new flow.",
    "Added a dedicated, always-available ad-hoc prompt runner in the web application for one-shot inferences with streaming output and preserved form state. The release moved the Run UI into its own workspace tab and standardized executions to stdin-based `opencode` runs with stricter default permissions and titled invocations.",
    "Added workspace forking and fork lifecycle management to support isolated work streams. The release introduced a root-only fork management mode, a kebab-case fork creation flow, automated fork merge (optionally via PR), fork renaming, and a dedicated endpoint/UI action to delete fork workspaces.",
    "Added persistent workspace pinning across the web UI and desktop menu experience to improve navigation. The release implemented pin/unpin state persistence, exposed pin/unpin controls on workspace pages (including root workspaces with forks), displayed pinned status in the tree view, and enabled pinned workspaces in the macOS menu bar.",
    "Added a React-based single-page dashboard application with supporting APIs for live updates and navigation. The release replaced the previous template-based UI with a Bun-built React SPA, added JSON APIs and an SSE watcher, and updated build/test/docs to account for the new webapp.",
    "Added a macOS status bar menu for monitoring and opening running factories. The release implemented Cocoa menu bar integration, improved dashboard base URL generation for wildcard listen addresses, and simplified menu contents and title formatting with supporting helpers and tests.",
    "Added an action to open the current workspace in the OpenCode terminal interface when a factory is running. The release introduced an “Open In OpenCode” command that launches the TUI via a terminal, requires `opencode` in `PATH`, and always uses LLM assistance for fork-merge metadata.",
    "Added a continuous workflow mode to keep work progressing without repeated manual starts. The release implemented backend continuous execution driven by `GOAL.md` `continuousModePrompt` and surfaced it in the UI as a dedicated Continuous Self-Drive control.",
    "Added a skill-based STPA workflow to standardize analysis behavior across runs. The release introduced a new `stpa-overview` skill and simplified the `stpa-analyst` agent startup to rely on the skill-driven workflow."
  ]
}

