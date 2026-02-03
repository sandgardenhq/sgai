# GOAL.md reference and authoring guide

Hey there!

`sgai` uses a `GOAL.md` file in your project directory to define what you want built. This page describes the supported `GOAL.md` structure and gives a practical, step-by-step authoring checklist.

It also points to the built-in **GOAL.md Composer** skill, which provides an interactive, phased workflow for writing a valid `GOAL.md`.

## What you'll learn

- The `GOAL.md` file layout (`YAML` frontmatter + Markdown body)
- How to write the `flow` graph and optional `models` mapping
- How `interactive` and `completionGateScript` affect execution
- A validation checklist (the stuff that usually trips people up)

## Prerequisites

- A project directory you can run `sgai` against
- A basic familiarity with YAML frontmatter and Markdown

## GOAL.md file layout

Create `GOAL.md` in your project root directory.

1. Add **YAML frontmatter** delimited by `---`.
2. Add a **Markdown body** that describes the goal, requirements, and a checkbox task list.

A minimal `GOAL.md` looks like this:

```markdown
---
flow: |
  "general-purpose"
interactive: yes
---

# My Project Goal

Describe what you want to build.

## Requirements

- List externally visible behavior and constraints.

## Tasks

- [ ] A concrete piece of work
```

For a larger example, see [`GOAL.example.md`](../../GOAL.example.md).

## Frontmatter fields

### `flow` (required)

Type: YAML multiline string containing a DOT-format directed acyclic graph (DAG).

Use one of these patterns:

- **Standalone agent** (no edges):

  ```yaml
  flow: |
    "general-purpose"
  ```

- **Dependency edge** (A runs before B):

  ```yaml
  flow: |
    "backend-go-developer" -> "go-readability-reviewer"
  ```

Notes:

- Quote agent names (for example, `"general-purpose"`).
- Keep the graph acyclic (no cycles).
- The `coordinator` agent is always present, but the flow must not include `"coordinator"`.

### `models` (optional)

Type: YAML map of `"agent-name"` → `"provider/model"`.

Example:

```yaml
models:
  "coordinator": "anthropic/claude-opus-4-5"
  "backend-go-developer": "anthropic/claude-sonnet-4-5"
```

Notes:

- Agent keys use the same quoted names as `flow`.
- Model strings may include a variant in parentheses (for example, `provider/model (variant)`).

### `interactive` (required)

Accepted values:

- `yes`
- `no`
- `auto`

This field controls how the workflow behaves when an agent needs clarification.

### `completionGateScript` (optional)

Type: string.

A shell command used as a completion gate. The workflow is only considered complete when the command exits successfully.

Example:

```yaml
completionGateScript: make test
```

## Markdown body conventions

Use the Markdown body to communicate the project specification.

A common structure is:

- A title (`# ...`)
- A short description
- `## Requirements`
- `## Tasks` with checkboxes

Task checkboxes use `- [ ]` and can be nested:

```markdown
## Tasks

- [ ] Top-level task
  - [ ] Nested task
```

## Reviewer pairing rules

Some coding agents have a required reviewer pairing:

- `backend-go-developer` → `go-readability-reviewer`
- `htmx-picocss-frontend-developer` → `htmx-picocss-frontend-reviewer`
- `shell-script-coder` → `shell-script-reviewer`

When a paired reviewer is present, the flow typically includes an edge from the coding agent to its reviewer.

## Validation checklist

Before running `sgai`, verify:

- `GOAL.md` has YAML frontmatter and a Markdown body.
- `flow` is present and uses quoted agent names.
- `flow` is a DAG (no cycles).
- `"coordinator"` is not listed in `flow`.
- `interactive` is set to `yes`, `no`, or `auto`.
- If `models` is present, it uses quoted agent keys.
- If a paired reviewer is used, include it and connect it appropriately in `flow`.
- `## Tasks` uses checkbox syntax (`- [ ]`).

## Use the GOAL.md Composer skill

The repository includes a GOAL authoring skill ("GOAL.md Composer") with a seven-phase interactive workflow.

Skill files:

- `cmd/sgai/skel/.sgai/skills/product-design/goal-md-composer/SKILL.md`
- `cmd/sgai/skel/.sgai/skills/product-design/goal-md-composer/REFERENCE.md`

The skill guides goal authoring through these phases:

1. Project description (questions asked one at a time)
2. Agent recommendations (including required reviewer pairing)
3. Flow builder (build the `flow` DAG)
4. Model configuration
5. Specification writing (title/description, requirements, tasks)
6. Options (`interactive`, `completionGateScript`)
7. Output and validation

If a workflow needs a fresh `GOAL.md` and the details are still fuzzy, use this composer process to gather inputs and produce a valid file.
