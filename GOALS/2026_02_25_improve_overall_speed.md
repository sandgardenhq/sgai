---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose" 
  "go-readability-reviewer" 
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6 (max)"
  "project-critic-council": ["anthropic/claude-opus-4-6 (max)"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

it is overall still very slow.

- [x] what if you were more efficient with jj calls?
- [x] also when I click around, for example, the UI doesn't update as quickly - when I click Pin, I don't see the pinned repository right away
- [x] using analytical skills and agents, figure out why it is very slow when I browse around