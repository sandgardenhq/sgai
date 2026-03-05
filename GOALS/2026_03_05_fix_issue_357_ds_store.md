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
  "go-readability-reviewer": ["anthropic/claude-opus-4-6","opencode/glm-5"]
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": ["anthropic/claude-opus-4-6","opencode/glm-5"]
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": "anthropic/claude-opus-4-6"
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

# Fix issue 357

Using GH, absorb issue https://github.com/sandgardenhq/sgai/issues/357

- [x] Fix issue
- [ ] copy GOAL.md into GOALS/ following the instructions from README.md; store the git path (by querying jj) into GIT_DIR, and using GH, make a draft PR for the commit at @ (jj); CRITICAL: commit message, the PR title and body, must adhere to the standard of previous commits - update all of these if necessary; once you are done, using bash(`open`), open the PR for me. REMEMBER: 'GOALS' must never be the used commit message or PR title prefix