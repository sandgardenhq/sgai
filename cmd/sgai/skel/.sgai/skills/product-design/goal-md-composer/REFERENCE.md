# GOAL.md Composer Reference

Complete reference documentation for composing `GOAL.md` files for SGAI.

## GOAL.md Format Specification

A `GOAL.md` file consists of YAML frontmatter followed by a markdown project specification.

```markdown
---
flow: |
  "agent-a" -> "agent-b"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
interactive: yes
completionGateScript: make test
---

# Project Goal

[Goal, requirements, tasks]
```

## Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow` | string (DOT format) | Yes | Directed acyclic graph defining routable agents and dependencies |
| `models` | map | No | Per-agent model assignments with optional variant syntax |
| `completionGateScript` | string | No | Shell command that must succeed for workflow completion |
| `interactive` | string | No | Human interaction mode: `yes`, `no`, or `auto` |

## Flow Rules

- Edges: `"agent-a" -> "agent-b"`
- Standalone: `"agent-name"`
- Use double quotes around agent names
- The graph must be a DAG with no cycles
- The `coordinator` is always present automatically; do not include it in `flow`
- Safety Analysis is a coordinator/reviewer `stpa-overview` skill workflow; do not include `stpa-analyst` in `flow`

Example:

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "general-purpose"
```

## Models

Example:

```yaml
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
```

Notes:

- Agents without explicit model assignments use defaults.
- Variant syntax such as `(max)` is passed through to the inference engine.
- The `coordinator` should typically use the strongest model.
- Do not assign a model to `stpa-analyst`.

## Available Agents

### Development Agents

| Agent | Description | Paired Reviewer |
|-------|-------------|-----------------|
| `backend-go-developer` | Expert Go backend developer for APIs, CLI tools, and services. | `go-readability-reviewer` |
| `htmx-picocss-frontend-developer` | Frontend developer using HTMX and PicoCSS. | `htmx-picocss-frontend-reviewer` |
| `shell-script-coder` | Production-quality POSIX/bash shell scripts. | `shell-script-reviewer` |
| `react-developer` | React/TypeScript frontend developer. | `react-reviewer` |
| `general-purpose` | Cross-domain tasks, research, and multi-step work. | None |
| `webmaster` | Marketing sites, landing pages, SEO, and accessibility. | None |

### Review Agents

| Agent | Description |
|-------|-------------|
| `go-readability-reviewer` | Reviews Go code for readability, idioms, and best practices. |
| `htmx-picocss-frontend-reviewer` | UI polish, accessibility, and visual consistency for HTMX/PicoCSS. |
| `react-reviewer` | React code review for best practices, performance, accessibility, and hooks usage. |
| `shell-script-reviewer` | Shell script correctness, portability, and security review. |

Any `*-reviewer` may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.

### Skill Workflows

| Skill | Description |
|-------|-------------|
| `stpa-overview` | Coordinator/reviewer STPA hazard and safety analysis workflow. Use for Safety Analysis instead of adding a routable agent. |

### Coordination

| Agent | Description |
|-------|-------------|
| `coordinator` | Always present. Orchestrates workflow, manages tasks, and communicates with the human. Never include in `flow`. |

### SDK Verification Agents

| Agent | Description |
|-------|-------------|
| `agent-sdk-verifier-py` | Validates Python Claude Agent SDK applications. |
| `agent-sdk-verifier-ts` | Validates TypeScript Claude Agent SDK applications. |
| `openai-sdk-verifier-py` | Validates Python OpenAI Agents SDK applications. |
| `openai-sdk-verifier-ts` | Validates TypeScript OpenAI Agents SDK applications. |

### Utility Agents

| Agent | Description |
|-------|-------------|
| `cli-output-style-adjuster` | Adjusts CLI output for minimal, plain-text style. |

## Mandatory Reviewer Pairing

Every coding agent must be paired with its corresponding reviewer.

| Development Agent | Required Reviewer |
|-------------------|-------------------|
| `backend-go-developer` | `go-readability-reviewer` |
| `htmx-picocss-frontend-developer` | `htmx-picocss-frontend-reviewer` |
| `react-developer` | `react-reviewer` |
| `shell-script-coder` | `shell-script-reviewer` |

Rules:

1. Auto-add the paired reviewer when selecting a development agent.
2. Auto-create the dependency edge from developer to reviewer.
3. Show a notice explaining why the reviewer was added.
4. Warn if the user tries to remove the required reviewer.

Pattern:

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
```

`general-purpose` has no dedicated reviewer. If Safety Analysis is needed, add `## Safety Analysis` guidance to the GOAL body rather than routing it to another agent.

## Safety Analysis Pattern

Safety Analysis must be represented as instructions, not an agent node.

```markdown
## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.
```

