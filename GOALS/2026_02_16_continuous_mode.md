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

I want to add a continuous mode. 

What's the continuous mode? It is a mode that the factory is always running, basically reacting to either changes in GOAL.md or to steering messages. 

In essence, it starts once and when the coordinator mark state:complete, the factory stores the checksum of GOAL.md in memory and keeps an eye in the message bus. 

If a new human partner message comes in, or, GOAL.md checksum changes, it starts a new workflow. 

When the reason to restart the workflow is the steering message, the factory prepends the message to the start of GOAL.md, right after the frontmatter. 

When is the continuous mode enabled? When the GOAL.md attribute "continuousModePrompt" is set. 

Basically, what happens is that when in continuous mode, that the coordinator decides to mark state:complete, it will then invoke opencode (using the same environment variables as regular workflow agents) using as prompt the "continuousModePrompt", only then store the GOAL.md checksum in memory and observe the message hub. 


- [x] Interview the human partner and expand the GOAL.md with the tasks necessary to carry the goal above.

## Implementation Tasks

### Backend Go Changes (cmd/sgai/)

- [x] Add `ContinuousModePrompt` field to `GoalMetadata` struct in `main.go`
- [x] Create new file `continuous.go` with:
  - [x] `runContinuousWorkflow()` - outer loop wrapping `runWorkflow()`
  - [x] `runContinuousModePrompt()` - invokes opencode with prompt (3 retries, logs task/progress/currentAgent to state.json for UI visibility)
  - [x] `watchForTrigger()` - polls GOAL.md checksum + Human Partner messages every 2s
  - [x] `prependSteeringMessage()` - inserts steering text after GOAL.md frontmatter, marks message as read
  - [x] `hasHumanPartnerMessage()` - checks for unread messages from "Human Partner"
- [x] Modify `serve.go` `startSession()` goroutine to call `runContinuousWorkflow()` when `continuousModePrompt` is set (always auto-mode), otherwise `runWorkflow()` as before
- [x] Add `continuousMode` boolean field to workspace detail API response in `serve_api.go`

### Frontend React Changes (cmd/sgai/webapp/)

- [x] Add `continuousMode: boolean` field to `ApiWorkspaceDetailResponse` in `types/index.ts`
- [x] Update `WorkspaceDetail.tsx` button rendering:
  - When `continuousMode` is true: show only "Continuous Self-Drive" + "Stop"
  - When `continuousMode` is false: show normal "Self-Drive" | "Start" | "Stop"

### Tests

- [x] Create `continuous_test.go` with unit tests for all new functions (watchForTrigger, prependSteeringMessage, hasHumanPartnerMessage, runContinuousModePrompt observability)
- [x] Update `WorkspaceDetail.test.tsx` with continuous mode button rendering tests
- [x] `make test` passes (all existing + new tests)
- [x] `make lint` passes
- [x] `make build` succeeds

### Verification

- [x] Playwright screenshot showing "Continuous Self-Drive" button when GOAL.md has continuousModePrompt
- [x] Playwright screenshot showing normal buttons when no continuousModePrompt
- [x] Progress entries visible in UI during continuous mode prompt step between cycles

### PR

- [x] Create a draft PR in sandgardenhq/sgai


