---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
  "project-critic-council": ["opencode/kimi-k2.5-free", "opencode/minimax-m2.1-free", "opencode/glm-4.7-free"]
interactive: yes
completionGateScript: make test
mode: graph
---

- [x] without using external dependencies, strip all ANSI codes that make this look wrong:
```
[94m[1m| [0m[90m webfetch  [0mhttps://example.com (text/html)
```
(note all the escape codes, strip them)
(only change serve_adhoc.go - leave the rest alone)

---

- [x] using simplification cascades and similar skills, simplify the changes you made (`jj diff --git -r vqxyxxyp`)
  - [x] KEEP the intended behavior, but use the most minimal implementation to achieve the current behavior
  - [x] https://cdn.jsdelivr.net/npm/idiomorph@0.3.0/dist/idiomorph.min.js vs https://cdn.jsdelivr.net/npm/idiomorph@0.3.0/dist/idiomorph-ext.min.js why did you add idiomorph.min.js?

---

BUGS
- [x] the models parser is showing options named "free" without a model name (possibly it is misreading the output of `opencode models`)
- [x] the internal tabs reloads, and the reload makes it impossible for me to type the message in the Run box because the content gets reset.
  - [x] possibly you need to use idiomorph here
        use playwright to see how (try alternating to the Internal tab, scrolling down to the Run box, and typing something, observe how it still disappears)
  - [x] still doesn't work, when navigating to Internals tab, the auto-refresh doesn't let me choose the model
        once the value is set, it remains; but the auto-refresh makes the select box close
        the right solution is to preserve the WHOLE Run box
    - [x] simplify the changes you made to the minimum necessary to be successful
      - [x] validate with playwright
      - [x] try running a prompt with opencode/big-pickle to prove that the output works and survives the automatic refreshes
  - [x] state not being preserved when I switch between tabs
  - [x] you have to mesh together the output from stderr and stdout
        (the equivalent of `opencode run -m 'user/chosen-model' 'command' 2>&1)

**CRITICAL** use tmux and playwright to validate the problem and the solution
**CRITICAL** use port 10091 for testing purposes

---

- [x] inside the internals tab, below the sequence box, I want a one-shot prompting interface
  - [x] this feature must be hidden behind either a CLI flag or a sgai.json flag
    - [x] CLI: `--enable-adhoc-prompt`
    - [x] `sgai.json`: `"enable-adhoc-prompt":true"`
  - [x] The box title is "Run""
  - [x] Simple one-line input text box
  - [x] Simple model chooser loading from `opencode models`
  - [x] One submit button
  - [x] When SUBMIT, print the output stream into the output box
  - [x] SUBMIT MUST BE DISABLED UNTIL THE CURRENT SUBMISSION IS COMPLETE <- CRITICAL
    - [x] In other words, prevent duplicated submission
  - [x] Page reloads must work
  - [x] SHELL COMMAND WILL BE `echo $USERINPUT | opencode run -m 'chosen model'`
    - [x] use Go's exec type properly with io.Reader and io.Writer
  - [x] the output must be safely translated into HTML
    - [x] fixed width output fonts, CSS styles, and configurations

