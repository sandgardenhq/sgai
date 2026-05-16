# GOAL.md Composer Reference

Complete reference documentation for composing GOAL.md files for SGAI.

---

## GOAL.md Format Specification

### Structure

A GOAL.md file consists of two parts:

```markdown
---
<YAML frontmatter>
---

<Markdown body: project description, requirements, tasks>
```

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow` | string (DOT format) | **Yes** | Directed acyclic graph defining agent execution order and dependencies |
| `models` | map[string]string | No | Per-agent model assignments with optional variant syntax |
| `completionGateScript` | string | No | Shell command that must succeed for workflow to be considered complete |
| `interactive` | string | No | Human interaction mode: `yes`, `no`, or `auto` |

---

## Frontmatter Field Details

### `flow` (Required)

DOT-format DAG defining which agents run and their dependencies.

**Syntax:**
- Edges: `"agent-a" -> "agent-b"` (agent-a must complete before agent-b starts)
- Standalone: `"agent-name"` (agent with no dependencies)
- Use double quotes around agent names
- Use `|` for multi-line YAML strings

**Example:**
```yaml
flow: |
  "go" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss" -> "stpa-analyst"
```

**Standalone agents (no dependencies):**
```yaml
flow: |
  "general-purpose"
  "htmx-picocss"
```

**Rules:**
- The graph must be a DAG (Directed Acyclic Graph) - no cycles allowed
- The `coordinator` agent is ALWAYS present automatically - do NOT include it in the flow
- All agents listed in the flow will be available for task delegation

---

### `models` (Optional)

Per-agent model assignments. Supports variant syntax in parentheses.

**Example:**
```yaml
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5"
  "general-purpose": "openai/gpt-5.5"
  "htmx-picocss": "openai/gpt-5.5"
```

**Notes:**
- Agents without explicit model assignments use `defaultModel` from `sgai.json` or system default
- Variant syntax (e.g., `(max)`) is passed through to the inference engine
- The `coordinator` should typically use the most capable model

---

### `completionGateScript` (Optional)

Shell command that determines if the workflow is complete. The workflow is only considered complete when this script exits with status 0.

**Examples:**
```yaml
completionGateScript: make test
```

```yaml
completionGateScript: go test ./... && npm run lint
```

```yaml
completionGateScript: ./scripts/verify-all.sh
```

---

### `interactive` (Optional)

Controls how agent questions are handled.

| Value | Behavior |
|-------|----------|
| `yes` | Agent questions appear in web UI; human responds interactively (recommended) |
| `no` | Workflow exits when an agent asks a question |
| `auto` | Self-driving mode; agents attempt to proceed without human input |

**Example:**
```yaml
interactive: yes
```

---

## Markdown Body

The body contains the project specification in markdown.

**Structure:**
```markdown
# Project Goal

Describe what you want to build here. Be specific about behavior,
not implementation. Focus on outcomes.

## Requirements

- What should happen when a user does X?
- What constraints exist?
- What does success look like?

## Tasks

- [ ] Task 1
- [ ] Task 2
  - [ ] Task 2.1
