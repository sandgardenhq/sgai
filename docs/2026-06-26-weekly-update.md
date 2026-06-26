# 2026-06-26 Weekly Update: June workflow and maintenance roundup

June split into two active weeks and two quiet ones. June 1 concentrated on diagnostics and usage visibility, and June 15 focused on orchestration, documentation, and maintenance. June 8 and June 22 stayed quiet, with no merged changes recorded by publication time.

## Week of June 1

June opened with better failure tracing and clearer usage data. Agent failure logging now captures more of the context needed to explain what went wrong, AskAndWait, the pause that waits for a human response, now logs through the configurable path, and the new usage page shows token usage, the total amount of text the system processed.

- Agent failure logging now keeps more surrounding context, which shortens the path from a failed run to the cause.
- AskAndWait now logs through the configurable path, so human response pauses stay visible alongside the rest of the run output.
- Global usage tracking now feeds the new `/usage` page and the backfill flow, so token consumption stays visible across the system.

## Week of June 8

No merged changes landed during the week of June 8.

## Week of June 15

June 15 carried the broadest set of changes. Workflow control shifted from the message bus to explicit navigation in MCP (`sgai mcp`, the workflow tool server), with the requested next step flowing through the shared workflow state file (`.sgai/state.json`). The related docs now match that model, while the rest of the week tightened UI checks, refreshed dependencies, fixed human response handling, corrected active subagent tracking, and made opencode runs more deterministic by setting `PWD`, the working directory environment variable, and `OPENCODE_CONFIG_DIR`.

- Workflow navigation and docs now use explicit handoff requests instead of the message bus, and the MCP and workflow state references now explain the `navigate` flow clearly.
- The webapp now handles `react-doctor`, the React checker used in CI, more reliably after the checker exposed a regression in its output format.
- Dependency refreshes brought in updated `go-isatty`, `modernc.org/sqlite`, `modernc.org/libc`, and `golang.org/x/sys` modules.
- Human response handling now processes `ask_user_question` replies directly, and active subagents, the currently tracked delegated tasks, now surface correctly in structured output.
- Usage refresh and reconciliation now update workspace token counts before deletion or reset, which keeps usage totals aligned with the current workspace state.
- `opencode` runs now set `PWD` and `OPENCODE_CONFIG_DIR` explicitly, so CLI invocation stays deterministic across environments.
- The obsolete explainer HTML file no longer ships with the repository.

## Week of June 22

No merged changes have landed as of June 26, so the week of June 22 remains quiet to date.

---

Written by [doc.holiday](https://doc.holiday) ✌️