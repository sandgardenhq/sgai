# Environment variables

This page lists environment variables that `sgai` reads.

## `EDITOR`

If `EDITOR` is set, `sgai` uses it to choose the default for `--interactive`.

## `SGAI_MCP_WORKING_DIRECTORY`

`sgai mcp` reads `SGAI_MCP_WORKING_DIRECTORY` to decide which directory contains `.sgai/state.json`.

If `SGAI_MCP_WORKING_DIRECTORY` is not set, the default working directory is `.`.

## `SGAI_NTFY`

If `SGAI_NTFY` is set, `sgai` sends remote notifications by posting the message body to that URL.

