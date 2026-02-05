---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
  "project-critic-council": ["anthropic/claude-opus-4-5", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-5 (max)"
interactive: yes
completionGateScript: make test
---

- [x] In Messages tab, the messages box doesn't need a title
  - [x] does it even need a box? why not inline?
- [x] In Run, the Run box doesn't need a title
  - [x] does it even need a box? why not inline?