# Web dashboard

The `sgai serve` command starts a web dashboard for monitoring workspaces, following progress, and responding when human input is needed.

## Prerequisites

- `sgai` is installed.
- A project directory exists with a `GOAL.md` file.
- The dashboard is running via `sgai serve`.

## Open the dashboard

1. Run:

   ```sh
   sgai serve
   ```

2. Open `http://127.0.0.1:8080` in a browser.

## Workspaces list

The dashboard shows a list of workspaces.

### Workspace refresh behavior

The workspaces list can refresh while keeping the existing list visible.

### Workspace row indicators

- **Waiting for response**

  A workspace row can show an inbox indicator with the tooltip text `Waiting for response`.

### Header indicators

The workspace list header can show two indicators:

- An inbox indicator with a numeric badge when one or more workspaces are waiting for a response.
- A factory status marker:
  - `●` with the tooltip text `Some factories are running`
  - `○` with the tooltip text `All factories stopped`

## Workspace detail: missing workspace

If a workspace cannot be loaded and the error message contains `workspace not found`, the dashboard navigates back to the main dashboard instead of showing an error page.

## Progress tab

The Progress tab includes workflow details and an events timeline.

### Agent models table

When workflow data includes agent-to-model assignments, the Progress tab shows an **Agent Models** table.

Each row shows:

- An agent name
- The models associated with that agent

Model values display in truncated form, with a tooltip that reveals the full model value on hover.

### Events timeline

The events timeline renders directly in the page layout (without an internal scroll container).

## Workflow completion gate

When a workflow uses a completion gate script (for example, a `completionGateScript` value in `GOAL.md`), the dashboard can show a human task message that includes text like:

- `running completionGateScript: …`

## Messages tab

Unread messages appear with bold styling in the message summary.
