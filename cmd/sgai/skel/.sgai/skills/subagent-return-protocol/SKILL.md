---
name: subagent-return-protocol
description: Shared return contract for SGAI Task subagents. Use before any delegated implementation, deployment, writing, verification, review, or project-critic subagent reports back to its caller.
---

# Subagent Return Protocol

Use this skill when a Task subagent reports completion, concerns, missing context, review findings, or blockers to the agent that invoked it.

## Core Rules

- Return exactly one envelope as the final answer.
- Do not call workflow-state tools unless your prompt explicitly permits them.
- Read `TASK_PACKET` paths first when a packet is provided.
- If context is insufficient, return the missing path, evidence item, decision, or question through `NEEDS_CONTEXT`.
- Use `NEXT_OWNER: invoking agent` when the caller should reconcile the result.
- Use `NEXT_OWNER: coordinator` only when the coordinator must decide or route the next step.
- Use a concrete owner such as `project-critic` only when your prompt names that wrapper as the caller.

## Execution Envelope

Use this for implementation, deployment, writing, cleanup, and other work-producing subagents:

```markdown
STATUS: DONE | DONE_WITH_CONCERNS | NEEDS_CONTEXT | BLOCKED
TASK: ...
READ_PATHS:
- ...
CHANGED_PATHS:
- ...
VERIFICATION:
- command: ...
  result: ...
EVIDENCE:
- ...
REVIEW_SCOPE:
- ...
CONCERNS:
- ...
NEXT_OWNER: invoking agent | coordinator
```

## Review Envelope

Use this for reviewer, verifier, safety-review, and project-critic subagents:

```markdown
REVIEW_STATUS: PASS | NEEDS_WORK | NEEDS_CONTEXT | BLOCKED
SCOPE_REVIEWED:
- ...
EVIDENCE_REVIEWED:
- ...
FINDINGS:
- file:line - required fix
BLOCKERS:
- ...
NEXT_OWNER: invoking agent | developer | coordinator
```

## Status Meanings

- `DONE`: completed the task with required verification.
- `DONE_WITH_CONCERNS`: completed the task, but caller must consider listed concerns.
- `PASS`: reviewed scope has no required fixes.
- `NEEDS_WORK`: caller must route listed findings for fixes.
- `NEEDS_CONTEXT`: caller must provide only the missing context listed.
- `BLOCKED`: caller must resolve the blocker before this work can complete.

## Reconciliation Guidance

- Put required fixes in `FINDINGS`, not prose outside the envelope.
- Put unknowns or unavailable resources in `BLOCKERS`.
- Put command output, screenshots, docs, and files read in evidence fields.
- Keep `NEXT_OWNER` about routing, not blame.
