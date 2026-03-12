# Forks and External Workspaces

SGAI can work with:

* **Root workspaces** (the main workspace for a project)
* **Fork workspaces** (a workspace that points back to a root)
* **Externally attached workspaces** (a workspace directory that lives outside the main root directory SGAI scans)

This page explains a reliability issue that shows up when forks or roots involve symlinks or live in external locations, plus what SGAI does to handle it.

## The problem this week’s changes address

Some workspace operations relied on comparing filesystem paths to decide how to group workspaces and whether something “belongs” under a root.

That can go wrong when the same directory can be referenced via multiple paths, for example:

* One path goes through a symlink, and the other is the symlink-resolved (real) path.
* A root workspace is scanned from one directory, but a fork is created or attached from another directory.

In these cases, two strings can refer to the same underlying directory, and a plain string comparison can mis-classify the workspace.

This mis-classification shows up as “fork/external repo” rough edges in day-to-day work:

* A fork that should be associated with a root can appear disconnected because the fork path and root path don’t match byte-for-byte.
* A fork of an external workspace can fail to be treated as external if the fork directory is tracked using the non-resolved path.

## What SGAI does now

### 1) Normalize paths by resolving symlinks

SGAI resolves symlinks and uses the resolved path when comparing root and workspace paths.

In `cmd/sgai/service_workspace.go`, root/workspace comparisons use symlink-resolved paths.

Related implementation details in this commit range:

* In `cmd/sgai/service_external.go`, the root workspace path is resolved via `resolveSymlinks(...)` before it’s used, and the function returns early if `getRootWorkspacePath(...)` returns an empty string.
* In `cmd/sgai/serve.go` and `cmd/sgai/serve_api.go`, grouping and validation logic uses `resolveSymlinks(...)` and compares resolved root paths instead of raw strings.

### 2) Treat forks of external workspaces as external

The weekly update also calls out “External workspace fork tracking”: when the target workspace path is external, fork directories are recorded as external using the symlink-resolved fork path.

In `cmd/sgai/service_workspace.go`, this is implemented by storing the fork’s resolved directory path in the service’s `externalDirs` map and attempting to persist the updated set (logging an error if persistence fails).

## Examples

### Example: root inside the scan path, fork created through a symlink

* Root workspace lives at a real path like `/Users/alex/work/root-project`.
* The same directory is also accessible at `/Users/alex/link-to-work/root-project` (where `link-to-work` is a symlink).

Without symlink normalization, those two paths look different even though they point at the same directory.

With symlink normalization, SGAI compares (and groups) workspaces using resolved paths so the root and fork stay connected.

If these paths refer to the same directory:

* `/Users/alex/work/root-project`
* `/Users/alex/link-to-work/root-project`

SGAI uses the resolved (real) path for comparisons instead of assuming the raw strings must match.

### Example: external root workspace, external fork

If a workspace is attached from outside the usual scan root, it is treated as external. When creating or tracking forks for that workspace, SGAI records the fork directory as external as well, using the symlink-resolved fork path.

## Related reading

* [Agent aliases](./agent-aliases.md)