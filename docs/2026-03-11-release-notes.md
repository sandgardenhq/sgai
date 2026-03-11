# 2026-03-11 0.0.0+20260311 — Weekly maintenance, usability, and reliability updates

Hi folks! Here’s what shipped this week.

- **Date**: 2026-03-11
- **Version**: 0.0.0+20260311
- **Summary**: Changes included in this release are listed below.

## Raw release notes data

```json
{
  "🚀 New Features": [
    "You can now use agent aliases to refer to reviewer agents more flexibly across workflows. This adds alias resolution in the agent selector and updates the continuous/retrospective workflow, UI, and docs to recognize alternate agent names.",
    "The system now handles forks and external repositories more safely during workspace operations. This hardens workspace/fork flows (including rollback) and improves compose wizard storage, error handling, and dirty-state logic for more reliable multi-repo work."
  ],
  "🛠 Internal Updates": [
    "Running `make test` now ensures the web app is built and tested as part of the same command. The `test` Makefile target now depends on both `webapp-build` and `webapp-test` to prevent false greens from stale builds.",
    "Reviewer agents now provide stricter, read-only feedback that is treated as blocking by default. The reviewer prompts were tightened to deny bash execution, remove softened language, and update workflows/skills/examples across the Go, React, HTMX/PicoCSS, and shell-script reviewers.",
    "The codebase is easier to maintain due to consolidation of duplicated helpers and test utilities. This removes unused helpers/components, centralizes server test helpers, consolidates React test utilities (fetch, Markdown editor, workspace fixtures), and updates call sites/tests to the simplified APIs while keeping behavior unchanged.",
    "Core services and persistence flows are more consistent by sharing common logic instead of duplicating it. This refactors message deletion into a shared service and improves compose-wizard persistence and error paths, including hardened storage handling and dirty-state tracking.",
    "Repository housekeeping and planning artifacts have been updated to reduce noise and track goals. This adds a global `.DS_Store` ignore pattern and introduces a new GOALS entry for issue 357."
  ]
}

```

---

Written by [doc.holiday](https://doc.holiday)