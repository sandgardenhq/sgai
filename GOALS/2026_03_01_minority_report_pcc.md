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
  "project-critic-council": ["anthropic/claude-opus-4-6","anthropic/claude-sonnet-4-6", "anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

The project-critic-council works reasonably well in guaranteeing that the project is on track. 

However, I often see models simply agreeing with each other.

Also, we artificially to say that the first model in the multi-model list is the "FrontMan" that has specific jobs.

I want to artificialize another role: "MinorityReport" -- it is always the _last_ model in the multi-model agent setup in project-critic-council

The "MinorityReport" jobs is to question the establishment and find holes in the current evaluation. Its job is to think out of the box, but at the same time, doing that in a responsible way that doesn't prevent the project-critic-council from converging to a decision.

- [x] Add the concept of the "MinorityReport" to the project-critic-council.md when multiple models are available
