---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
  "project-critic-council": ["anthropic/claude-opus-4-5", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-5 (max)"
interactive: yes
completionGateScript: make test
---

Question: "Hard-code project critic council? Be even more strict about goal?"

I want to improve the sgai ability to see a feature done to the end. And to that effect we are going to make few changes.

# Change 1: Make project-critic-council mandatory
In the sgai internal clockwork, there is a logic that forces/prepends the coordinator to all left-most nodes.
We are going to add one mandatory relationship
- [x] add `"coordinator" -> "project-critic-council"`
- [x] update `coordinator.md` model to default to "anthropic/claude-opus-4-5 (max)" (refer to https://opencode.ai/docs/agents#markdown)
  - [x] update `coordinator.md` to include a step in its master plan to ask `project-critic-council` agent whether the project is complete
- [x] update `project-critic-council.md` model to default to "anthropic/claude-opus-4-5 (max)" (refer to https://opencode.ai/docs/agents#markdown)
  - [x] update `project-critic-council.md` to include forceful instructions to always read PROJECT_MANAGEMENT.md and GOAL.md to make proper evaluations and decisions
  - [x] update `project-critic-council.md` to include forceful instructions to always communicate back with the coordinator

# Change 2: Ask Human Partner about validation criteria
- [x] update the `cmd/sgai/skel/.sgai/skills/product-design/brainstorming/SKILL.md` skill to interview the human partner with validation questions that are supposed to be used by the project-critic-council; make sure that inside the brainstorming skill there are instructions to update PROJECT_MANAGEMENT.md with the human partner answers as it evolves.

