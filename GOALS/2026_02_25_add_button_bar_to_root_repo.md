---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
  "c4-code"
  "c4-component"
  "c4-container"
  "c4-context"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "c4-code": "anthropic/claude-sonnet-4-6"
  "c4-component": "anthropic/claude-sonnet-4-6"
  "c4-container": "anthropic/claude-sonnet-4-6"
  "c4-context": "anthropic/claude-sonnet-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
  "stpa-analyst": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] Add to the Root Repository in Forked Mode, the same button bar available in Forked Repository / Standalone Repository