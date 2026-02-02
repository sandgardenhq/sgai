---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
  "project-critic-council": ["opencode/kimi-k2.5-free", "opencode/minimax-m2.1-free", "opencode/glm-4.7-free"]
interactive: yes
completionGateScript: make test
mode: graph
---

- [x] Interview me to ask what I think of what you've done so far.

---

## Reduce Accidental Complexity (all 4 phases approved)

Zero behavior change. Pure file splits first, then deeper simplifications. `make test` + `make lint` must pass after every step. One commit per file split.

### Phase 1: Split serve.go (3800+ → ~1000 lines)
- [x] Step 1.1: Extract `serve_jjlog.go` (~290 lines — JJ log parsing & rendering)
- [x] Step 1.2: Extract `serve_retrospective.go` (~800 lines — all retro web handlers, types, parsing)
- [x] Step 1.3: Extract `serve_workspace.go` (~200 lines — workspace scanning, grouping, fork detection)
- [x] Step 1.4: Extract `serve_viewmodels.go` (~250 lines — view model structs & preparation)
- [x] Step 1.5: Extract `serve_skills_snippets.go` (~150 lines — skills/snippets browsing handlers)
- [x] Step 1.6: Extract `serve_actions.go` (~200 lines — workspace CRUD action handlers)

### Phase 2: Split main.go (2690 → ~1400 lines)
- [x] Step 2.1: Extract `stream_writer.go` (~370 lines — prefixWriter, jsonPrettyWriter, streamEvent types)
- [x] Step 2.2: Extract `human_interaction.go` (~200 lines — editor, terminal multichoice, state transition)
- [x] Step 2.3: Extract `frontmatter.go` (~250 lines — YAML parsing, GoalMetadata, model validation)
- [x] Step 2.4: Extract `cmd_status.go` (~220 lines — cmdStatus, cmdListAgents CLI commands)
- [x] Step 2.5: Extract `retrospective_io.go` (~150 lines — file copy, session export, retro dir management)
- [x] Step 2.6: Extract `layer_overlay.go` (~60 lines — sgai layer folder overlay)

### Phase 3: Split mcp.go (1243 → ~970 lines)
- [x] Step 3.1: Extract `mcp_skills.go` (~135 lines — findSkills search logic)
- [x] Step 3.2: Extract `mcp_snippets.go` (~135 lines — findSnippets search logic)

### Phase 4: Deeper Simplifications
- [x] Step 4.1: Simplify `routeWorkspace()` — 35 switch cases → route table pattern
- [x] Step 4.2: Reduce `sessionData` fields — 30+ field struct → smaller focused structs
- [x] Step 4.3: Unify tab rendering pattern — identical load-state→build-data→execute-template
- [x] Step 4.4: Extract SVG rendering — DOT generation + Graphviz invocation into own file

----

- [x] stress-test the last few changes, prove they are correct and the behavior is correct.

---

## Fix Confirmation: `pyxkuutx` (cmd/sgai: attempt to fix interview regression)

**Status: CONFIRMED WORKING** — all tests pass, lint passes.

### What the fix does
1. Adds `isHumanPendingStatus(s string) bool` helper in `mcp.go`
2. In `updateWorkflowState`, checks if current status is human-pending before allowing status transition
3. If status IS human-pending (`human-communication` or `waiting-for-human`), preserves it — only task/progress are updated
4. Returns a clear message to the agent explaining the preservation
5. Adds 212-line test file with 12 test cases covering all scenarios

### Why it fixes the regression
After the simplification that removed `humanMessage` from `update_workflow_state`, agents could still overwrite `human-communication`/`waiting-for-human` statuses by calling `updateWorkflowState(status: "working")`. This caused the interview loop to break because the state machine lost track of being in "waiting for human" mode.

## Code Cleanup Analysis

### Opportunity 1: Move `isHumanPendingStatus` to `pkg/state`
- Currently private in `cmd/sgai/mcp.go`
- Same boolean condition duplicated inline at:
  - `serve.go:1425` — `wfState.Status == state.StatusHumanCommunication || wfState.Status == state.StatusWaitingForHuman`
  - `main.go:2200-2202` — same check
- **Recommendation**: Create `state.IsHumanPending(status string) bool`, replace all 3 occurrences
- **Impact**: Removes duplication, makes concept first-class in the state package

