---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "react-developer": "openai/gpt-5.2-codex (xhigh)"
  "react-reviewer": "openai/gpt-5.2-codex (xhigh)"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
interactive: yes
completionGateScript: make test
---

There is a continuation message that's fed back into the agent.

We need to make two changes.

- [x] when the agent sends a message with `sgai_send_message`, we instruct the agent that to be able to see a response it MUST yield the control back (as part of the response to `sgai_send_message)
- [x] in the continuation message, if the agent has an outbox with pending messages to be delivered, we must add a instruction in the continuation message that says something like "Listen, you sent a message; for the other agent get back to you, have to yield control by marking state:agent-done`
