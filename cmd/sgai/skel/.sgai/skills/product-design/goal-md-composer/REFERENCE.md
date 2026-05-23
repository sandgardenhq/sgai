# GOAL.md Composer Reference

Complete reference documentation for composing `GOAL.md` files for SGAI.

## Models

Recommended GPT-5.5 split:

```yaml
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5 (low)"
  "react": "openai/gpt-5.5 (low)"
  "general-purpose": "openai/gpt-5.5 (low)"
```

Notes:

- Agents without explicit model assignments use defaults.
- Variant syntax such as `(xhigh)` and `(low)` is passed through to the inference engine.
- The `coordinator` should typically use `openai/gpt-5.5 (xhigh)` because orchestration and safety-gate reasoning dominate workflow quality.
- Non-coordinator implementation, reviewer, utility, specialist, and general-purpose agents should default to `openai/gpt-5.5 (low)` for GPT-5.5 cost/performance efficiency.
- Users can still override any agent model in `GOAL.md`.
- Aliases remain the recommended way to run the same role at multiple cost tiers.
- Do not assign a model to `stpa-analyst`.

## Model Selection Guidelines

| Agent Type | Recommended Model | Reason |
|------------|-------------------|--------|
| Coordinator | `openai/gpt-5.5 (xhigh)` | Orchestration and safety-gate reasoning |
| Go Developer | `openai/gpt-5.5 (low)` | Cost-conscious GPT-5.5 implementation baseline |
| Go Reviewer | `openai/gpt-5.5 (low)` | Routine review baseline; override only for exceptional workflows |
| Frontend Developer | `openai/gpt-5.5 (low)` | Cost-conscious GPT-5.5 UI implementation baseline |
| Frontend Reviewer | `openai/gpt-5.5 (low)` | Routine review baseline; override only for exceptional workflows |
| General Purpose | `openai/gpt-5.5 (low)` | Cross-domain default baseline for routine work |
| Utility Agents | `openai/gpt-5.5 (low)` | Simple targeted work |

## Complete Examples

### Simple Go Workflow

```markdown
---
flow: |
  "go"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5 (low)"
interactive: yes
completionGateScript: go test ./...
---

# JSON Validator CLI

Create a command-line tool that validates JSON files against a schema.

## Tasks

- [ ] Create main.go with argument parsing
- [ ] Implement validation
- [ ] Write tests
```

### Full-Stack Workflow with Safety Analysis

```markdown
---
completionGateScript: make test
flow: |
  "go"
  "react"
  "general-purpose"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5 (low)"
  "react": "openai/gpt-5.5 (low)"
  "general-purpose": "openai/gpt-5.5 (low)"
interactive: yes
---

# Task Management Dashboard

Build a web-based task management application for small teams.

## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.

## Tasks

- [ ] Set up Go backend
- [ ] Build React frontend
- [ ] Add Playwright tests
```

### Exploratory Research

```markdown
---
flow: |
  "general-purpose"
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "general-purpose": "openai/gpt-5.5 (low)"
interactive: yes
---

# API Design Research

Research and document best practices for REST API versioning.
```
