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

- [x] use rich editor for Edit GOAL (assume markdown)
  - [x] ensure the rich editor always has word-wrap turned on by default
  - [x] add basic markdown operations buttons
- [x] use rich editor for Compose GOAL (assume markdown) wizard staged
  - [x] ensure the rich editor always has word-wrap turned on by default
  - [x] add basic markdown operations buttons
- [x] use rich editor for Respond to Agent (assume markdown)
  - [x] ensure the rich editor always has word-wrap turned on by default
  - [x] add basic markdown operations buttons
- [x] use rich editor for AdHoc Runs (assume markdown)
  - [x] ensure the rich editor always has word-wrap turned on by default
  - [x] add basic markdown operations buttons
- [x] Fix the resize - in http://localhost:8080/workspaces/rich-editor-monaco/goal/edit, when I resized the editor to be taller, the container resized, but the embedded editor didn't.
- [x] Close https://github.com/sandgardenhq/sgai/issues/72
