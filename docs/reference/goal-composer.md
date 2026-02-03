# GOAL.md Composer (web UI)

The GOAL composer is a web-based editor for building a `GOAL.md` file with a live preview, optional AI suggestions, and a “command bar” for natural-language edits.

## Overview

Use the composer when writing a new project goal from scratch, or when a goal needs structured edits across multiple sections (frontmatter, agent selection, workflow graph, and tasks).

The compose page is available at:

- `/compose?workspace=<workspace>`

`workspace` is required.

## Prerequisites

- Run `sgai serve`.
- Open the dashboard in a browser.
- Know the workspace name you want to edit (the composer uses the `workspace` query parameter).

## Getting started

1. Start the web server:

   ```sh
   sgai serve
   ```

2. Open the composer for a workspace:

   ```text
   http://127.0.0.1:8080/compose?workspace=<workspace>
   ```

3. Use the left side to edit fields and the right side to review the rendered `GOAL.md` preview.

4. Select **Save** to write the composed content back to `GOAL.md` on disk.

## Step-by-step guide

### 1) Edit the project description

Open **Project Description** and write the main goal text.

- The panel includes an **AI Assist** button that can propose an updated description.

### 2) Configure frontmatter

Open **Frontmatter Configuration**.

- **Interactive Mode**: `yes`, `no`, or `auto`.
- **Completion Gate Script**: optional command (for example, `make test`).

### 3) Pick agents and models

Open **Agent Selection & Models**.

- Use checkboxes to select agents.
- Use the per-agent dropdown to pick a model, or leave it on **Default model**.
- Select **AI Suggest Agents** to ask for a proposed set of agents.

### 4) Define the workflow DAG

Open **Workflow DAG (DOT Format)**.

- Edit the `flow` text area.
- The panel shows a validation error block when the workflow cannot be parsed.
- Select **AI Generate DAG** to request a proposed workflow graph.

### 5) Maintain a task list

Open **Task List**.

- Use markdown checkbox syntax:

  ```markdown
  - [ ] Task description
  ```

- Select **AI Generate Tasks** to request a proposed task list.

## Natural-language command bar

The command bar applies a free-form command to the full configuration, then shows a **Command Preview** modal.

- **Proposed Changes** lists a summary of changes.
- **View Full Diff** shows a before/after preview of the generated `GOAL.md` content.
- Select **Apply Changes** to replace the current composer state with the proposed result.

## Save and reset

- **Save** writes the composer content to `GOAL.md`.
- **Reset** reloads the composer from the current `GOAL.md` on disk.

## Notes

- The composer also exposes JSON endpoints for discovery:
  - `/compose/agents?workspace=<workspace>`
  - `/compose/models?workspace=<workspace>`
