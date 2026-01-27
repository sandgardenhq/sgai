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
  "status": "working" | "agent-done" | "complete" | "human-communication",
  "message": "Status message describing current state",
  "task": "Current task description or null",
  "progress": ["note1", "note2", "note3"],
  "humanMessage": "A message for the human to see, must be set with status:human-communication"
}
```

### Field Meanings

**`status`** - Overall workflow status
- `"working"` - Actively working, may need another iteration
- `"agent-done"` - Agent finished its work and the sgai should move to the next agent.
- `"complete"` - Work is done successfully
- `"human-communication"` - Return control to human partner to communicate

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

**`humanMessage`** - Message you want the human to see so it can communicate back to you
- Set when you need human input
- Leave as `null` during normal work
- Examples:
  - "Should I use PostgreSQL or MongoDB for this feature?"
  - "The API credentials are missing. Where should I find them?"
  - "Design choice: REST or GraphQL for this endpoint?"
- After human responds, clear this field and continue working

### How to Update State File

**IMPORTANT:** Use the `sgai_update_workflow_state` tool to update the state file. Do NOT use bash/jq commands directly.

**Tool: sgai_update_workflow_state**

Available parameters:
- `status` (required): "working" | "agent-done" | "complete" | "human-communication"
- `task` (required): Current task being worked on. Use empty string to clear.
- `addProgress` (required): Add a progress note (appended to progress array). Document your steps frequently.
- `humanMessage` (optional): A message for the human to see, must be set with status:human-communication, or null

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
Request help (blocked):
Tool: sgai_update_workflow_state
Parameters: {
  "status": "human-communication",
  "humanMessage": "Should I use refresh tokens (complex, better UX) or simple JWT (simpler)?",
  "addProgress": "Paused to ask about authentication strategy"
}
```

```
Continue working after help received:
Tool: sgai_update_workflow_state
Parameters: {
  "status": "working",
  "humanMessage": null,
  "task": "Implementing refresh token logic",
  "addProgress": "Human chose refresh tokens, proceeding with implementation"
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

---

## Message-Driven Navigation

In flow mode, navigation between agents is driven by inter-agent messages using `sgai_send_message()` and `sgai_check_inbox()`.

### How Navigation Works

1. **Send a message to route work**: Use `sgai_send_message({toAgent: "agent-name", body: "..."})` to send work to another agent.

2. **Message-based routing**: When you set `status: "agent-done"`, the system checks for pending messages and routes to the agent with the oldest unread message.

3. **Default to coordinator**: When no messages are pending, control returns to the coordinator.

4. **Terminal Nodes**: If your agent has no successors (terminal node), control returns to the coordinator.

### Key Points

- Messages drive the workflow - send a message to an agent to have work routed to them
- Only coordinator can set `status: "complete"` to end the entire workflow
- The DAG (flow definition) represents intention and provides predecessor/successor information for context
- Any agent can message any other agent

---
