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
  "project-critic-council": ["anthropic/claude-opus-4-6 (max)", "anthropic/claude-opus-4-5", "anthropic/claude-sonnet-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---


- [x] Fix the Diff Tab (the view full diff is broken)
  - [x] It used to work, check the previous changes to figure out why