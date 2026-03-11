# 2026-03-11 Release Notes: Maintenance, performance, and documentation quality improvements

This release focuses on smoother automation configuration, more solid workspace operations, and cleanup to keep local development and tests predictable.

## 🚀 New Features

This week’s customer-facing changes focus on making automation configuration more flexible across different projects and teams.

The main theme is reducing friction when you want to refer to the same agent using different names in different contexts.

- **Added agent aliases** - Configure agent aliases so the same automation can be referenced by multiple names in different contexts.

## 🚧 Bug Fixes

This week’s fixes focus on making workspace flows more resilient, especially in scenarios that involve external repos and recovery.

We also tightened the compose wizard’s behavior so saving settings and recovering from errors is more consistent across sessions.

- **Fixed workspace and fork recovery flows** - Improved workspace and fork handling to avoid inconsistent state during recovery.
- **Fixed compose wizard save and recovery** - Improved settings persistence and error handling so incomplete wizard state does not leak across sessions.

## 🛠 Internal Updates

This week’s internal updates focus on keeping the codebase and docs easier to maintain, with special attention to tests, automation guardrails, and removing duplication.

The net effect is a cleaner developer experience: fewer flaky prerequisites, clearer documentation terminology, and more consistent reviewer automation.

- **Updated skill guides terminology** - Updated skill documentation to use OpenCode terminology consistently.
- **Improved local dev defaults** - Added a global `.DS_Store` ignore pattern and recorded a GOALS entry for issue 357.
- **Fixed `make test` prerequisites** - Updated the Makefile so `test` depends on `webapp-test` and `webapp-build`.
- **Refined automated reviewer guardrails** - Tightened reviewer automation to keep reviews read-only, block shell execution, and treat findings as blocking issues.
- **Hardened continuous and retrospective workflows** - Improved agent workflows to run more reliably across forks and external repositories.
- **Consolidated test helpers and utilities** - Centralized server test helpers and consolidated React test utilities to reduce duplication while preserving runtime behavior.
- **Simplified internal helpers and components** - Removed unused helpers/components and consolidated overlapping utilities to simplify APIs without changing behavior.

---

Written by <a href="https://doc.holiday">doc.holiday</a>