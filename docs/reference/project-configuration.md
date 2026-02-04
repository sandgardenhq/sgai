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

## How `sgai` picks an editor

`sgai` selects an editor in the following order:

1. The `editor` field in `sgai.json` (when set)
2. The `$VISUAL` environment variable
3. The `$EDITOR` environment variable
4. The `code` preset

### Availability checks and fallback

`sgai` only enables "Open in Editor" when the selected editor command is available.

If `$VISUAL` / `$EDITOR` points at a command that is not available, `sgai` falls back to the default preset (`code`) when that preset is available.

## Custom commands

- If your custom command contains `{path}`, it will be replaced with the file path.
- If your custom command does not contain `{path}`, the path is appended to the end.
- Environment variable values are treated as custom commands.
- Custom commands are assumed to be GUI editors (not terminal editors).

## Terminal editors

Some presets (for example, `vim` and `nvim`) run in a terminal.

The web interface still treats the editor as available based on whether the configured command exists on the machine.

If "Open in Editor" does not behave as expected with a terminal-based editor, pick a GUI editor preset (for example, `code` or `cursor`) or configure a GUI command explicitly.

