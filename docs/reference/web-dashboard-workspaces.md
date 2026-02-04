# Web dashboard: workspace tabs

The `sgai serve` web dashboard organizes each workspace into a set of tabs.

## What you can do

- Review session output
- Run an ad-hoc prompt against a selected model from the **Run** tab

## Workspace tabs

Open a workspace, then use the navigation at the top:

- **Session**: View session content.
- **Run**: Submit an ad-hoc prompt and watch its output stream in-place.

## Use the Run tab

1. Open the workspace.
2. Select the **Run** tab.
3. Choose a model.

   If no model has been selected for Run yet, the UI selects the coordinator model from `GOAL.md` when available. If no coordinator model is set, the first available model is selected.

4. Enter a prompt.
5. Select **Submit**.

The output area updates while the prompt is running.

## Notes

- The Run tab is always available. No configuration flag is required to enable it.
