# Project configuration (`sgai.json`)

`sgai` reads an optional `sgai.json` file from the project root (as a sibling to the `.sgai` directory).

## File name and location

- File name: `sgai.json`
- Location: project root directory

## Schema

### `defaultModel`

Type: string

If set, `defaultModel` becomes the default model for any agent in `GOAL.md` that does not already have a model configured.

### `disable_retrospective`

Type: boolean

If `true`, `sgai` does not create or resume a retrospective directory for the workflow run.

### `mcp`

Type: object (`map[string]json.RawMessage`)

`mcp` allows defining additional MCP entries that are merged into `.sgai/opencode.jsonc` (only if the entry name does not already exist in that file).

## Notes

- If `sgai.json` does not exist, `sgai` proceeds without configuration.
- If `sgai.json` exists but cannot be read due to permissions, `sgai` reports a "permission denied reading config file" error.
- If `sgai.json` contains invalid JSON syntax or type mismatches, `sgai` reports an error that includes the file path and the failing offset or field.
