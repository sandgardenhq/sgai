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
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6","anthropic/claude-sonnet-4-6","anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

# Improve UX for SGAI

OK - I am using sgai very extensively, and I think I like what Codex App does (refer to https://github.com/openai/codex)

We are going to make a set of changes to experiment with what.

- [x] in the Root Repository in Forked Mode, there will be a rich editor the can be used for the user to write a GOAL.md
  - [x] on save, it create a new fork with that GOAL.md, and the folder name should be random name with this form: `<adjective>-<color>-<random 4 numbers + vowels >`
- [x] on the left, instead of showing the directory+description, you are going to show the description only
  - [x] the description has to be the first phrase of the GOAL.md, stripped into plaintext, up to 255 (if 256 or more, then add `...` at the end)
- [x] in the left, when I select a repository, make sure that you show it highlighted
- [x] on the left, forked repositories must be nested under the Root Repository
  - [x] the name of the Root Repository entry must be the name of the directory of the root repository (instead of the first 255 chars of GOAL.md)
  - [x] the nesting must be slightly indented to the right

- [x] rebase against `main@origin` (make sure you fetch first)

- [x] in `/workspaces/$workspace/forks`, the text area must be either be prefilled with the previous fork creation, OR, in at least the embedded GOAL.example.md (the one stored in the binary) as a starting point.

- [x] the rich text editor autocomplete forces the display of a scrollbar in the container. Do we need a container for the Rich Editor? I don't think so, prove that it can work without the container in all cases and remove them as much as possible.
  - [x] Also, make the autocomplete help with feeling up the right agent names in frontmatter `flow`, and the model names in frontmatter `models`

- [x] In Root Repository in Forked Mode, in the table that you list the forks, the title should be the same rule used in the tree on the left, with the name of the folder in the mouse over

- [x] in the Forked Repository, the title should be the same left tree summary you use

- [x] drop the summarization of repositories
  - [x] remove from the left tree
  - [x] remove from the workspace page
  - [x] the server side implementation
    - [x] including prompts and customized calls and debouncers

- [x] the left tree must be collapsible (so that the main workspace area grows larger)
  - [x] the left tree must be resizeable
 
- [x] observe http://192.168.0.65:8080/ - replicate the state for dynamolock, and fix why description is empty.

- [x] the rich editor in Edit GOAL, the autocomplete is hiding under the bottom of the container, and therefore I don't see all options. 

- [x] remove the subtitles in Edit GOAL page

- [x] Remove the "GOAL.md content" container title FROM Edit GOAL page 

- [x] The pinned repositories are lost on restart
  - [x] Is it reading from XDG env vars?

- [x] In Edit GOAL page, the save flash indicator (`Saved! Redirecting...`) looks ugly. I suggest update to make the `Save GOAL.md` button to change to "Saving..." instead of adding this extra visual component.