---
name: goal-md-composer
description: Interactive wizard to compose valid GOAL.md files for SGAI with step-by-step guidance through 7 phases. Use when creating GOAL.md, configuring agents, model lists, flow DAGs, safety-analysis guidance, or starting an SGAI project. Safety Analysis must use coordinator/reviewer stpa-overview skill guidance, not stpa-analyst flow nodes.
---

# GOAL.md Composer

Guide the user through a complete `GOAL.md` using seven phases: purpose, agents, flow, model configuration, specification writing, options, and validation.

## Phase 4: Model Configuration

Configure per-agent model assignments.

Default GPT-5.5 recommendations:

- `coordinator`: `openai/gpt-5.5 (xhigh)` because orchestration quality dominates workflow quality.
- Development, review, frontend, utility, specialist, and general-purpose agents: `openai/gpt-5.5 (low)` by default because GPT-5.5 low is the cost-conscious baseline for routine implementation and review.
- Users may override any agent model when a specific workflow warrants a higher tier.
- Aliases remain the recommended way to run the same role at multiple cost tiers.

Do not add a model entry for `stpa-analyst`.

Example:

```yaml
models:
  "coordinator": "openai/gpt-5.5 (xhigh)"
  "go": "openai/gpt-5.5 (low)"
  "react": "openai/gpt-5.5 (low)"
  "general-purpose": "openai/gpt-5.5 (low)"
```

Ask: "Do you want to adjust any model assignments?"

## Safety Analysis Guidance

Safety analysis is coordinator/reviewer skill guidance, not a routable flow node.

```markdown
## Safety Analysis

- The coordinator must load/use `stpa-overview` when safety, hazard, risk, external input, filesystem, concurrency, or unsafe state-transition concerns are relevant.
- `*-reviewer` agents may load/use `stpa-overview` when circumstances warrant hazard or safety analysis.
```

## Validation Checklist

- [ ] Flow has no cycles
- [ ] Coordinator is not in the flow
- [ ] `stpa-analyst` is not in the flow
- [ ] `stpa-analyst` is not in models
- [ ] Safety Analysis, if selected, appears as coordinator/reviewer `stpa-overview` guidance
- [ ] Models are assigned to agents that appear in the flow or to the coordinator
- [ ] `coordinator` uses `openai/gpt-5.5 (xhigh)` unless the user explicitly overrides it
- [ ] Non-coordinator agents use `openai/gpt-5.5 (low)` unless the user explicitly overrides them
- [ ] Interactive mode is set
- [ ] Specification has clear requirements and checkbox tasks

## Full-Featured Template with Safety Analysis

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
