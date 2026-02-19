---
name: retrospective
description: Post-completion factory improvement analysis. Guides the retrospective agent through artifact discovery, pattern analysis, suggestion generation, and presenting proposed changes for approval. Use when the retrospective agent starts its post-completion phase.
---

# Retrospective Analysis

## Overview

This skill guides the retrospective agent through analyzing a completed session and producing actionable improvements to the factory. The goal is to make the factory better over time by examining what happened, identifying patterns, and proposing concrete changes to skills, agent prompts, and AGENTS.md.

**Core principle:** Evidence-based improvement. Every suggestion must be grounded in artifacts from the session, not speculation.

## When to Use

- Use when the retrospective agent starts (post-completion phase)
- Use when analyzing session artifacts to find improvement opportunities
- Don't use during normal workflow execution
- Don't use for modifying application source code

## IMPORTANT: Understanding `state.json` Paths

There are TWO different `state.json` files in the system:

1. **Session copy**: `.sgai/retrospectives/<session-id>/state.json` — A snapshot of the workflow state captured at session end. This file MAY NOT always exist (it depends on whether the session completed normally and the copy was made).
2. **Main workflow state**: `.sgai/state.json` — The live workflow state file. This file is ALWAYS present after the factory starts.

**Fallback logic (use this whenever a step says to "read state.json"):**
- First, try to read `.sgai/retrospectives/<session-id>/state.json` (the session copy)
- If it does not exist or is unreadable, fall back to `.sgai/state.json` (always present)
- Document which file you actually read in your analysis log

## Process

### Step 1: Artifact Discovery

Read artifacts in THIS ORDER (priority matters — richest signal sources first):

#### 1a. Read session `state.json` FIRST (highest priority)

- [ ] Read `.sgai/PROJECT_MANAGEMENT.md` frontmatter to find the retrospective session directory path (key: `Retrospective Session: .sgai/retrospectives/<session-id>`)
- [ ] List all files in the session directory
- [ ] **Read session `state.json` FIRST** — This is the single richest signal source. Use fallback logic: try `.sgai/retrospectives/<session-id>/state.json` first; if missing, fall back to `.sgai/state.json`. It contains:
  - Visit counts per agent (how many times each agent ran)
  - Inter-agent message log (every message sent between agents)
  - Agent sequence (order of agent execution)
  - Progress notes from each agent
  - **If BOTH `.sgai/retrospectives/<session-id>/state.json` AND `.sgai/state.json` are missing or unreadable, STOP and report this in your analysis log — do NOT proceed to Step 2 without acknowledging this gap**

#### 1b. Read Goal and Project Management artifacts

- [ ] Read `GOAL.md` copy (what was supposed to happen)
- [ ] Read `PROJECT_MANAGEMENT.md` copy (what actually happened — decisions, issues, backtracks)

#### 1c. Read ALL session JSON files (mandatory — no exceptions)

- [ ] Read **ALL** session JSON files (numbered `NNNN-<agent>-<timestamp>.json`) — these contain the full conversation transcripts
- [ ] You MUST read every single session JSON file in the directory, not just a subset
- [ ] Count the total number of session JSON files and track how many you have read
- [ ] **If there are more than 10 session JSONs, you may skim the later ones, but you MUST still open and read at least the first 500 lines of each one**

**Reading session JSONs:** Process files in numerical order (0001, 0002, ...). Each contains the full transcript for one agent iteration. Focus on:
- What the agent was asked to do
- What problems it encountered
- How long it took (number of tool calls)
- Whether it needed reviewer feedback or corrections
- Any patterns of rework, confusion, or inefficiency

#### 1d. Read logs

- [ ] Read `stdout.log` and `stderr.log` for build/test output signals

**Session directory structure:**
```
.sgai/retrospectives/<session-id>/
├── GOAL.md
├── PROJECT_MANAGEMENT.md
├── state.json                        # Session copy (MAY NOT EXIST — use .sgai/state.json as fallback)
├── stdout.log
├── stderr.log
├── screenshots/
└── NNNN-<agent>-<timestamp>.json   # Session transcripts
```

### Step 1.5: Mandatory Analysis Log

**GATE: You MUST complete this step before proceeding to Step 2. You may NOT skip this step.**

After reading all artifacts in Step 1, write a structured analysis summary to your progress notes using `sgai_update_workflow_state`. This summary MUST contain:

