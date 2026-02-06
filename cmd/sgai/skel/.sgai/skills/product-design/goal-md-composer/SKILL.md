---
name: goal-md-composer
description: Interactive wizard to compose valid GOAL.md files for SGAI with step-by-step guidance through 7 phases. When your human partner wants to create a new GOAL.md file, start a new SGAI project, configure agents for a software factory, or asks "help me set up GOAL.md", "I want to configure agents", "let's start a new project with SGAI".
---

# GOAL.md Composer

## Overview

An interactive wizard that guides you through composing valid, well-structured GOAL.md files for SGAI (Software Generation AI). This skill walks through 7 phases to help configure agents, models, workflows, and project specifications.

**Announce at start:** "I'm using the GOAL.md Composer skill to help you create a valid GOAL.md file."

## Prerequisites

Before starting, load the REFERENCE.md companion document for the complete agent catalog and format specification:
- `skills/product-design/goal-md-composer/REFERENCE.md`

## The 7-Phase Process

### Phase 1: Project Description

Understand what the user is building.

**Ask these questions (one at a time):**

1. "What type of project are you building?"
   - Options: API/Backend, CLI Tool, Web Application, Full-Stack App, Documentation, Library/SDK, Other

2. "What programming language(s) will you use?"
   - Options: Go, TypeScript, Python, Shell/Bash, Multiple languages, Other

3. "What's the scope of this project?"
   - Options: Quick prototype, MVP, Production-ready application, Exploratory research

4. "Are there any specific constraints?"
   - Free-form answer about: external dependencies, testing requirements, security needs, etc.

**Log answers in:** `@.sgai/PROJECT_MANAGEMENT.md` under "## GOAL.md Composer Session"

**Hand control back to human after each question.**

---

### Phase 2: Agent Recommendations

Based on project description, recommend appropriate agents.

**CRITICAL - Mandatory Reviewer Pairing Rules:**
When selecting a development agent, you MUST automatically include its paired reviewer:

| Development Agent | Required Reviewer |
|-------------------|-------------------|
| `backend-go-developer` | `go-readability-reviewer` |
| `htmx-picocss-frontend-developer` | `htmx-picocss-frontend-reviewer` |
| `react-developer` | `react-reviewer` |
| `shell-script-coder` | `shell-script-reviewer` |

**Process:**

1. Based on Phase 1 answers, present recommended agents organized by category
2. For each development agent recommended, show: "This will automatically include [reviewer] for code review"
3. Present the full agent list with descriptions (see REFERENCE.md)
4. Ask: "Do you want to add or remove any agents from this selection?"

**Example recommendation for Go Backend API:**
```
Recommended agents for your Go Backend API:
- backend-go-developer (Go development)
- go-readability-reviewer (automatically paired)
- general-purpose (cross-domain tasks)
- stpa-analyst (safety analysis)

Do you want to add or remove any agents?
```

**Hand control back to human for confirmation.**

---

### Phase 3: Flow Builder

Generate the DOT syntax DAG defining agent execution order and dependencies.

**Auto-generation rules:**

1. Development agents always flow into their paired reviewers:
   ```
   "backend-go-developer" -> "go-readability-reviewer"
   ```

2. Reviewers typically flow into terminal analysis agents:
   ```
   "go-readability-reviewer" -> "stpa-analyst"
   ```

3. Standalone agents (no dependencies) are listed without arrows:
   ```
   "general-purpose"
   ```

**Present the generated flow:**
```yaml
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
```

**Ask:** "Does this workflow look correct? Would you like to add or modify any dependencies?"

**Hand control back to human for adjustments.**

---

### Phase 4: Model Configuration

Configure per-agent model assignments.

**Available models (see REFERENCE.md for full list):**
- `anthropic/claude-opus-4-6` - Most capable, highest cost ($$$$)
- `anthropic/claude-opus-4-6 (max)` - Extended thinking variant
- `anthropic/claude-sonnet-4-5` - Balanced capability/cost ($$$)
- `anthropic/claude-sonnet-4-5 (max)` - Extended thinking variant
- `google/gemini-2.0-flash-001` - Fast, lower cost ($$)
- `openai/gpt-4.1` - Alternative high-capability model ($$$)

