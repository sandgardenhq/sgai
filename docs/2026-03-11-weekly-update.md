# 2026-03-11 Weekly Update: Multi-workspace reliability improvements

This week’s work focuses on making multi-workspace workflows more predictable and easier to operate. Highlights include agent aliases (so a workflow can reuse an agent prompt under a different model setting) and more robust workspace handling when forks live outside the main workspace root.

## 🚀 New Features

Two themes stand out in this week’s feature work: more flexible agent configuration, and better handling of workspaces that don’t live exactly where the factory expects (for example, forks created into external directories).

* **Agent aliases** - Workflows can define `alias:` mappings in `GOAL.md` frontmatter so a workflow can refer to an agent by an alternate name. Alias resolution is used when SGAI reads an agent prompt from disk and when it parses snippets for the current agent.
* **Beefed up workspace support for forks/external repos** - Workspace and fork operations normalize paths by resolving symlinks before comparing. Fork deletion also treats both “root workspace” and “fork workspace” inputs as valid, and uses the symlink-resolved root path as the canonical directory for root-level operations.

## 🧯 Bug Fixes

This week includes regression fixes and hardening around workflows and workspace handling.

* **Fix regressions from #356** - Workflow handling is hardened across continuous/retrospective mode and workspace/fork operations.

## 🛠️ Internal Updates

Maintenance work this week tightens reviewer standards, simplifies code, consolidates tests, and smooths out developer workflows.

* **Stricter reviewer agents** - Reviewer prompts include a mandatory “blocking issues only” contract, are read-only, and deny bash.
* **Code simplification and consolidation** - Overlapping helpers are consolidated and call sites are updated to use simplified APIs.
* **Consolidated test suite** - Fragmented tests are replaced with a consolidated Go/React test suite.
* **Centralized server test helpers** - Server test helpers are consolidated to reduce duplication across the test suite.
* **Consolidated React test utilities** - React test helpers are consolidated for fetch, Markdown editor, and workspace fixtures.
* **Compose wizard hardening** - Compose wizard storage and error handling are made more robust, including dirty-state logic.
* **Shared service for message deletion** - Message deletion is refactored into a shared service.
* **`make test` includes webapp checks** - The `test` Makefile target depends on both `webapp-test` and `webapp-build`.
* **Repository housekeeping** - `.gitignore` ignores `cover*.out`.

---
Written by [doc.holiday](https://doc.holiday)