1. **Files read count**: "Read X session JSONs out of Y total" (X must equal Y, or you must explain why not)
2. **Visit count summary**: From session `state.json` (or `.sgai/state.json` fallback) — which agents ran, how many visits each
3. **Message count summary**: From session `state.json` (or `.sgai/state.json` fallback) — total inter-agent messages, notable message patterns
4. **Per-category observations** (at least 1 observation per category):
   - **Efficiency**: At least 1 observation about visit counts, handoff patterns, or iteration depth
   - **Quality**: At least 1 observation about reviewer feedback, test failures, or backtracks
   - **Knowledge gaps**: At least 1 observation about missing information, repeated mistakes, or tool misuse
   - **Process gaps**: At least 1 observation about missing skills, skill violations, or convention drift

**If you cannot produce at least 1 observation per category:**
- You MUST re-read the session artifacts more carefully
- Focus on the session JSONs — patterns are buried in the transcripts, not in summary artifacts
- A "clean" session (all goals complete, tests passing) does NOT mean there are no observations. Every session has patterns worth noting.
- If after a second careful read you still have a category with no observations, you may write "No observations found after thorough review" for that specific category — but this should be rare

**Example analysis log:**
```
Analysis Summary:
- Files: Read 15/15 session JSONs, session state.json (from .sgai/retrospectives/<session-id>/state.json), GOAL.md, PROJECT_MANAGEMENT.md, stdout.log, stderr.log
- Visits: coordinator(8), backend-go-developer(3), go-readability-reviewer(2), react-developer(1), project-critic-council(1)
- Messages: 19 inter-agent messages, 3 reviewer feedback rounds
- Efficiency: Backend developer visited 3 times due to reviewer feedback — could skills reduce this?
- Quality: Reviewer caught SQL formatting issues 3 times — suggests missing skill
- Knowledge gaps: Agent asked about migration workflow mid-session — no skill exists for this
- Process gaps: stpa-analyst.md was a 17-line stub that needed emergency expansion in-session
```

### Step 2: Pattern Analysis

Analyze the artifacts for these signal types:

#### Efficiency Signals
- [ ] **Visit counts** — From session `state.json` (or `.sgai/state.json` fallback), how many times was each agent visited? High counts suggest rework or unclear instructions.
- [ ] **Handoff patterns** — Were there unnecessary back-and-forth between agents? Could some handoffs have been avoided?
- [ ] **Iteration depth** — How many iterations did the workflow take? What drove the iteration count?

#### Quality Signals
- [ ] **Reviewer feedback** — What did reviewers (go-readability-reviewer, react-reviewer) find? Are there patterns in what they catch?
- [ ] **Test failures** — Were there build or test failures? What caused them?
- [ ] **Backtracks** — Did any agent need to undo or redo work? Why?

#### Knowledge Gaps
- [ ] **Missing information** — Did agents ask questions that should have been pre-documented in skills or AGENTS.md?
- [ ] **Repeated mistakes** — Did multiple agents make the same kind of error?
- [ ] **Tool misuse** — Did agents use tools incorrectly or inefficiently?

#### Process Gaps
- [ ] **Missing skills** — Were there situations where a skill would have helped but none existed?
- [ ] **Skill violations** — Did agents ignore or misapply existing skills?
- [ ] **Convention drift** — Were there style or convention inconsistencies that AGENTS.md should address?

### Step 3: Generate Suggestions

For each pattern identified in Step 2, produce a concrete suggestion. Each suggestion must have:

1. **Category** — One of: `new-skill`, `modify-skill`, `new-agent-prompt`, `modify-agent-prompt`, `update-agents-md`
2. **Evidence** — The specific artifact and pattern that motivated it
3. **Proposal** — What to create or change (be specific)
4. **Rationale** — Why this improvement will help future sessions
5. **Diff Preview** — For suggestions that modify existing files, include the unified diff showing what will change (read the file first, then compute the diff). For new files, show the proposed content.

#### Path Validation Rule

Before presenting any suggestion, verify the target path:
- Target path MUST NOT be under `.sgai/` (except `.sgai/SGAI_NOTES.md`)
- If a suggestion targets `.sgai/`, translate it to the `sgai/` overlay equivalent:
  - `.sgai/agent/foo.md` -> `sgai/agent/foo.md`
  - `.sgai/skills/bar/SKILL.md` -> `sgai/skills/bar/SKILL.md`
