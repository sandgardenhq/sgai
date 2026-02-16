# Webapp Forks tab (Ad-hoc Prompt)

The webapp workspace **Forks** tab includes an **Ad-hoc Prompt** section below the forks list. This section provides an inline “run box” for submitting a prompt against a selected model and viewing the resulting output.

## Where to find it

1. Open the webapp.
2. Open a workspace.
3. Select the **Forks** tab.
4. Scroll below the forks list to the **Ad-hoc Prompt** section.

The **Ad-hoc Prompt** section renders even when the workspace has no forks (the empty state message still appears above it).

## Use the run box

### 1) Choose a model

Use the **Model** selector to choose from the available models for the current workspace.

- The model list is loaded from the webapp models endpoint (`/api/v1/models`).
- If a default model is provided by the endpoint, it is auto-selected (as long as it exists in the returned model list).

### 2) Enter a prompt

Enter your text in the **Prompt** field.

### 3) Submit

Select **Submit** to start the ad-hoc run.

While a run is in progress:

- The submit button shows a loading state (“Running…”) and is disabled.
- The model selector and prompt field are disabled.

## View output

An **Output** section appears when either:

- a run is in progress, or
- output text is available.

Output is displayed in a scrollable, monospaced block and auto-scrolls as new output is appended.

## Errors and loading behavior

- While models are loading, a skeleton/loading layout is shown.
- If the model list fails to load, an error message is shown instead of the run box.
- If the ad-hoc run fails, an alert is shown with the error message.

## What the webapp calls

The run box uses these workspace API calls:

- Start an ad-hoc run: `api.workspaces.adhoc(workspaceName, prompt, model)`
- Poll for status/output while running: `api.workspaces.adhocStatus(workspaceName)`
