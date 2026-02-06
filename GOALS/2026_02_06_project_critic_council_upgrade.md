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
  "coordinator": "openai/gpt-5.2-codex (xhigh)"
  "backend-go-developer": "openai/gpt-5.2-codex"
  "go-readability-reviewer": "openai/gpt-5.2-codex"
  "general-purpose": "openai/gpt-5.2-codex"
  "htmx-picocss-frontend-developer": "openai/gpt-5.2-codex"
  "htmx-picocss-frontend-reviewer": "opencode/kimi-k2.5"
  "stpa-analyst": "openai/gpt-5.2-codex"
  "project-critic-council": ["openai/gpt-5.2-codex", "openai/gpt-5.2", "opencode/kimi-k2.5"]
  "skill-writer": "openai/gpt-5.2-codex (xhigh)"
interactive: yes
completionGateScript: make test
---

The project-critic-council is a very neat idea. I love it, and it really helps with ensuring the project gets to a completion.

However, no matter what model I use - GPT-5.2 or Opus 4.5 or Kimi K2.5 or Big Pickle... I can see often the agents over communicate.

In a sense, what I want is a more or less straightforward journey

0. The coordinator asks the Project Critic Council to execute an evaluation and deliver the message to its FrontMan.
1. FrontMan Sibling Mode asks all peers to execute their own evaluations
2. The Siblings talk to each other to communicate their own assessments
3. They all communicate to FrontMan their results
4. FrontMan evaluates these results, and it is biased to communicate back to the coordinator

- [x] Evaluate changes in the Project Critic Council that could more closely adhere to these goals
- [x] Improve Web Interface so that the ORDER of models displayed in the UI is the same as the one written in GOAL.md

## Council Plan

- [x] Define FrontMan and Sibling roles explicitly; map each step (0–4) to a single expected message.
- [x] Standardize peer evaluation and aggregation message templates to reduce over-communication.
- [x] Ensure only the FrontMan communicates back to the coordinator.
- [x] Clarify when/why peers communicate with each other (step 2) and how to keep it minimal.

## UI Ordering Plan

- [x] Identify UI view(s) that display model order for project-critic-council.
- [x] Trace the data flow to confirm where model list order is sourced.
- [x] Update the source to use the GOAL.md frontmatter `models` list order for project-critic-council.
- [x] Define fallback behavior for missing/unknown models.
- [x] Verify UI display order matches GOAL.md.