- The `.sgai/` directory is the runtime directory rebuilt from skeleton + overlay on every startup — changes there are lost immediately

#### Suggestion Categories

**New Skills** (`sgai/skills/<name>/SKILL.md`)
- Agents repeatedly needed guidance that no skill provides
- A process pattern emerged that should be standardized
- Example: "Create a skill for database migration testing — agents spent 3 iterations figuring out the migration workflow"

**Modify Existing Skills** (`sgai/skills/<name>/SKILL.md`)
- An existing skill was unclear or incomplete
- Agents violated a skill due to ambiguity
- Example: "Add a section to go-code-review about SQL formatting — reviewer flagged this 3 times"

**New/Modified Agent Prompts** (`sgai/agent/<name>.md`)
- An agent's behavior needs adjustment
- An agent's permissions were too broad or too narrow
- Example: "Add explicit instruction to backend-go-developer about running make lint before marking done"

**Update AGENTS.md**
- A style rule emerged from reviewer feedback
- A business rule was discovered during brainstorming
- A convention was established mid-session that should persist
- Example: "Add rule: Go error variable names must use err prefix pattern (errClose, errRead)"

### Step 4: Prioritize and Group

- [ ] Sort suggestions by impact (high/medium/low)
- [ ] Group suggestions into exactly 3 category buckets:
  - **Skills** — Categories `new-skill` and `modify-skill`
  - **Agent Prompts** — Categories `new-agent-prompt` and `modify-agent-prompt`
  - **AGENTS.md** — Category `update-agents-md`
- [ ] Discard suggestions that are too vague or not actionable
- [ ] Limit to the top 10 most impactful suggestions (quality over quantity)

**Impact assessment:**
| Impact Level | Criteria |
|-------------|----------|
| **High** | Would have saved 3+ iterations or prevented a major backtrack |
| **Medium** | Would have improved clarity or reduced one review round |
| **Low** | Nice-to-have, minor improvement |

### No Suggestions Case

**PREREQUISITES — You may NOT invoke this case unless ALL of the following are true:**

1. You have read the session `state.json` (via `.sgai/retrospectives/<session-id>/state.json`, or the `.sgai/state.json` fallback) and recorded visit counts and message counts in your analysis log
2. You have read at least 3 session JSON files (or all of them if fewer than 3 exist)
3. You have completed the Step 1.5 Mandatory Analysis Log with observations in all 4 signal categories (efficiency, quality, knowledge gaps, process gaps)
4. You have completed Steps 2-4 (Pattern Analysis, Generate Suggestions, Prioritize and Group)

**If ALL prerequisites are met** and you genuinely have zero actionable suggestions after thorough analysis, send `RETRO_COMPLETE` and exit:

```
sgai_send_message({
  toAgent: "coordinator",
  body: "RETRO_COMPLETE: No actionable improvements identified for this session. Analysis summary: Read X/Y session JSONs, session state.json (from [path used]) showed Z agent visits and W messages. Per-category findings: [brief summary of each category observation from Step 1.5]."
})
sgai_update_workflow_state({ status: "agent-done", task: "", addProgress: "No actionable suggestions found after thorough analysis. Sent RETRO_COMPLETE." })
// STOP HERE. Make NO more tool calls. Your turn is OVER.
```

**CRITICAL:** The RETRO_COMPLETE message MUST include your analysis summary (files read, visit counts, per-category observations) as proof that you actually completed the analysis. A bare "No actionable improvements identified" without evidence is NOT acceptable.

### Step 5: Present Changes for Approval

**MANDATORY YIELD PROTOCOL:** After every `sgai_send_message()` call in this step, you MUST:
1. Immediately call `sgai_update_workflow_state({status: "agent-done"})`
2. STOP making tool calls — your turn is over
3. Do NOT call `check_inbox` or `check_outbox` — the coordinator cannot respond until you yield

**MANDATORY:** You MUST send at least one `RETRO_QUESTION:` message to the coordinator during your run. This is NOT optional. If you found zero suggestions, follow the "No Suggestions Case" above instead.

Present proposed changes to the human partner by sending `RETRO_QUESTION:` messages to the coordinator, grouped by category. For each non-empty category bucket (Skills, Agent Prompts, AGENTS.md), send ONE message containing ALL proposals in that category.

#### Presentation Format

For each non-empty category, send a single `RETRO_QUESTION` with this structure:

