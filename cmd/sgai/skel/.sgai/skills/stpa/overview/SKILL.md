---
name: stpa/overview
description: Entry point for STPA (System Theoretic Process Analysis) hazard analysis. Guides through all 4 steps sequentially.
when_to_use: When starting a new STPA analysis session. When the human partner mentions safety analysis, hazard analysis, or risk assessment. When analyzing control systems for potential failures.
version: 1.0.0
tags: stpa, safety, hazard-analysis, control-theory
---

# STPA Overview

## What is STPA?

STPA (System Theoretic Process Analysis) is a hazard analysis method that:
- Treats safety as a **control problem**, not just a failure problem
- Uses **control-feedback loops** to model complex systems
- Identifies **unsafe control actions** that could lead to hazards
- Discovers complex, unintended system interactions
- Works on **software**, **physical**, and **AI-driven** systems

## Announce at Start

"I'm using the STPA Overview skill to guide you through a systematic hazard analysis. We'll work through 4 steps, asking questions along the way."

## The 4 Steps

### Step 1: Define Purpose of Analysis
Load: `skills({"name":"stpa/step1-define-purpose"})`
- Identify **Losses** (unacceptable outcomes)
- Define **System-Level Hazards** (states leading to losses)
- Establish **System-Level Constraints** (behaviors to prevent hazards)

### Step 2: Model the Control Structure
Load: `skills({"name":"stpa/step2-control-structure"})`
- Create hierarchical control-feedback diagrams
- Identify controllers, control actions, and feedback paths
- Use Graphviz/DOT format with `rankdir=TB` and `node [shape=box]`

### Step 3: Identify Unsafe Control Actions (UCAs)
Load: `skills({"name":"stpa/step3-unsafe-control-actions"})`
- Analyze each control action for 4 types of UCAs:
  1. Not provided when needed
  2. Provided when not needed
  3. Wrong timing (too early/late/out of sequence)
  4. Wrong duration (stopped too soon/applied too long)

### Step 4: Identify Loss Scenarios
Load: `skills({"name":"stpa/step4-loss-scenarios"})`
- Trace causal pathways for each UCA
- Identify why UCAs might occur
- Develop recommendations and mitigations

## Process Flow

```
[Step 1: Purpose] → [Step 2: Control Structure] → [Step 3: UCAs] → [Step 4: Scenarios]
       ↑                                                                      |
       └──────────────────────── (Iterate as needed) ─────────────────────────┘
```

## Interactive Approach

At each step:
1. Load the step-specific skill
2. Ask ONE question at a time
3. Use `human-communication` status to wait for responses
4. Record answers in PROJECT_MANAGEMENT.md under `## STPA Analysis`
5. Continue to next question or step

## Documentation Structure

### In PROJECT_MANAGEMENT.md:
```markdown
## STPA Analysis

### Step 1: Purpose Definition
#### Losses (L)
- L-1: [description]

#### Hazards (H)
- H-1: [system] [unsafe condition] [→ L-1]

#### System-Level Constraints (SC)
- SC-1: [condition to enforce] [→ H-1]

### Step 2: Control Structure
[Graphviz/DOT diagram]

### Step 3: Unsafe Control Actions
[UCA table]

### Step 4: Loss Scenarios
[Scenario descriptions and recommendations]
```

### In GOAL.md (final summary):
```markdown
## STPA Findings
- [X] STPA analysis completed on [date]
- Key hazards identified: [count]
- Unsafe control actions found: [count]
- Critical recommendations: [list]
```

## When Analysis is Complete

After completing all 4 steps:
1. Summarize findings in GOAL.md under `## STPA Findings`
2. Mark the STPA analysis checkbox as complete
3. Set `status: "agent-done"` to return control

## Related Skills

- `stpa/step1-define-purpose` - Detailed Step 1 guidance
- `stpa/step2-control-structure` - Control structure modeling
- `stpa/step3-unsafe-control-actions` - UCA identification tables
- `stpa/step4-loss-scenarios` - Causal scenario analysis
