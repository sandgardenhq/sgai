# Agents

Agents are specialised AI workers that perform distinct roles within a workflow. Each agent has its own identity, constraints, and instructions defined in a markdown file. The coordinator dispatches work to agents; agents communicate with each other through a messaging system.

## How agents work

When a session starts, `sgai` reads the `agents` list from `GOAL.md` frontmatter and makes those agents available for dispatch. The coordinator runs first and routes work to other agents by name using inter-agent messaging. Each agent's `.md` file is loaded as its system prompt when that agent is activated.

Agents have access to a shared set of MCP tools for reading and writing files, running shell commands, and communicating with other agents. The permission block in an agent's frontmatter can restrict which tools the agent may use.

## File location

Each agent is a single `.md` file in the workspace `.sgai/agent/` directory:

```
<workspace>/
  .sgai/
    agent/
      coordinator.md
      <agent-name>.md
      <agent-name>.md
```

One file = one agent. The filename (without `.md`) is the agent's name. When `GOAL.md` lists `agents: [coordinator, backend-go-developer]`, `sgai` looks for `coordinator.md` and `backend-go-developer.md` in `.sgai/agent/`.

## File format

Every agent file must begin with a YAML frontmatter block, followed by the agent's system prompt as the body:

```markdown
---
description: <one-line description shown in the web UI>
mode: <primary or all>
permission:
  <capability>: <allow or deny>
---

# Agent Name

System prompt content here...
```

### Frontmatter fields

#### `description`

Type: string — required

A one-line description of the agent's role. This is the text shown in the Agents tab of the web UI and read by other agents when they look up available agents.

#### `mode`

Type: string — `primary` or `all`

- `primary` — the agent runs only when explicitly dispatched to
- `all` — the agent participates in all turns alongside any primary agent

#### `permission`

Type: object — optional

Restricts which tools the agent may use. Any capability not listed is permitted by default. Example values:

```yaml
permission:
  bash: deny                        # cannot run shell commands
  edit: deny                        # cannot edit any file
  doom_loop: deny                   # cannot trigger doom-loop detection bypass
  external_directory: deny          # cannot access paths outside the workspace
  question: deny                    # cannot use ask_user_question
  plan_enter: deny                  # cannot enter plan mode
  plan_exit: deny                   # cannot exit plan mode
  todowrite: deny                   # cannot write to the todo list
  todoread: deny                    # cannot read the todo list
```

The `edit` permission supports path-level granularity:

```yaml
permission:
  edit:
    "*": deny                              # deny all edits by default
    "*/GOAL.md": allow                     # allow editing GOAL.md
    "*/.sgai/PROJECT_MANAGEMENT.md": allow # allow editing this specific file
```

## Built-in agents

`sgai` ships with approximately 20 built-in agents including:

- `coordinator` — orchestrates the workflow; the only agent that communicates with the human
- `general-purpose` — handles tasks that don't fit a specialist agent
- `project-critic-council` — multi-model council that verifies GOAL.md items are genuinely complete
- `retrospective` — analyses session artifacts and produces improvement suggestions
- Language/framework specialists (Go, React, HTMX, shell scripting, etc.)
- Verifier agents for Claude and OpenAI SDK applications

These are stored in `cmd/sgai/skel/.sgai/agent/` and unpacked into every workspace at session start.

## The coordinator

`coordinator.md` is the only protected agent file. When the overlay directory (`<workspace>/sgai/agent/`) is applied at session start, `coordinator.md` is explicitly skipped — the built-in coordinator is never overwritten by an overlay file of the same name.

All other agent files can be overridden by placing a file with the same name in the overlay directory.

## Adding your own agents

### When to add an agent

Add a custom agent when you need a specialist that the built-in agents do not cover. Good candidates:

- A language or framework specialist for a stack not covered by built-ins (e.g. C#, Godot, Rust)
- A domain expert that reads project-specific knowledge files before producing output (e.g. a rules interpreter that reads a game ruleset)
- A read-only reviewer for a technology that the built-in reviewers do not cover
- A QA or validation agent that verifies correctness against known-good data
- A workflow agent that handles project-specific operations (e.g. committing, deploying)

Do not add agents for tasks that `general-purpose` handles adequately, or that the existing specialist agents already cover.

### Where to put custom agents

Write custom agents to the workspace overlay directory:

```
<workspace>/
  sgai/
    agent/
      <agent-name>.md
```

Files in `<workspace>/sgai/agent/` are copied into `<workspace>/.sgai/agent/` when a session starts. The overlay runs after the skeleton unpack, so custom agents are added on top of all built-in agents.

To make an agent immediately available without waiting for a session start, also write it directly to `<workspace>/.sgai/agent/<agent-name>.md`.

### Naming

- Use lowercase kebab-case: `csharp-godot-developer`, `rules-interpreter`
- The filename is the name used in `GOAL.md` and in inter-agent messages — keep it clear and specific
- Avoid names that conflict with built-in agents unless you intend to replace them (only `coordinator.md` is protected from replacement)

### Example agent

```markdown
---
description: Read-only reviewer for C# Godot 4 code. All feedback is mandatory and blocking.
mode: all
permission:
  bash: deny
  edit: deny
  doom_loop: deny
  external_directory: deny
  question: deny
  plan_enter: deny
  plan_exit: deny
---

# C# Godot Reviewer

## MANDATORY REVIEW CONTRACT

Every issue you raise is mandatory. There are no suggestions...
```

## Inter-agent communication

Agents communicate using three MCP tools available to all agents:

- **`sgai_send_message({toAgent: "name", body: "..."})`** — send a message to a named agent
- **`sgai_check_inbox()`** — read messages sent to the current agent
- **`sgai_check_outbox()`** — check messages already sent (to avoid sending duplicates)

The coordinator is the only agent that communicates directly with the human (via `ask_user_question`). Other agents route human communication through the coordinator.

## How agents are loaded at session start

When a session starts in a workspace, `sgai` runs two steps in order:

1. **Skeleton unpack** — all built-in agents from `skel/.sgai/agent/` are copied into `<workspace>/.sgai/agent/`, overwriting any files with matching names
2. **Overlay apply** — all files from `<workspace>/sgai/agent/` are copied into `<workspace>/.sgai/agent/` on top, with one exception: `coordinator.md` is skipped

Custom agents with names not matching any built-in agent are unaffected by step 1. Custom agents with the same name as a built-in agent will replace that agent (since the overlay runs second), except for `coordinator.md`.

## Agents in the web UI

The Agents tab in the sgai web UI reads from `<workspace>/.sgai/agent/` at request time. Each entry shows the agent's filename (as its name) and its `description` frontmatter field. The full agent file content is not rendered in the UI — the description is the only field surfaced there.

## Referencing agents in GOAL.md

Agents must be listed in the `agents` frontmatter field of `GOAL.md` to be available for dispatch during a session:

```yaml
---
agents:
  - coordinator
  - csharp-godot-developer
  - csharp-godot-reviewer
  - rules-interpreter
  - godot-qa-tester
  - git-manager
---
```

An agent file that exists in `.sgai/agent/` but is not listed in `GOAL.md` will not be dispatched to during that session.
