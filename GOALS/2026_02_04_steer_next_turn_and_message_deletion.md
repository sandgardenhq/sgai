---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
  "project-critic-council": ["opencode/kimi-k2.5", "opencode/minimax-m2.1", "opencode/glm-4.7"]
  "skill-writer": "anthropic/claude-opus-4-5 (max)"
interactive: yes
completionGateScript: make test
---

- [x] inside the Internals tab, I want a new box name "Steer Next Turn"
  - [x] it takes a text input and a submit button, the submit should be part of a group with the text input, so it is all in the same line (visually) - refer to https://picocss.com/docs/group
  - [x] when I submit, it adds a message right before the oldest unread message
        From "Human Parter"
        To "Coordinator"
        Re-steering instruction: "$message"
    - [x] if ALL messages are read, then it goes to the top of the list
    - [x] on finishing the submission, the form has to be reset
- [x] in the Messages tab, add command to delete a message from the state file
