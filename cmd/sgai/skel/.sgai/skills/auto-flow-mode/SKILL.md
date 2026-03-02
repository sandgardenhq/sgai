---
name: auto-flow-mode
description: Use when GOAL.md has no flow defined or flow is set to "auto" - guides coordinator through surveying agents, analyzing workspace technologies, selecting and pairing agents, and updating GOAL.md with the flow configuration
---

# Auto-Flow Mode

## Overview

When GOAL.md has no explicit agent flow configured (empty frontmatter or `flow: "auto"`), this skill guides the coordinator through automatically detecting the right agents for the project and writing the flow configuration.

**Core principle:** Survey the available agents, analyze the workspace, pick agents that match the technologies present, pair developers with reviewers, and write the configuration to GOAL.md so the DAG hot-reload picks it up.

## When to Use

- Use when GOAL.md has no frontmatter at all
- Use when GOAL.md frontmatter has no `flow:` key
- Use when GOAL.md frontmatter has `flow: "auto"` or `flow: auto`
- Use when the system nudge tells you: "The GOAL.md has no explicit agent flow configured"

**Do NOT use when:**
- GOAL.md already has a valid `flow:` with agent edges defined
- The human partner has explicitly configured the flow

## Process

### Step 1: Survey Available Agents

Read all agent definition files from `.sgai/agent/` directory to build a catalog.

- [ ] Use the Task tool with `explore` subagent type to list and read all `.sgai/agent/*.md` files
- [ ] For each agent file, extract the YAML frontmatter fields:
  - `description` - what the agent does
  - `mode` - either `primary` (developer/coder) or `all` (reviewer)
- [ ] Build a table of agents with their capabilities:

```
| Agent Name | Mode | Description |
|------------|------|-------------|
| backend-go-developer | primary | Expert Go backend developer... |
| go-readability-reviewer | all | Reviews Go code... |
| ... | ... | ... |
```

### Step 2: Analyze Workspace

Scan the workspace to determine what technologies are in use.

- [ ] Use the Task tool with `explore` subagent type to analyze the workspace:
  - File extensions present (`.go`, `.tsx`, `.jsx`, `.ts`, `.js`, `.py`, `.sh`, `.html`, `.css`, etc.)
  - Build files (Makefile, package.json, go.mod, pyproject.toml, Cargo.toml, etc.)
  - Project structure (cmd/, internal/, src/, components/, etc.)
  - Frameworks and libraries (check imports, dependencies)
- [ ] Summarize the technologies detected

### Step 3: Select Agents

Based on the workspace analysis, select agents that match the detected technologies.

- [ ] Apply the following technology-to-agent mapping as guidance:

| Technology Signal | Developer Agent | Notes |
|-------------------|----------------|-------|
| `.go` files, `go.mod`, `Makefile` | `backend-go-developer` | Go backend code |
| `.tsx`, `.jsx`, `package.json` with React | `react-developer` | React frontend code |
| `.html` with HTMX attributes, PicoCSS | `htmx-picocss-frontend-developer` | HTMX+Pico frontend |
| `.sh`, `.bash` scripts, shell-heavy Makefiles | `shell-script-coder` | Shell scripting |
| `.py` files, `pyproject.toml` | Consider `general-purpose` | Python work |
| Mixed or unclear | `general-purpose` | Catch-all |

- [ ] Always include `general-purpose` as a catch-all agent
- [ ] If no technology is identifiable at all, default to just `general-purpose`
- [ ] Use your judgment — the table above is guidance, not exhaustive. If the workspace has technologies that match other agents in the catalog, include them.

### Step 4: Pair Developers with Reviewers

For each selected developer agent, find and include its corresponding reviewer.

- [ ] Apply the known developer-reviewer pairings:

| Developer Agent | Reviewer Agent |
|----------------|---------------|
| `backend-go-developer` | `go-readability-reviewer` |
| `react-developer` | `react-reviewer` |
| `htmx-picocss-frontend-developer` | `htmx-picocss-frontend-reviewer` |
| `shell-script-coder` | `shell-script-reviewer` |

- [ ] For each pairing, create a flow edge: `"developer-agent" -> "reviewer-agent"`
- [ ] If a developer agent has no known reviewer, include it as a standalone node: `"agent-name"`
- [ ] The `general-purpose` agent has no dedicated reviewer — include it as a standalone node

### Step 5: Set Default Models

Assign a default model to each selected agent.

- [ ] Use `anthropic/claude-sonnet-4-6` as the default model for all agents
- [ ] Use `anthropic/claude-sonnet-4-6` for the coordinator as well
- [ ] Format each entry as: `"agent-name": "anthropic/claude-sonnet-4-6"`

### Step 6: Update GOAL.md

Write the flow configuration into GOAL.md's YAML frontmatter.

- [ ] **CRITICAL: Preserve ALL existing body content** — everything after the frontmatter closing `---` must remain unchanged
- [ ] Read the current GOAL.md content
- [ ] Construct the new frontmatter with:
  - `flow: |` followed by indented edge specifications
  - `models:` section with each agent and its model
- [ ] If GOAL.md already has frontmatter (even if empty/auto), replace only the frontmatter
- [ ] If GOAL.md has no frontmatter, add it at the top

**Flow syntax format** (DOT-like edge specification):
```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
```

**Models syntax format:**
```yaml
models:
  "coordinator": "anthropic/claude-sonnet-4-6"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-sonnet-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
```

