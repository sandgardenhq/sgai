# Web Dashboard: Workspace Details

This page describes what the web dashboard shows on the **Workspace Details** view.

## Status line visibility

The Workspace Details view can show a status line when at least one of these values is available:

- A model label
- A status text line

The status line is hidden when viewing the **root repository** in **forked mode**.

### Notes for contributors

In the frontend code, these values are surfaced as `agentModelLabel` and `statusLine`.