**Default recommendations:**
- `coordinator`: `anthropic/claude-opus-4-6 (max)` (always use strongest for coordination)
- Development agents: `anthropic/claude-opus-4-6` (complex reasoning)
- Review agents: `anthropic/claude-opus-4-6` (thorough analysis)
- Frontend agents: `anthropic/claude-sonnet-4-5` (good balance)
- Utility agents: `anthropic/claude-sonnet-4-5` (cost-effective)

**Present configuration:**
```yaml
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
```

**Ask:** "Do you want to adjust any model assignments? (e.g., use a faster/cheaper model for certain agents)"

**Hand control back to human for adjustments.**

---

### Phase 5: Specification Writing

Guide through writing the markdown body with Goal, Requirements, and Tasks.

**Structure to generate:**

```markdown
# [Project Title]

[1-2 paragraph description of what to build, focusing on outcomes not implementation]

## Requirements

- [Behavioral requirement 1]
- [Behavioral requirement 2]
- [Constraint 1]

## Tasks

- [ ] Task 1
  - [ ] Subtask 1.1
  - [ ] Subtask 1.2
- [ ] Task 2
- [ ] Task 3
```

**Guidance:**
1. Ask: "Describe the main goal in 1-2 sentences. Focus on WHAT, not HOW."
2. Ask: "What are the key requirements? Think about: user actions, constraints, success criteria."
3. Ask: "What are the major tasks? Break down into actionable items."

**Tips to share:**
- Focus on behavior, not implementation
- Use checkboxes (`- [ ]`) for tasks - the coordinator manages these
- Be specific about constraints (e.g., "no external dependencies", "must pass CI")
- Nested checkboxes are supported for subtasks

**Hand control back to human after each section.**

---

### Phase 6: Options

Configure optional settings.

**Ask about each option:**

1. **Interactive mode:**
   "How should agent questions be handled?"
   - `yes` - Questions appear in web UI; human responds interactively (RECOMMENDED)
   - `no` - Workflow exits when an agent asks a question
   - `auto` - Self-driving mode; agents attempt to proceed without human input

2. **Completion gate script:**
   "Should a command verify workflow completion?"
   - Examples: `make test`, `go test ./...`, `npm run lint && npm test`
   - Leave empty if no verification needed

**Present configuration:**
```yaml
interactive: yes
completionGateScript: make test
```

**Hand control back to human for confirmation.**

---

### Phase 7: Output & Validation

Generate the complete GOAL.md and validate.

**Generate the complete file:**

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

**Validation checklist (present to user):**

- [ ] All development agents have paired reviewers in the flow
- [ ] Flow has no cycles (DAG is valid)
- [ ] Coordinator is NOT in the flow (it's always present automatically)
- [ ] Models are assigned to agents that appear in the flow
- [ ] Interactive mode is set
- [ ] Specification has Goal, Requirements, and Tasks sections
- [ ] Tasks use checkbox format (`- [ ]`)

**Ask:** "I've generated the GOAL.md. Would you like me to:
1. Write it to ./GOAL.md
2. Show me the complete file first
3. Make adjustments"

**Hand control back to human for final decision.**

---

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

### Full-Featured Template
```markdown
---
completionGateScript: make test
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
interactive: yes
---

# Project Goal

[Description]

## Requirements

- [Requirement 1]
- [Requirement 2]

## Tasks

- [ ] Task 1
  - [ ] Subtask 1.1
- [ ] Task 2
```

---

## Remember

- Announce skill usage at start
- Ask ONE question at a time during gathering phases
- Auto-enforce reviewer pairing (mandatory, not optional)
- Present configuration incrementally for validation
- Log all decisions in `@.sgai/PROJECT_MANAGEMENT.md`
- Hand control back to human between phases
- Validate the final output before writing
- Use `sgai_ask_user_question` for structured questions:
  ```
  sgai_ask_user_question({
    questions: [{
      question: "**Phase 1: Project Description**\n\nWhat type of project are you building?",
      choices: ["API/Backend", "CLI Tool", "Web Application", "Full-Stack App", "Documentation", "Other"],
      multiSelect: false
    }]
  })
  ```