Use this pattern when the project involves safety-critical behavior, physical systems, AI autonomy, external input, filesystem writes, concurrency, state machines, workflow completion, deployment, or risk assessment.

## Model Selection Guidelines

| Agent Type | Recommended Model | Reason |
|------------|-------------------|--------|
| Coordinator | Strongest available model, often `(max)` | Orchestration and safety-gate reasoning |
| Go Developer | Strong coding model | Complex code generation |
| Go Reviewer | Strong reasoning model | Thorough code analysis |
| Frontend Developer | Balanced capable model | UI implementation |
| Frontend Reviewer | Strong reasoning/model with visual analysis capability | Detailed review |
| General Purpose | Strong flexible model | Varied tasks |
| Utility Agents | Cost-effective model | Simple targeted work |

## Complete Examples

### Example 1: Simple Go CLI Tool

```markdown
---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
interactive: yes
completionGateScript: go test ./...
---

# JSON Validator CLI

Create a command-line tool that validates JSON files against a schema.

## Requirements

- Accept JSON file path and schema file path as arguments
- Output validation errors with line numbers
- Exit with code 0 on valid, 1 on invalid, 2 on error

## Tasks

- [ ] Create main.go with argument parsing
- [ ] Implement validation
- [ ] Write tests
```

### Example 2: Full-Stack Web Application with Safety Analysis

```markdown
---
completionGateScript: make test
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "general-purpose"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-sonnet-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
interactive: yes
---

# Task Management Dashboard

Build a web-based task management application for small teams.

## Requirements

- Users can create, edit, and delete tasks
- Drag-and-drop changes task status
- Data persists to SQLite

## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.

## Tasks

- [ ] Set up Go backend
- [ ] Build HTMX frontend
- [ ] Add Playwright tests
```

### Example 3: Exploratory Research

```markdown
---
flow: |
  "general-purpose"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "general-purpose": "anthropic/claude-opus-4-6"
interactive: yes
---

# API Design Research

Research and document best practices for REST API versioning.

## Requirements

- Survey common versioning strategies
- Document pros and cons
- Recommend an approach

## Tasks

- [ ] Research URL-based versioning
- [ ] Research header-based versioning
- [ ] Write recommendation
```

### Example 4: Shell Script Project

```markdown
---
flow: |
  "shell-script-coder" -> "shell-script-reviewer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "shell-script-coder": "anthropic/claude-opus-4-6"
  "shell-script-reviewer": "anthropic/claude-opus-4-6"
interactive: yes
completionGateScript: shellcheck scripts/*.sh
---

# Deployment Scripts

Create deployment automation scripts for a Go application.

## Requirements

- POSIX-compliant where possible
- Support staging and production
- Rollback capability

## Tasks

- [ ] Create deploy.sh
- [ ] Add rollback.sh
- [ ] Document usage
```

## Common Patterns

### Go Backend Only

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
```

### Go Backend with Safety Analysis

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
```

Add `## Safety Analysis` body guidance using `stpa-overview`.

### HTMX Frontend Only

```yaml
flow: |
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
```

### Full-Stack Go + HTMX with Safety Analysis

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
```

Add `## Safety Analysis` body guidance using `stpa-overview`.

### React Frontend Only

```yaml
flow: |
  "react-developer" -> "react-reviewer"
```

### Full-Stack Go + React with Safety Analysis

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
```

Add `## Safety Analysis` body guidance using `stpa-overview`.

### Documentation

```yaml
flow: |
  "c4-code" -> "c4-component"
  "c4-component" -> "c4-container"
  "c4-container" -> "c4-context"
```

## Validation Checklist

Before finalizing a GOAL.md, verify:

- [ ] Flow DAG is valid and acyclic
- [ ] Coordinator is not in flow
- [ ] `stpa-analyst` is not in flow
- [ ] Reviewer pairing is enforced
- [ ] Reviewer edges exist from developer to reviewer
- [ ] Models are assigned correctly
- [ ] `stpa-analyst` is not in models
- [ ] Interactive mode is set
- [ ] Specification is complete
- [ ] Tasks use checkbox format
- [ ] Safety Analysis, if needed, is represented as `stpa-overview` coordinator/reviewer guidance
- [ ] Completion gate command is runnable if present

## Troubleshooting

### "Agent not found" Error

- Verify agent name spelling and quotes.
- Check that the agent is listed in Available Agents.
- If the missing agent is `stpa-analyst`, remove it from `flow`/`models` and add `stpa-overview` Safety Analysis guidance instead.

### "Cycle detected" Error

- Review the flow for circular dependencies.
- Ensure reviewers do not flow back to developers.
- Keep Safety Analysis out of the DAG.

### Tasks not being tracked

- Ensure tasks use checkbox format: `- [ ] Task description`.
- Nested tasks should be indented with two spaces.
- Do not use other list formats for trackable tasks.
