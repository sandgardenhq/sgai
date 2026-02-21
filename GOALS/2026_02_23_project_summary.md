---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "project-critic-council"
  "stpa-analyst"
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

I really like what we are doing here, but we need to improve the HCI/UX.

Specifically, I want to add english text single sentence summary of the projects

- [x] compose a summary and update the UI
  - [x] when GOAL.md is saved, or when a summary is missing
  - [x] when project is started and a summary is missing
  - [x] BUG: when I click `Edit GOAL.md`, and then save it, it is not updating.

- [x] allow the user to edit the summary manually if they want to
  - [x] manual summaries are sticky, they shouldn't be overriden by automatic summaries

- [x] show the summary on the left tree
  - [x] in the tree
  - [x] in the In Progress (pinned and in-progress sessions)
- [x] show the summary on the workspace page
- [x] show the summary on the Root Repository in Forked Mode
  - [x] on each Forked Repository line inside the Root Repository in Forked Mode Page

- [x] using the events systems already in place, make sure that `Edit GOAL.md` doesn't block (right now, it takes several seconds between saving and redirecting, becasue it needs to call opencode to compose the summary message)
  - [x] do that in the background
  - [x] handle debouncing and last-one-wins implementation, so that multiple edits are handled correctly (and avoid duplicated calls to opencode)

