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
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] in the internal tab, in the todo box, the project todo and agent todo boxes don't need an interval vertical scroll bar

- [x] in progress tab, the button bar must wrap around on mobile instead of scroll

- [x] in internal tab, in steering box, there's no need for the internal title "instructions"

- [x] somewhere shift+enter is submitting the form, change it so that only ctrl+enter (windows and Linux) or cmd+enter (Mac) submits the forms

- [x] update the code so that every time sgai transition between agents, it copies the contents of `sgai/` into `.sgai` (it does it already once on start, I want once on start and at every agent transition)

- [x] update retrospective agent and skills to populate `.sgai/SGAI_NOTES.md` more often

- [x] in Root Repository in Forked Mode, add the button bar under the Fork tab too so that the root repository is able to run actions

- [x] in diff tab, remove the form that allows me to update the commit message
  - [x] remove also the server side inplementation 
  - [x] ensure that the diff is always the delta between the workspace origin (default@) and the head of the workspace in question

- [x] print more lines in the Log tab; keep the scrollbar hidden.