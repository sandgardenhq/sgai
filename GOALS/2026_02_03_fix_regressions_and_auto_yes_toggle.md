---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
  "project-critic-council": ["opencode/kimi-k2.5-free", "opencode/minimax-m2.1-free", "opencode/glm-4.7-free"]
interactive: yes
completionGateScript: make test
mode: graph
---

- [x] Once the session is started, add a button to allow me to alternate between auto and yes.

Using gh look at https://github.com/sandgardenhq/sgai/actions/runs/21652033466/job/62418618478?pr=96
note how the tests are not passing:
- [x] fix regressions

Using gh look at https://github.com/sandgardenhq/sgai/actions/runs/21657705908/job/62435865772?pr=96
note how the tests are not passing:
- [x] fix regressions
