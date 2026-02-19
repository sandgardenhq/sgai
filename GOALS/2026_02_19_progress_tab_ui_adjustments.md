---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "opencode/kimi-k2.5"
  "backend-go-developer": "opencode/kimi-k2.5"
  "go-readability-reviewer": "opencode/kimi-k2.5"
  "general-purpose": "opencode/kimi-k2.5"
  "react-developer": "opencode/kimi-k2.5"
  "react-reviewer": "opencode/kimi-k2.5"
  "stpa-analyst": "opencode/kimi-k2.5"
  "project-critic-council": ["opencode/kimi-k2.5", "opencode/minimax-m2.5-free", "opencode/glm-5-free"]
  "skill-writer": "opencode/kimi-k2.5"
completionGateScript: make test
---

The progress tab is good but it needs few adjustments:

- [x] the model table must be displayed at the right of the flow graph
- [x] the model table must also be updated when the GOAL.md is edited by the user (possibly poll? not sure what's the best here)
- [x] remove the "Agent Models title"
- [x] remove the scrollbar caused by the horizontal overflow, the best would be to break the line (or print in a bulleted list for multi model agents)
