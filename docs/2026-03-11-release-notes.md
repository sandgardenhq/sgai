# 2026-03-11 0.0.0+20260311 — Weekly maintenance, usability, and reliability updates

Hi folks! Here’s what shipped this week.

- **Date**: 2026-03-11
- **Version**: 0.0.0+20260311
- **Summary**: Changes included in this release are listed below.

## Raw release notes data

```json
{
  "🚀 New Features": [
    "You can now use agent aliases to refer to reviewer agents more flexibly across workflows.",
    "The system now handles forks and external repositories more safely during workspace operations."
  ],
  "🛠 Internal Updates": [
    "Running `make test` now ensures the web app is built and tested as part of the same command.",
    "Reviewer agents now provide stricter, read-only feedback that is treated as blocking by default.",
    "The codebase is easier to maintain due to consolidation of duplicated helpers and test utilities.",
    "Core services and persistence flows are more consistent by sharing common logic instead of duplicating it.",
    "Repository housekeeping and planning artifacts have been updated to reduce noise and track goals."
  ]
}

```

---

Written by [doc.holiday](https://doc.holiday)