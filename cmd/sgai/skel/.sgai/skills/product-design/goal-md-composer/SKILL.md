---
name: goal-md-composer
description: Interactive wizard to compose valid GOAL.md files for SGAI with step-by-step guidance through 7 phases. Use when creating GOAL.md, configuring agents, model lists, flow DAGs, safety-analysis guidance, or starting an SGAI project. Safety Analysis must use coordinator/reviewer stpa-overview skill guidance, not stpa-analyst flow nodes.
---

# GOAL.md Composer

## Overview

An interactive wizard that guides you through composing valid, well-structured `GOAL.md` files for SGAI. It configures project scope, agents, workflow flow, models, options, and final validation.

**Announce at start:** "I'm using the GOAL.md Composer skill to help you create a valid GOAL.md file."

## Prerequisites

Before starting, load the companion reference when you need the full catalog or examples:

- `skills/product-design/goal-md-composer/REFERENCE.md`

## Non-Negotiable Safety Analysis Rule

Safety Analysis is **not** a workflow agent. Do not add `stpa-analyst` to `flow`, `models`, examples, generated GOAL files, or model recommendations.

When the user wants Safety Analysis, add coordinator/reviewer guidance in the GOAL body instead:

```markdown
## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.
```

## The 7-Phase Process

### Phase 1: Project Description

Ask one question at a time:

1. "What type of project are you building?"
   - Options: API/Backend, CLI Tool, Web Application, Full-Stack App, Documentation, Library/SDK, Other
2. "What programming language(s) will you use?"
   - Options: Go, TypeScript, Python, Shell/Bash, Multiple languages, Other
3. "What's the scope of this project?"
   - Options: Quick prototype, MVP, Production-ready application, Exploratory research
4. "Are there any specific constraints, including safety, security, filesystem, concurrency, or external-input concerns?"

Log answers in `.sgai/PROJECT_MANAGEMENT.md` under `## GOAL.md Composer Session`.

### Phase 2: Agent Recommendations

Recommend agents from the reference catalog.

**Mandatory reviewer pairing:**

| Development Agent | Required Reviewer |
|-------------------|-------------------|
| `backend-go-developer` | `go-readability-reviewer` |
| `htmx-picocss-frontend-developer` | `htmx-picocss-frontend-reviewer` |
| `react-developer` | `react-reviewer` |
| `shell-script-coder` | `shell-script-reviewer` |

Process:

1. Present recommended agents organized by category.
2. For each development agent, show its automatically included reviewer.
3. If Safety Analysis is requested, say: "Safety Analysis will be handled by coordinator/reviewer use of `stpa-overview`; it does not add a workflow agent."
4. Ask: "Do you want to add or remove any agents from this selection?"

Example recommendation for a Go Backend API with Safety Analysis:

```markdown
Recommended agents:
- backend-go-developer (Go development)
- go-readability-reviewer (automatically paired)
- general-purpose (cross-domain tasks)

Safety Analysis:
- The coordinator will be instructed to load/use `stpa-overview` when safety or hazard analysis is relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant it.
```

### Phase 3: Flow Builder

Generate the DOT syntax DAG defining agent execution order and dependencies.

Rules:

1. Development agents always flow into their paired reviewers:
   ```yaml
   flow: |
     "backend-go-developer" -> "go-readability-reviewer"
   ```
2. Do not include `coordinator` in the flow; it is always present automatically.
3. Do not include `stpa-analyst`; Safety Analysis is represented in GOAL body guidance, not the flow DAG.
4. Standalone agents with no dependencies are listed without arrows:
   ```yaml
   flow: |
     "general-purpose"
   ```

Example generated flow:

```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "general-purpose"
```

Ask: "Does this workflow look correct? Would you like to add or modify any dependencies?"

### Phase 4: Model Configuration

Configure per-agent model assignments.

Default recommendations:

- `coordinator`: strongest available model for orchestration
- Development agents: strong coding model
- Review agents: strong reasoning model
- Frontend agents: capable balanced model
- Utility agents: cost-effective model

Do not add a model entry for `stpa-analyst`.

Example:

```yaml
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
```

Ask: "Do you want to adjust any model assignments?"

### Phase 5: Specification Writing

Guide the markdown body. Focus on **what** to build, not implementation details.

Suggested structure:

```markdown
# [Project Title]

[1-2 paragraph description]

## Requirements

- [Behavioral requirement]
- [Constraint]

## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.

## Tasks

- [ ] Task 1
- [ ] Task 2
```

Only include `## Safety Analysis` when requested or warranted by project risk.

### Phase 6: Options

Ask about:

1. **Interactive mode** — `yes`, `no`, or `auto`
2. **Completion gate script** — for example `make test`, `go test ./...`, `npm run lint && npm test`

Example:

```yaml
interactive: yes
completionGateScript: make test
```

### Phase 7: Output & Validation

Generate the complete file:

```markdown
---
flow: |
  [generated flow from Phase 3]
models:
  [generated models from Phase 4]
interactive: [from Phase 6]
completionGateScript: [from Phase 6, if set]
---

[generated specification from Phase 5]
```

Present this validation checklist:

- [ ] All development agents have paired reviewers in the flow
- [ ] Flow has no cycles
- [ ] Coordinator is not in the flow
- [ ] `stpa-analyst` is not in the flow
- [ ] `stpa-analyst` is not in models
- [ ] Safety Analysis, if selected, appears as coordinator/reviewer `stpa-overview` guidance
- [ ] Models are assigned to agents that appear in the flow or to the coordinator
- [ ] Interactive mode is set
- [ ] Specification has clear requirements and checkbox tasks

Ask: "I've generated the GOAL.md. Would you like me to write it to `./GOAL.md`, show the complete file first, or make adjustments?"

## Quick Reference

### Minimal GOAL.md Template

```markdown
---
flow: |
  "general-purpose"
interactive: yes
---

# Project Goal

[Description]

## Requirements

- [Requirement]

## Tasks

- [ ] Task
```

### Full-Featured Template with Safety Analysis

```markdown
---
completionGateScript: make test
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-sonnet-4-5"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
interactive: yes
---

# Project Goal

[Description]

## Requirements

- [Requirement 1]
- [Requirement 2]

## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.

## Tasks

- [ ] Task 1
- [ ] Task 2
```

## Red Flags - STOP

- Adding `stpa-analyst` to `flow`
- Adding `stpa-analyst` to `models`
- Describing Safety Analysis as a terminal workflow agent
- Making reviewers flow into STPA
- Omitting coordinator/reviewer `stpa-overview` guidance after the user requested Safety Analysis

## Remember

- Ask one question at a time during gathering phases
- Auto-enforce reviewer pairing
- Safety Analysis means `stpa-overview` skill guidance, not an agent node
- Log decisions in `.sgai/PROJECT_MANAGEMENT.md`
- Hand control back to the human between phases
- Validate before writing final output
