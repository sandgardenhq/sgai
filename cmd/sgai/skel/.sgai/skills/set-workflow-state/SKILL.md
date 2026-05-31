---
name: set-workflow-state
description: You MUST USE THIS SKILL TO COMMUNICATE TO THE ENVIRONMENT WHAT YOU ARE DOING, WHERE YOU ARE GOING, AND WHETHER YOU ARE BLOCKED (IE NEED TO SEND THE HUMAN PARTNER A MESSAGE)
---

# Set Workflow State

## State File Protocol

**CRITICAL:** You must maintain the workflow state file throughout your work using the `sgai_update_workflow_state` tool.

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
- `"agent-done"` - Agent finished its work and the sgai should move to the next agent.
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

Agent handoffs are driven by durable notes in `.sgai/PROJECT_MANAGEMENT.md` plus optional `sgai_update_workflow_state({status: "agent-done", navigate: {to, reason}})` requests when the current runtime has made another agent available.

### How Handoffs Work

1. **Record the handoff**: Append the work request, blocker, or decision to `.sgai/PROJECT_MANAGEMENT.md`.

2. **Request a specific next agent only when appropriate**: Set `status: "agent-done"` with `navigate: {to: "agent-name", reason: "..."}` only for an agent that the coordinator or current handoff explicitly made available.

3. **Default to coordinator**: When no messages are pending, control returns to the coordinator.

4. **Default handoff target**: If no explicit next agent is requested, control returns to the coordinator to inspect the ledger and decide the next delegation.

### Key Points

- `.sgai/PROJECT_MANAGEMENT.md` is the durable coordination ledger
- Only coordinator can set `status: "complete"` to end the entire workflow
- The current handoff and available-agent list provide context for any explicit navigation request
- Agents never navigate to themselves
- The coordinator must never navigate to `coordinator`; it should record its own status with `sgai_update_workflow_state` or `.sgai/PROJECT_MANAGEMENT.md`

---
