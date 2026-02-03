---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-sonnet-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
interactive: yes
---

# GOAL.md Composer Skill

Create a comprehensive skill document that enables LLMs (including the sgai coordinator and external coding assistants) to compose valid, well-structured GOAL.md files. This skill should encode all the knowledge from issue #67 in a format that can be loaded into any LLM context.

## Requirements (from Brainstorming)

1. **Interactive Wizard**: Step-by-step prompts guiding users through 7 phases
2. **Layered Structure**: Two files - SKILL.md (interactive) + REFERENCE.md (documentation)
3. **Auto-enforce Reviewer Pairing**: Automatically add reviewers when developer agents are selected
4. **Target**: GOAL.md files for SGAI (skill can be run by any LLM)
5. **Location**: `cmd/sgai/skel/.sgai/skills/product-design/goal-md-composer/`

## Wizard Phases

1. **Project Description** - Understand what user is building (type, language, scope)
2. **Agent Recommendations** - Suggest agents based on project, auto-add paired reviewers
3. **Flow Builder** - Generate DOT syntax DAG with dependencies
4. **Model Configuration** - Configure per-agent model assignments
5. **Specification Writing** - Guide through Goal, Requirements, Tasks sections
6. **Options** - Set interactive mode, completionGateScript
7. **Output & Validation** - Generate complete GOAL.md with validation checklist

## Acceptance Criteria

- [x] SKILL.md created with interactive 7-phase wizard
- [x] REFERENCE.md created with complete agent catalog and format documentation
- [x] Mandatory reviewer pairing rules documented and enforced in wizard
- [x] All frontmatter fields documented (flow, models, completionGateScript, interactive)
- [x] DOT syntax examples included for flow DAG
- [x] Complete GOAL.md examples included
- [x] Skill follows existing skill conventions (announce usage, hand control back)