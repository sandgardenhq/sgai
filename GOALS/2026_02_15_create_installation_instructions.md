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

- [x] create installation instructions (INSTALLATION.md) that I can tell people to simply load into a claude prompt like: `claude -p "install SGAI using the instructions from https://github.com/sandgardenhq/sgai/"`
  - [x] The README.md redirect to INSTALLATION.md
    - [x] Update README.md
  - [x] The INSTALLATION.md must
    - [x] install dependencies
    - [x] install sgai
    - [x] create a directory name "SGAI"
    - [x] start sgai in "SGAI" (make sure it survives the stop of Claude Code)
    - [x] create one demo project called TodoApp in "SGAI"
    - [x] and add a constraint on start that says "when INSTALLATIONS_NOTES.md is present, read that file to understand how to start sgai"

- [x] Proof read "INSTALLATION.md" please, I think a lot of these instructions are wrong. Make sure you are using the right sources to make them correct.
  - [x] only install deps if they don't exist first
  - [x] use `opencode` instead of `claude` in the documentation README.md / INSTALLATION.md
  - [x] scan all the skills in `cmd/sgai/skel` and see other CLI tools that need to be installed
    - [x] Update README.md
    - [x] Update INSTALLATION.md
  - [x] the cmd/sgai/webapp is not shipped with the Go code, you have to make sure bun is correctly generating the web application before building the binary.
