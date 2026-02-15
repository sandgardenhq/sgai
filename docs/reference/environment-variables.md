# Environment variables

This page lists environment variables that `sgai` reads.

## `EDITOR`

If `EDITOR` is set, `sgai` uses it to choose the default for `--interactive`.

- When `EDITOR` is set, the default interactive mode is `yes`.
- When `EDITOR` is not set, the default interactive mode is `auto`.

## `SGAI_MCP_EXECUTABLE`

Used by the OpenCode plugin configuration to locate the `sgai` binary.

In the skeleton plugin, the MCP server command is configured as:

```ts
command: [process.env.SGAI_MCP_EXECUTABLE || "sgai", "mcp"]
```

## `SGAI_MCP_INTERACTIVE`

Controls the interactive mode used by the MCP integration.

In the skeleton plugin, the environment is configured as:

```ts
environment: {
  SGAI_MCP_WORKING_DIRECTORY: directory,
  SGAI_MCP_INTERACTIVE: process.env.SGAI_MCP_INTERACTIVE || "yes",
}
```

## `SGAI_MCP_WORKING_DIRECTORY`

`sgai mcp` reads `SGAI_MCP_WORKING_DIRECTORY` to decide which directory contains `.sgai/state.json`.

- If `SGAI_MCP_WORKING_DIRECTORY` is not set, the default working directory is `.`.
- `sgai mcp` loads state from `<working-dir>/.sgai/state.json`.


