# CLI commands

This page describes the `sgai` command-line interface.

## Usage

```sh
sgai [--interactive] [--fresh] <target_directory>
```

`sgai` expects a `GOAL.md` file in the target directory.

## Global options

- `--interactive`

  Interactive mode.

  Accepted values:

  - `yes` (open `$EDITOR` for human responses)
  - `no`
  - `auto` (self-driving)

- `--fresh`

  Force a fresh start (do not resume existing workflow).

## Commands

### `sgai serve`

Start the web server for session management.

```sh
sgai serve [--listen-addr addr]
```

Options:

- `--listen-addr`

  HTTP server listen address.

  Default: `127.0.0.1:8080`

## Troubleshooting

### Workspace looks like it is still running after a session ends

Some session views keep per-workspace “ever started” state to decide whether to show a running indicator (for example, a spinner).

If a workspace is already complete, `sgai serve` clears that “ever started” marker when the session ends. If you still see a workspace shown as running:

1. Check the workspace workflow state to confirm the status is complete.
2. If the workflow state file is missing or the status is not complete, the running indicator may stay on because `sgai serve` does not clear the marker in those cases.

See [Workflow state](./workflow-state.md) for where the state file lives and which statuses are considered complete.

### `sgai sessions`

List all sessions in `.sgai/retrospectives`.

```sh
sgai sessions
```

### `sgai status`

Show workflow status summary.

```sh
sgai status [target_directory]
```

If `target_directory` is omitted, `sgai` uses the current directory.

### `sgai retrospective`

Work with retrospective sessions.

#### `sgai retrospective analyze`

Analyze a session.

```sh
sgai retrospective analyze [session-id]
```

If `session-id` is omitted, `sgai` analyzes the most recent session.

#### `sgai retrospective apply`

Apply improvements from a session.

```sh
sgai retrospective apply <session-id>
```

### `sgai list-agents`

List available agents.

```sh
sgai list-agents [target_directory]
```

### `sgai mcp`

Start the MCP server.

```sh
sgai mcp
```
