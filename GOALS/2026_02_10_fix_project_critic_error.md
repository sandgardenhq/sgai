---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "react-developer": "openai/gpt-5.2-codex (xhigh)"
  "react-reviewer": "openai/gpt-5.2-codex (xhigh)"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
interactive: yes
completionGateScript: make test
---



observe this log:
```
[dasdas] [sgai                  ] agent project-critic-council still working, re-running...
[dasdas] [project-critic-council:1302] 1095 |     const info = provider.models[modelID]
[dasdas] [project-critic-council:1302] 1096 |     if (!info) {
[dasdas] [project-critic-council:1302] 1097 |       const availableModels = Object.keys(provider.models)
[dasdas] [project-critic-council:1302] 1098 |       const matches = fuzzysort.go(modelID, availableModels, { limit: 3, threshold: -10000 })
[dasdas] [project-critic-council:1302] 1099 |       const suggestions = matches.map((m) => m.target)
[dasdas] [project-critic-council:1302] 1100 |       throw new ModelNotFoundError({ providerID, modelID, suggestions })
[dasdas] [project-critic-council:1302]                    ^
[dasdas] [project-critic-council:1302] ProviderModelNotFoundError: ProviderModelNotFoundError
[dasdas] [project-critic-council:1302]  data: {
[dasdas] [project-critic-council:1302]   providerID: "anthropic",
[dasdas] [project-critic-council:1302]   modelID: "claude-opus-4-6 (max)",
[dasdas] [project-critic-council:1302]   suggestions: [],
[dasdas] [project-critic-council:1302] },
[dasdas] [project-critic-council:1302]
[dasdas] [project-critic-council:1302]       at getModel (src/provider/provider.ts:1100:13)
[dasdas] [project-critic-council:1302]
[dasdas] [project-critic-council:1302] [error]
````

observe this GOAL definition:
```
---
flow: |
  "shell-script-coder" -> "shell-script-reviewer"
models:
  "coordinator": "anthropic/claude-opus-4-6"
  "shell-script-coder": "anthropic/claude-opus-4-6"
  "shell-script-reviewer": "anthropic/claude-opus-4-6"
interactive: yes
---

- [] make me a simple address book: save and list
```
(observe that project-critic-council is absent)


- [x] fix the injection of implicit project-critic-council
  - [x] when implicit, it must use the same model and the same variation from coordinator
  - [x] the injector is not handling that correctly
