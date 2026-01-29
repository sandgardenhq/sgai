# Web dashboard: opening an in-progress workspace

Hey there, fellow developer!

When a workspace is already running, the web dashboard lets you jump back in quickly. This page explains where the dashboard takes you when you select an **in-progress** workspace.

## What you’ll learn

- What “needs input” means in the dashboard
- Where an in-progress workspace link takes you

## How selection works for in-progress workspaces

When you click an in-progress workspace/project in the dashboard:

- If the workspace **needs input**, the dashboard opens the **Respond** screen.
- If the workspace **does not need input**, the dashboard opens the workspace **Progress** screen.

### Relevant routes

Depending on the workspace state, the UI navigates to one of these URLs:

- Respond screen:

  ```text
  /respond?dir=<project directory>
  ```

- Workspace progress screen:

  ```text
  /workspaces/<workspace name>/progress
  ```

## Troubleshooting

### Clicking an in-progress workspace does not show a place to reply

The workspace might not be waiting for human input. Open the progress view instead, and look for a prompt that requests input.

## Next steps

- Start the dashboard with `sgai serve`, then use the in-progress list to navigate back to active work.
