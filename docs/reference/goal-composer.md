# GOAL.md Composer (web UI)

The GOAL composer is a web-based editor for building a `GOAL.md` file with a live preview.

## Overview

Use the composer to create a new `GOAL.md` from scratch, or to make structured edits across multiple sections (description, frontmatter, workflow graph, and tasks).

Open the composer at:

- `/compose?workspace=<workspace>`

`workspace` is required.

## Prerequisites

- Run `sgai serve`.
- Open the dashboard in a browser.
- Know the workspace name to edit (the composer uses the `workspace` query parameter).

## Getting started

1. Start the web server:

   ```sh
   sgai serve
   ```

2. Open the composer for a workspace:

   ```text
   http://127.0.0.1:8080/compose?workspace=<workspace>
   ```

   The dashboard also includes a **Compose GOAL** action that links to `/compose` with the current workspace filled in.

3. Use the left side to edit fields and the right side to review the rendered `GOAL.md` preview.

4. Select **Save** to write the composed content back to `GOAL.md` on disk.

## Step-by-step guide

### 1) Edit the project description

Open **Project Description** and write the main goal text.

### 2) Configure frontmatter

Open **Frontmatter Configuration**.

### 3) Define the workflow graph

Open **Workflow DAG (DOT Format)**.

The preview area can show a workflow validation error block when the workflow cannot be parsed.

### 4) Maintain a task list

Open **Task List**.

Use markdown checkbox syntax:

```markdown
- [ ] Task description
```

## Save and reset

- **Save** writes the composer content to `GOAL.md`.
- **Reset** reloads the composer from the current `GOAL.md` on disk.

## Notes

- The composer also exposes JSON endpoints for discovery:
  - `/compose/agents?workspace=<workspace>`
  - `/compose/models?workspace=<workspace>`
