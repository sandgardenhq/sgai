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
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
```

**Standalone agents (no dependencies):**
```yaml
flow: |
  "general-purpose"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
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
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-sonnet-4-5"
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
| `backend-go-developer` | Expert Go backend developer for APIs, CLI tools, and services with idiomatic Go patterns. Works with go-readability-reviewer for code quality. | `go-readability-reviewer` |
| `htmx-picocss-frontend-developer` | Frontend developer using HTMX and PicoCSS for lightweight web interfaces. No custom JavaScript. | `htmx-picocss-frontend-reviewer` |
| `shell-script-coder` | Production-quality POSIX/bash shell scripts with proper error handling. | `shell-script-reviewer` |
| `react-developer` | Frontend developer specializing in React for building modern, component-based web applications. TypeScript, hooks, modern patterns. | `react-reviewer` |
| `general-purpose` | Cross-domain tasks, research, multi-step operations. No dedicated reviewer. | None |
| `webmaster` | Marketing sites, landing pages with Bootstrap/Tailwind/PicoCSS. SEO and accessibility focus. | None |

### Review Agents

| Agent | Description |
|-------|-------------|
| `go-readability-reviewer` | Reviews Go code for readability, idioms, and best practices. Read-only - sends fixes via messaging. |
| `htmx-picocss-frontend-reviewer` | UI polish, accessibility, visual consistency for HTMX/PicoCSS interfaces. Read-only. |
| `react-reviewer` | React code review for best practices, performance, accessibility, hooks usage, and anti-patterns. Read-only. |
| `shell-script-reviewer` | Shell script correctness, portability, security review. Read-only. |
| `project-critic-council` | Multi-model council that validates completion claims with strict standards. |

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
| `coordinator` | **Always present** - orchestrates workflow, manages tasks, human communication. Never include in flow. |

### SDK Verification Agents

| Agent | Description |
|-------|-------------|
| `agent-sdk-verifier-py` | Validates Python Claude Agent SDK applications. |
| `agent-sdk-verifier-ts` | Validates TypeScript Claude Agent SDK applications. |
| `openai-sdk-verifier-py` | Validates Python OpenAI Agents SDK applications. |
| `openai-sdk-verifier-ts` | Validates TypeScript OpenAI Agents SDK applications. |

### Retrospective Agents

| Agent | Description |
|-------|-------------|
| `retrospective-session-analyzer` | Analyzes sessions for SGAI improvement opportunities. |
| `retrospective-code-analyzer` | Extracts reusable code snippets from sessions. |
| `retrospective-refiner` | Deduplicates and formats improvement suggestions. |
| `retrospective-applier` | Applies approved suggestions via delegation. |
| `skill-writer` | Creates validated skill files from suggestions. |
| `snippet-writer` | Creates code snippet files from suggestions. |

### Utility Agents

| Agent | Description |
|-------|-------------|
| `cli-output-style-adjuster` | Adjusts CLI output for Unix philosophy compliance (minimal, plain-text). |

---

## Mandatory Reviewer Pairing

> **CRITICAL CONSTRAINT:** Every coding agent **must** be paired with its corresponding reviewer agent.

| Development Agent | Required Reviewer |
|-------------------|-------------------|
| `backend-go-developer` | `go-readability-reviewer` |
| `htmx-picocss-frontend-developer` | `htmx-picocss-frontend-reviewer` |
| `react-developer` | `react-reviewer` |
| `shell-script-coder` | `shell-script-reviewer` |

### Enforcement Rules

When selecting a development agent:
1. **Auto-add** the paired reviewer to the agent selection
2. **Auto-create** the dependency edge in the flow DAG
3. **Show notice** explaining why the reviewer was added
4. **Warn** if user tries to remove reviewer while developer is selected

### Flow Pattern

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
```

### Exception: `general-purpose`

The `general-purpose` agent does not have a dedicated reviewer since it handles cross-domain tasks. It may optionally flow into `stpa-analyst` for safety analysis.

---

## Available Models

### Anthropic Models

| Model | Variant | Cost | Description |
|-------|---------|------|-------------|
| `anthropic/claude-opus-4-6` | - | $$$$ | Most capable, best reasoning |
| `anthropic/claude-opus-4-6` | `(max)` | $$$$ | Extended thinking for complex tasks |
| `anthropic/claude-sonnet-4-5` | - | $$$ | Balanced capability and cost |
| `anthropic/claude-sonnet-4-5` | `(max)` | $$$ | Extended thinking variant |

### Google Models

| Model | Cost | Description |
|-------|------|-------------|
| `google/gemini-2.0-flash-001` | $$ | Fast responses, good for simpler tasks |
| `google/gemini-2.5-pro-preview-05-06` | $$$ | Advanced reasoning |

### OpenAI Models

| Model | Cost | Description |
|-------|------|-------------|
| `openai/gpt-4.1` | $$$ | High capability, good reasoning |
| `openai/gpt-4.1-mini` | $$ | Faster, lower cost |
| `openai/o3` | $$$$ | Advanced reasoning model |

### Model Selection Guidelines

| Agent Type | Recommended Model | Reason |
|------------|-------------------|--------|
| Coordinator | `anthropic/claude-opus-4-6 (max)` | Needs best reasoning for orchestration |
| Go Developer | `anthropic/claude-opus-4-6` | Complex code generation |
| Go Reviewer | `anthropic/claude-opus-4-6` | Thorough code analysis |
| Frontend Dev | `anthropic/claude-sonnet-4-5` | Good balance for UI work |
| Frontend Reviewer | `anthropic/claude-opus-4-6` | Detailed visual analysis |
| General Purpose | `anthropic/claude-opus-4-6` | Varied complex tasks |
| STPA Analyst | `anthropic/claude-opus-4-6` | Safety-critical analysis |
| Utility Agents | `anthropic/claude-sonnet-4-5` | Cost-effective for simple tasks |

---

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
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-sonnet-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
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
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "general-purpose": "anthropic/claude-opus-4-6"
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
  "shell-script-coder" -> "shell-script-reviewer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "shell-script-coder": "anthropic/claude-opus-4-6"
  "shell-script-reviewer": "anthropic/claude-opus-4-6"
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
- [ ] **Reviewer pairing enforced** - Every developer agent has its reviewer
- [ ] **Reviewer edges exist** - Developer flows into reviewer (e.g., `"backend-go-developer" -> "go-readability-reviewer"`)
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
  "backend-go-developer" -> "go-readability-reviewer"
```

### Go Backend with Safety Analysis
```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
```

### HTMX Frontend Only
```yaml
flow: |
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
```

### Full-Stack Go + HTMX
```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
```

### React Frontend Only
```yaml
flow: |
  "react-developer" -> "react-reviewer"
```

### Full-Stack Go + React
```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
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
- Ensure reviewers don't flow back to developers
- Use a topological sort tool to verify DAG validity

### "Model not available" Error
- Check model name spelling
- Verify model is available in your opencode configuration
- Try using a different model variant

### Tasks not being tracked
- Ensure tasks use checkbox format: `- [ ]`
- Nested tasks should be indented with 2 spaces
- Don't use other list formats (*, 1., etc.) for trackable tasks
