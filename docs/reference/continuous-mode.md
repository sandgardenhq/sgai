# Continuous mode

Continuous mode keeps a workspace running after a workflow completes. Instead of stopping, the workflow starts a new cycle and waits for a trigger before running again.

## Overview

Continuous mode is enabled per workspace via `GOAL.md` YAML frontmatter.

While continuous mode is enabled, `sgai` watches for:

- Updates to `GOAL.md`
- New steering messages from the human partner

When a trigger happens, `sgai` starts the next workflow cycle.

## Prerequisites

- A workspace directory with a `GOAL.md`
- The dashboard running via `sgai serve`

## Enable continuous mode

1. Open the workspace `GOAL.md`.
2. Add a non-empty `continuousModePrompt` value in the YAML frontmatter.

```markdown
---
continuousModePrompt: "Review the current state and plan next steps"
---

# Project Goal

...
```

3. Start the dashboard:

```sh
sgai serve
```

`sgai` treats continuous mode as enabled when `continuousModePrompt` is present and not empty.

## What to expect in the dashboard

- The workspace detail page shows a **Continuous Self-Drive** button.
- When the workspace is running, a **Stop** button is available.

## Notes

- Continuous mode is surfaced in the workspace detail API response as `continuousMode`.
- Continuous mode writes additional progress entries using the agent name `continuous-mode`.
