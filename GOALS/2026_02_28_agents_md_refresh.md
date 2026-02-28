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
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] I don't see either the project-critic-council _or_ the retrospective agent doing any meaningful updates to AGENTS.md; at every session, the retrospective agent should consume AGENTS.md and propose changes to it, especially:
  - [x] suggest the creation of AGENTS.md when missing
  - [x] suggest meaningful updates when the user asked for something that contradicts AGENTS.md
  - [x] when AGENTS.md gets too large, find opportunities of instructions to remove
  - [x] evaluate AGENTS.md in a way that it could be re-structured into smaller files to avoid file size bloat.
