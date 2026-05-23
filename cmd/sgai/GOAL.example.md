---
flow: |
  "go"
  "react"
  "general-purpose"
  "skill-writer"
alias:
  # "go-lite": "go"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  # GPT-5.5 low is the recommended default for non-coordinator agents.
  "go": "openai/gpt-5.5 (low)"
  "general-purpose": "openai/gpt-5.5 (low)"
  "react": "openai/gpt-5.5 (low)"
  "project-critic-council": ["openai/gpt-5.5 (low)"]
  "skill-writer": "openai/gpt-5.5 (low)"
  # "go-lite": "openai/gpt-5.5 (low)"
---

# Title of your Goal

One or two paragraphs explaining what you want to do.

- [ ] a list of verifiable checks to help agents to communicate their progress
  - [ ] they can even be nested
