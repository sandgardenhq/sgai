# Weekly maintenance and reliability updates

Hi folks! Quick update: here’s what we shipped this week.

- **Date**: 2026-03-11
- **Version**: 0.0.0+20260311
- **Summary**: This release includes more rock-solid multi-repo workspace handling, improved testing reliability, and sharper reviewer-agent behavior.

## 🚀 New Features

This week’s feature work makes day-to-day workflows smoother and reduces friction when working across different repos. Better yet, you can reference reviewer agents more flexibly, and workspace operations are more resilient when forks or external repositories are involved.

- **Rolled out agent aliases for reviewer selection** - You can now use agent aliases to refer to reviewer agents more flexibly across workflows.
- **Beefed up workspace support for forks/external repos** - Workspace operations sometimes treated the *same* directory as two different workspaces when the input path came in different forms (for example, a symlinked path vs. the symlink-resolved path). In practice, the mismatch came from comparing and recording *raw* paths.

  That caused a few concrete problems:
  - Forks could be mis-classified as being “inside” or “outside” the workspace when the fork dir and the workspace dir were logically the same location but had different path strings.
  - Delete-fork requests could behave differently depending on whether the request originated from a root-workspace path or a fork-workspace path.
  - Forks created from an attached external workspace could be recorded inconsistently when the external directory contained symlinks.

  The resolution is to canonicalize paths before using them for decisions:
  - Root/workspace comparisons use symlink-resolved paths (for example, comparing the symlink-resolved root workspace path to the symlink-resolved workspace path).
  - External workspace tracking records the *resolved* fork path as external (and persists that external directory set), so external forks stay consistently tracked even when symlinks are involved.

## 🚧 Bug Fixes

No customer-facing bug fixes were shipped this week.

## 🛠 Internal Updates

We streamlined how tests run, tightened reviewer-agent prompts, and consolidated repeated helpers to keep the codebase easier to evolve. On top of that, we refined shared service logic and did a bit of repository housekeeping to reduce noise.

- **Streamlined `make test` to include the web app** - Running `make test` now ensures the web app is built and tested as part of the same command.
- **Refined reviewer agents to be stricter and read-only** - Reviewer agents now provide stricter, read-only feedback that is treated as blocking by default.
- **Streamlined helpers and test utilities** - The codebase is easier to maintain due to consolidation of duplicated helpers and test utilities.
- **Refined shared service logic for consistency** - Core services and persistence flows are more consistent by sharing common logic instead of duplicating it.
- **Tackled repo housekeeping and planning updates** - Repository housekeeping and planning artifacts have been updated to reduce noise and track goals.

---

Written by [doc.holiday](https://doc.holiday)