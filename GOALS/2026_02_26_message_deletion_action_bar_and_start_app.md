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
  "coordinator": "opencode/kimi-k2.5"
  "backend-go-developer": "opencode/kimi-k2.5"
  "go-readability-reviewer": "opencode/kimi-k2.5"
  "general-purpose": "opencode/kimi-k2.5"
  "react-developer": "opencode/kimi-k2.5"
  "react-reviewer": "opencode/kimi-k2.5"
  "stpa-analyst": "opencode/kimi-k2.5"
  "project-critic-council": ["opencode/kimi-k2.5"]
  "skill-writer": "opencode/kimi-k2.5"
completionGateScript: make test
---

- [x] add button to remove message from the message bus
- [x] move the Internal's button bar into the Progress tab
- [x] add a `Start Application` button in the internal definitions
  - [x] add a `Start Application` button in the sgai.example.json too
- [x] pending messages counter seems to be incorrect, counting double.