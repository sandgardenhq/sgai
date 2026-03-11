# Workspaces: forks and external repos

Some workspace operations need to handle forks that live outside the main workspace root (for example, when a fork directory is created in an external location, or when symlinks cause the same path to have multiple representations).

This page explains the problem and the resolution that is summarized in the 2026-03-11 weekly update and implemented in `cmd/sgai/service_workspace.go`.

## The problem: path comparisons break with symlinks and external locations

Workspace and fork operations sometimes need to compare:

* the “root workspace” path
* a “fork workspace” path
* the “target workspace” path

When paths include symlinks (or when a fork lives outside the main workspace root), comparing raw path strings can produce incorrect results.

## The resolution: normalize paths and persist external fork directories

Two concrete changes address this:

1. **Normalize paths before comparing**

   `cmd/sgai/service_workspace.go` resolves symlinks before it compares the “root workspace path” and the workspace path.

2. **Record forked workspaces as external when needed**

   When the target workspace path is considered external, `cmd/sgai/service_workspace.go` records the fork directory as external by storing the symlink-resolved fork path, and then persists the updated external directory set.

## Examples

### Example: symlinked workspace paths

If the same directory can be addressed through two different paths (for example, a real path vs. a symlinked path), SGAI resolves symlinks before comparing paths.

### Example: a fork that lives outside the workspace root

If a fork is created into a directory that SGAI considers external, SGAI records the fork directory as external using the symlink-resolved fork path.
