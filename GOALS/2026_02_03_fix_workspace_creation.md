# Fix Workspace Creation Pre-population

When creating a new workspace via the web UI `[+]` button, the workspace should be immediately usable without requiring a manual engine run first.

## Problem

The web UI `handleNewWorkspacePost` creates only the directory and a `GOAL.md`, but does not:
1. Unpack the `.sgai` skeleton (agents, skills, snippets, plugin, opencode.jsonc)
2. Initialize git/jj version control
3. Add `/.sgai` to `.git/info/exclude`

This means the workspace appears broken until the first engine run: agents/skills/snippets tabs are empty, the fork button fails, and `.sgai` files are tracked by git.

## Requirements

- [ ] When a new workspace is created, immediately pre-populate `.sgai` from the embedded skeleton
- [ ] Initialize `jj` (with `jj git init --colocate`) in the new workspace
- [ ] Add `/.sgai` to `.git/info/exclude` so it is not tracked
- [ ] The fork button should work immediately after workspace creation
- [ ] Existing workspace initialization (engine startup) should continue to work
- [ ] The same initialization logic should apply to `handleWorkspaceInit` and `handleWorkspaceFork`
