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

#### macos menu bar factory monitor

On macOS (`darwin`), `sgai serve` also starts a menu bar status item.

The status item:

- Shows an attention badge in the title.
  - `● <count>` when one or more factories need attention
  - `○ sgai` when none need attention
- Provides a dropdown menu with:
  - **Open Dashboard** (opens the `sgai serve` base URL)
  - A list of factories that need attention
    - `⚠ <name> (Needs Input)` opens `/workspaces/<name>/respond`
    - `■ <name> (Stopped)` opens `/workspaces/<name>/progress`
  - A status line: `<running> running, <attention> need attention`

The menu updates on server-sent events (SSE) from `sgai serve`.

On non-macOS platforms (`!darwin`), the menu bar implementation is not available.

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
