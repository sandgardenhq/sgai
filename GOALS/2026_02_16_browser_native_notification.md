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
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

Refer to https://github.com/sandgardenhq/sgai/issues/69 (using gh)

- [x] Implement Browser Notification
  - [x] Notification should only ring when the browser is open and in the page
- [x] copy GOAL.md (after all checkboxes are updated) into GOALS/ following the instructions from README.md
- [x] using GH, make a draft PR for the commit at @ (jj) - from here
- [x] it didn't work - not sure why
  - [x] it never asked for permission either, using playwright check the console logs to check for the existence of permission errors.
  - [x] only if strictly necessary, add a check for notification, and a yellow top bar, on top of everything to let the user to click to enable the notification or to dismiss it (store in browser the dismissal)
