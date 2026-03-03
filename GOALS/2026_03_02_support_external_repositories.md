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
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6","anthropic/claude-sonnet-4-6","anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

# Several SGAI Improvements

Right now, sgai will rely on the current working directory to show repositories on the left tree.

We need to expand, and allow the addition of external repositories.

You already use XDG env vars to handle configuration, so the idea is that the `[ + ]` button would allow to add repositories to the current directory OR allow to add a directory anywhere else from the computer.

- [x] Add option to `[ + ]` to alternate between adding new workspace or adding/attaching an external workspace
    - [x] the dialog must support navigating the OS and then choosing a directory (not a file)
    - [x] Upon adding an external repository/workspace, if GOAL.md is present then create .sgai and navigate to Edit GOAL
    - [x] Upon adding an external repository/workspace, if GOAL.md is absent then create .sgai and navigate to Compose GOAL


- [x] when retrospective agent runs, that it messages coordinator, sometimes, it gets stuck in status:agent-done and the clockwork doesn't return to coordinator, it seems correlated to some kind of tool timeout. Investigate previous entries in `.sgai/retrospectives` for examples of this problem.
  - [x] ensure that the model that retrospective use either comes from the frontmatter `models` or, in its absence, that it uses the same model as the coordinator

- [x] when deleting a workspace, you must
  - [x] if it is current working directory workspace, the delete button must delete the workspace from disk
  - [x] if it is an external workspace, and it is a Root Repository or a Standalone Repository, the delete button simply removes from the interface, leaving the disk untouched
  - [x] if it is an external workspace, and it is a Forked Repository, the delete button must remove from the disk

- [x] the pinned repositories are being lost on restart
      I see this in the output:
```
2026/03/03 09:36:26 pruning stale pinned path: /private/var/folders/9_/xr1r7kx92z1_bp3z7r6n6qjw0000gn/T/TestHandleAPIForkWorkspaceautoGeneratesUniqueName1971444493/002/calm-teal-02ee
2026/03/03 09:36:26 pruning stale pinned path: /private/var/folders/9_/xr1r7kx92z1_bp3z7r6n6qjw0000gn/T/TestHandleAPIForkWorkspaceautoGeneratesUniqueName1971444493/002/warm-green-o0e1
```
      I think the problem is that on shutdown, you run the prunning, and you should only really prune on the start; so to avoid the race condition on shutdown.

- [x] I still see a lot of timeouts in the log:
```
[00:07:00.356][true-onyx-56u0][coordinator            :0001] sgai_ask_user_question(questions: [map[choices:[Yes, it looks complete - just verify tests pass and mark it done No, there's additional behavior needed (I'll describe) I'm not sure - run the tests first and let me know the result] multiSelect:false question:**Phase 1: Understanding (Item 1 of 2)**
I found that `ensureImplicitRetrospectiveModel()` is already implemented in `cmd/sgai/main.go:2254` and called at line 151. It handles:
- If retrospective has no explicit model in frontmatter, copies coordinator's model
- If retrospective has an explicit model, doesn't override it
- If coordinator has no model, does nothing
- Tests exist in `main_test.go:504-601` covering all 4 scenarios
**Is the retrospective model selection sub-item already complete?** Should I just verify tests pass and mark it done, or is there additional behavior you need?] map[choices:[It's only happening in tests - the test infrastructure has a race condition It's happening in real usage too - repos I pin disappear after restarting sgai The issue IS in the test - fix the test to not prune on shutdown/cleanup Let me describe the scenario more precisely] multiSelect:false question:**Phase 1: Understanding (Item 2 of 2)**
Regarding 'pinned repositories being lost on restart':
Your GOAL.md says: "I think the problem is that on shutdown, you run the pruning, and you should only really prune on the start; so to avoid the race condition on shutdown."
From code analysis:
- `loadPinnedProjects()` (which does pruning) is called once at startup (`serve.go:593`)
- The pruning log messages you showed are from test temp dirs (`TestHandleAPIForkWorkspace`)
- I don't see pruning called on shutdown in production code
**Where exactly are you seeing the pinned repos being lost?** This will help me understand the actual bug.]]) ERROR: TimeoutError: The operation timed out.
[00:12:00.354][true-onyx-56u0][coordinator            :0001] sgai_ask_user_question(questions: [map[choices:[Yes - verify tests and mark done No - additional work needed Run tests first, then decide] multiSelect:false question:**Brainstorming Phase 1 - Item 1: Retrospective Model**
`ensureImplicitRetrospectiveModel()` already exists in main.go with 4 test scenarios. Should I just verify tests pass and mark it done?] map[choices:[Only in tests - fix test race condition In real usage too - repos disappear after restart Fix the test to not prune on shutdown/cleanup Let me describe more precisely] multiSelect:false question:**Brainstorming Phase 1 - Item 2: Pinned Repos Lost**
Your GOAL.md says pruning on shutdown causes the issue. Where are you seeing this?]]) ERROR: TimeoutError: The operation timed out.
```
  I wonder if, internally, you have some kind of queueing system that is getting clogged - and makes you wait for me to answer, but I never get the notification because the message wasn't really handled correctly inside SGAI.


