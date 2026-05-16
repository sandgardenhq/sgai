---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
alias:
  # "backend-go-developer-lite": "backend-go-developer"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "backend-go-developer": "openai/gpt-5.5"
  "go-readability-reviewer": "openai/gpt-5.5"
  "general-purpose": "openai/gpt-5.5"
  "react-developer": "openai/gpt-5.5"
  "react-reviewer": "openai/gpt-5.5"
  "stpa-analyst": "openai/gpt-5.5"
  "project-critic-council": ["openai/gpt-5.5"]
  "skill-writer": "openai/gpt-5.5 (xhigh)"
  # "backend-go-developer-lite": "openai/gpt-5.4-mini"
---

# Title of your Goal

One or two paragraphs explaining what you want to do.

- [ ] a list of verifiable checks to help agents to communicate their progress
  - [ ] they can even be nested
