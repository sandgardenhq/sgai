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
  "coordinator": "openai/gpt-5.2-codex"
  "backend-go-developer": "openai/gpt-5.2-codex"
  "go-readability-reviewer": "openai/gpt-5.2-codex (xhigh)"
  "general-purpose": "openai/gpt-5.2-codex"
  "htmx-picocss-frontend-developer": "openai/gpt-5.2-codex"
  "htmx-picocss-frontend-reviewer": "openai/gpt-5.2-codex"
  "stpa-analyst": "openai/gpt-5.2-codex"
  "project-critic-council": ["openai/gpt-5.2-codex", "openai/gpt-5.2", "openai/gpt-5.2-codex (high)"]
  "skill-writer": "openai/gpt-5.2-codex"
interactive: yes
completionGateScript: make test
---

The Fork button is really useful. I love it. It creates some frictions though, and we are going to address them now.

- [x] when I click the fork button, I must be asked for the name of the fork
- [x] use kebab case when creating the jj workspace with it

Once a project is forked,
- [x] the root MUST NOT be used to execute agentic work, ONLY the forks can do work;
- [x] the root MUST alternate to a different view in which I see all the forks, their commit lists, and for each, an automated "merge" button that ensure everything is commited, creates the jj bookmark, push and create the PR; and
  - [x] upon confirmation, erases the fork.
  - [x] in terms of interface
        - [x] that means, when in Root Mode, Progress, Diffs, Commits, Messages should be gone
        - [x] that means, when in Root Mode, new tabs
                - [x] Forks - in which I can see all the forks, the deltas in them, with the merge button

