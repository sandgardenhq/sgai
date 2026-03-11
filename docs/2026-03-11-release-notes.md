# 2026-03-11 Release Notes: Maintenance, performance, and documentation quality improvements

This week’s update includes the following changes across new features, bug fixes, and internal updates.

```json
{
  "🆕 New Features": [
    "You can now configure and use agent aliases so the same automation can be referenced by multiple names in different contexts. This adds alias resolution for agents and updates related UI and documentation to recognize and display those aliases consistently."
  ],
  "🐜 Bug Fixes": [
    "Workspace and fork handling is now more reliable, especially when working with external repositories and recovery flows. This hardens the continuous/retrospective workflows and adds safer fork-workspace rollback behavior to prevent inconsistent state during workspace/fork operations.",
    "The compose wizard now behaves more predictably when saving settings or recovering from errors. This hardens wizard storage and error handling, improves dirty-state tracking, and tightens related workflows so incomplete state does not leak across sessions."
  ],
  "🛠 Internal Updates": [
    "Documentation now consistently refers to OpenCode rather than Claude, reducing confusion when following the skill guides. This updates skill documentation language and examples to use OpenCode terminology throughout.",
    "Local development defaults are now a bit cleaner and more consistent across environments. This adds a global `.DS_Store` ignore pattern and records a new GOALS entry for issue 357.",
    "The `test` build target now runs the steps it actually depends on, which reduces flaky results from missing prerequisites. This updates the Makefile so `test` depends on both `webapp-test` and `webapp-build`.",
    "Reviewer automation now enforces stricter guardrails so review feedback is consistently actionable and non-destructive. This tightens reviewer prompts to be read-only, deny bash execution, and treat findings as mandatory blocking issues, with updated workflows/skills/examples across Go, React, HTMX/PicoCSS, and shell-script reviewers.",
    "The test and helper codebase is now easier to maintain with fewer duplicated utilities and clearer ownership boundaries. This centralizes server test helpers, consolidates Go/React test suites and React test utilities (fetch, Markdown editor, workspace fixtures), and refactors message deletion into a shared service while keeping runtime behavior the same.",
    "Internal helpers and UI components have been simplified to reduce overlap and make future changes safer. This removes unused helpers/components, consolidates overlapping Go and React/TS utilities, and updates call sites and tests to use the simplified APIs without changing behavior."
  ]
}

```

---

Written by <a href="https://doc.holiday">doc.holiday</a>