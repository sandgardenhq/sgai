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

The web interface needs some improvement on the directory list on the left.

- [x] on the left tree, below the pinned repositories list, list only directories that has either .sgai or GOAL.md
   - [x] if a directory has the GOAL.md present but `.sgai` directory is absent, then create an empty `.sgai` folder and show the workspace