```
sgai_send_message({
  toAgent: "coordinator",
  body: "RETRO_QUESTION [MULTI-SELECT]: **Skills Changes** (N proposals)\n\n### 1. [Title of first proposal]\nEvidence: [1-line evidence from session artifacts]\n```diff\n--- a/[file path]\n+++ b/[file path]\n@@ ... @@\n[unified diff content]\n```\nRationale: [why this helps future sessions]\n\n### 2. [Title of second proposal]\nEvidence: [1-line evidence]\n[full proposed file content for new files, or diff for modifications]\nRationale: [why this helps]\n\nSelect which to approve (multi-select):\n- 1. [Title of first proposal]\n- 2. [Title of second proposal]"
})
// Then yield immediately
sgai_update_workflow_state({ status: "agent-done", task: "Waiting for human response via coordinator", addProgress: "Sent Skills category RETRO_QUESTION to coordinator" })
// STOP HERE. Make NO more tool calls. Do NOT check inbox or outbox. Your turn is OVER.
```

#### Full Example

```
sgai_send_message({
  toAgent: "coordinator",
  body: "RETRO_QUESTION [MULTI-SELECT]: **Skills Changes** (2 proposals)\n\n### 1. Add SQL formatting section to go-code-review\nEvidence: Reviewer flagged SQL formatting 3 times in session\n```diff\n--- a/sgai/skills/go-code-review/SKILL.md\n+++ b/sgai/skills/go-code-review/SKILL.md\n@@ -45,6 +45,12 @@\n+## SQL Formatting\n+- Align VALUES with INSERT columns\n+- Each column on its own line\n```\nRationale: Prevents repeated reviewer catches\n\n### 2. Create db-migration-testing skill\n[full proposed file content]\nRationale: Standardizes migration testing workflow\n\nSelect which to approve (multi-select):\n- 1. Add SQL formatting section to go-code-review\n- 2. Create db-migration-testing skill"
})
sgai_update_workflow_state({ status: "agent-done", task: "Waiting for human response via coordinator", addProgress: "Sent Skills RETRO_QUESTION to coordinator" })
// STOP HERE. Your turn is OVER.
```

#### Processing Responses

When the coordinator relays the human's response (which numbered items were selected):
- Track which individual changes were approved vs rejected
- Proceed to the next non-empty category with another `RETRO_QUESTION [MULTI-SELECT]:`
- After all categories have been presented, proceed to Step 6

#### Key Presentation Rules

1. **One message per category** — Never send individual proposals one-at-a-time
2. **Include `[MULTI-SELECT]` marker** — So the coordinator knows to use `multiSelect: true` when relaying
3. **Show diffs for ALL file modifications** — When suggesting changes to existing files, you MUST read the file first and show the unified diff. For new files, show the full proposed content.
4. **Respect rejections** — If user deselects a proposal, do NOT re-present it
5. **Respect "skip all"** — If user selects nothing in a category, that entire category is skipped
6. **Yield after every send** — The IRON LAW applies here without exception

### Step 6: Apply Approved Changes

Apply only the individually-approved changes. Skip any rejected or unselected changes entirely.

#### Overlay Directory Workflow

The `sgai/` directory is an **overlay** — files placed there wholly replace their skeleton defaults. This has critical implications for how you apply changes:

**For MODIFYING an existing skill, agent prompt, or snippet:**
1. READ the current version from `.sgai/` (the live runtime directory — skeleton + overlay merged)
2. Copy the ENTIRE file content into memory
3. Apply your modifications to the copy
4. Write the COMPLETE modified file to `sgai/` (the overlay directory)

**For CREATING a new skill, agent prompt, or snippet:**
1. Write the entire new file directly to `sgai/`

**CRITICAL:** Partial edits are NOT possible via the overlay. Every file in `sgai/` must be a complete, self-contained version of the file it overrides. If you write only your changes without the rest of the file content, the original content will be lost.

#### Checklist

For each approved change:

- [ ] For modifications: READ the current file from `.sgai/` first, then write the COMPLETE modified version to `sgai/`
- [ ] For new files: Write the entire file directly to `sgai/`
- [ ] Write the change to the correct location (`sgai/` overlay or `AGENTS.md`)
- [ ] For new skills: Create proper `SKILL.md` with YAML frontmatter (name, description)
- [ ] For AGENTS.md changes: Append to the appropriate section, don't overwrite existing content
- [ ] For agent prompt modifications: Use the `sgai/agent/` overlay directory
- [ ] Verify each written file is syntactically valid

#### Writing Skills

New skills must follow the Agent Skills spec:
```markdown
---
name: <skill-name>
description: <when to use this skill>
---

