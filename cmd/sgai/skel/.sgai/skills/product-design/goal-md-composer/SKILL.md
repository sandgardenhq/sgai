---
name: goal-md-composer
description: Interactive wizard to compose valid GOAL.md files for SGAI with step-by-step guidance through 7 phases. Use when creating GOAL.md, configuring agents and model, safety-analysis guidance, or starting an SGAI project. Safety Analysis must use coordinator/reviewer stpa-overview skill guidance, not stpa-analyst as a GOAL agent.
---

# GOAL.md Composer

Guide the user through a complete `GOAL.md` using seven phases: purpose, agents, model configuration, specification writing, options, and validation.

## Phase 4: Model Configuration

Configure the shared model and list available delegate agents.

Default GPT-5.5 recommendation:

- A single `model` is used for all agents. The coordinator runs with `openai/gpt-5.5 (xhigh)` for orchestration quality; the OpenCode runtime handles model propagation to subagents.
- The `agents` list enumerates non-coordinator delegate agents available for OpenCode subagent delegation.

Do not add `stpa-analyst` to the agents list.

Example:

```yaml
agents:
  - "go"
  - "react"
  - "general-purpose"
model: "openai/gpt-5.5 (xhigh)"
```

Ask: "Which delegate agents should be available, and what model should be used?"

## Safety Analysis Guidance

Safety analysis is coordinator/reviewer skill guidance, not a routable flow node.

```markdown
## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.
```

## Validation Checklist

- [ ] `stpa-analyst` is not in the agents list
- [ ] Safety Analysis, if selected, appears as coordinator/reviewer `stpa-overview` guidance
- [ ] Agent names are valid OpenCode subagent identifiers
- [ ] Interactive mode is set
- [ ] Specification has clear requirements and checkbox tasks

## Full-Featured Template with Safety Analysis

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

# Project Goal

[Description]

## Requirements

- [Requirement]

## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.

## Tasks

- [ ] Task
```
