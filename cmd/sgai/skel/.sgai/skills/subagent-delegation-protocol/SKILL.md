---
name: subagent-delegation-protocol
description: Shared delegation contract for SGAI agents that invoke Task subagents. Use before delegating implementation, review, verification, deployment, writing, safety review, or project-critic work.
---

# Subagent Delegation Protocol

Use this skill when an agent delegates work to one or more Task subagents.

## Core Rules

- Delegate with bounded paths and sections instead of pasted large context blobs.
- Do not delegate broad `read the repo` work unless discovery itself is the task.
- Tell subagents to read packet paths first.
- Tell subagents to return `NEEDS_CONTEXT` with the exact missing path, evidence item, decision, or question when the packet is insufficient.
- Split broad work before fanout.
- Parallelize only independent work units.
- Reconcile every subagent return envelope before claiming completion.
- Use `subagent-return-protocol` skill for the return envelope named in the packet.

## Task Packet

Every Task prompt must include this packet shape:

```markdown
TASK_PACKET:
- Task: ...
- Owner agent: ...
- Goal path: GOAL.md
- Ledger path: .sgai/PROJECT_MANAGEMENT.md
- Required context paths:
  - ...
- Relevant ledger sections:
  - ...
- Expected edit boundaries:
  - ...
- Out-of-scope paths:
  - ...
- Verification commands:
  - ...
- Review expectations:
  - ...
- Return envelope: Execution Envelope | Review Envelope
```

## Work Split

Before broad fanout, create this split map:

```markdown
WORK_SPLIT:
- Work unit: ...
  Owner agent: ...
  Input paths:
    - ...
  Edit paths:
    - ...
  Output expected: ...
  Dependencies:
    - ... or none
  Can run in parallel with:
    - ...
  Must serialize because:
    - ... or none
```

## Splitting Rules

- Split by independently verifiable deliverable: package, command, route, page, component, template, script, deployment target, artifact, test file, safety concern, review concern, or verification concern.
- Parallelize only when edit paths and runtime resources do not conflict.
- Serialize work that shares files, APIs, generated artifacts, lockfiles, migrations, servers, databases, ports, browser sessions, package installs, deployment targets, or required outputs from sibling tasks.
- Include the concrete dependency reason for every serialized unit.

## Reconciliation Rules

- Compare each subagent status, changed paths, read paths, verification evidence, review findings, blockers, concerns, and next owner.
- If any unit returns `NEEDS_CONTEXT`, provide only the requested missing path, evidence, decision, or answer.
- If any unit returns `BLOCKED`, record the blocker and route it to the appropriate owner; do not claim completion for that unit.
- If parallel units changed overlapping files or made incompatible assumptions, stop integration and dispatch a focused reconciliation task.
- Proceed to completion or project-critic only after required work units have compatible evidence and no unresolved blockers.
