# Agent template settings (mode, tool permissions, and model)

This page describes a few agent-template settings as they appear in the `sgai` skeleton under `cmd/sgai/skel/.sgai/agent/*.md`.

## Overview

The skeleton includes multiple agent template files. Some of these templates set a `mode` field, and some list tool permissions using per-tool `...: allow` entries.

## Mode

Several skeleton agent templates set `mode: all`.

The following templates use `mode: all` in the current skeleton:

- `go-readability-reviewer`
- `htmx-picocss-frontend-reviewer`
- `shell-script-reviewer`
- `skill-writer`
- `snippet-writer`

## Tool permissions

Some skeleton agent templates include per-tool permission entries like:

- `edit: allow`
- `bash: allow`
- `skill: allow`
- `webfetch: allow`

In the current skeleton, some templates remove previously listed `...: allow` entries.

Examples from the current skeleton:

- `go-readability-reviewer` no longer lists `bash: allow`, `skill: allow`, or `webfetch: allow`.
- `htmx-picocss-frontend-developer` no longer lists `edit: allow`, `bash: allow`, `skill: allow`, or `webfetch: allow`.
- `htmx-picocss-frontend-reviewer` no longer lists `edit: allow`, `bash: allow`, `skill: allow`, or `webfetch: allow`.
- `retrospective-code-analyzer` no longer lists `edit: allow`, `bash: allow`, or `skill: allow`.
- `retrospective-refiner` no longer lists `edit: allow`, `bash: allow`, or `skill: allow`.

Some templates explicitly allow tools.

Examples from the current skeleton:

- `retrospective-applier` lists `edit: allow`, `bash: allow`, `skill: allow`, and `webfetch: allow`.
- `retrospective-session-analyzer` lists `edit: allow`, `bash: allow`, and `skill: allow`.

## Model

The `stpa-analyst` template does not set an explicit model override line for `anthropic/claude-opus-4-5`.
