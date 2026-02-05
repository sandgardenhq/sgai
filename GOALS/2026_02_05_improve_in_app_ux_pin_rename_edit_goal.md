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


We have to improve the in-app experience of sgai.

We have few UI/UX issues to fix:
- [x] add ability to pin projects in the left tree
  - [x] pinned projects always show in the In Progress section
  - [x] use XDG config via `github.com/adrg/xdg` library
  - [x] store pins in `XDG_CONFIG_HOME/sgai/pinned_projects.json`
  - [x] JSON format: `{"pinned": [{"path": "...", "pinnedAt": "..."}]}`
  - [x] pin icon in left tree (next to project names)
  - [x] Pin/Unpin button in header actions bar
- [x] add ability to rename projects
      on the session/project page, I should be able to rename the project;
      the renaming _IS_ renaming in the file system.
  - [x] click on project title triggers HTMX swap to input form
  - [x] submit renames directory on filesystem
  - [x] pure HTMX implementation, no custom JavaScript
- [x] always show "Edit GOAL" when "Compose GOAL" is present
      the problem is that if it loads GOAL.md and the body is empty, it doesn't allow me to edit GOAL, only to compose GOAL. I want to edit the GOAL file.
  - [x] show BOTH "Compose GOAL" and "Edit GOAL" buttons always
  - [x] remove the `HasEditedGoal` condition check