- [ ] Task 3
```

**Key Points:**
- Checkboxes (`- [ ]`) are managed by the coordinator agent
- Nested checkboxes are supported for subtasks
- Focus on *what* to build, not *how*
- Be specific about behavior and constraints

---

## Available Agents

### Development Agents

| Agent | Description | Paired Reviewer |
|-------|-------------|-----------------|
| `go` | Primary Go wrapper that delegates implementation to `go-developer` and review to `go-reviewer`. | Handled internally |
| `htmx-picocss` | Primary HTMX/PicoCSS wrapper that delegates implementation to `htmx-picocss-developer` and review to `htmx-picocss-reviewer`. | Handled internally |
| `shell-script` | Primary shell script wrapper that delegates implementation to `shell-script-developer` and review to `shell-script-reviewer`. | Handled internally |
| `react` | Primary React wrapper that delegates implementation to `react-developer` and review to `react-reviewer`. | Handled internally |
| `general-purpose` | Cross-domain tasks, research, multi-step operations. No dedicated reviewer. | None |
| `webmaster` | Marketing sites, landing pages with Bootstrap/Tailwind/PicoCSS. SEO and accessibility focus. | None |

### Analysis Agents

| Agent | Description |
|-------|-------------|
| `stpa-analyst` | STPA hazard analysis for safety-critical software, physical, and AI systems. |
| `c4-code` | C4 Code-level documentation from source files (lowest C4 level). |
| `c4-component` | C4 Component-level architecture synthesis from code documentation. |
| `c4-container` | C4 Container-level deployment documentation. |
| `c4-context` | C4 System context diagrams for stakeholders (highest C4 level). |

### Coordination

| Agent | Description |
|-------|-------------|
| `coordinator` | **Always present** - orchestrates workflow, manages tasks, human communication, and owns the internal completion review gate. Never include in flow. |

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
| `cli-output-style-adjuster` | Adjusts CLI output for Unix philosophy compliance (minimal, plain-text). |

---

## Mandatory Coding Wrappers

> **CRITICAL CONSTRAINT:** Coding work must use public wrapper agents. Wrapper agents delegate to hidden developer and reviewer subagents internally.

| Coding Capability | Public Wrapper |
|-------------------|-------------------|
| Go | `go` |
| HTMX/PicoCSS | `htmx-picocss` |
| React | `react` |
| Shell scripts | `shell-script` |
| Websites | `webmaster` |

### Enforcement Rules

When selecting a coding capability:
1. **Select** the public wrapper agent only
2. **Do not add** hidden developer or reviewer subagents to the GOAL.md flow
3. **Do not assign** models to hidden developer or reviewer subagents in GOAL.md
4. **Show notice** explaining that implementation and review are handled internally by the wrapper

### Flow Pattern

```yaml
flow: |
  "go"
```

### Exception: `general-purpose`

The `general-purpose` agent does not have a dedicated reviewer since it handles cross-domain tasks. It may optionally flow into `stpa-analyst` for safety analysis.

---

## Available Models

### Google Models

| Model | Cost | Description |
|-------|------|-------------|
| `google/gemini-2.0-flash-001` | $$ | Fast responses, good for simpler tasks |
| `google/gemini-2.5-pro-preview-05-06` | $$$ | Advanced reasoning |

### OpenAI Models

| Model | Cost | Description |
|-------|------|-------------|
| `openai/gpt-5.5` | $$$ | High capability, good reasoning |
| `openai/gpt-5.4-mini` | $$ | Faster, lower cost |
| `openai/gpt-5.5 (xhigh)` | $$$$ | Advanced reasoning model |

### Model Selection Guidelines

| Agent Type | Recommended Model | Reason |
|------------|-------------------|--------|
| Coordinator | `openai/gpt-5.5 (xhigh)` | Needs best reasoning for orchestration |
| Go Developer | `openai/gpt-5.5` | Complex code generation |
| Go Reviewer | `openai/gpt-5.5` | Thorough code analysis |
| Frontend Dev | `openai/gpt-5.5` | Good balance for UI work |
| Frontend Reviewer | `openai/gpt-5.5` | Detailed visual analysis |
| General Purpose | `openai/gpt-5.5` | Varied complex tasks |
| STPA Analyst | `openai/gpt-5.5` | Safety-critical analysis |
| Utility Agents | `openai/gpt-5.5` | Cost-effective for simple tasks |

---

## Complete Examples

### Example 1: Simple Go CLI Tool

```markdown
---
flow: |
  "go"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5"
interactive: yes
completionGateScript: go test ./...
---

# JSON Validator CLI

Create a command-line tool that validates JSON files against a schema.

## Requirements

- Accept JSON file path and schema file path as arguments
- Support JSON Schema draft-07
- Output validation errors with line numbers
- Exit with code 0 on valid, 1 on invalid, 2 on error
- No external dependencies beyond standard library

## Tasks

- [ ] Create main.go with argument parsing
- [ ] Implement JSON schema validation
- [ ] Add error formatting with line numbers
- [ ] Write unit tests
- [ ] Add integration tests with sample files
```

### Example 2: Full-Stack Web Application

```markdown
---
completionGateScript: make test
flow: |
  "go" -> "stpa-analyst"
  "htmx-picocss" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5"
  "htmx-picocss": "openai/gpt-5.5"
  "general-purpose": "openai/gpt-5.5"
  "stpa-analyst": "openai/gpt-5.5"
interactive: yes
---

# Task Management Dashboard

Build a web-based task management application for small teams.

## Requirements

- Users can create, edit, and delete tasks
- Tasks have title, description, status (todo/in-progress/done), and assignee
- Dashboard shows tasks grouped by status in Kanban-style columns
- Drag-and-drop to change task status (using HTMX)
- Simple authentication with username/password
- Data persisted to SQLite database
- Responsive design that works on mobile

