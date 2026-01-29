# Environment variables

This page lists environment variables that `sgai` reads.

## `EDITOR`

If `EDITOR` is set, `sgai` uses it to choose the default for `--interactive`.

- When `EDITOR` is set, the default interactive mode is `yes`.
- When `EDITOR` is not set, the default interactive mode is `auto`.

## `SGAI_MCP_WORKING_DIRECTORY`

`sgai mcp` reads `SGAI_MCP_WORKING_DIRECTORY` to decide which directory contains `.sgai/state.json`.

- If `SGAI_MCP_WORKING_DIRECTORY` is not set, the default working directory is `.`.
- `sgai mcp` loads state from `<working-dir>/.sgai/state.json`.

## `SGAI_MCP_EXECUTABLE`

When `sgai` runs `opencode`, it sets `SGAI_MCP_EXECUTABLE` to the absolute path of the `sgai` executable.

This is used by the `sgai` MCP integration to locate the `sgai` binary.

## `SGAI_NTFY`

If `SGAI_NTFY` is set, `sgai` sends remote notifications by posting the message body to that URL.