### Opportunity 2: Replace string literals with constants in `serve.go`
- `serve.go` uses `"waiting-for-human"` string literal **7 times** instead of `state.StatusWaitingForHuman`
- `main.go:878` also uses the string literal instead of the constant
- **Recommendation**: Replace all with `state.StatusWaitingForHuman`
- **Impact**: Prevents typo bugs, enables IDE navigation, makes refactoring safe

### Opportunity 3: Extract `needsInput` computation to `state.Workflow` method
- The expression `wfState.Status == "waiting-for-human" && (wfState.MultiChoiceQuestion != nil || wfState.HumanMessage != "")` is duplicated **5 times** in `serve.go` (lines 403, 688, 1831, 2342, 2662)
- **Recommendation**: Create `func (w Workflow) NeedsHumanInput() bool` method on the Workflow struct
- **Impact**: DRYs up 5 identical expressions, centralizes definition, makes it testable

### Opportunity 4: Consider unifying `StatusHumanCommunication` and `StatusWaitingForHuman`
- These represent the same concept ("waiting for human") at different lifecycle stages:
  - `human-communication`: set by MCP tools (askUserQuestion/askUserWorkGate)
  - `waiting-for-human`: set by main loop after processing the above
- The transition is trivial: `main.go:877-878` just renames the status
- **This is the riskiest cleanup** and may not be worthwhile — the two-phase approach might be intentional for UI rendering/polling reasons
- **Recommendation**: Discuss with human partner before pursuing

### Brainstorming Note
This is a review/cleanup task rather than a new feature design. Adapting brainstorming to be a cleanup scope discussion rather than full product design cycle.

## Human Partner Clarifications (2026-02-02)

### Fix Concerns
Q: Do you agree the fix in pyxkuutx is correct?
A: "No, I have concerns about the fix" — Human wants less code. The fix works but should be simplified through the unification of the two human-pending statuses. Less code is better.

### Cleanup Scope
Q: Which cleanup opportunities to implement?
A: "All of 1-4 (including the risky unification)" — Do all four.

### Desired Behavior Specification (AUTHORITATIVE)
The human defines three interactive modes:

**INTERACTIVE: NO**
- Print the message to stdout
- Exit with error code 2

**INTERACTIVE: AUTO**
- Tools `ask_user_question` and `ask_user_work_gate` are hidden (not registered) so agents can't ask questions
- Self-driving: no human interaction needed
- autoResponseMessage is NOT needed back (confirmed 2026-02-02)

**INTERACTIVE: YES** (Two phases)
- **Phase 1 — Interview**: `ask_user_question` lets the coordinator agent interview the human partner. Only the coordinator agent can ask questions.
- **Phase 2 — Build**: After work-gate approval, the system transitions to self-driving (auto) behavior. The `ask-work-gate` tool unifies this transition between coordinator and brainstorming skills.

### AUTO Mode Clarification
Q: Should autoResponseMessage be restored for AUTO mode?
A: No. Current behavior is correct — tools are hidden, agent can't ask questions, just keeps working. autoResponseMessage is NOT needed back.

### Additional Directives
- **Remove `auto-session`**: This mode was previously added but should be removed. When work-gate is approved in `interactive:yes` mode, just switch to `auto` behavior directly.
- **Keep `ask-work-gate`**: Maintains the transition pattern between coordinator/brainstorming and the build phase.
- **Principle**: Less code is better, as long as the user gets the desired behavior.

## Brainstorming: COMPLETE
All requirements clarified. Consensus reached on scope: 4 cleanup items + auto-session removal.

## Agent Decisions Log (2026-02-02)

### Decision 1: Scope includes all 4 cleanup items + auto-session removal
- Cleanup #1: Move `isHumanPendingStatus` → `state.IsHumanPending()` (simplifies to single status check after #4)
- Cleanup #2: Replace string literals with constants
- Cleanup #3: Extract `NeedsHumanInput()` method on Workflow struct
- Cleanup #4: Unify `StatusHumanCommunication` + `StatusWaitingForHuman` → single `StatusWaitingForHuman`
- Extra: Remove `auto-session` mode entirely

### Decision 2: The fix in pyxkuutx should be simplified, not just confirmed
- The guard logic is correct but should be simpler after unification
- After unifying statuses, `isHumanPendingStatus` becomes a single comparison
- The concept should live in `pkg/state` as `IsHumanPending()`

