---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "stpa-analyst"
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

I need an action bar inside Internals tab.

What's an action bar?

An action bar is a set of prompts+model+variant+name buttons that when I click, they are executed for me.

This should be part of the configuration that would go into sgai.json

Basically:
```
{
...
"actions": [
    {
        "name": "Create PR",
        "model": "anthropic/claude-opus-4-6 (max)",
        "prompt": "using GH make a prompt"
    }
]
...
}
```
(note how it uses the same variant parsing logic of the attribute `models` in the frontmatter)

- [x] Server side: Add the support for the Action Bar mechanic based on Run tab (adhoc runs)
- [x] Add the button bar in the UI: Action Bar, inside Internal Tab, on the very top of the tab.
- [x] The buttons can live freely on top, no need to encapsulate them in a box.
- [x] variant handling, for example `"model": "anthropic/claude-opus-4-6 (max)",` becomes model claude-opus-4-6, variant max (the same logic that models attribute use to parse model names)
- [x] print the CLI command you're going to run in the sgai stdout/stderr
- [x] it must survive me to browse away and back, like Run tab does (maybe it's already done)
- [x] I wonder if there is some kind of global lock blocking the whole clockwork? sometimes I browse and the browser doesn't load the page.
