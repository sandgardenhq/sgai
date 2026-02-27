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
  "project-critic-council": ["anthropic/claude-opus-4-6 (max)", "anthropic/claude-opus-4-5", "anthropic/claude-sonnet-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---


- [x] do we still need `sgai was interrupted while working. Reset state to start fresh.`?
      hunt it in the source code and brainstorm with me.

- [x] in `tooltip-trigger` the name of the agent and model is not showing up
      is it loading from the internal state? are you calling `state.Save()` often enough?

- [x] the dot graphviz SVG is not showing currently active agent
      is it loading from the internal state? are you calling `state.Save()` often enough?

- [x] also, all progress messages at the bottom box show coordinator EVEN WHEN they were made by another agent
      is it loading from the internal state? are you calling `state.Save()` often enough?

- [x] the messages tab don't seem to be being marked as read.
      is it loading from the internal state? are you calling `state.Save()` often enough?

- [x] make `pkg/state.Load` and `pkg/state.Save` private so that no external entity is able to use them except through the Coordinator type.

- [x] when I hit the stop button, make sure that
  - [x] the GOAL hash is updated with the latest state of the GOAL.md
  - [x] correctly flushes into .sgai/state.json before exiting

- [x] sometimes, the agent tight loops in a error, like this:
```
[00:02:05.285][concurrency-issues][project-critic-council :0002] [error]
[concurrency-issues][sgai                   ] agent project-critic-council still working, re-running...
[00:00:02.307][concurrency-issues][project-critic-council :0003] [error]
[concurrency-issues][sgai                   ] agent project-critic-council still working, re-running...
[00:00:02.291][concurrency-issues][project-critic-council :0004] [error]
[concurrency-issues][sgai                   ] agent project-critic-council still working, re-running...
```
      and when I kill opencode or something I see it's because the context window went overboard:
```
[00:01:40.161][ask-user-blocking-call][backend-go-developer   :0076] /Users/ucirello/bin/opencode: line 4: 45170 Terminated: 15          /opt/homebrew/bin/opencode "$@"

=== RAW AGENT OUTPUT (last 1000 lines) ===
{"type":"error","timestamp":1772218889093,"sessionID":"ses_35f9c32a4ffePRHLhXc1g9Fwan","error":{"name":"ContextOverflowError","data":{"message":"prompt is too long: 286648 tokens > 200000 maximum","responseBody":"{\"type\":\"error\",\"error\":{\"type\":\"invalid_request_error\",\"message\":\"prompt is too long: 286648 tokens > 200000 maximum\"},\"request_id\":\"req_011CYZ6Cw9wQzERnw8NKxynp\"}"}}}
```
      add something that if the same agent errors more than 10 times; it discards the current session_id and open a new session_id to allow it to recover.

---

Let's improve how the models communicate with humans and the enviroment

- [x] sendMessage should be getting the list of agents from the DAG and not from the state.json

- [x] make `sgai_ask_user_question` a blocking call, that is, it waits for the human partner to react before returning control to the agent

- [x] make `sgai_ask_user_work_gate` a blocking call
      that is, it waits for the human partner to react before returning control to the agent

- [x] For `sgai_ask_user_question` and `sgai_ask_user_work_gate`, keep the existing response substance exactly as it behaves today; only change timing so the call blocks until human response is received.

- [x] make `sgai_update_workflow_state({"status":"agent-done"})` to softly stop the agent
  - [x] when `sgai_update_workflow_state({"status":"agent-done"})` is called, a watchdog should wait for 1 minute before canceling the context.
  - [x] repeated calls to `sgai_update_workflow_state({"status":"agent-done"})` must be ignored

- [x] remove all file-based response transport for `sgai_ask_user_question` and `sgai_ask_user_work_gate`
      including `.sgai/response.txt` reads/writes and any on-disk answer handoff

- [x] make question/answer flow avoid disk writes entirely for this flow,
      including no `.sgai/state.json` updates as part of ask/answer handoff

- [x] preserve blocking tool-call behavior with direct answer return:
      tool call waits -> human answers -> tool call returns that answer to the model

- [x] harden work-gate approval parsing to reject `Selected:` line-injection from free-text answer content
      (only explicit selected approval choice should authorize transition)

- [x] propagate legacy API respond fallback state load/save failures in `cmd/sgai/serve_api.go`
      as explicit errors (no silent success/no-pending path)

- [x] drop `state.Load()` -- all state must be managed in memory
  - [x] conversely always keep `state.Save` because `.sgai/state.json` must be used for retrospectives
