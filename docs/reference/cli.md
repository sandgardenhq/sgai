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

  Note: When the host is a wildcard address (for example, `0.0.0.0:8080` or `[::]:8080`), `sgai` opens the dashboard URL using a loopback host (`127.0.0.1` / `::1`).

#### macOS menu bar status

On macOS, `sgai` shows a compact status summary in the menu bar.

The summary uses this format:

- Normal: `⏺ <running> / <total>`
- Warning: `⚠ <running> / <total>`

The menu only shows per-factory entries such as `<factory> (Needs Input)` and `<factory> (Stopped)`.

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
