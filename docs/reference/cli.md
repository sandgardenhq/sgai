# CLI commands

This page describes the `sgai` command-line interface.

## Usage

```sh
sgai [--fresh] <target_directory>
```

`sgai` expects a `GOAL.md` file in the target directory.

## Global options

- `--fresh`

  Force a fresh start (do not resume existing workflow).

## Interaction behavior

`sgai` prompts for human input when the workflow asks a question and standard input is a TTY.

After the workflow passes its completion gate (the point where a run is considered approved), `sgai` switches to a self-driving mode and stops prompting for human input.

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
