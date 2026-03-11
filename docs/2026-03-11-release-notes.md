# 2026-03-11 Release Notes: Maintenance, performance, and documentation quality improvements

This week’s update includes the following changes across new features, bug fixes, and internal updates.

```json
{
  "🆕 New Features": [
    "You can now configure and use agent aliases so the same automation can be referenced by multiple names in different contexts."
  ],
  "🐜 Bug Fixes": [
    "Workspace and fork handling is now more reliable, especially when working with external repositories and recovery flows.",
    "The compose wizard now behaves more predictably when saving settings or recovering from errors."
  ],
  "🛠 Internal Updates": [
    "Documentation now consistently refers to OpenCode rather than Claude, reducing confusion when following the skill guides.",
    "Local development defaults are now a bit cleaner and more consistent across environments.",
    "The `test` build target now runs the steps it actually depends on, which reduces flaky results from missing prerequisites.",
    "Reviewer automation now enforces stricter guardrails so review feedback is consistently actionable and non-destructive.",
    "The test and helper codebase is now easier to maintain with fewer duplicated utilities and clearer ownership boundaries.",
    "Internal helpers and UI components have been simplified to reduce overlap and make future changes safer."
  ]
}

```

---

Written by <a href="https://doc.holiday">doc.holiday</a>