## Tasks

- [ ] Set up Go backend with HTTP server
- [ ] Create SQLite database schema
- [ ] Implement user authentication
  - [ ] Login/logout endpoints
  - [ ] Session management
- [ ] Build task CRUD API endpoints
- [ ] Create HTMX frontend
  - [ ] Kanban board layout
  - [ ] Task cards with drag-and-drop
  - [ ] Create/edit task modals
- [ ] Add Playwright tests for UI
- [ ] Write integration tests for API
```

### Example 3: Exploratory Research

```markdown
---
flow: |
  "general-purpose"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "general-purpose": "openai/gpt-5.5"
interactive: yes
---

# API Design Research

Research and document best practices for REST API versioning.

## Requirements

- Survey common versioning strategies (URL, header, query param)
- Document pros/cons of each approach
- Find real-world examples from major APIs
- Recommend approach for our use case (internal microservices)

## Tasks

- [ ] Research URL-based versioning (/v1/, /v2/)
- [ ] Research header-based versioning (Accept-Version)
- [ ] Research query parameter versioning (?version=1)
- [ ] Document findings with examples
- [ ] Write recommendation document
```

### Example 4: Shell Script Project

```markdown
---
flow: |
  "shell-script"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "shell-script": "openai/gpt-5.5"
interactive: yes
completionGateScript: shellcheck scripts/*.sh
---

# Deployment Scripts

Create deployment automation scripts for our Go application.

## Requirements

- POSIX-compliant for maximum portability
- Support for staging and production environments
- Rollback capability if deployment fails
- Health check verification after deployment
- Logging to both file and stdout

## Tasks

- [ ] Create deploy.sh with environment selection
- [ ] Add rollback.sh for failed deployments
- [ ] Create healthcheck.sh for verification
- [ ] Add common.sh with shared functions
- [ ] Document usage in README
```

---

## Validation Checklist

Before finalizing a GOAL.md, verify:

- [ ] **Flow DAG is valid** - No cycles, all edges point forward
- [ ] **Coordinator not in flow** - It's always present automatically
- [ ] **Wrapper pairing enforced** - Coding work uses wrapper agents that handle implementation and review internally
- [ ] **Review path exists** - Coding wrapper agents delegate to hidden developer and reviewer subagents internally
- [ ] **Models assigned correctly** - All agents in flow have model assignments (or use defaults)
- [ ] **Interactive mode set** - Explicitly set to `yes`, `no`, or `auto`
- [ ] **Specification complete** - Has Goal description, Requirements, and Tasks
- [ ] **Tasks use checkboxes** - Format: `- [ ] Task description`
- [ ] **No implementation details** - Focus on WHAT, not HOW
- [ ] **CompletionGateScript valid** - If set, command should be runnable

---

## Common Patterns

### Go Backend Only
```yaml
flow: |
  "go"
```

### Go Backend with Safety Analysis
```yaml
flow: |
  "go" -> "stpa-analyst"
```

### HTMX Frontend Only
```yaml
flow: |
  "htmx-picocss"
```

### Full-Stack Go + HTMX
```yaml
flow: |
  "go" -> "stpa-analyst"
  "htmx-picocss" -> "stpa-analyst"
```

### React Frontend Only
```yaml
flow: |
  "react"
```

### Full-Stack Go + React
```yaml
flow: |
  "go" -> "stpa-analyst"
  "react" -> "stpa-analyst"
```

### Research/Exploration
```yaml
flow: |
  "general-purpose"
```

### Documentation
```yaml
flow: |
  "c4-code" -> "c4-component"
  "c4-component" -> "c4-container"
  "c4-container" -> "c4-context"
```

---

## Troubleshooting

### "Agent not found" Error
- Verify agent name is spelled correctly (case-sensitive)
- Check that agent is listed in the Available Agents section
- Ensure agent name is in double quotes in the flow

### "Cycle detected" Error
- Review the flow for circular dependencies
- Ensure terminal analysis agents do not flow back to coding wrappers
- Use a topological sort tool to verify DAG validity

### "Model not available" Error
- Check model name spelling
- Verify model is available in your opencode configuration
- Try using a different model variant

### Tasks not being tracked
- Ensure tasks use checkbox format: `- [ ]`
- Nested tasks should be indented with 2 spaces
- Don't use other list formats (*, 1., etc.) for trackable tasks
