---
name: stpa-overview
description: Entry point for STPA (System Theoretic Process Analysis) hazard and safety analysis. Use for full 4-step STPA sessions, focused project-critic safety reviews, coordinator safety gates, reviewer hazard checks, risk assessment, unsafe state transitions, external input, filesystem, concurrency, physical systems, AI-driven systems, or when tempted to route work to retired stpa-analyst.
metadata:
  tags: "stpa, safety, hazard-analysis, risk-assessment, control-theory, project-critic, reviewer, retired-stpa-analyst"
---

# STPA Overview

## Core Principle

STPA is a **skill workflow**, not a routable workflow agent.

When safety, hazards, risk, unsafe state transitions, external input, filesystem effects, concurrency, physical systems, or AI-driven control loops are relevant, load and use this skill. Do **not** route to `stpa-analyst`, add `stpa-analyst` to GOAL `flow`, add a `stpa-analyst` model entry, or send `QUALITY_REPORT_REQUEST` to a retired agent.

## What is STPA?

STPA (System Theoretic Process Analysis) is a hazard analysis method that:

- Treats safety as a **control problem**, not just a failure problem
- Uses **control-feedback loops** to model complex systems
- Identifies **unsafe control actions** that could lead to hazards
- Discovers unintended interactions across software, humans, physical processes, and AI systems

## When to Use

Use this skill in either mode:

1. **Full 4-step STPA mode** — for a new safety/hazard analysis session or when the human partner asks for safety analysis, hazard analysis, risk assessment, STPA, or control-loop analysis.
2. **Focused safety-review mode** — for coordinator gates, project-critic gates, quality reports, or `*-reviewer` reviews where a quick hazard check is warranted but a full STPA session would be too heavy.

Do not use this skill for ordinary style-only review with no safety, risk, control-flow, validation, concurrency, or state-transition implications.

## Mandatory Routing Rules

1. **No retired-agent routing** — Never route safety work to `stpa-analyst`; STPA lives here.
2. **Coordinator gate** — The coordinator must load/use this skill when safety/hazard analysis is relevant before delegating, validating, or declaring a safety-sensitive workflow complete.
3. **Project-critic gate** — Project critics must request or perform focused safety-review evidence through this skill, not by targeting `stpa-analyst`.
4. **Reviewer permission** — Any `*-reviewer` may load/use this skill when circumstances warrant hazard or safety analysis. If safety risk is material, include focused STPA findings in the review report.
5. **Coordinator-owned checkboxes** — If a GOAL checkbox is completed by STPA work, report evidence to the coordinator; do not edit GOAL checkboxes unless you are explicitly the coordinator.

## Choose the Mode

| Situation | Mode | Output |
|-----------|------|--------|
| Human asks for STPA, hazard analysis, risk assessment, or safety analysis | Full 4-step STPA | Losses, hazards, constraints, control structure, UCAs, loss scenarios, mitigations |
| Coordinator sees external input, filesystem changes, concurrent operations, unsafe state transitions, or safety-critical behavior | Focused safety-review first; escalate to full STPA if needed | Gate note or task plan with hazards/constraints |
| Project critic needs a quality report | Focused safety-review | PASS/NEEDS WORK safety report |
| `*-reviewer` sees safety or hazard implications | Focused safety-review inside normal review | Review comments with safety findings |

## Full 4-Step STPA Mode

Announce: "I'm using the STPA Overview skill to guide a systematic hazard analysis. We'll work through four STPA steps and record decisions as we go."

### Step 1: Define Purpose of Analysis

Load: `skills({"name":"stpa-step1-define-purpose"})`

- Identify **Losses** (unacceptable outcomes)
- Define **System-Level Hazards** (states leading to losses)
- Establish **System-Level Constraints** (behaviors that prevent hazards)

### Step 2: Model the Control Structure

Load: `skills({"name":"stpa-step2-control-structure"})`

- Create hierarchical control-feedback diagrams
- Identify controllers, controlled processes, control actions, and feedback paths
- Use Graphviz/DOT format with `rankdir=TB` and `node [shape=box]`

### Step 3: Identify Unsafe Control Actions (UCAs)

Load: `skills({"name":"stpa-step3-unsafe-control-actions"})`

- Analyze each control action for four UCA types:
  1. Not provided when needed
  2. Provided when not needed
  3. Wrong timing or order
  4. Stopped too soon or applied too long

### Step 4: Identify Loss Scenarios

Load: `skills({"name":"stpa-step4-loss-scenarios"})`

- Trace causal pathways for each UCA
- Identify why UCAs might occur
- Develop recommendations and mitigations

### Full-Mode Process

- [ ] Load each step-specific skill before doing that step
- [ ] Ask one question at a time using the environment's human-question mechanism
- [ ] If your role cannot ask the human directly, send the coordinator `QUESTION: ...`
- [ ] Record answers in `.sgai/PROJECT_MANAGEMENT.md` under `## STPA Analysis`
- [ ] Iterate when later steps reveal missing losses, hazards, controls, or feedback
- [ ] Summarize findings and notify the coordinator when STPA work completes

