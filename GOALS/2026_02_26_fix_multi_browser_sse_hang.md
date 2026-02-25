---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "go-readability-reviewer"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6 (max)"
  "project-critic-council": ["anthropic/claude-opus-4-6 (max)"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---


start the sgai in a test port, but serving from `/go/src/github.com/sandgardenhq` - let's make some changes to the diff tab

- [x] instead of showing the diff, show the diff file stat; which should be quicker to load.
  - [x] add a `View Full Diff` button
    - [x] when user clicks the button, opens a new page - in the new page, the whole diff is loaded

- [x] for example, when I click `Start` nothing happens, I have to do a hard refresh to see that a workspace has been started.

- [x] now I see duplicated entries in the pinned workspace block on the top left.

---

somehow, you made everything worse -- start the sgai in a test port, but serving from `/go/src/github.com/sandgardenhq`

- [x] Browse using playwright to see how bad it is
- [x] the state endpoint takes forever to start, and it seems to be returning something other than the JSON of the state necessary for the frontend application to work.

---

- [x] the implementation below is severely incomplete, I still see endpoints like:
      `/api/v1/workspaces/concurrency-issues/todos`
      ALL the state load must be done from the single `/api/v1/state` endpoint

- [x] Be more exhaustive with your tests

---

- [x] It seems the problem has been improved, but still happens.
  - [x] Rethink the architecture:
    - [x] Create a single `/api/v1/state` endpoint that returns the ENTIRE factory state needed to render any page. Use singleflight behind the scenes to debounce concurrent calls (N tabs hitting simultaneously = 1 computation).
    - [x] Keep a lightweight SSE (or WebSocket) endpoint whose ONLY job is to send "reload" signals — no data payload, just a notification that state has changed so clients should re-fetch `/api/v1/state`.
    - [x] Remove the current per-event-type SSE broker infrastructure (sseBroker, per-workspace brokers, multiplexed streams).
    - [x] Update the React frontend to use the new `/api/v1/state` endpoint as the single source of truth. Two-layer update strategy: (1) Baseline: poll `/api/v1/state` every ~3s (always-on, short-lived requests, unlimited tabs). (2) Accelerator: try SSE `/api/v1/signal` with 5s timeout — if connected, triggers instant re-fetch; if timeout, rely on polling only. Page Visibility API to pause/slow hidden tabs.
  - [x] Acceptance criteria:
    - [x] I am able to open as many tabs as I want
    - [x] The state is always fresh and up-to-date
    - [x] It is snappy and fast

---

- [x] When I open sgai in many browsers, either all of them hang or few take a very very long time to work.
  - [x] use all analytical skills and agents to figure out why
    - [x] make sure you use Playwright and Chrome's console to investigate what is blocking what.
  - [x] fix the problem
