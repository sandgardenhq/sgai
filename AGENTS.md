THE ONLY ACCEPTABLE PLACE FOR PROJECT_MANAGEMENT.md IS `.sgai/PROJECT_MANAGEMENT.md` -- never place `cmd/sgai/skel/.sgai/PROJECT_MANAGEMENT.md`.

Every time you are asked to make a source code (or prompt) modification  to `/.sgai` you have to make the modification to `sgai/` (the overlay directory) instead.

In term of Go code style, I prefer total absence of inline comments; organize functions and if blocks in a way that they have intention revealing names, and use that instead.

In term of Go code style, I prefer very private functions over public functions; private struct over public structs; local types and structs over global structs; public functions and structs must have godoc comments.

In terms of Go code style, error variable names must use the err prefix pattern: errSpecificName (e.g., errClose, errRead), not the suffix pattern (closeErr, readErr).

Always run the code reviewer when one is available.

Always make sure you use tmux and playwright to test the changes.

For tests:
use listen address `-listen-addr 127.0.0.1:0` (observe the port number and use that from this moment on)
use directory ./verification
use `make build` to generate the binary
Do not add tests whose only purpose is to assert that removed implementation details remain absent. Prefer tests for the new positive behavior and delete tests for behavior that no longer exists.

In terms of layout, UI, style, when something doesn't fit a container, use ellipsis with tooltip - refer to https://picocss.com/docs/tooltip

CRITICAL: use playwright screenshots (and the skill to operate playwright) to verify the application is working correctly.

For React/TypeScript code in cmd/sgai/webapp/, use bun for building, testing, and running scripts. Build command: `bun run build`. Dev server: `bun run dev.ts`. Tests: `bun test`.

React components must use shadcn/ui components where possible. Do not create custom implementations when a shadcn component exists. Reference: https://ui.shadcn.com/docs

React tests: bun test for unit/component tests (vitest-compatible API), Playwright for E2E tests.

Use useSyncExternalStore for external data sources (SSE store). Use useReducer+Context for app state management. Do NOT use Redux, Zustand, or other state management libraries. No optimistic updates for critical workflow actions.

When modifying cmd/sgai/webapp/, always run `bun run build && make build` to verify both the React build and Go binary compile correctly.

CRITICAL(code quality): ensure good Go code quality by calling `make lint`

You must use the skill `browser-bug-testing-workflow` - remember to use visual diffs and screenshots to evaluate the problem
You must use the skill `run-long-running-processes-in-tmux`


# Terminology

- Standalone Repository: a repository that has only _one_ `jj workspace` -- itself.
- Root Repository: a repository that has more than one `jj workspace`, and it is the root (it is the one in which `.jj/repo` is a directory and not a file)
- Forked Repository: a repository that is part of a `jj workspace, and it is not the root (it is the one in which `.jj/repo` is a text file, whose content points to the parent).

- Repository Mode: is when a repository is served by SGAI in a way that it can actually run software.
- Forked Mode: is when a root repository has at least one child, it displays the fork (dashboard-style) mode.
**CRITICAL** when a Root Repository run out of children, it must revert back from Forked Mode to Repository Mode.


# Safe Assumptions

"OpenCode" (aka `opencode`) is always installed and available.

When implementing new features that handle external input, interact with the filesystem, or manage concurrent operations, the coordinator should load and use the `stpa-overview` skill to identify unsafe control actions and loss scenarios before implementation begins. STPA is a skill workflow, not a routable `stpa-analyst` agent.

# Code Auditing Guidance

When auditing for dead routes, check both literal usage (API endpoint calls from frontend) AND semantic liveness (does the route lead to functionality that has been replaced by inline components or other mechanisms). A route that is technically reachable but leads to replaced functionality is dead.



----

# Karpathy Guidelines

Behavioral guidelines to reduce common LLM coding mistakes, derived from [Andrej Karpathy's observations](https://x.com/karpathy/status/2015883857489522876) on LLM coding pitfalls.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.
