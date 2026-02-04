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
  "project-critic-council": ["opencode/kimi-k2.5-free", "opencode/minimax-m2.1-free", "opencode/glm-4.7-free"]
  "skill-writer": "anthropic/claude-opus-4-5 (max)"
interactive: yes
completionGateScript: make test
mode: graph
---

Using gh, check these CI/CD errors https://github.com/sandgardenhq/sgai/pull/106/checks

- [x] fix ubuntu
- [x] fix macos

---

- [x] When I create a new workspace (`[ + ]`) few problems exist:
  - [x] Agents, Skills, Snippets are empty -- that means that when you create a workspace, you must immediately pre-populate .sgai for me
  - [x] when I create a new workspace, it seems that `.sgai` is not being correctly inserted into `.git/info/exclude`
  - [x] when I create a new workspace, fork button fails -- which indicates that .git and .jj weren't initialized yet
  - [x] create a branch
  - [x] copy GOAL.md to GOALS/ following the instructions from README.md


use GH to create a PR in sandgardenhq/sgai
