---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "backend-go-developer" -> "stpa-analyst"
  "react-reviewer" -> "stpa-analyst"
models:
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "coordinator": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
---

There is a bug in SGAI where when you rename a workspace, it appears twice in the left navigation. It appears once as its renamed name and another time as its original name.

- [x] after renaming the workspace it should only appear once in the left nav. 
I have a screenshot that I will attach.