# <Skill Title>

## Overview
[What this skill is for]

## When to Use
[Trigger conditions]

## Process
[Step-by-step guide]
```

#### Writing to AGENTS.md

When adding to AGENTS.md:
1. Read the current AGENTS.md first
2. Find the appropriate section (or create one)
3. Add new content without removing existing content
4. Use the same formatting style as existing entries

### Step 7: Completion

- [ ] Verify all approved changes were written successfully
- [ ] Summarize what was changed in the workflow state progress log
- [ ] Set status to `agent-done`

## Rules

1. **Evidence-based only** — Every suggestion must reference specific artifacts. No speculation.
2. **User approval required** — Never write changes without explicit human approval via `RETRO_QUESTION:` messages to the coordinator.
3. **sgai/ and AGENTS.md only** — Only modify files in the `sgai/` overlay directory and `AGENTS.md`. Nothing else.
4. **Read everything** — Read ALL session artifacts before producing suggestions. Incomplete analysis produces bad suggestions.
5. **Quality over quantity** — Limit to 10 suggestions max. Better to have 3 great suggestions than 10 mediocre ones.
6. **Graceful exit** — If the user rejects everything during the approval process, mark done without making changes. That is a valid outcome. But you MUST have sent at least one `RETRO_QUESTION:` message to the coordinator before exiting (or a `RETRO_COMPLETE` in the no-suggestions case).
7. **No source code** — You do not modify Go, TypeScript, test files, or any application code. Period.
8. **No `.sgai/` suggestions** — Never suggest changes targeting `.sgai/` paths (except `.sgai/SGAI_NOTES.md`). Always translate to `sgai/` overlay equivalents. The `.sgai/` directory is rebuilt from skeleton + overlay on every startup — changes there are lost.
9. **Mandatory analysis log** — You MUST complete Step 1.5 before proceeding to Step 2. Skipping the analysis log is a skill violation.

### Common Rationalizations to REJECT
- "I'll suggest modifying `.sgai/agent/foo.md` directly" — NO. Always target `sgai/agent/foo.md` (overlay).
- "I'll suggest changes to `.sgai/skills/bar/SKILL.md`" — NO. Target `sgai/skills/bar/SKILL.md` instead.
- "The `.sgai/` path is where the file currently lives" — Irrelevant. You READ from `.sgai/`, but SUGGEST and WRITE to `sgai/`.
- "Everything looks clean, no need to dig deeper" — NO. Clean-looking sessions often have the most interesting buried patterns. Every session has patterns worth noting, even successful ones.
- "The session was successful so there's nothing to improve" — NO. Success does not mean there are no improvement opportunities. Dig into the transcripts.
- "I've read enough to conclude there are no suggestions" — NO, unless you have met ALL prerequisites for the No Suggestions Case (session `state.json` read via fallback logic, 3+ session JSONs read, analysis log complete with all 4 categories).

## Checklist

Before marking done, verify:

- [ ] Read session `state.json` FIRST (tried `.sgai/retrospectives/<session-id>/state.json`, fell back to `.sgai/state.json` if needed) and recorded visit counts + message counts
- [ ] Read ALL session JSON files (count: X out of Y total)
- [ ] Completed Step 1.5 Mandatory Analysis Log with observations in all 4 categories
- [ ] Read all session artifacts (GOAL.md, PM, session `state.json` via fallback logic, session JSONs, stdout.log, stderr.log)
- [ ] Identified patterns from at least 2 signal categories (efficiency, quality, knowledge, process)
- [ ] Produced concrete suggestions with evidence, diffs, and rationale
- [ ] Grouped suggestions into category buckets (Skills, Agent Prompts, AGENTS.md)
- [ ] Sent at least one `RETRO_QUESTION [MULTI-SELECT]:` message per non-empty category to the coordinator (or `RETRO_COMPLETE` if zero suggestions)
- [ ] Applied only individually-approved changes; skipped all rejected changes
- [ ] Applied changes to correct locations (sgai/ overlay or AGENTS.md)
- [ ] Set workflow state to agent-done
- [ ] After EVERY sgai_send_message() call, immediately called sgai_update_workflow_state({status: "agent-done"}) and stopped
- [ ] Never called check_inbox or check_outbox between sending a message and yielding
