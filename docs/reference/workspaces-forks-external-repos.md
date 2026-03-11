# Workspaces, forks, and external directories

SGAI workspaces can be forked. In some setups, a fork target directory can live outside the normal workspace root (for example, under a different directory tree).

This page explains how SGAI handles forks that land in “external” locations, and what changed to make this more reliable.

## The problem this addresses

Two related issues can make fork handling unreliable:

1. **Symlink vs. real path mismatches**

   Workspace paths can be expressed in multiple ways (for example, a symlinked path vs. the resolved filesystem path). If comparisons use raw strings, SGAI can treat two paths as “different” even when they point to the same directory.

2. **Forks that live outside the workspace root**

   A fork can be created into a directory that is considered external. When that happens, it needs to be recorded as an external directory so later operations know it’s outside the normal workspace root.

## How it works now

### Path normalization uses symlink resolution

Workspace root path handling normalizes paths by resolving symlinks before comparing:

- The root workspace path is derived and then passed through symlink resolution.
- Fork directory validation compares the symlink-resolved root path for the fork against the symlink-resolved target workspace path.

### External fork targets are recorded

When an external workspace is detected during a fork operation:

- The symlink-resolved fork path is stored in the server’s external directory set.
- The updated external directory set is persisted.

## Examples

### Example: symlinked workspace root

If the workspace root is reached through a symlink (for example, `/work` pointing to `/mnt/workspaces`), comparisons now resolve both sides before deciding whether a fork directory is “in” the workspace.

### Example: fork created into an external directory

If a fork is created into a directory outside the normal workspace root, the fork directory is recorded as external using the symlink-resolved fork path.
