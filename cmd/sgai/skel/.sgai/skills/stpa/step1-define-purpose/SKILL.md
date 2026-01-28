---
name: stpa/step1-define-purpose
description: STPA Step 1 - Define the purpose of analysis by identifying losses, hazards, and system-level constraints.
when_to_use: When beginning STPA analysis. After loading stpa/overview. When you need to establish what could go wrong and what must be prevented.
version: 1.0.0
tags: stpa, safety, losses, hazards, constraints
---

# STPA Step 1: Define Purpose of Analysis

## Objective

Establish the foundation for the entire STPA analysis by identifying:
1. **Losses** - What we absolutely cannot afford to happen
2. **Hazards** - System states that could lead to those losses
3. **System-Level Constraints** - What the system must do/not do to prevent hazards

## Announce at Start

"I'm using the STPA Step 1 skill to define the purpose of our analysis. We'll identify losses, hazards, and constraints through a series of questions."

## Phase 1: Identify Losses (L)

### Questions to Ask (ONE at a time)

**Q1: What are the unacceptable outcomes for this system?**
- Examples for software: data loss, unauthorized access, system downtime, financial loss
- Examples for physical: injury, property damage, environmental harm
- Examples for AI: biased decisions, safety violations, unintended actions

**Q2: For each loss, how severe is the impact?**
- Critical: Immediate danger or catastrophic failure
- Serious: Significant damage or major disruption
- Moderate: Notable impact but recoverable
- Minor: Inconvenience with minimal lasting effect

### Recording Losses

```markdown
#### Losses (L)
- L-1: [Loss of human life or serious injury]
- L-2: [Loss of data integrity]
- L-3: [Loss of service availability]
- L-4: [Financial loss exceeding $X]
```

## Phase 2: Identify Hazards (H)

### Definition
A hazard is a **system state or set of conditions** that, together with a worst-case set of environmental conditions, will lead to a loss.

### Questions to Ask

**Q1: What system states could lead to [Loss L-1]?**
- Think about the system's behavior, not component failures
- Focus on what the SYSTEM does, not what COMPONENTS do

**Q2: Under what environmental conditions would this hazard lead to the loss?**
- Worst-case scenarios
- Concurrent conditions that compound risk

### Hazard Template

```
H-[number]: [System] [unsafe condition/behavior] [leading to L-X]
```

### Examples

**Software System:**
- H-1: Authentication service grants access to unauthorized users [→ L-2, L-4]
- H-2: Database writes incomplete transaction records [→ L-2]

**Physical System:**
- H-1: Vehicle accelerates when driver intends to stop [→ L-1]
- H-2: Pressure vessel exceeds safe operating limits [→ L-1, L-3]

**AI System:**
- H-1: ML model recommends unsafe medication dosage [→ L-1]
- H-2: Autonomous agent takes actions outside approved scope [→ L-1, L-4]

### Recording Hazards

```markdown
#### Hazards (H)
- H-1: [System] [condition] [→ L-X, L-Y]
- H-2: [System] [condition] [→ L-X]
```

## Phase 3: Define System-Level Constraints (SC)

### Definition
System-level constraints specify what the system must do or must NOT do to prevent hazards.

### Derivation Rule
For each hazard, derive at least one constraint that, if enforced, prevents the hazard.

### Constraint Template

```
SC-[number]: [System] must [always/never] [behavior] [to prevent H-X]
```

### Examples

**From H-1 (auth service):**
- SC-1: Authentication service must verify credentials before granting access [→ H-1]
- SC-2: Authentication service must never bypass verification checks [→ H-1]

**From H-1 (vehicle):**
- SC-1: Vehicle must decelerate when brake is applied [→ H-1]
- SC-2: Vehicle must never accelerate during brake application [→ H-1]

### Recording Constraints

```markdown
#### System-Level Constraints (SC)
- SC-1: [System] must [behavior] [→ H-X]
- SC-2: [System] must never [behavior] [→ H-X]
```

## Validation Questions

Before proceeding to Step 2, verify:

1. **Completeness**: "Are there any other losses you can think of?"
2. **Traceability**: "Does each hazard trace to at least one loss?"
3. **Coverage**: "Does each hazard have at least one constraint?"
4. **Clarity**: "Are the constraints specific enough to be verifiable?"

## Output Template

Record in PROJECT_MANAGEMENT.md:

```markdown
## STPA Analysis

### Step 1: Purpose Definition

#### Losses (L)
- L-1: [description] - Severity: [Critical/Serious/Moderate/Minor]
- L-2: [description] - Severity: [level]

#### Hazards (H)
- H-1: [System] [unsafe condition] [→ L-1, L-2]
- H-2: [System] [unsafe condition] [→ L-1]

#### System-Level Constraints (SC)
- SC-1: [System] must [behavior] [→ H-1]
- SC-2: [System] must never [behavior] [→ H-1]
```

## When to Proceed to Step 2

Move to Step 2 when:
- [ ] All significant losses are identified
- [ ] Each loss has at least one associated hazard
- [ ] Each hazard has at least one constraint
- [ ] Human partner confirms the analysis looks complete

Load: `skills({"name":"stpa/step2-control-structure"})`