### Decision 3: Implementation sequencing
1. First: Unify statuses (foundational change, #4)
2. Second: Replace string literals with constants (#2)
3. Third: Move helper to pkg/state and simplify (#1, now trivial after #4)
4. Fourth: Extract NeedsHumanInput() method (#3)
5. Fifth: Remove auto-session mode
6. Throughout: Update tests, run `make test`, `make lint`

## Work Gate: APPROVED (2026-02-02)
Human partner approved: "DEFINITION IS COMPLETE, BUILD MAY BEGIN"
Delegating to backend-go-developer for implementation.

## Implementation Status (2026-02-02)

### All GOAL.md items marked complete ✅
- Status unification: `StatusHumanCommunication` removed, only `StatusWaitingForHuman` remains
- String literals: all replaced with `state.StatusWaitingForHuman` constant
- `IsHumanPending()`: extracted to `pkg/state/state.go`
- `NeedsHumanInput()`: method on `Workflow` struct in `pkg/state/state.go`
- `auto-session`: removed, work-gate now switches directly to `auto`
- `ValidStatuses` doc comment: updated to explain exclusion of `StatusWaitingForHuman`
- Tests: all pass (`make test`)
- Lint: clean (`make lint`, 0 issues)

### Remaining code quality issue from go-readability-reviewer
- Duplicate `TestIsHumanPending` in `cmd/sgai/updateworkflowstate_test.go:147-167` (already tested in `pkg/state/state_test.go:129-148`)
- Sent to backend-go-developer for removal
- ✅ RESOLVED: backend-go-developer removed duplicate test, all tests pass, lint clean

## Project Completion (2026-02-02)

All GOAL.md items verified complete (21/21 checked, 0 unchecked):
- Status unification: `StatusHumanCommunication` removed, single `StatusWaitingForHuman` remains
- String literals: all replaced with `state.StatusWaitingForHuman` constant
- `IsHumanPending()`: extracted to `pkg/state/state.go` with godoc
- `NeedsHumanInput()`: method on `Workflow` struct in `pkg/state/state.go` with godoc
- `auto-session`: removed, work-gate switches directly to `auto`
- Human communication interface simplified (removed `humanMessage`, `human-communication` status, free-text templates)
- `ValidStatuses` doc comment updated to explain `StatusWaitingForHuman` exclusion
- Go-readability-reviewer feedback fully addressed (duplicate test removed)
- All tests pass (`make test`), lint clean (`make lint`)


---


- [x] Confirm fix done in `jj diff --git -r pyxkuutx`
  - [x] what could you do to make the code overall cleaner, simpler, smaller?
- [x] Unify `StatusHumanCommunication` + `StatusWaitingForHuman` → single status
- [x] Replace `"waiting-for-human"` string literals with `state.StatusWaitingForHuman` constant (8 occurrences)
- [x] Move `isHumanPendingStatus` → `state.IsHumanPending()` in pkg/state (simplifies after unification)
- [x] Extract `NeedsHumanInput()` method on `state.Workflow` struct (DRYs 5 expressions)
- [x] Remove `auto-session` mode — when work-gate approved in `yes` mode, switch directly to `auto`
- [x] All tests pass (`make test`) and lint clean (`make lint`)

---

It seems that after the simplification below, the "start" mode is no longer interviewing me.

- [x] Is it the case?
  - [x] Why?
- [x] Propose fixes
  - [x] implement the fixes

IMPORTANT CONCEPT: the "self-drive" mode (interactive:auto) is VERY distinct from "start" mode (interactive:yes)

The problem may have been introduced in this range: `jj log -r orxtoxsk::okrpvrwy/0`

---
- [x] simplify the human communication interface
  - [x] remove `humanMessage` field and `human-communication` status from `update_workflow_state` MCP tool
  - [x] `ask_user_question` is the sole human communication channel (already hidden in auto/auto-session mode)
  - [x] `ask_user_work_gate` kept as separate tool (already hidden in auto/auto-session mode)
  - [x] remove `autoResponseMessage` constant and all auto-select/auto-respond logic in main.go
  - [x] delete `response_dialog.html` and `response_modal.html` (free-text templates)
  - [x] simplify `pageRespond` handler (remove free-text vs multi-choice branching)
  - [x] update coordinator prompt in dag.go and set-workflow-state skill docs
  - [x] all tests pass (`make test`)
