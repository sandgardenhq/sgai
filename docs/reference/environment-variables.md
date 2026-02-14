# Environment variables

This page lists environment variables that `sgai` reads.

## `EDITOR`

If `EDITOR` is set, `sgai` uses it when opening an editor for human responses.

## `SGAI_MCP_EXECUTABLE`

Used by the OpenCode plugin configuration to locate the `sgai` binary.

In the skeleton plugin, the MCP server command is configured as:

```ts
command: [process.env.SGAI_MCP_EXECUTABLE || "sgai", "mcp"]
```

## `SGAI_MCP_WORKING_DIRECTORY`

`sgai mcp` reads `SGAI_MCP_WORKING_DIRECTORY` to decide which directory contains `.sgai/state.json`.

- If `SGAI_MCP_WORKING_DIRECTORY` is not set, the default working directory is `.`.
- `sgai mcp` loads state from `<working-dir>/.sgai/state.json`.

## `SGAI_NTFY`

If `SGAI_NTFY` is set, `sgai` sends remote notifications by posting the message body to that URL.

