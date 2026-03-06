# Release Notes

## 0.0.0+00010101 — Release notes update

- **Date**: 0001-01-01
- **Version**: 0.0.0+00010101
- **Summary**: Quick update: see the items below.

### Notes (raw)

```json
{
  "🚀 New Features": [
    "You can now connect external workspaces and manage them without losing data. This adds external workspace attach/detach support, shows external workspaces in the UI with non-destructive removal semantics, and introduces backend delete flows with UI confirmation dialogs for workspace deletion.",
    "You can now view a full workspace diff from the server to better understand changes across a project. This introduces a workspace full-diff API endpoint backed by `jj diff --git` and documents the intended usage goal for the endpoint.",
    "You can now edit markdown content with a richer in-browser editor experience. This adds a Monaco-based markdown editor with toolbar and preview, reuses it across markdown inputs, and updates markdown rendering and tests accordingly.",
    "You can now see YAML frontmatter as a structured preview while editing markdown. This adds a YAML-based frontmatter table preview and a Monaco Ctrl/Cmd+A select-all action, along with new dependencies and test coverage.",
    "You can now create forks and immediately continue configuring their goals without extra navigation. This updates `NewFork` to navigate to the workspace goal editor after fork creation, and refreshes the related tests and goals documentation.",
    "You can now use an MCP-compatible HTTP endpoint that matches the existing web API tooling behavior. This adds an external MCP HTTP endpoint with tool parity to sgai’s web API, refactors server logic into service methods, and documents an agentskills.io-compliant HTTP skills interface plus MCP configuration."
  ],
  "🚧 Bug Fixes": [
    "Renaming a workspace no longer leaves behind stale or mismatched state that can make the UI behave incorrectly. This re-keys and clears workspace-related caches and state on rename (pins, adhoc/composer state, and classification/bookmark caches) and documents the bugfix goal that drove the change.",
    "Pinned workspaces now behave consistently even when directories are referenced through symlinks. This stores and manages pinned workspace directories using symlink-resolved paths and adds tests plus a GOALS document to lock in the expected symlink behavior.",
    "Waiting for human input is now more reliable when a question times out or an MCP config file is updated. This fixes `opencode.jsonc` MCP updates to preserve all fields, ensures `AskAndWait` preserves question state on timeout with improved logging, refines waiting-for-human handling in the agent loop, and adds tests for these cases.",
    "Retrospective flows now transition modes more predictably and follow stricter PR-creation guidance. This fixes retrospective mode transition behavior, adds explicit retrospective-mode prompt/flow sections with tests, and tightens PR creation prompt constraints.",
    "Long-running “completion gate” scripts are less likely to hang a workflow and can be cancelled safely. This makes the completion gate execution context-aware and killable, adds cancellation tests, and includes a new goal configuration file describing the expected behavior.",
    "The workbench plugin is less likely to fail when an interactive user response takes longer than expected. This increases the MCP HTTP transport timeout and adds a goal configuration file that documents user question timeout expectations."
  ],
  "🛠 Internal Updates": [
    "Workflows now have a more consistent and recoverable coordination model for human-in-the-loop operation. This introduces a state `Coordinator` with blocking human interaction, removes file-based ask/answer and reset APIs, adds tight-loop recovery, and wires coordinator state through MCP, server, agents, and the UI.",
    "Workflow execution is now easier to reuse and test across different entry points. This refactors workflow execution into a reusable runner, tightens coordinator/watchdog and `AskAndWait` behavior, restructures skills/snippets lookup, and updates UI hooks/state handling, configs, agents, and tests.",
    "Workspace metadata is now structured around goals rather than ad-hoc summaries to unify behavior across creation and forking. This replaces workspace summaries with GOAL-based descriptions, hardens workspace/fork creation, adds fork GOAL templates, and updates the web UI for inline forking, an editable sidebar, and richer GOAL editing with autocomplete.",
    "The system’s “actions” configuration is now more explicit and the internal UI is less cluttered. This makes an explicit empty `actions` array override default actions, removes the Start Application button bar from the internal session tab UI (and its tests), and adds a GOALS description file.",
    "Review agents now apply stricter, more consistent standards when evaluating changes. This tightens reviewer prompts to be read-only, deny bash, and treat findings as mandatory blocking issues, with workflow/skill/example updates across Go, React, HTMX/PicoCSS, and shell-script reviewers.",
    "Retrospectives now include a required health-analysis section with clearer documentation about where information should live. This makes AGENTS.md health analysis a mandatory structured part of retrospectives and clarifies the roles of `SGAI_NOTES.md` vs `AGENTS.md`.",
    "The critic council protocol now supports explicit dissent to surface alternative viewpoints during review. This adds a MinorityReport dissent role to the `project-critic-council` multi-model protocol and updates steps, roles, and templates to support it.",
    "The UI and workflow experience for syncing, logging, and navigation has been streamlined and modernized. This updates SGAI workflow syncing and logging, removes commit message editing, adds fork actions in the root repo view, adjusts key/scroll/layout behaviors, expands retrospective and Go review skills, and relocates the Go review skill into the skeleton.",
    "Retrospective agent wiring is now easier to reason about and better covered by tests. This restores conditional retrospective agent wiring in the workspace DAG, routes the coordinator prompt through `project-critic-council` before retrospectives, and adds tests plus a goals scenario file to validate the flow.",
    "Build and test automation has been tightened to reduce accidental gaps in webapp validation. This updates the `test` Makefile target to depend on both `webapp-test` and `webapp-build`.",
    "Dependency and SDK updates have been applied to keep the Go stack current. This updates `github.com/segmentio/asm` from v1.1.3 to v1.2.1 and updates Go dependencies for the modelcontextprotocol go-sdk and its JWT dependency.",
    "Repository hygiene and contributor guidance have been improved to reduce noise and make contributions more spec-driven. This adds a global `.DS_Store` ignore, updates `absorb-sgai` to skip `README.md` and `.DS_Store` via `find` filters, and clarifies README contribution guidelines around spec-based contributions and naming conventions.",
    "Documentation and goal specifications were expanded to support new deployment and coordination workflows. This adds goal and deployment agent specifications for Vercel, Cloudflare Workers, and exe.dev, adds platform-specific permission-aware installation steps, and introduces multiple GOALS configuration files for tracked issues (including issues 322 and 357) and workflow/model selection.",
    "Fork, compose, and messaging subsystems were refactored to reduce duplication and harden behavior without changing user-facing semantics. This adds safer fork workspace rollback, centralizes message deletion into a shared service, consolidates server test helpers, hardens compose wizard storage/error handling and dirty-state logic, and unifies React test utilities for fetch, Markdown editor, and workspace fixtures while removing unused helpers/components and simplifying APIs.",
    "Workspace rename support was removed in favor of explicit attach/detach flows for external workspaces. This removes all workspace rename functionality while ensuring external workspaces can be surfaced and managed through attach/detach semantics in the UI and backend."
  ]
}

```

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
