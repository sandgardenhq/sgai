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
  "project-critic-council": ["anthropic/claude-opus-4-6","anthropic/claude-sonnet-4-6","anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

`ERROR: TimeoutError: The operation timed out.` -- I am seeing a lot of this error message in the projects I am doing.

But at the same time, I don't get any notification that the LLM wants to ask me a question.

- [x] find the root cause (use all analytical skills)
    - [x] Track 1: Fix `opencodeConfig` struct in `config.go` to preserve all fields (likely root cause - drops `experimental.mcp_timeout` when re-writing `opencode.jsonc`)
    - [x] Track 2: Add diagnostic logging to `AskAndWait()` for future debugging
    - [x] Track 3: Fix notification persistence - don't clear question state on timeout so ⚠ stays visible
    - [x] Track 4: Add macOS desktop notification when question is pending
      - [x] revert Track 4

*Human Partner Guess: I wonder if it has to do with the fact I am running multiple workspaces at the same time*
*Critical* run a second pass to confirm you have enough logging.
