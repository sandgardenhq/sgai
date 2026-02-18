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
  "coordinator": "opencode/kimi-k2.5"
  "backend-go-developer": "opencode/glm-5"
  "go-readability-reviewer": "opencode/kimi-k2.5"
  "general-purpose": "opencode/kimi-k2.5"
  "react-developer": "opencode/glm-5"
  "react-reviewer": "opencode/kimi-k2.5"
  "stpa-analyst": "opencode/kimi-k2.5"
  "project-critic-council": ["opencode/kimi-k2.5-free", "opencode/glm-5-free", "opencode/minimax-m2.5-free"]
  "skill-writer": "opencode/kimi-k2.5"
completionGateScript: make test
---

I like the Continuous Mode, and I think we can keep improving it.

I have a desiderata; each desire should be its own commit, OK?

- [x] I want a new frontmatter parameter to let the continuousMode loop itself
      `continuousModeAuto: "duration of wait done compatible with Go's time.ParseDuration"`
      Basically, when `continuousModeAuto` is set, it is going to wait for the duration set in `continuousModeAuto`, and it will let the workflow run again.

- [x] I want a new frontmatter parameter to let the continuousMode loop itself on a periodic basis
      `continuousModeCron: "crontab style to define when to run a factory or not"`
      Basically, when `continuousModeCron` is set, it is going to wait for the deadline defined in `continuousModeCron`, and it will let the workflow run again.
  - [x] use this dep https://github.com/adhocore/gronx?tab=readme-ov-file#next-tick


# Post Success Activities
- [x] general-purpose: "copy GOAL.md into GOALS/ following the instructions from README.md"
- [x] general-purpose: "using GH, make a draft PR for the commit at @ (jj) - from here"

# PR Reference
- PR #276: https://github.com/sandgardenhq/sgai/pull/276
  - Status: Draft
  - Title: cmd/sgai: add auto and cron scheduling to continuous mode
  - Commit: 65191a944e01a5e18623287e4932e0e26f3195fe
