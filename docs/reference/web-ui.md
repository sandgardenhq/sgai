# Web UI

The web UI includes a dashboard with a sidebar header that shows an inbox indicator when one or more workspaces are waiting on a human response.

## Open the next workspace that needs input

Select the inbox indicator in the sidebar header.

The web UI navigates to the first workspace that needs input, using this route pattern:

`/workspaces/:name/respond`

## Accessibility notes

The inbox indicator is a button (not decorative text).

- Screen readers announce it using an `aria-label` that mentions how many workspaces are waiting for a response.
- Keyboard users can tab to it and see a visible focus outline.