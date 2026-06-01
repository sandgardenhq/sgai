# Skills

Skills are markdown documents that provide agents with on-demand instructions, checklists, domain knowledge, or procedural guidance. An agent searches for and loads a skill at runtime — skills are not injected automatically.

## How skills work

Skills are exposed to every agent as two MCP tools:

- **`find_skills({"name":""})`** — searches available skills by name or keywords, returns a list of matching skill names and descriptions
- **`skill({"name":"skill-name"})`** — loads the full content of a specific skill into the agent's context

Every agent's system prompt instructs it to search for relevant skills before doing any work. When a skill is found, the agent calls `skill()` to load the full content, then follows it as part of completing its task.

Skills are scoped to the workspace — an agent can only find and load skills from the workspace it is running in.

## File location

Skills are stored in the workspace `.sgai/skills/` directory:

```
<workspace>/
  .sgai/
    skills/
      <skill-name>/
        SKILL.md
      <category>/
        <skill-name>/
          SKILL.md
```

Each skill lives in its own subdirectory containing a single `SKILL.md` file. Skills can be placed at the top level of `skills/` or nested one level deep inside a category directory. The directory structure determines how skills are grouped in the web UI.

The skill name used with the `skill()` tool is the path relative to `skills/`, for example:
- `skill({"name":"test-driven-development"})` — top-level skill
- `skill({"name":"coding-practices/go-code-review"})` — categorised skill

## File format

Every `SKILL.md` must begin with a YAML frontmatter block:

```markdown
---
name: <skill name — should match the parent directory name>
description: <what this skill does and when an agent should use it>
compatibility: <any requirements or constraints, e.g. tool dependencies>
---

# Skill Title

Skill content here...
```

The `name` and `description` fields are required. `find_skills()` returns these fields — the description is what an agent reads when deciding whether to load the full skill. A clear, specific description is important: it determines whether an agent will recognise the skill as relevant to its task.

The body of `SKILL.md` is free-form markdown. It is loaded verbatim into the agent's context when `skill()` is called.

## How `find_skills` searches

When an agent calls `find_skills({"name":"some query"})`, the search runs in this order:

1. **Exact match** — checks if any skill's relative path exactly matches the query
2. **Prefix match** — checks if any skill path starts with the query
3. **Basename match** — checks if any skill's directory name matches the query
4. **Fuzzy match** — returns skills whose name or description contains any of the query words

Calling `find_skills({"name":""})` (empty string) returns all available skills.

## Built-in skills

`sgai` ships with approximately 35 built-in skills covering topics including:

- Test-driven development
- Code review (requesting and receiving)
- Systematic debugging
- Verification before completion
- Deployment patterns
- Frontend design
- Retrospective processes
- Using JJ instead of git

These are stored in `cmd/sgai/skel/.sgai/skills/` in the sgai source tree and are unpacked into every workspace's `.sgai/skills/` directory when a session starts.

## Adding your own skills

### When to add a skill

Add a skill when agents would otherwise have to rediscover the same knowledge from scratch on every run. Good candidates:

- **Domain knowledge** specific to your project (data formats, business rules, terminology)
- **Process checklists** that must be followed consistently (deployment steps, review criteria)
- **Patterns and conventions** used in your codebase that are not universally known

Skills are not necessary for general programming knowledge that models already have. They are most valuable for project-specific information that exists nowhere else.

### Where to put custom skills

Write custom skills to the workspace overlay directory:

```
<workspace>/
  sgai/
    skills/
      <skill-name>/
        SKILL.md
```

Files in `<workspace>/sgai/skills/` are copied into `<workspace>/.sgai/skills/` when a session starts. This means your skill file is the source of truth, and `.sgai/` is the runtime copy.

To make a skill immediately available without waiting for a session start, also write it directly to `<workspace>/.sgai/skills/<skill-name>/SKILL.md`.

### Naming

- Use lowercase kebab-case for directory names: `analyze-rules`, `go-code-review`
- Category directories (optional, one level only): `coding-practices/go-code-review`
- The directory name becomes the skill name used in `skill()` calls — keep it short and descriptive
- Do not use names that conflict with the 35 built-in skill names (they would be overwritten on session start)

### Example skill

```markdown
---
name: unit-data-format
description: Describes how unit stat files in data/ are structured. Use before reading or writing any unit data file.
compatibility: Requires access to the workspace data/ directory.
---

# Unit Data Format

Unit stats are stored as markdown files in `data/`.

## File naming

`<Unit Name> <Variant>.md` — for example: `Panzer PzKpfW I.md`, `Foot Platoon (Rifle).md`

## Frontmatter fields

Each file begins with YAML frontmatter:

...
```

## How skills are loaded at session start

When a session starts in a workspace, `sgai` runs two steps in order:

1. **Skeleton unpack** — all built-in skills from `skel/.sgai/skills/` are copied into `<workspace>/.sgai/skills/`, overwriting any files with matching paths
2. **Overlay apply** — all files from `<workspace>/sgai/skills/` are copied into `<workspace>/.sgai/skills/` on top, overwriting any skel files with matching paths

Custom skills with names that do not conflict with built-in skills are unaffected by step 1. Custom skills with the same name as a built-in skill will override it (since the overlay runs second).

## Skills in the web UI

The Skills tab in the sgai web UI reads from `<workspace>/.sgai/skills/` at request time. Skills are grouped by their category directory. Each skill shows its `name` and `description` from frontmatter, and its full rendered content is available on click.
