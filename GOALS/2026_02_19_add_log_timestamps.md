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
  "coordinator": "opencode/glm-5"
  "backend-go-developer": "opencode/glm-5"
  "go-readability-reviewer": "opencode/glm-5"
  "general-purpose": "opencode/glm-5"
  "react-developer": "opencode/glm-5"
  "react-reviewer": "opencode/glm-5"
  "stpa-analyst": "opencode/glm-5"
  "project-critic-council": ["opencode/glm-5", "opencode/kimi-k2.5", "opencode/minimax-m2.5"]
  "skill-writer": "opencode/glm-5"
completionGateScript: make test
---

Currently, in the logs, everywhere I have this format

[factoryname][agentname:####]

- [x] add timestamp so that I can see how long each line is taking to run
