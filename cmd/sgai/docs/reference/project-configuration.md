# Project configuration (`sgai.json`)

`sgai` reads an optional `sgai.json` file from the project root (as a sibling to the `.sgai` directory).

## File name and location

- File name: `sgai.json`
- Location: project root directory

## Related examples

The repository includes example configuration files you can use as starting points:

- [`GOAL.example.md`](../../GOAL.example.md): Example `GOAL.md` frontmatter with a workflow graph and per-agent model selection.
- [`sgai.example.json`](../../sgai.example.json): Example MCP configuration snippet for a project-level `sgai.json`.
- [`opencode.json`](../../opencode.json): Example OpenCode configuration with a local MCP entry.

## Schema

### `defaultModel`

Type: string

If set, `defaultModel` becomes the default model for any agent in `GOAL.md` that does not already have a model configured.

Notes:

- `defaultModel` is validated using the base model name. If the value includes a variant in parentheses (for example, `provider/model (variant)`), only the base model is validated.

### `disable_retrospective`

Type: boolean

If `true`, `sgai` does not create or resume a retrospective directory for the workflow run.

### `editor`

Type: string

Configures the editor used for the "Open in Editor" button in the web interface.

**Preset names:**

| Preset | Command | Terminal? |
|--------|---------|-----------|
| `code` | `code {path}` | No |
| `cursor` | `cursor {path}` | No |
| `zed` | `zed {path}` | No |
| `subl` | `subl {path}` | No |
| `idea` | `idea {path}` | No |
| `emacs` | `emacsclient -n {path}` | No |
| `nvim` | `nvim {path}` | Yes |
| `vim` | `vim {path}` | Yes |
| `atom` | `atom {path}` | No |

**Examples:**

```json
{"editor": "cursor"}
```

```json
{"editor": "myeditor --open {path}"}
```

**Fallback chain:**

When not set, `sgai` uses the following fallback chain:

1. `$VISUAL` environment variable
2. `$EDITOR` environment variable
3. `code` (VS Code)

**Custom commands:**

- If your custom command contains `{path}`, it will be replaced with the file path.
- If your custom command does not contain `{path}`, the path is appended to the end.
- Environment variable values are treated as custom commands.
- Custom commands are assumed to be GUI editors (not terminal editors).

**Terminal editors:**

Terminal-based editors (`vim`, `nvim`) cannot be opened from the web interface. When a terminal editor is configured, the "Open in Editor" button is hidden.

### `mcp`

Type: object (`map[string]json.RawMessage`)

`mcp` allows defining additional MCP entries that are merged into `.sgai/opencode.jsonc`.

Merge rules:

- `sgai` only adds an MCP entry when the entry name does not already exist in `.sgai/opencode.jsonc`.
- If `sgai.json` contains MCP entries but all of them already exist in `.sgai/opencode.jsonc`, `sgai` does not rewrite `.sgai/opencode.jsonc`.

## Notes

- If `sgai.json` does not exist, `sgai` proceeds without configuration.
- If `sgai.json` exists but cannot be read due to permissions, `sgai` reports a "permission denied reading config file" error.
- If `sgai.json` contains invalid JSON syntax or type mismatches, `sgai` reports an error that includes the file path and the failing offset or field.
