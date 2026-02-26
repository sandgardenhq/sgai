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
  "project-critic-council": ["anthropic/claude-opus-4-6 (max)"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

The Root Repository in Forked Mode is still very slow.

Let's make it faster by simplifying it.

- [x] Instead of having multiple cards, make it a compact interface;
- [x] add the button bar buttons into the list (the ones from sgai.json / actions)

in the new cards:
- [x] drop the internal title: "commits"
- [x] drop the pill with `# ahead`