**Complete frontmatter example:**
```yaml
---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
models:
  "coordinator": "anthropic/claude-sonnet-4-6"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-sonnet-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
---

[existing GOAL.md body content preserved here]
```

### Step 7: Signal Completion

- [ ] Set `agent-done` status so the DAG hot-reload picks up the new configuration
- [ ] The system will automatically rebuild the DAG from the updated GOAL.md on the next iteration

## Rules

1. **Always preserve body content** — The GOAL.md body (everything after the frontmatter) contains the user's actual goals. Losing it is unacceptable.
2. **Always include coordinator in models** — The coordinator is always part of the flow (the system injects coordinator edges automatically).
3. **Always include general-purpose** — It serves as a catch-all for work that doesn't match specialized agents.
4. **Don't over-select agents** — Only include agents relevant to the detected technologies. Including unnecessary agents adds noise to the workflow.
5. **Don't under-select agents** — If a technology is clearly present, include its matching agent. Missing agents means work won't get done.
6. **Reviewer comes with developer** — Never include a reviewer without its corresponding developer, and always include the reviewer when including the developer.
7. **Use the explore subagent** — Don't try to guess what technologies are present. Actually scan the workspace using the Task tool with `explore` subagent type.

## Automatic Injections

The system automatically adds these to the DAG regardless of what you write in the flow:
- `coordinator -> <all entry nodes>` — coordinator is always the entry point
- `coordinator -> project-critic-council` — always injected for quality gates

You do NOT need to include these edges in the flow specification. They are handled by the system.

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "GOAL.md already tells me what technologies are used" | GOAL.md describes goals, not the full technology stack. The workspace may have shell scripts, Makefiles, or frameworks not mentioned in the goals. Always scan. |
| "I'll skip reviewers to save time on this urgent fix" | Urgency makes reviewers MORE important, not less. Auth bugs and security issues need a second pair of eyes. |
| "Including all agents is safer than missing one" | Extra agents add noise and wasted cycles. Be precise. Only include agents for detected technologies. |
| "This is just a small project, I can eyeball it" | Even small projects have build files and dependencies that reveal technology choices. Use the explore subagent. |
| "I'll add the reviewer later if needed" | "Later" never comes. Pair them now. The flow is written once and reused for the entire session. |

## Red Flags - STOP

- You're about to write a flow without scanning the workspace first — **STOP and analyze**
- You're including agents that don't match any detected technology — **STOP and reconsider**
- You're writing the frontmatter and realize you might lose the body content — **STOP and verify**
- You're skipping the reviewer for a developer agent that has one — **STOP and pair them**
- You're about to hardcode specific agent names without surveying `.sgai/agent/` first — **STOP and survey**

## Examples

### Good Example: Go + React Project

Workspace analysis reveals: `go.mod`, `.go` files in `cmd/` and `internal/`, `package.json` with React, `.tsx` files in `src/`.

```yaml
---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
models:
  "coordinator": "anthropic/claude-sonnet-4-6"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-sonnet-4-6"
  "react-developer": "anthropic/claude-sonnet-4-6"
  "react-reviewer": "anthropic/claude-sonnet-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
---
```

### Good Example: Pure Go Project

Workspace analysis reveals: only `go.mod`, `.go` files, `Makefile`.

```yaml
---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "general-purpose"
models:
  "coordinator": "anthropic/claude-sonnet-4-6"
  "backend-go-developer": "anthropic/claude-sonnet-4-6"
  "go-readability-reviewer": "anthropic/claude-sonnet-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
---
```

### Good Example: Unknown/Empty Workspace

Workspace analysis reveals nothing identifiable.

```yaml
---
flow: |
  "general-purpose"
models:
  "coordinator": "anthropic/claude-sonnet-4-6"
  "general-purpose": "anthropic/claude-sonnet-4-6"
---
```

### Bad Example: Including Everything

**DON'T** include all 28 agents because "more is better":

```yaml
# BAD - includes agents for technologies not present in workspace
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "shell-script-coder" -> "shell-script-reviewer"
  "stpa-analyst"
  "general-purpose"
```

This wastes agent cycles on technologies that aren't in the workspace.

### Bad Example: Missing Reviewer

**DON'T** include a developer without its reviewer:

```yaml
# BAD - backend-go-developer has no paired reviewer
flow: |
  "backend-go-developer"
  "general-purpose"
```

The reviewer is essential for code quality.

### Bad Example: Losing Body Content

**DON'T** write only frontmatter and lose the goals:

```yaml
# BAD - where did the user's goals go?
---
flow: |
  "general-purpose"
models:
  "coordinator": "anthropic/claude-sonnet-4-6"
---
```

The body content MUST be preserved after the closing `---`.

## Checklist

Before completing, verify:

- [ ] Surveyed `.sgai/agent/` directory for available agents
- [ ] Analyzed workspace for technologies using explore subagent
- [ ] Selected agents matching detected technologies
- [ ] Paired every developer with its reviewer
- [ ] Included `general-purpose` as catch-all
- [ ] Included `coordinator` in models section
- [ ] Used `anthropic/claude-sonnet-4-6` as default model
- [ ] Preserved ALL existing GOAL.md body content
- [ ] Flow syntax uses correct DOT-like format with quoted agent names
- [ ] Set `agent-done` status after writing GOAL.md
