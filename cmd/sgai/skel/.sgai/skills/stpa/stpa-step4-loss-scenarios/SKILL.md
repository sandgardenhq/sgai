---
name: stpa-step4-loss-scenarios
description: STPA Step 4 - Identify Loss Scenarios by tracing causal pathways back through control loops to understand why UCAs might occur. After completing STPA Step 3. When you need to understand WHY unsafe control actions might happen. When developing recommendations and mitigations.
metadata:
   tags: stpa, safety, loss-scenarios, causal-analysis, mitigations
---

# STPA Step 4: Identify Loss Scenarios

## Objective

For each Unsafe Control Action (UCA) identified in Step 3:
1. Trace back through the control structure to find **why** it might occur
2. Identify **causal factors** at each level
3. Develop **recommendations** to prevent or mitigate

## Announce at Start

"I'm using the STPA Step 4 skill to identify loss scenarios. We'll trace each UCA back through the control structure to understand what could cause it."

## Two Types of Scenarios

### Scenario Type 1: Why the UCA occurs

The controller issues an unsafe control action because of:
- **Incorrect process model**: Controller has wrong understanding of system state
- **Inadequate control algorithm**: Logic errors or missing conditions
- **External factors**: Environmental interference, component failures
- **Communication issues**: Commands lost, delayed, or corrupted

### Scenario Type 2: Why the control action is improperly executed

Even if the correct control action is issued:
- **Actuator failure**: The mechanism can't execute the command
- **Delayed execution**: Action happens too late
- **Process doesn't respond**: Physical constraints prevent action
- **Conflicting controls**: Another controller countermands

## Causal Factor Categories

For each UCA, consider these causal categories:

### Controller-Related

1. **Unsafe control algorithm**
   - Missing or incorrect logic
   - Race conditions
   - Unhandled edge cases

2. **Incorrect process model**
   - Wrong assumptions about system state
   - Stale information
   - Missing information
   - Incorrect understanding of physics/behavior

3. **Inadequate feedback**
   - Feedback not received
   - Feedback delayed
   - Feedback incorrect
   - Feedback misinterpreted

### Path-Related

4. **Control action not executed**
   - Command lost in transmission
   - Actuator failure
   - Conflicting commands

5. **Feedback not provided**
   - Sensor failure
   - Communication failure
   - Feedback suppressed

### Process-Related

6. **Process behavior**
   - Unexpected process response
   - External disturbances
   - Component failures within process

## Analysis Process

### For Each High-Priority UCA

**Step A: State the UCA**
```
UCA-X: [Controller] [action context] leading to [H-X]
```

**Step B: Ask "Why might this happen?"**

Walk backward through the control loop:

1. **Why might the controller issue this UCA?**
   - What must the controller believe for this to seem correct?
   - What information might it be missing?
   - What could go wrong with its decision logic?

2. **Why might the controller have wrong information?**
   - How does feedback reach the controller?
   - What could corrupt or delay this feedback?
   - What assumptions does the controller make?

3. **Why might the action not execute correctly?**
   - What could prevent proper execution?
   - What could delay execution?
   - What could cause partial execution?

**Step C: Document each causal pathway**

### Scenario Template

```markdown
### UCA-X: [description]

**Scenario 1: [Brief name]**
[Controller] [reason for issuing UCA] because [root cause].
The controller's process model was incorrect because [specific gap].

**Recommendation:** [Mitigation measure]

**Scenario 2: [Brief name]**
Even though correct command was issued, [reason execution failed] because [root cause].

**Recommendation:** [Mitigation measure]
```

## Example Analysis

### UCA-2: Auth Service issues token when credentials are invalid

**Scenario 1: Credential Check Bypassed**
Auth Service issues token without proper verification because the validation logic contains a short-circuit that skips checks under high load conditions. The controller's algorithm prioritizes availability over security.

**Recommendation:** Remove short-circuit logic; ensure all credential validation is mandatory regardless of load. Add circuit breaker that fails closed (denies access) rather than fails open.

**Scenario 2: Cached Valid Status**
Auth Service issues token because it uses a cached "valid" status from a previous request, not realizing the credentials were revoked. The process model is stale.

**Recommendation:** Implement cache invalidation on credential revocation. Add freshness checks before using cached authentication status.

**Scenario 3: Response Confusion**
Auth Service issues token because response from identity provider was corrupted in transit and misinterpreted as "valid". Communication path has no integrity checking.

**Recommendation:** Add checksum/signature verification on all identity provider responses. Implement retry with fresh request on verification failure.

## Questions to Guide Analysis

For each UCA, work through:

**Q1: What must be true in the controller's model for this UCA to happen?**
(Identify the incorrect belief or assumption)

**Q2: How could the controller's model become incorrect?**
- Feedback never received?
- Feedback received but wrong?
- Feedback received but misinterpreted?
- Process model never updated?

**Q3: What could prevent proper execution of the correct action?**
- Actuator issues?
- Timing problems?
- Interference?

**Q4: What would prevent/detect each scenario?**
(These become your recommendations)

## Recommendation Categories

### Prevention (Eliminate the scenario)
- Design changes to make scenario impossible
- Adding missing feedback paths
- Fixing control algorithm logic

### Detection (Catch it when it happens)
- Monitoring and alerts
- Redundant checks
- Anomaly detection

### Mitigation (Reduce impact when it occurs)
- Fallback behaviors
- Graceful degradation
- Recovery procedures

## Output Template

Record in .sgai/PROJECT_MANAGEMENT.md:

```markdown
### Step 4: Loss Scenarios

#### UCA-1: [description]

**Scenario 1: [Name]**
[Controller] [behavior] because [cause].
The [aspect] was [incorrect/missing/delayed] because [root cause].

**Recommendation:** [Mitigation]

**Scenario 2: [Name]**
[Description of causal pathway]

**Recommendation:** [Mitigation]

---

#### UCA-2: [description]

[Repeat for each high-priority UCA]

---

### Recommendation Summary

| ID | Recommendation | Addresses | Priority |
|----|---------------|-----------|----------|
| R-1 | [recommendation] | UCA-1, UCA-3 | High |
| R-2 | [recommendation] | UCA-2 | Medium |
```

## Completing the STPA Analysis

When all scenarios are documented:

1. **Summarize in GOAL.md:**

```markdown
## STPA Findings
- [X] STPA analysis completed on [date]
- Key hazards identified: [count]
- Unsafe control actions found: [count]
- Loss scenarios analyzed: [count]
- Critical recommendations:
  - R-1: [brief description]
  - R-2: [brief description]
```

2. **Update .sgai/PROJECT_MANAGEMENT.md** with complete analysis

3. **Set status: "agent-done"** to return control

## Iteration

STPA is iterative. If during Step 4 you discover:
- New hazards → Return to Step 1
- Missing control paths → Return to Step 2
- New UCAs → Return to Step 3

Document any iterations and the insights that triggered them.
