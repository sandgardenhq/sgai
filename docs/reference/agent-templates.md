# Agent template settings (mode and tool permissions)

This page describes a few agent-template settings as they appear in the `sgai` skeleton under `cmd/sgai/skel/.sgai/agent/*.md`.

## Overview

The skeleton includes multiple agent template files. Commit `f2ab442683ad28f673061c3459309240ea7ee2e5` normalizes how agent `mode` and tool permissions are expressed.

## Mode

At least one skeleton agent template uses a `mode` setting.

- One template changes its `mode` value from `primary` to `all`.

## Tool permissions

Several skeleton agent templates listed individual tool permissions as separate entries (for example: `edit: allow`, `bash: allow`, `skill: allow`, `webfetch: allow`). In this commit, many templates remove these per-tool `allow` entries.

One skeleton template changes its tool-permission section to:

- remove `edit: allow`, `bash: allow`, and `skill: allow`
- add `tools: allow`
- keep `webfetch: allow`

## Notes

- Some templates still include individual tool permissions (for example, one template retains `edit`, `bash`, `skill`, and `webfetch` entries but removes trailing whitespace).
