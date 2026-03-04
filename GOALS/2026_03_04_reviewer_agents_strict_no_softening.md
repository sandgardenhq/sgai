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
  "project-critic-council": ["anthropic/claude-opus-4-6", "anthropic/claude-sonnet-4-6", "anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

# Reviewer Agents Issue

The reviewer agents are actually very helpful and useful. However, they often emit observations that they deem to not be blocking.

- [x] Update reviewer agents to never "soften the blow" no issue is small enough. The whole point of the reviewer agents is to highlight ALL problems. Let the developer agent to decide what to react to or not in that particular iteration.

## Agreed Definition (2026-03-04)

- Scope: apply to `go-readability-reviewer`, `react-reviewer`, `htmx-picocss-frontend-reviewer`, and `shell-script-reviewer` only.
- Approach: hybrid rollout (shared strict contract + targeted conflict cleanup per file).
- Acceptance criteria: all four reviewer prompts enforce no-softening policy and remove severity-downplaying language.
- Verification: prompt-content checks plus behavior spot-checks for no issue downplaying.
- Required evidence artifact: `REVIEWERS_UPDATE.md` completion report.
