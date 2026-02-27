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
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6 (max)"
  "project-critic-council": ["anthropic/claude-opus-4-6 (max)", "anthropic/claude-opus-4-5", "anthropic/claude-sonnet-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] In the Internal tab there is a button bar that shouldn't be there. Not sure why it's there. The only button bars that should exist are:
  - [x] In the standalone / forked repository, button bar in the progress tab
  - [x] In the root repository in forked mode
    - [x] button bar in each forked repository
    - [x] button bar at the top, below the forks tab - that applies to the root repository itself.

- [x] The button bar has an internal default
  - [x] it is fully overwritten (full replacement) by the value in sgai.json