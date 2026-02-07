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
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
interactive: yes
completionGateScript: make test
---

Terminology:
- Standalone Repository: a repository that has only _one_ `jj workspace` -- itself.
- Root Repository: a repository that has more than one `jj workspace`, and it is the root (it is the one in which `.jj/repo` is a directory and not a file)
- Forked Repository: a repository that is part of a `jj workspace, and it is not the root (it is the one in which `.jj/repo` is a text file, whose content points to the parent).

- Repository Mode: is when a repository is served by SGAI in a way that it can actually run software.
- Forked Mode: is when a root repository has at least one child, it displays the fork (dashboard-style) mode.
**CRITICAL** when a Root Repository run out of children, it must revert back from Forked Mode to Repository Mode.

Expected Behaviors:
- [x] Standalone Repositories
    - [x] Must always display the Fork Button
    - [x] Must always use the Repository Mode
- [x] Root Repositories
    - [x] Must always display the Fork Button
    - [x] Must always use the Forked Mode
    - [x] Forked Repositories of a Root Repository must NEVER show the Fork Button
    - [x] Forked Repositories of a Root Repository must ALWAYS be displayed in Repository Mode
