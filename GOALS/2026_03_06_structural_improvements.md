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
  "go-readability-reviewer": "anthropic/claude-opus-4-6 (max)"
  "general-purpose": "anthropic/claude-opus-4-6 (max)"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6 (max)"
  "stpa-analyst": "anthropic/claude-opus-4-6 (max)"
  "project-critic-council": ["opencode/glm-5", "opencode/kimi-k2.5", "opencode/minimax-m2.5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

# Structural Improvements

## Support for Agent Alias

In frontmatter, I want support for agent alias in a new section name `alias`. It would look like this:
```
---
... other fields ...
flow: |
    "backend-go-developer-lite" -> "go-readability-reviewer"
alias:
    "backend-go-developer-lite": "backend-go-developer"
models:
    "backend-go-developer-lite": "anthropic/claude-haiku-4-5"
... other fields ...
---
```

The idea is that I can create agent alias that reuse whatever agent that's already defined with a custom model configuration. When I do that, for all purposes, they look like agents like any other native non-aliased agent.

- [x] implement support for agent alias
    - [x] update README.md

## UI Improvements
- [x] any button clicked should trigger a state reload so to make the UI snappy
    - [x] use debouncing logic so that the last-call always wins.
- [x] delete buttons should only be a simple confirmation dialog
- [x] editors modifying or creating GOAL.md, the autocomplete dictionary must only be agent names and model names
    - [x] for all other editors, autocomplete must be disabled.



**CRITICAL** scan previous sessions in `.sgai/retrospective` to figure out what happened and where you currently are.

## Structural Regressions
- [x] it seems retrospective agent is not triggering anymore in interactive mode (`Start` button)
- [x] the tree on the left is not correctly showing the repositories
    - [x] forks are showned at the top but not in the bottom part of the tree
    - [x] it seems certain forks are being displayed more than once (refer to http://127.0.0.1:8080/)
    - [x] it seems there are some dead routes like /forks/new -- find all dead routes and remove them both from frontend (react) and backend (go)
    - [x] when I try to delete a fork that shows in the pinned repositories block, it errors out with the message saying: "workspave not a root"
NOTE: all these bugs seems to have been introduced in these changes:
```
◆  onrnoown/0 ulderico 2026-03-05 10:29:23 10b6f20d (divergent) cmd/sgai: simplify code by reducing unnecessary lines across Go and React/TS
◆  ppzpoxzz/0 ulderico 2026-03-05 10:29:23 f9f1afc3 (divergent) cmd/sgai: simplify code by reducing unnecessary lines across Go and React/TS
```
- [x] I am seeing duplicated repositories (like sgai root - refer to http://127.0.0.1:8080/workspaces/sgai/forks on the left, it shows twice on the bottom)
- [x] the root repository name in the pinned repository block on the left has the wrong name
- [x] I see pinned repositories that aren't showing in the left bar on the left (and they should)
- [x] in Edit GOAL page `/goal/edit` - make sure you add the description (with the folder name in the tool tip) at the top. Use CSS ellipsis to handle displays that are too narrow.
- [x] I see forks not nested under their respective roots
    - [x] For example, http://192.168.0.65:8080/workspaces/soft-teal-1a80/progress should be nested under http://192.168.0.65:8080/workspaces/sgai/forks
    - [x] For example, http://192.168.0.65:8080/workspaces/true-mint-8e79/progress should be nested under http://192.168.0.65:8080/workspaces/sgai/forks
- [x] I have to hit the `Stop` button multiple times before a workspace stops
- [x] In http://192.168.0.65:8080/ I see a workspace named '.' -- it shouldn't exist or be there
- [x] I am unable to fork externally attached repositories - I should be able to
    - [x] in terms of filesystem location, the fork must ALWAYS be a sibling of the root repository, for example, if the root repo is `/full/path/to/root/repo` then the fork must be in `/full/path/to/root/fork`

**CRITICAL** never kill `sgai-base` (or the tmux session running `sgai-base`)

## Improving Tests

There are hundreds both of Go and React tests. The truth is that these tests didn't do anything useful to prevent bugs.

In Order.
- [x] 1. Remove All Go And React tests
- [x] 2. Add New Tests only as much as necessary to prove the application is free of bugs
    - [x] 2.1 Aim at a code coverage of 80%+ for Go
        - [x] *CRITICAL* 2.1.1 there are tests that need call the program `code` (or something similar) in the shell (`exec.Command` and `exec.Command`), use a mock call instead of actually shelling it out; currently, whenever the test suite runs, it makes the computer unusable because it keeps opening vscode windows over and over.
              for example: it opening vscode (aka `code`) windows named `test-ws` and `editor-ws` - they shouldn't be opening and they are! or for example: /var/folders/9_/xr1r7kx92z1_bp3z7r6n6qjw0000gn/T/TestMCPToolOpenEditorGoalExists2274727513/001/editgoal-mcp/GOAL.md and /var/folders/9_/xr1r7kx92z1_bp3z7r6n6qjw0000gn/T/TestMCPToolOpenEditorPMExists3900521434/001/editpm-mcp/.sgai/PROJECT_MANAGEMENT.md
    - [x] 2.2 Aim at a code coverage of 80%+ for React

*HUMAN PARTNER NOTE*: file names like `coverage_boost2` (like coverage boost?) it's in very bad taste. All the tests must be grouped in files that correlate with the original source code files they are testing. You spread too many tests into too many files, make sure the test files are, as much as possible, one to one with the source code.

- [x] 3. Reorganize and consolidate test files
    - [x] 3.1 Reorganize and consolidate test files for Go
    - [x] 3.2 Reorganize and consolidate test files for React

## Post-refactor regressions

- [x] Pinned repositories not being displayed
- [x] when deleting a fork, it must redirect / navigate to the Root Repository page
- [x] forks of externally attached repositories must be placed as siblings
    - [x] in terms of filesystem location, the fork must ALWAYS be a sibling of the root repository, for example, if the root repo is `/full/path/to/root/repo` then the fork must be in `/full/path/to/root/fork`
- [x] fix the build https://github.com/sandgardenhq/sgai/actions/runs/22784701250/job/66098783469?pr=361 - use GH
