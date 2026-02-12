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
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
interactive: yes
completionGateScript: make test
---

- [x] The `statusLine` in WorkspaceDetails.tsx should not be displayed for the Root Repository in Forked Mode
