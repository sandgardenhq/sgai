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

- [x] retrospective agent correctly sets `status:"agent-done"` but it never stops, it must stop to let the outer clockwork to tick
      and when that happens, I have to manually sigterm opencode to let the clockwork resume; make sure that the retrospective agent has strong prompt instructions to stop after changing the state.