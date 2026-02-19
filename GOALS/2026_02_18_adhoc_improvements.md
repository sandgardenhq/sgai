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
  "project-critic-council": ["opencode/glm-5-free", "opencode/kimi-k2.5-free",  "opencode/minimax-m2.5-free"]
  "skill-writer": "opencode/glm-5"
completionGateScript: make test
---

The adhoc runners in Root Repository in Forked Mode and Run tab in Standalone/Forked Repositories need some improvements

- [x] runs must be stoppable - Add Stop button to all adhoc UIs (RunTab, InlineRunBox in ForksTab, AdhocOutput) with backend DELETE /api/v1/workspaces/{name}/adhoc endpoint
  - [x] the stop button must ensure the underlying opencode process is properly stopped
- [x] layout should be better - Make model selector and prompt textarea side-by-side layout
- [x] shift+enter should work - Shift+Enter submits, plain Enter adds new line (code editor style)
