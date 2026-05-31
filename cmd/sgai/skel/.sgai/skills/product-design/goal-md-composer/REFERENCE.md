# GOAL.md Composer Reference

Complete reference documentation for composing `GOAL.md` files for SGAI.

## Model and Agents Configuration

GOAL.md uses a single `model` and an `agents` list of delegate agents. The coordinator is implicit and runs at the top level with the configured model; the OpenCode runtime handles subagent delegation.

Recommended GPT-5.5 setup:

```yaml
agents:
  - "go"
  - "react"
  - "general-purpose"
model: "openai/gpt-5.5 (xhigh)"
```

Notes:

- `agents` lists the non-coordinator agents available for OpenCode subagent delegation.
- `model` is a single model string used by the coordinator; the OpenCode runtime manages model propagation to subagents.
- Variant syntax such as `(xhigh)` and `(low)` is passed through to the inference engine.
- The coordinator runs with the configured model — use `openai/gpt-5.5 (xhigh)` for orchestration quality.
- Do not add `stpa-analyst` to the agents list.

## Model Selection Guidelines

A single `model` is used for all agents. The coordinator runs with this model; the OpenCode runtime propagates it to subagents as appropriate.

| Recommended Model | Reason |
|-------------------|--------|
| `openai/gpt-5.5 (xhigh)` | Orchestration, safety-gate reasoning, and implementation quality |

## Complete Examples

### Simple Go Project

```markdown
---
agents:
  - "go"
model: "openai/gpt-5.5 (xhigh)"
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

### Full-Stack Project with Safety Analysis

```markdown
---
agents:
  - "go"
  - "react"
  - "general-purpose"
model: "openai/gpt-5.5 (xhigh)"
completionGateScript: make test
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
agents:
  - "general-purpose"
model: "openai/gpt-5.5 (xhigh)"
interactive: yes
---

# API Design Research

Research and document best practices for REST API versioning.
```
