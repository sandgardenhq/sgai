This is the overlay directory - agents, skills, and snippets you add here, will be copied into .sgai working directory on start.

The files here can also be downloaded and fine-tuned to your specific case.

## Internal Completion Review Subagents

The completion review architecture exposes one coordinator-invoked wrapper subagent:

- The visible wrapper is the `mode: subagent` entrypoint. It acts as the FrontMan/orchestrator/aggregator and may Task-invoke only the internal critic role agents plus reviewer agents ending in `-reviewer` when it needs specific domain opinions.

The internal role agents are hidden subagents:

- Sibling Evaluator: independent strict completion assessment.
- MinorityReport: adversarial dissent focused on evidence gaps and overlooked risks.

OpenCode `mode: subagent` behavior matters for visibility. Non-hidden subagents can be manually invoked by users with `@` mention. Setting `hidden: true` hides a subagent from `@` autocomplete while preserving Task invocation by agents whose `permission.task` allows it. Therefore the internal critic role agents use `mode: subagent` and `hidden: true`, while the visible wrapper remains non-hidden.

The wrapper itself is the FrontMan. Do not add a separate hidden FrontMan child agent; that would increase session transitions and undermine the token-reduction intent of this design.
