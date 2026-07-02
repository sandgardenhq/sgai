---
name: set-workflow-state
description: Coordinator and retrospective workflow-state protocol. Use only when the current agent has permission to call sgai_update_workflow_state; Task subagents should return STATUS or REVIEW_STATUS envelopes instead.
---

# Set Workflow State

## State File Protocol

**CRITICAL:** Use `sgai_update_workflow_state` only from agents that have permission to call it, primarily `coordinator` and `retrospective`. Developer, reviewer, verifier, deployer, writer, and hidden role subagents should return their `STATUS` or `REVIEW_STATUS` envelopes to the caller instead.

### State Schema

```json
{
  "status": "working" | "agent-done" | "complete",
  "message": "Status message describing current state",
  "task": "Current task description or null",
  "progress": ["note1", "note2", "note3"]
}
```

### Field Meanings

**`status`** - Overall workflow status
- `"working"` - Actively working, may need another iteration
- `"agent-done"` - Current primary workflow agent finished its turn and SGAI should return control to coordinator or the next workflow phase.
- `"complete"` - Work is done successfully

**`message`** - Status message
- Brief description of current state
- When complete, provide a summary of what was accomplished
- Be clear and specific

**`task`** - Current task being worked on (required)
- Set to describe your current task: "Writing tests for login endpoint"
- Set to empty string (`""`) when no specific task or when complete
- Helps human understand your progress in real-time

**`progress`** - Progress log (array of strings)
- Append entries as you work - this is your audit trail
- Document major steps completed
- Note decisions made and why
- Track issues encountered and how you resolved them
- This helps humans understand your thought process

### How to Update State File

**IMPORTANT:** Use the `sgai_update_workflow_state` tool to update the state file. Do NOT use bash/jq commands directly.

**Tool: sgai_update_workflow_state**

Available parameters:
- `status` (required): "working" | "agent-done" | "complete"
- `task` (required): Current task being worked on. Use empty string to clear.
- `addProgress` (required): Add a progress note (appended to progress array). Document your steps frequently.

**Examples:**

```
Update task and add progress note:
Tool: sgai_update_workflow_state
Parameters: {
  "task": "Writing tests for authentication endpoints",
  "addProgress": "Created User model with bcrypt password hashing"
}
```

```
Mark as complete:
Tool: sgai_update_workflow_state
Parameters: {
  "status": "complete",
  "task": "",
  "addProgress": "All 15 tests passing, feature verified working"
}
```

```
Signal work is done for this agent:
Tool: sgai_update_workflow_state
Parameters: {
  "status": "agent-done",
  "task": "",
  "addProgress": "Review complete, ready for next step"
}
```

**Update frequency:**
- After completing each major step
- When changing tasks
- Before requesting another iteration
- Before marking complete
- At least every few turns of work

### Human Communication

Use `ask_user_question` to present structured multi-choice questions to the human partner. This is the sole channel for human communication.

---

## Coordinator Handoffs

Workflow handoffs are driven by durable notes in `.sgai/PROJECT_MANAGEMENT.md` plus `sgai_update_workflow_state({status: "agent-done"})` to yield control when the current agent is a workflow owner. Task subagent handoffs are driven by their final envelope response.

### How Handoffs Work

1. **Record the handoff**: Append the work request, blocker, or decision to `.sgai/PROJECT_MANAGEMENT.md` only if the current agent has permission and the prompt explicitly owns that ledger write.

2. **Yield control when done**: Set `status: "agent-done"` after recording durable handoff notes in `.sgai/PROJECT_MANAGEMENT.md`.

3. **Default to coordinator**: When no messages are pending, control returns to the coordinator.

4. **Default handoff target**: If no explicit next agent is requested, control returns to the coordinator to inspect the ledger and decide the next delegation.

### Key Points

- `.sgai/PROJECT_MANAGEMENT.md` is the durable coordination ledger
- Only coordinator can set `status: "complete"` to end the entire workflow
- The current handoff and available-agent list provide context for coordinator delegation
- Agents never route to themselves
- The coordinator should record its own status with `sgai_update_workflow_state` or `.sgai/PROJECT_MANAGEMENT.md`

### Task Subagent Return Protocol

If you are a Task subagent, do not call `sgai_update_workflow_state` unless your prompt explicitly permits it. Load and follow `subagent-return-protocol` skill; implementation, deployment, writing, and cleanup agents return its Execution Envelope, while reviewer, verifier, safety-review, and project-critic agents return its Review Envelope.

---
