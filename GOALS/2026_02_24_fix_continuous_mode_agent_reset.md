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
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---


I see this error: ```
[00:00:00.564][merge-dependabot-prs][continuous-mode        :0001] !  agent "continuous-mode" not found. Falling back to default agent
```

- [x] Is the current code broken?
  - [x] if so, what the problem is? `resetWorkflowForNextCycle` in `cmd/sgai/continuous.go:282` resets `Status` and `InteractionMode` but does not reset `CurrentAgent`, so the stale value `"continuous-mode"` (set at line 99 as a progress-tracking label) leaks into the next workflow cycle at `main.go:270-272`, causing `opencode run --agent continuous-mode` to be invoked against a nonexistent agent definition.
