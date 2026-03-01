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

I want a new auto-mode. Basically, I am trying to fix the issue of the GOAL.md maybe not having a frontmatter block.

Basically, if `flow: "auto"` OR the frontmatter is empty, then the automatic flow mode must kick in.

What I would expect the coordinator to use a skill that:
- [x] look at the workspace source code, and correctly pick the right agents for me
  - [x] it would survey the agents from withing `.sgai/agent` subdirectory
- [x] use the default model for these agents
- [x] when picking up agents, correct pair agents (like developer with reviewer)


In term of sequence of events: user writes GOAL without frontmatter, user starts sgai, the coordinator detects that frontmatter is missing or lacking flow, go shop for agents, update the GOAL.md with the agents, then resume as if GOAL.md had the flow defined all along.

- [x] rebuildIfFlowChanged is conceptually incorrect, I guess. On every loop, you have to rebuild the DAG and the model configuration from GOAL.md, unconditionally. When you add tests like this, you make it the code more complex and less reliable. 