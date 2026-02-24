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
  "general-purpose": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

# Retrospective Issues

- [x] `.sgai/stderr` and `.sgai/stdout` are being mixed up (note that this is the retrospective of `sgai-mcp-server`, but, it has log lines of other workspaces)
- [x] the messages tab should display all messages
  - [x] unread messages in the messages tab list should be bold/strong to indicate they are unread
- [x] consider 2026-02-23-22-36.io13.zip -- for some reason, it reached the retrospective agent, but it never circled back to the coordinator
  - [x] Explain the problem here:
```
ROOT CAUSE ANALYSIS:

When the retrospective agent called sgai_update_workflow_state({status: "complete"}),
the workflow terminated instead of returning control to the coordinator. The workflow
loop in runWorkflow (main.go) checks wfState.Status == state.StatusComplete after each
agent run and terminates if no pending messages exist. There was no enforcement
preventing non-coordinator agents from setting StatusComplete. The retrospective agent,
believing its work was done, set StatusComplete — which the MCP server accepted without
restriction. The workflow loop detected this, found no pending messages, and terminated
the entire workflow.

STPA INSIGHT:

This is a "wrong control action applied" scenario — the retrospective agent applied a
control action (StatusComplete) that is only valid when issued by the coordinator
controller. The system had no enforcement (process model) to prevent this unauthorized
control action from causing the loss (workflow termination without coordinator approval).

FIX:

In runMultiModelAgent (main.go), a guard was added: when a non-coordinator agent sets
StatusComplete, it is downgraded to StatusAgentDone with a warning log. Only the
coordinator can set StatusComplete to terminate the workflow.
```
  - [x] Fix the issue

# Continuous Mode issues

Consider this log line:
`[00:00:00.491][merge-dependabot-prs][continuous-mode        :0001] !  agent "continuous-mode" not found. Falling back to default agent`

- [x] what's this `continuous-mode` agent? I don't think I ever spec'd it.


# Structural Refactor

Right now, we have a rat nest between Self-Drive mode and Interact (aka Start) mode.

I want you to refactor the application into three distinct source logical branches:

1. Continuous Mode branch (aka `Continuous Self-Drive` button) - in which the continuousMode attribute is used; in this mode, the user never interacts with the application directly; In this Continuous Mode, retrospectives are NEVER run (and if the user sets `retrospective: true` in the frontmatter, it must error out)

2. Interactive branch (aka `Start` button); in this mode, the user is interviewed until the work-gate is cleared. When crossing the work-gate, the tools to talk to the user are disable (ask_user_question and siblings), then the application run autonmously until the retrospective phase. In the retrospective phase, the clockwork re-enables `ask_user_question` and guide the user through the retrospective questions. **CRITICAL: the user may decide to skip retrospective by adding `retrospective: false` in the frontmatter**

3. Self-Drive branch (aka `Self-Drive` button); it works like the interactive branch but all user interaction tools are disabled, and the LLM running the workflow should figure out by itself what to do.

- [x] Code `Continuous Mode` logical branch
- [x] Code `Interactive` logical branch
- [x] Code `Self-Drive` logical branch
- [x] Code dispatcher that clearly activates one branch over another
- [x] Refactor prompts and continuation messages such that they can be rendered for each of the logical branches above
