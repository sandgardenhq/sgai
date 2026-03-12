# 2026-03-11 Weekly Update: Reliability and workflow polish

This week’s work focuses on reliability: workflows behave more consistently across different modes, reviewer feedback is stricter by default, and multi-workspace operations handle forks and external directories more predictably.

On top of that, the project picked up a few quality-of-life improvements: agent aliases make it easier to reuse an existing agent prompt under a new name, and developer workflows have fewer sharp edges due to test and tooling cleanup.

## 🚀 New Features

Two themes stand out in this week’s feature work: more flexible agent configuration, and better handling of workspaces that do not live exactly where the factory expects.

Agent aliases are now a first-class workflow feature.

An *agent alias* is a workflow-level name that points at an existing *base agent*. When SGAI runs an agent, it resolves the alias to the base agent name before loading the agent definition from `.sgai/agent/<agent>.md`.

This is useful when the workflow should treat two names as separate roles (for readability or intent), but both roles should share the same prompt/tools/snippets. It also supports a common pattern: keep the base agent definition the same, but assign a different model to the alias name via `models:` in `GOAL.md` frontmatter.

* **Agent aliases** - Define `alias:` mappings in `GOAL.md` frontmatter so a workflow can refer to an existing agent prompt under an alternate name. The alias resolves to a base agent name when SGAI runs an agent.
* **External workspace fork tracking** - Record fork directories as external when the target workspace path is external, using the symlink-resolved fork path.
* **Expanded MCP external tool coverage** - Exercise the external MCP server’s tool surface area with a broad set of success and error-path tests (workspace, state, session, knowledge, compose, ad-hoc, editor, and model tools).

## 🧯 Bug Fixes

This week includes regression fixes and guardrails around workflow completion and workspace path handling.

Workflow completion can now be blocked until a configured retrospective step runs. When the workflow is about to finish but the retrospective agent has not run yet, SGAI injects a redirect message to the `retrospective` agent and persists the updated state.

Workspace root detection is also more defensive. Root paths are normalized via symlink resolution before comparisons, and the external-root resolver returns early when no root workspace path is available.

* **Fix regressions from #356** - Harden workflow behavior across continuous/retrospective handling, workspace/fork operations, and external repo handling.
* **Block completion until retrospective runs** - Intercept workflow completion when a `retrospective` node exists but has not run yet, then append a redirect message and save state.
* **Symlink-normalized workspace comparisons** - Resolve symlinks for root/workspace path comparisons to avoid false mismatches.
* **Defensive root path detection** - Return early when no root workspace path is available, and avoid reporting a root when required filesystem paths do not exist.

## 🛠 Internal Updates

Maintenance work this week tightens reviewer standards, consolidates tests, and reduces duplicated utilities across Go and the web app.

Reviewer agents are now configured with stricter defaults. The updated prompts emphasize read-only review behavior, disallow bash, and treat findings as blocking issues.

The codebase also went through simplification passes: shared logic is factored into common services, repeated helpers are removed, and tests are rewritten into larger consolidated suites.

* **Make `make test` include web app build and tests** - Update the Makefile so the `test` target depends on both `webapp-test` and `webapp-build`.
* **Use stricter reviewer agent rules** - Tighten reviewer prompts so reviews are read-only, deny bash, and treat findings as mandatory blocking issues.
* **Consolidate Go and React test utilities** - Centralize server test helpers, React test utilities, and workspace fixtures.
* **Simplify code without behavior changes** - Remove unused helpers and consolidate overlapping utilities while updating call sites and tests.
* **Update code auditing guidance** - Expand `AGENTS.md` guidance to check both literal usage and semantic liveness when auditing for dead routes.
* **Add global ignore patterns** - Update `.gitignore` to ignore `cover*.out`.
* **Update planning artifacts** - Add and update `GOALS/` entries for ongoing cleanup and issue tracking.

---
Written by [doc.holiday](https://doc.holiday)