- [x] Remove the ability to rename a fork or a standalone repository or a root repository
  - [x] remove UI affordances
  - [x] remove server side implementations

- [x] retrospective transitions but it seems it never starts:
```
[true-onyx-56u0][sgai                   ] agent coordinator done:
[true-onyx-56u0][sgai                   ] pending messages for retrospective - redirecting before completion
[true-onyx-56u0][sgai                   ] coordinator -> retrospective
[true-onyx-56u0][sgai                   ] transitioning to retrospective mode
```
  - [x] also it seems it is not correctly passing the models into the call.
```
24001 s001  S      0:00.00 /bin/sh /Users/ucirello/bin/opencode run --format=json --agent retrospective --title retrospective
24002 s001  S      0:01.75 /opt/homebrew/bin/opencode run --format=json --agent retrospective --title retrospective
```
(look at `.sgai/retrospectives` for signal of what you could be going wrong)

- [x] will retrospective run correctly if triggered by project-critic-council?

- [x] I saw this happening just now:
```
[00:09:00.374][true-onyx-56u0][coordinator            :0001] sgai_ask_user_question(questions: [map[choices:[Write a test that explicitly verifies PCC-to-retrospective triggering works end-to-end, then mark done Your analysis is sufficient - just mark it done I have concerns - let me describe what I'm worried about I want you to actually test it live (build + run + playwright verification)] multiSelect:false question:**Brainstorming Phase 1: Understanding the remaining GOAL item**

The one unchecked item is:
> `- [ ] will retrospective run correctly if triggered by project-critic-council?`

I traced the code flow in `cmd/sgai/main.go` and found:

1. **Message routing is trigger-agnostic** - `findFirstPendingMessageAgent()` routes to any agent with unread messages, regardless of who sent them
2. **`unlockInteractiveForRetrospective()` fires whenever `currentAgent == "retrospective"`** - it doesn't check who triggered the transition
3. **Model selection** is done at startup via `ensureImplicitRetrospectiveModel()`, so it works regardless of trigger source
4. **The RETRO_QUESTION relay** (retrospective → coordinator → human) works via the generic message bus

**My conclusion**: The code should handle this correctly already. The mechanism is designed to be trigger-agnostic.

What do you want for this item?]]) ERROR: TimeoutError: The operation timed out.
2026/03/03 12:57:40 askandwait: question state set, status changed to waiting-for-human
2026/03/03 12:57:40 askandwait: blocking for human answer
[00:14:00.369][true-onyx-56u0][coordinator            :0001] The question timed out. Let me try again with a more concise question.
[00:14:00.370][true-onyx-56u0][coordinator            :0001] sgai_ask_user_question(questions: [map[choices:[Write a test to verify PCC-to-retrospective triggering, then mark done Analysis is sufficient - just mark it done I have concerns about something specific Test it live with build+run+playwright] multiSelect:false question:Remaining GOAL item: 'will retrospective run correctly if triggered by project-critic-council?'

My code analysis shows the mechanism is trigger-agnostic and should work correctly. What should I do?]]) ERROR: TimeoutError: The operation timed out.
2026/03/03 13:02:41 askandwait: question state set, status changed to waiting-for-human
2026/03/03 13:02:41 askandwait: blocking for human answer
2026/03/03 13:04:25 askandwait: response queued for delivery
2026/03/03 13:04:25 askandwait: answer received from human
2026/03/03 13:04:25 askandwait: clearing waiting state
[00:19:00.363][true-onyx-56u0][coordinator            :0001] sgai_ask_user_question(questions: [map[choices:[Yes, write a test No, just mark it done Other] multiSelect:false question:Last GOAL item: retrospective triggered by PCC. Write a test to verify it works?]]) ERROR: TimeoutError: The operation timed out.
```
I saw the timeout, and the UI didn't show the question; if you are using channel, you probably want to use channel in a channel setup (on the tool call you send a channel that shows the question in the UI, then when the user responds, the interface feed the response in a channel inside the channel, with a buffer size of 1)

- [x] Simplify clockwork
  - [x] How can runContinuousWorkflow be simpler (fewer branches and corner cases)
  - [x] How can runWorkflow be simpler (fewer branches and corner cases)
  - [x] Why do have these extra flags handling?
        if everything runs from the main server, why the extra parameter and flag handling?
    - [x] `log.Fatalln("usage: sgai [--fresh] <target_directory>")`?
    - [x] `log.Fatalln("usage: sgai <target_directory>")`?

- [x] in the notification bar (Mac) icon, instead of use the same naming convention as in the web interface's left bar

- [x] Progress tab's button bar runs from distinct workspaces seem to be contaminating each other
      for example, right now in `http://127.0.0.1:8080/workspaces/merge-dependabot-prs/progress` if I hit `Upstream Sync`, I am able to see the output in `http://127.0.0.1:8080/workspaces/auto-flow-mode/progress`, and I shouldn't; also, it blocks me from calling other actions in button bars from other workspaces.useAdhocRun
