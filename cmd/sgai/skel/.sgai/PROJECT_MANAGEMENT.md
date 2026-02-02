# Project management

This file is the durable, shared project log for this workspace.

Use it to capture decisions, open questions, and the current plan so work can continue across sessions and across agents.

## What this file is (and isn’t)

- This file is the **shared memory** for the workspace.
- This file is **not** a scratchpad. Keep raw notes elsewhere and copy only high-signal outcomes here.
- This file is **not** a replacement for direct communication. Use it to make decisions and context durable.

## How to use this file

1. Read `GOAL.md` and `.sgai/PROJECT_MANAGEMENT.md` before doing anything else.
2. Add an entry as soon as something changes (decision, constraint, question, plan update).
3. Keep entries small and scannable. Prefer bullets.
4. Include evidence (file paths, command outputs, links) when possible.
5. Append a new entry instead of rewriting history.

## Getting started (5 minutes)

1. Fill out **Project overview** (3 bullets is enough).
2. Add any hard **Known constraints** (deadlines, tech restrictions, must-not-change areas).
3. Create a first **Plan / task breakdown** with checkboxes.
4. Add at least one **Open question** if anything is unclear.
5. Set **Current status** so another agent can pick up the work.

At this point, another agent should be able to answer: “What are we doing, what’s the plan, and what should happen next?” by reading this file.

## Writing good entries

- Start bullets with an action verb: “Decide…”, “Investigate…”, “Verify…”, “Ask…”, “Ship…”.
- Prefer outcomes over process:
  - ✅ “Decided to store X in Y because Z.”
  - ❌ “Looked at a bunch of files.”
- Include just enough context that someone can continue work without redoing your investigation.

## Quick start checklist

- [ ] Add a short **Project overview**.
- [ ] Capture any **Known constraints**.
- [ ] Add a first-pass **Plan / task breakdown**.
- [ ] Record **Open questions** to ask the human partner.
- [ ] If a human answers a question, log it under **Human partner clarifications**.

## Project overview

- **Goal:** 
- **Non-goals:** 
- **Success criteria:** 

## Known constraints

- 

## Current status

- **Workflow step:** 
- **State:** 
- **Last updated:** 

## Plan / task breakdown

Write tasks in execution order. Use checkboxes so progress is visible.

- [ ] 

## Open questions

Log questions here before asking them.

For each question, include:

- **Context:** why this question matters
- **Question:** what is needed from the human partner
- **Options:** if there are likely answers, list them

### Questions

1. **Context:** 
   **Question:** 
   **Options:** 

## Human Partner Clarifications

When the human partner answers a question, copy the Q/A here so it persists.

Use this format:

- **Date:** YYYY-MM-DD
- **Context:** what prompted the question
- **Question:** 
- **Answer:** 
- **Impact:** what changes in the plan, requirements, or acceptance criteria

## Agent Decisions Log

Append decisions with dates. Include alternatives considered when helpful.

- **Date:** YYYY-MM-DD
  - **Decision:** 
  - **Reasoning:** 
  - **Impact:** 

## Evidence log

Paste high-signal evidence that supports decisions or completion claims.

- 

## Council evaluation (YYYY-MM-DD)

Use this section when a council-style review happens.

When adding council feedback, include:

- What was reviewed (files, commands, or artifacts)
- What is blocked vs. what is approved
- Concrete next steps (checkbox tasks) to resolve issues

### Items verified

- 

### Items reverted

- 

## STPA Analysis

Use this section to store outputs from an STPA analysis workflow.

Keep each step’s output in its own dated subsection so it stays scannable.

- **Date:** YYYY-MM-DD
  - **Step:** (overview | 1-define-purpose | 2-control-structure | 3-unsafe-control-actions | 4-loss-scenarios)
  - **Notes / output:**
    - 
