---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-sonnet-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
interactive: yes
completionGateScript: make test
---

- [x] Move the Run from inside Internal and make its own Tab
- [x] The default model for the Run tab must be the same model of "coordinator"(it is safe to rely only on the model name and ignore the variant within parenthesis)
- [x] As the Run box runs, the output scrolls up over and over, and I can't keep up with the output.
- [x] Run should use stdin instead of parameter `echo $msg | opencode run` instead of `opencode run $msg`
- [x] update opencode.json in sgai template to automatically deny doom_loop and external_directory permissions (refer to https://opencode.ai/docs/permissions/#granular-rules-object-syntax )