## Focused Safety-Review Mode

Announce: "I'm using STPA Overview in focused safety-review mode to check hazards without running the full four-step STPA process."

Use this mode when a coordinator, project critic, or reviewer needs a quick safety/hazard assessment.

### Scope

Focus on:

- **Control-flow safety** — unclear control paths, unguarded transitions, missing feedback
- **Unsafe state transitions** — inconsistent states, stale state, invalid workflow transitions
- **Error handling adequacy** — silent failures, swallowed errors, misleading success states
- **Input and boundary validation** — external input, filesystem paths, permissions, untrusted data
- **Concurrency and timing** — races, double actions, cancellation, retries, ordering problems
- **Human/AI control loops** — missing human confirmation, hallucinated authority, unsafe autonomy

### Process

- [ ] Read `GOAL.md` and `.sgai/PROJECT_MANAGEMENT.md` to understand expected behavior and risk scope
- [ ] Inspect relevant files, diffs, reports, or reviewer notes for the scope above
- [ ] Identify hazards as `H-n`, constraints as `SC-n`, and concrete issues with file/line evidence when possible
- [ ] Escalate to full 4-step STPA mode if the focused review finds systemic hazards or unclear losses
- [ ] Send or include the structured report below

### Focused Report Format

```markdown
QUALITY_REPORT from stpa-overview focused safety review

**Scope Reviewed:** [brief scope]

**Hazards / Issues Found:**
- H-1: [hazard or issue, with file:line if applicable]

**Recommended Safety Constraints:**
- SC-1: [constraint or mitigation]

**Verdict:** PASS | NEEDS WORK

**Unresolved Concerns:**
- [unknowns, assumptions, or follow-up needed]
```

## Documentation Structure

Record full STPA sessions like this:

```markdown
## STPA Analysis

### Step 1: Purpose Definition
#### Losses (L)
- L-1: [description]

#### Hazards (H)
- H-1: [system] [unsafe condition] [→ L-1]

#### System-Level Constraints (SC)
- SC-1: [condition to enforce] [→ H-1]

### Step 2: Control Structure
[Graphviz/DOT diagram]

### Step 3: Unsafe Control Actions
[UCA table]

### Step 4: Loss Scenarios
[Scenario descriptions and recommendations]
```

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "Routing to the dedicated STPA agent is faster." | `stpa-analyst` is retired. Loading this skill is the supported path. |
| "The quality-report behavior lived in the old wrapper." | Focused safety-review mode now owns that behavior. Use this skill's report format. |
| "STPA is outside reviewer scope." | `*-reviewer` agents may load/use STPA when safety or hazard risk warrants it. |
| "Full STPA is too heavy for a project-critic gate." | Use focused safety-review mode; escalate only if needed. |
| "Safety Analysis should add an agent to GOAL flow." | Safety analysis is coordinator/reviewer guidance plus this skill, not a flow node. |

## Red Flags - STOP

- Adding `stpa-analyst` to GOAL `flow` or `models`
- Sending `QUALITY_REPORT_REQUEST` to `stpa-analyst`
- Treating safety review as impossible because full STPA is too large
- Saying reviewers cannot use STPA when they see hazard/safety implications
- Completing a safety-relevant coordinator or project-critic gate without STPA evidence
- Using slash-form step skill names instead of actual `stpa-step...` skill names

## Examples

### Good: Coordinator Gate

The coordinator sees a task involving external input, filesystem writes, and concurrent workers. It loads `stpa-overview`, runs focused safety-review mode, records hazards and constraints, then delegates implementation with those constraints.

### Good: Reviewer

A `go-readability-reviewer` sees a workflow state transition that can mark work complete after a failed validation. The reviewer loads `stpa-overview`, uses focused mode, and reports `H-1` plus `SC-1` in the review.

### Bad: Retired Agent Routing

The project critic sends `QUALITY_REPORT_REQUEST` to `stpa-analyst` because an old wrapper described that mode. This is wrong: use focused safety-review mode in this skill and report back directly.

## Completion Checklist

Before finishing STPA work, verify:

- [ ] You chose full mode or focused mode intentionally
- [ ] You did not route to `stpa-analyst`
- [ ] You loaded actual step skills by name when running full mode
- [ ] You recorded findings or returned a structured focused report
- [ ] You notified the coordinator of completed GOAL work instead of editing checkboxes yourself

## Related Skills

- `stpa-step1-define-purpose` — Detailed Step 1 guidance
- `stpa-step2-control-structure` — Control structure modeling
- `stpa-step3-unsafe-control-actions` — UCA identification tables
- `stpa-step4-loss-scenarios` — Causal scenario analysis
