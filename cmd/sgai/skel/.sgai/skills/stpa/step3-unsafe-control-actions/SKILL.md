---
name: stpa/step3-unsafe-control-actions
description: STPA Step 3 - Identify Unsafe Control Actions (UCAs) using the 4-type analysis framework.
when_to_use: After completing STPA Step 2. When analyzing control actions for potential safety issues. When you need to systematically identify what could go wrong with each control action.
version: 1.0.0
tags: stpa, safety, uca, unsafe-control-actions
---

# STPA Step 3: Identify Unsafe Control Actions

## Objective

For each control action identified in Step 2, determine if it could be hazardous under any of these 4 conditions:

1. **Not Provided** - Control action needed but not given
2. **Provided** - Control action given when it shouldn't be
3. **Wrong Timing** - Control action given too early, too late, or out of sequence
4. **Wrong Duration** - Control action stopped too soon or applied too long

## Announce at Start

"I'm using the STPA Step 3 skill to identify Unsafe Control Actions. We'll analyze each control action for 4 types of potential hazards."

## The 4 Types of UCAs

### Type 1: Not Provided (When Needed)

**Question**: What happens if [Controller] does NOT send [Control Action] when it should?

**Examples**:
- Brake pedal pressed but braking command not sent
- Authentication required but verification not performed
- ML model output not checked before action executed

### Type 2: Provided (When Not Needed)

**Question**: What happens if [Controller] sends [Control Action] when it shouldn't?

**Examples**:
- Braking applied when driver is accelerating
- Access granted to unauthorized user
- Emergency shutdown triggered during normal operation

### Type 3: Wrong Timing

**Question**: What happens if [Control Action] is given too early, too late, or out of sequence?

**Sub-types**:
- **Too Early**: Action taken before system is ready
- **Too Late**: Action taken after the safe window has passed
- **Out of Sequence**: Action taken in wrong order relative to other actions

**Examples**:
- Airbag deploys before crash occurs (too early)
- Airbag deploys after crash is over (too late)
- Payment processed before identity verified (out of sequence)

### Type 4: Wrong Duration

**Question**: What happens if [Control Action] is stopped too soon or applied too long?

**Sub-types**:
- **Stopped Too Soon**: Action ends before objective achieved
- **Applied Too Long**: Action continues beyond safe period

**Examples**:
- Braking stops before vehicle is at safe speed
- Medication pump runs longer than prescribed
- Database lock held until timeout causes failures

## Analysis Process

### For Each Control Action

1. State the control action clearly
2. Consider each of the 4 UCA types
3. Ask: "In what context would this be hazardous?"
4. Link back to hazards from Step 1 (H-1, H-2, etc.)
5. Record as UCA-[number]

### UCA Template

```
UCA-[number]: [Controller] [does/does not] [control action] [context], leading to [H-X]
```

### Examples

**Control Action: Auth Service issues access token**

| Type | UCA | Hazard |
|------|-----|--------|
| Not Provided | UCA-1: Auth Service does not issue token when user provides valid credentials, causing service denial | H-3 |
| Provided | UCA-2: Auth Service issues token when credentials are invalid, allowing unauthorized access | H-1 |
| Wrong Timing | UCA-3: Auth Service issues token before credential verification completes | H-1 |
| Wrong Duration | UCA-4: Auth Service issues token that never expires, allowing indefinite access | H-1, H-2 |

## Questions for Each Control Action

Work through systematically:

**Q1: What is the control action we're analyzing?**
(From Step 2's control structure)

**Q2: Type 1 - What if it's NOT provided when needed?**
- When would it be needed?
- What happens if it's missing?
- Does this lead to any hazard from Step 1?

**Q3: Type 2 - What if it's provided when NOT needed?**
- When should it NOT be given?
- What harm could it cause?
- Does this lead to any hazard from Step 1?

**Q4: Type 3 - What if the timing is wrong?**
- Too early: What if given before X happens?
- Too late: What if given after Y happens?
- Out of sequence: What if given before/after Z?

**Q5: Type 4 - What if the duration is wrong?**
- Too short: What if stopped before completion?
- Too long: What if it continues too long?

**Q6: Are any of these combinations N/A?**
Mark as N/A with brief explanation if truly not applicable.

## UCA Analysis Table

```markdown
| Control Action | Not Provided | Provided | Wrong Timing | Wrong Duration |
|---------------|--------------|----------|--------------|----------------|
| [CA-1] | UCA-1: [desc] → H-X | UCA-2: [desc] → H-Y | UCA-3: [desc] → H-X | N/A |
| [CA-2] | N/A | UCA-4: [desc] → H-X | UCA-5: [desc] → H-Y | UCA-6: [desc] → H-X |
```

## Prioritization

After identifying all UCAs, prioritize by:

1. **Severity**: Which hazards are most severe (from Step 1)?
2. **Likelihood**: Which UCAs are most likely to occur?
3. **Detectability**: Which UCAs are hardest to detect?

High priority = High severity + High likelihood + Low detectability

## Output Template

Record in .sgai/PROJECT_MANAGEMENT.md:

```markdown
### Step 3: Unsafe Control Actions

#### UCA Analysis Table

| Control Action | Not Provided | Provided | Wrong Timing | Wrong Duration |
|---------------|--------------|----------|--------------|----------------|
| [CA from Step 2] | [UCA or N/A] | [UCA or N/A] | [UCA or N/A] | [UCA or N/A] |

#### UCA Details

**UCA-1:** [Controller] [action context] leading to [H-X]
- Type: [Not Provided / Provided / Wrong Timing / Wrong Duration]
- Priority: [High / Medium / Low]

**UCA-2:** [Controller] [action context] leading to [H-X]
- Type: [type]
- Priority: [priority]

#### UCA Summary
- Total UCAs identified: [count]
- High priority: [count]
- Medium priority: [count]
- Low priority: [count]
```

## When to Proceed to Step 4

Move to Step 4 when:
- [ ] All control actions from Step 2 have been analyzed
- [ ] Each control action checked against all 4 UCA types
- [ ] All UCAs linked to hazards from Step 1
- [ ] UCAs prioritized by severity/likelihood
- [ ] Human partner confirms analysis is complete

Load: `skills({"name":"stpa/step4-loss-scenarios"})`
