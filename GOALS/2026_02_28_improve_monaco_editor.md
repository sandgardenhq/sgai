---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "anthropic/claude-sonnet-4-6", "anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
---

Refer to https://microsoft.github.io/monaco-editor/

- [x] In "Edit GOAL", in preview, please print a table with the frontmatter 
- [x] Update Monaco editor component to enable "Select All"