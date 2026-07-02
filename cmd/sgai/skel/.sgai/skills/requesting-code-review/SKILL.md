---
name: requesting-code-review
description: Use when completing tasks, implementing major features, or before merging to verify work meets requirements - dispatches bounded reviewer subagents with TASK_PACKET and REVIEW_STATUS envelopes before proceeding
---

# Requesting Code Review

Dispatch the appropriate reviewer subagent with a bounded `TASK_PACKET` to catch issues before they cascade.

## Core Principle

Review early, review often, and make every reviewer output actionable through `REVIEW_STATUS`.

## When to Request Review

Mandatory:
- After completing a major feature
- Before marking a GOAL item complete
- Before merge or release
- After resolving non-trivial reviewer findings

Optional but valuable:
- When stuck and needing a fresh read-only perspective
- Before refactoring to establish baseline concerns
- After fixing a complex bug

## How to Request

1. Identify the exact review scope:
- Changed paths or `jj diff` scope
- Requirement or GOAL item being verified
- Developer `STATUS` envelope evidence
- Verification commands already run
- Screenshots, deployment URLs, or artifacts when relevant

2. Dispatch the domain reviewer subagent:
- `go-reviewer` for Go code
- `react-reviewer` for React/TypeScript UI code
- `htmx-picocss-reviewer` for HTMX/PicoCSS UI
- `shell-script-reviewer` for shell scripts
- `webmaster-reviewer` for website/UI/content/accessibility/SEO readiness

3. Use a bounded packet:

Load and follow `subagent-delegation-protocol` skill, then send the reviewer its Task Packet. Set expected edit boundaries to read-only review and set the return envelope to the Review Envelope from `subagent-return-protocol` skill.

4. Act on feedback:
- `PASS`: reviewer found no required fixes in scope
- `NEEDS_WORK`: route findings back to the developer; every finding is blocking
- `NEEDS_CONTEXT`: provide only the missing path, evidence item, decision, or question requested
- `BLOCKED`: record the blocker and route it to the owner

## Example

```markdown
[Developer returned STATUS: DONE for conversation index verification]

[Dispatch go-reviewer]
Use `subagent-delegation-protocol` skill to send a Task Packet for `internal/index/verify.go` and `internal/index/verify_test.go`, with read-only edit boundaries, `go test ./internal/index` as evidence, and the Review Envelope as the return contract.

[Reviewer returns]
Load and follow `subagent-return-protocol` skill, then return its Review Envelope exactly. Example finding: `internal/index/verify.go:82 - add progress indication for long repair runs`.
```

## Integration With Workflows

Executing plans:
- Review after each independent work unit or batch of non-overlapping units
- Loop `NEEDS_WORK` findings back to the developer
- Re-review until `PASS` or an explicit blocker remains

Ad-hoc development:
- Review before merge or completion
- Review when stuck

## Red Flags

Never:
- Skip review because "it's simple"
- Ignore `NEEDS_WORK`
- Proceed with unresolved reviewer findings
- Ask a reviewer to rediscover the whole repository when paths are known
- Send review requests without bounded paths or evidence

If reviewer is wrong:
- Push back with technical reasoning
- Show code/tests that prove it works
- Request clarification with a focused packet

## Reviewer Output

Reviewers must return this envelope exactly:

Load and follow `subagent-return-protocol` skill, then return its Review Envelope exactly.
