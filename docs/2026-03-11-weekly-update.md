# Weekly maintenance and reliability updates

Hi folks! Quick update: here’s what we shipped this week.

- **Date**: 2026-03-11
- **Version**: 0.0.0+20260311
- **Summary**: This release includes more rock-solid multi-repo workspace handling, improved testing reliability, and sharper reviewer-agent behavior.

## 🚀 New Features

This week’s feature work makes day-to-day workflows smoother and reduces friction when working across different repos. Better yet, you can reference reviewer agents more flexibly, and workspace operations are more resilient when forks or external repositories are involved.

- **Rolled out agent aliases for reviewer selection** - You can now use agent aliases to refer to reviewer agents more flexibly across workflows.
- **Beefed up workspace support for forks/external repos** - Some workspace operations treated the *same* workspace as different directories depending on which path form you started with (for example, a symlinked path vs. its resolved path). That led to a few concrete issues:
  - Forks could be mis-classified as being “inside” or “outside” a workspace when root/fork paths were compared without normalizing them.
  - Delete-fork requests were sensitive to whether the request started from the root workspace path or a fork workspace path.
  - Forks created from an attached external workspace weren’t handled consistently (placement/recording) when the external directory involved symlinks.

  Workspace handling now resolves symlinks before it compares or groups paths, and fork deletion first resolves the canonical root (whether the request starts from a root or a fork). For external workspaces, fork handling uses the external directory’s resolved path so fork placement and external tracking stay consistent.

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