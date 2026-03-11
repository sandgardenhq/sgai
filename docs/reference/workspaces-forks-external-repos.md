# Workspaces, forks, and external directories

SGAI workspaces can be forked. In some setups, a fork target directory can live outside the normal workspace root (for example, under a different directory tree).

This page explains how SGAI handles forks that land in “external” locations, and what changed to make this more reliable.

## The problem this addresses

Two related issues can make fork handling unreliable:

1. **Symlink vs. real path mismatches**

   Workspace paths can be expressed in multiple ways (for example, a symlinked path vs. the resolved filesystem path). If comparisons use raw strings, SGAI can treat two paths as “different” even when they point to the same directory.

2. **Forks that live outside the workspace root**

   A fork can be created into a directory that is considered external. When that happens, it needs to be recorded as an external directory so later operations know it’s outside the normal workspace root.

In practice, these show up as:

- A fork directory that *should* be accepted as belonging to a workspace gets rejected because the “workspace root” path is expressed differently.
- A fork directory that lives outside the workspace root isn’t treated consistently as “external”, so later operations don’t have a stable, canonical path to work with.

## How it works now

### Path normalization uses symlink resolution

Workspace root path handling normalizes paths by resolving symlinks before comparing:

- The root workspace path is derived and then passed through symlink resolution.
- Fork directory validation compares the symlink-resolved root path for the fork against the symlink-resolved target workspace path.

This shows up in a few places:

- Workspace list/grouping uses `resolveSymlinks(...)` to normalize directory keys before grouping entries.
- Fork deletion validation compares the fork directory’s symlink-resolved root to the request’s symlink-resolved root.

### External fork targets are recorded

When an external workspace is detected during a fork operation:

- The symlink-resolved fork path is stored in the server’s external directory set.
- The updated external directory set is persisted.

### Fork deletion treats “root workspace” and “fork workspace” as valid starting points

The fork deletion handler accepts either:

- a workspace path that classifies as a **root**, or
- a workspace path that classifies as a **fork** (in which case the root workspace path is derived)

It then uses the symlink-resolved root path as the canonical working directory for operations that need to run “at the workspace root”.

## Examples

### Example: symlinked workspace root

If the workspace root is reached through a symlink (for example, `/work` pointing to `/mnt/workspaces`), comparisons now resolve both sides before deciding whether a fork directory is “in” the workspace.

Concrete example:

- Workspace root directory: `/work/my-root` (symlink)
- The same directory on disk: `/mnt/workspaces/my-root` (resolved)

If a fork operation or deletion request references one path form and another internal step uses the other path form, comparisons are performed on the symlink-resolved paths.

### Example: fork created into an external directory

If a fork is created into a directory outside the normal workspace root, the fork directory is recorded as external using the symlink-resolved fork path.

### Example: delete a fork when the request starts from the fork path

Some requests can start from a fork directory path rather than the root workspace path.

If both of these are true:

- the provided `workspacePath` classifies as a fork, and
- the fork directory’s symlink-resolved root matches the workspace root

then fork deletion proceeds using the symlink-resolved root directory as the canonical workspace directory.
