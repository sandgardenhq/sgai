# 2026-03-11 Weekly Update: Workflow reliability improvements

This week focused on tightening up workflow behavior and making multi-workspace operations more reliable. The biggest user-visible pieces are agent aliases (so the same role can be run under different model settings) and more robust workspace handling when forks live outside the main workspace root.

## 🚀 New Features

Two themes stand out in this week’s feature work: more flexible agent configuration, and better handling of workspaces that don’t live exactly where the factory expects (like forks in external directories).

* **Agent aliases** - Workflows can define agent aliases so an alias name resolves to a base agent when the agent is invoked. This allows reusing an existing agent’s prompt/tools/snippets while using separate model configuration for the alias.
* **Workspace support for forks/external repos** - Forked workspaces are recorded as external when the fork target is in an external location, and workspace path comparisons normalize paths by resolving symlinks.

## 🧯 Bug Fixes

This week included regression fixes and hardening around continuous/retrospective workflows and workspace/fork handling.

* **Fix regressions from #356** - Continuous/retrospective workflow and workspace/fork handling (including external repos) is hardened.

## 🛠️ Internal Updates

Maintenance work this week focused on tightening reviewer standards, simplifying code, consolidating tests, and smoothing out developer workflows.

* **Stricter reviewer agents** - Reviewer agents are tightened to be read-only, deny bash, treat findings as mandatory blocking issues with no softening, and update workflows/skills/examples accordingly.
* **Code simplification** - Unused helpers/components are removed, overlapping utilities are consolidated, and call sites/tests are updated to use simplified APIs while keeping behavior the same.
* **Consolidated test suite** - Fragmented tests are replaced with a consolidated Go/React test suite.
* **`make test` runs webapp checks** - The `test` Makefile target depends on both `webapp-test` and `webapp-build`.
* **Repository housekeeping** - Global `**/.DS_Store` ignore pattern is added.

---
Written by [doc.holiday](https://doc.holiday)
