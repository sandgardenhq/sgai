# 2026-06-26 Weekly Update: June workflow and maintenance roundup

June split into two active weeks and two quiet ones. June 1 concentrated on diagnostics and usage visibility, and June 15 focused on orchestration, documentation, and maintenance. June 8 and June 22 stayed quiet, with no merged changes recorded by publication time.

## Week of June 1

June opened with better failure tracing and clearer usage data. The week stays focused on diagnostics first and usage visibility second.

### Logging and diagnostics

Agent failure logging now keeps more surrounding context. AskAndWait, the pause that waits for a human response, now logs through the configurable path so the pause stays visible alongside the rest of the run output.

- Agent failure logging now captures more surrounding context, which shortens the path from a failed run to the cause.
- AskAndWait now logs through the configurable path, so human response pauses stay visible alongside the rest of the run output.

### Usage visibility

Global usage tracking now feeds the new `/usage` page and the backfill flow. Token usage, the total amount of text the system processed, stays visible across the system.

- Global usage tracking now feeds the new `/usage` page and the backfill flow, so token consumption stays visible across the system.

## Week of June 8

No merged changes landed during the week of June 8.

## Week of June 15

June 15 carried the broadest set of changes. Workflow control moved away from the message bus, and the rest of the week focused on runtime reliability, usage reconciliation, dependency maintenance, and cleanup.

### Workflow navigation and docs

Workflow control now uses explicit `navigate` handoffs instead of the message bus. The surrounding docs now describe the flow through MCP and the shared workflow state file.

- Workflow navigation now uses explicit handoff requests, and the MCP and workflow state references now explain the `navigate` flow.

### Runtime reliability and UI fixes

The webapp now handles `react-doctor`, the React checker used in CI, more reliably after a regression in its output format. Human response handling now processes `ask_user_question` replies directly, active subagents now surface correctly in structured output, and `opencode` runs now set `PWD` and `OPENCODE_CONFIG_DIR` explicitly so CLI invocation stays deterministic across environments.

- The webapp now handles `react-doctor`, the React checker used in CI, more reliably after a regression in its output format.
- Human response handling now processes `ask_user_question` replies directly, and active subagents, the currently tracked delegated tasks, now surface correctly in structured output.
- `opencode` runs now set `PWD` and `OPENCODE_CONFIG_DIR` explicitly, so CLI invocation stays deterministic across environments.

### Usage refresh and reconciliation

Usage refresh and reconciliation now update workspace token counts before deletion or reset. That keeps usage totals aligned with the current workspace state.

- Usage refresh and reconciliation now update workspace token counts before deletion or reset, which keeps usage totals aligned with the current workspace state.

### Maintenance and internal updates

Dependency refreshes brought in updated `go-isatty`, `modernc.org/sqlite`, `modernc.org/libc`, and `golang.org/x/sys` modules. The obsolete explainer HTML file no longer ships with the repository.

- Dependency refreshes brought in updated `go-isatty`, `modernc.org/sqlite`, `modernc.org/libc`, and `golang.org/x/sys` modules.
- The obsolete explainer HTML file no longer ships with the repository.

## Week of June 22

No merged changes have landed as of June 26, so June 22 remains quiet to date.

---

Written by [doc.holiday](https://doc.holiday) ✌️