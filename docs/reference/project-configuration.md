# Project configuration (`sgai.json`)

`sgai` reads an optional `sgai.json` file from the project root (as a sibling to the `.sgai` directory).

## File name and location

- File name: `sgai.json`
- Location: project root directory

## Related examples

The repository includes example configuration files you can use as starting points:

- [`GOAL.example.md`](../../GOAL.example.md): Example `GOAL.md` frontmatter with a workflow graph and per-agent model selection.
- [`sgai.example.json`](../../sgai.example.json): Example `sgai.json` snippet.
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

If `editor` is not set, the app uses this fallback chain:

1. `$VISUAL` environment variable
2. `$EDITOR` environment variable
3. `code`

#### Presets

Use one of these preset values to select a common editor command:

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

#### Examples

```json
{"editor": "cursor"}
```

```json
{"editor": "myeditor --open {path}"}
```

#### Custom commands

Custom editor commands support a `{path}` placeholder.

- If the command contains `{path}`, the placeholder is replaced with the file path.
- If the command does not contain `{path}`, the file path is appended to the end.

Environment variable values from `$VISUAL` and `$EDITOR` are treated as custom commands.

Custom commands are assumed to be GUI editors (not terminal editors).

#### Terminal editors

Terminal-based editors (for example, `vim` and `nvim`) cannot be opened from the web interface.

When a terminal editor is configured, the web interface shows the workspace path or file path in a read-only text field so it can be selected and copied, instead of showing an "Open in Editor" button.

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
