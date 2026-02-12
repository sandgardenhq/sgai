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
completionGateScript: make test
---

- [x] Menu bar should be the ` ⏺ $RunningFactories / $TotalFactories `
  - [x] Menu bar should be the ` ⚠ $RunningFactories / $TotalFactories `

---

- [x] accessing the web interface through 0.0.0.0 mostly hangs
  - [x] is it a goroutine pinning issue?
  - [x] is it a mutex deadlocking issue?

Definition decisions (2026-02-12) for 0.0.0.0 hang:
- Investigation approach: parallel dual-track.
  - Track A: concurrency diagnostics and root-cause tracing.
  - Track B: network-path diagnostics through `0.0.0.0` with tmux + Playwright artifacts.
- Acceptance criteria: hang is no longer reproducible via agreed repro steps and all required checks pass.
- Response-time criterion: no fixed latency threshold; stability/no hangs only.

Validation refinements (2026-02-12) for 0.0.0.0 hang:
- Required checks:
  - `make test` (completion gate), `make lint`, `make build`.
  - Targeted regression tests for the `0.0.0.0` hang path.
  - Playwright browser verification.
  - Soak/reliability run: repeated `0.0.0.0` access for 10+ minutes without hanging.
  - `go test -race` for touched Go packages.
  - Explicit goroutine leak check before/after regression sequence.
- Required evidence:
  - Command outputs for required checks.
  - Root-cause narrative mapped to hypotheses (goroutine pinning vs mutex deadlock).
  - Playwright screenshot + console + network capture.
  - tmux session logs for reproduction/verification.
  - Before/after timing or stability metrics.
  - Goroutine dump and/or mutex profile linked to former hang point.
  - File-to-hypothesis mapping table.
- Edge-case requirements:
  - Repeated refresh/polling on `0.0.0.0` remains responsive.
  - Both localhost and remote-host paths are validated.
  - Both hypotheses are explicitly ruled in/out with evidence.
  - No regressions in menu-bar behavior related to factory status text.

---

In the Mac menu bar, there is a duplication of messages: "No factories need attention"/"# factory(ies) need attention" AND "# running, # need attention". We are going to simplify this.


- [x] Drop "No factories need attention"/"# factory(ies) need attention"
  - [x] KEEP `$factory (Needs Input)` and `$factory (Stopped)`

Definition decisions (2026-02-12):
- Keep top summary behavior unchanged as `# running, # need attention`.
- Implementation approach: Approach A (minimal surgical change, lowest risk).
- Completion requires completion-gate checks to pass (`make test`).
- Evidence required: test results plus Playwright screenshot evidence; when native macOS menu-bar text cannot be captured directly, code+tests+browser artifacts are an acceptable proxy.

Validation refinements (2026-02-12):
- Acceptance criteria: `make test` passes and evidence confirms duplicated line is removed while `# running, # need attention` remains; if native menu-bar text cannot be captured directly, code+tests+browser artifacts are accepted as proxy evidence.
- Evidence depth: include Playwright screenshot plus browser console and network capture.
- Edge-case preservation checks must explicitly confirm `$factory (Needs Input)` and `$factory (Stopped)` still render correctly.
