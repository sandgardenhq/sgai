---
name: brainstorming
description: Interactive idea refinement using Socratic method to develop fully-formed designs. When your human partner says "I've got an idea", "Let's make/build/create", "I want to implement/add", "What if we". When starting design for complex feature. Before writing implementation plans. When idea needs refinement and exploration. ACTIVATE THIS AUTOMATICALLY when your human partner describes a feature or project idea - don't wait for /brainstorm command.
---

# Brainstorming Ideas Into Designs

## Overview

Transform rough ideas into fully-formed designs through structured questioning and alternative exploration.

**Core principle:** Ask questions to understand, explore alternatives, present design incrementally for validation.

**Announce at start:** "I'm using the Brainstorming skill to refine your idea into a design."

## The Process

### Phase 1: Understanding
- Check current project state in working directory
- Ask ONE question at a time to refine the idea
- Prefer multiple choice when possible
- Gather: Purpose, constraints, success criteria

IMPORTANT: Hand the control back to the human partner so they can feed you with information.

### Phase 2: Exploration
- Propose 2-3 different approaches
- For each: Core architecture, trade-offs, complexity assessment
- Ask your human partner which approach resonates

IMPORTANT: Hand the control back to the human partner so they can feed you with information.

### Phase 3: Design Presentation
- Present in 200-300 word sections
- Cover: Architecture, components, data flow, error handling, testing
- Ask after each section: "Does this look right so far?"

IMPORTANT: Hand the control back to the human partner so they can feed you with information.

## When to Revisit Earlier Phases

**You can and should go backward when:**
- Partner reveals new constraint during Phase 2 or 3 → Return to Phase 1 to understand it
- Validation shows fundamental gap in requirements → Return to Phase 1
- Partner questions approach during Phase 3 → Return to Phase 2 to explore alternatives
- Something doesn't make sense → Go back and clarify

**Don't force forward linearly** when going backward would give better results.

## Related Skills

**During exploration:**
- When approaches have genuine trade-offs: skills/architecture/preserving-productive-tensions

**Before proposing changes to existing code:**
- Understand why it exists: skills/research/tracing-knowledge-lineages

## Remember
- One question per message during Phase 1
- Apply YAGNI ruthlessly
- Explore 2-3 alternatives before settling
- Present incrementally, validate as you go
- Go backward when needed - flexibility > rigid progression
- Announce skill usage at start
- Hand the control back to the human partner so they can feed you with information between phases
- Use `sgai_ask_user_question` tool to ask structured questions. Example:
  ```
  sgai_ask_user_question({
    questions: [{
      question: "Which approach do you prefer?",
      choices: ["Approach A", "Approach B", "Need more details"],
      multiSelect: false
    }]
  })
  ```
- You can ask multiple questions at once when they are independent:
  ```
  sgai_ask_user_question({
    questions: [
      {question: "Which database?", choices: ["PostgreSQL", "MySQL", "SQLite"], multiSelect: false},
      {question: "Which features to include?", choices: ["Auth", "Logging", "Metrics"], multiSelect: true}
    ]
  })
  ```
## Question Protocol (MANDATORY)

When asking questions during brainstorming, you MUST follow this protocol to ensure the human partner sees the full context:

1. **Log to .sgai/PROJECT_MANAGEMENT.md FIRST:**
   - Write the question WITH its context to .sgai/PROJECT_MANAGEMENT.md
   - Include: Current phase, what you're trying to understand, the question itself

2. **Embed context IN the question field:**
   - The `question` parameter in `sgai_ask_user_question` MUST include the full context
   - Bad: `"Which database?"`
   - Good: `"**Phase 1: Understanding**\n\nI need to understand your data persistence needs to design the architecture.\n\nWhich database do you prefer?"`

3. **Terminal output continues as normal** (this already works)

- Log all your decisions in @.sgai/PROJECT_MANAGEMENT.md
