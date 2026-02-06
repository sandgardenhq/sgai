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

- [x] using a combination of file rename and `jj workspace rename`, add the ability to rename forks
   - [x] make it a link from the title into its own screen (like when creating a new workspace)