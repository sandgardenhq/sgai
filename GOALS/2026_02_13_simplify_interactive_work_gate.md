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
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6 (max)"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6 (max)"
  "stpa-analyst": "anthropic/claude-opus-4-6 (max)"
  "project-critic-council": ["anthropic/claude-opus-4-6", "anthropic/claude-opus-4-6", "anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6"
interactive: yes
completionGateScript: make test
---


# BUG

- [ ] sgai doesn't start ```
Mac:sandgardenhq ucirello$ ./sgai/bin/sgai-base serve --listen-addr 0.0.0.0:8080
2026/02/13 21:18:58 sgai serve listening on http://0.0.0.0:8080
[auto-mode-after-work-gate] flag provided but not defined: -interactive
[auto-mode-after-work-gate] Usage of sgai:
[auto-mode-after-work-gate]   -fresh
[auto-mode-after-work-gate]     	force a fresh start (delete state.json and PROJECT_MANAGEMENT.md)
```
---

# Human Critique

The code is getting too complex, and you solved this GOAL by adding code instead of deleting code.

You have to apply a simplification pass now.

- [x] Remove unnecessary abstraction
- [x] Reduce the combinatorial explosion of variables and the functions that evaluate them
- [x] Achieve the same behavior with a simpler and cleaner design
- [x] Achieve the same behavior with a simpler and cleaner implementation
- [x] Refactor the code so to pretend that the `--interactive` CLI flag never existed.

The idea is: DELETE and assess.

---

# Previous Implementations

We have a problem right now, when we clear the work-gate (`sgai_ask_user_work_gate`) -- the change to FSD (`interactive: auto`) doesn't seem to be working correctly.

- [x] Drop the support for "interactive:" in the frontmatter
- [x] Drop the support for "--interactive" in the CLI
- [x] Drop support for the internal state of `interactive: no/false` -- the only acceptable values are `interactive: true/yes` and `interactive: auto`
- [x] Fix the internal clockwork so pre-gate default behaves as `interactive: yes`, and once I clear the work-gate (`sgai_ask_user_work_gate`) the transition to `interactive: auto` becomes permanent until workflow end; in post-gate mode both `Start` and `Self-drive` must work as equivalent triggers, clicking one disables the other for the current session, and both re-enable only in the next workflow session/run.

## Past Decisions (2026-02-12)

- `interactive: no/false` is invalid and must fail fast with clear migration guidance.
- Any usage of removed frontmatter `interactive:` or CLI `--interactive` must fail fast with clear migration guidance.
- Validation requires parser/config tests, state-transition tests, Playwright UI tests, and passing project gate tests (`make test`).
- Completion evidence must include test outputs, Playwright screenshots, and per-checkbox mapping to code/tests.

## Brainstorming Clarifications (2026-02-13, revised)

- **Start** = interactive interview → work-gate approval → transitions to self-drive. **Self-Drive** = immediate self-drive, no interview.
- Buttons have DISTINCT purposes, NOT equivalent triggers. The `triggerOwner` mechanism is unnecessary — DELETE it entirely.
- "Locking" is simply: if session is running, both buttons are disabled (natural behavior).
- Source of truth consolidated: `InteractiveAutoLock` (persistent, set on work-gate clear) + `session.interactiveAuto` (simple boolean at creation: false for Start, true for Self-Drive).
- DELETE: `triggerOwner` field/constants/helpers, `effectiveInteractiveAuto`, `normalizeTriggerOwner`, duplicated lock-state blocks, `TriggerOwner` from API responses, frontend `actionInFlightRef`/lock variables/`isCriticalActionPending`, all triggerOwner-related tests.
- Completion requires behavior parity and NET code deletion (~400+ lines). Zero new abstractions.
- Mandatory validation: `make test`, `make lint`, `bun test`, `bun run build`, Playwright screenshots, `jj diff --stat` shows net deletion.
