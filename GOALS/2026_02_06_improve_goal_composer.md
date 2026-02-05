---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "openai/gpt-5.2-codex (xhigh)"
  "backend-go-developer": "openai/gpt-5.2-codex"
  "go-readability-reviewer": "openai/gpt-5.2-codex"
  "general-purpose": "openai/gpt-5.2-codex"
  "htmx-picocss-frontend-developer": "openai/gpt-5.2-codex"
  "htmx-picocss-frontend-reviewer": "opencode/kimi-k2.5"
  "stpa-analyst": "openai/gpt-5.2-codex"
  "project-critic-council": ["openai/gpt-5.2-codex", "openai/gpt-5.2", "opencode/kimi-k2.5"]
  "skill-writer": "openai/gpt-5.2-codex (xhigh)"
interactive: yes
completionGateScript: make test
---

- [x] /compose/wizard/step/2?workspace=$workspace
      adjust the size of the buttons so they don't visually overlap (verify with screenshots)

---

- [x] when creating a new workspace, after I give a name, redirect to /compose?workspace=$workspace
  - [x] in other words, `/workspaces/new` submission in not redirecting to /compose?workspace=$workspace
  - [x] the Edit GOAL.md that's current in the pages `/compose/wizard/step/` should actually be in `/compose?workspace=$workspace`
  - [x] when the wizard is over, that I save the GOAL.md, the page should redirect to the project page in the Progress tab

---

- [x] I need a way to add the general-purpose agent to the composer in /compose/wizard/step/2?workspace=$workspace
- [x] when I create a new workspace, after I fill up with the name, start with the wizard
- [x] add option to edit GOAL.md at the bottom of the wizard as a escape valve for advanced users

---

in /compose/wizard/step/2?workspace=$workspace
- [x] when I click Docker or PostgreSQL and few other technology choices, the generated GOAL.md doesn't react correctly
  - [x] it should have added the agents in the flow section
  - [x] it should have added the models for the agents in the models section
  - [x] in the preview box, after I click, the padding is wrong

in /compose/wizard/step/3?workspace=$workspace
- [x] when I click among the options I don't see the preview updating either.
  - [x] in the preview box, after I click, the padding is wrong

- [x] There shouldn't be a step 4 code review -- code review is ALWAYS mandatory, if I add Go as a stack, I must ALWAYS get both agents: go backend engineer and the go readability reviewer, for example.

/compose/wizard/step/5?workspace=$workspace
- [x] when I toggle STPA analysis on and off, I don't see flow getting updated.

- [x] check what other agents could be used to create more templates,  and more options in steps 1, 2, 3, and 5.

in compose-command-bar in /compose/editor?workspace=$workspace
- [x] the button is too large, the input text too small
- [x] `Agent Selection & Models` the contents don't fit the box, make them fit (probably by making them smaller?)
- [x] `AI Suggest Agents` doesn't work
- [x] `AI Generate DAG` doesn't work
- [x] `AI Generate Tasks` doesn't work
- [x] consider the changes necessary to remove this particular mode please.

---
## Redesign Compose GOAL Flow

Redesign `/compose?workspace=$workspace` with a unified approach combining templates, wizard, and live preview.

### Design Decision (from Brainstorming 2026-02-05)

**Chosen Approach:** Combined Templates + Wizard + Live Preview
- Landing page offers template gallery OR guided wizard
- Templates pre-fill wizard steps (user can customize)
- All paths lead to same wizard with live preview on right side
- Focus on NEW GOAL.md creation, with simple edit capability (toggle agents)

### Implementation Tasks

- [x] **Landing Page Redesign**
  - [x] Create template gallery with 5 templates (Backend, Frontend, FullStack, Research, Custom)
  - [x] Add "Start Guided Wizard" entry point
  - [x] Style with PicoCSS, no custom JavaScript

- [x] **Wizard Implementation (4 steps)**
  - [x] Step 1: Project Description (textarea)
  - [x] Step 2: Tech Stack (checkboxes - Go, HTMX, React, Python, etc.)
  - [x] Step 3: Safety Analysis toggle (adds STPA agent)
  - [x] Step 4: Settings (interactive mode, completion gate script)
  - [x] Back/Next navigation between steps
  - [x] Progress indicator (● ○ ○ ○)

- [x] **Live Preview Panel**
  - [x] Always visible on right side (40% width)
  - [x] Updates in real-time as wizard progresses
  - [x] Shows rendered GOAL.md preview

- [x] **Final Review & Save**
  - [x] Summary of all choices
  - [x] Save button writes GOAL.md to workspace

- [x] **Bug Fixes (during redesign)**
  - [x] Fix model selector (`opencode models` integration)
  - [N/A] Fix any session management issues (SKIPPED per human partner)

- [x] **Template Definitions**
  - [x] Backend Development (Go developer + reviewer + STPA)
  - [x] Frontend Development (HTMX/PicoCSS developer + reviewer)
  - [x] Full Stack (backend + frontend + reviewers)
  - [x] Research/Analysis (general-purpose + critic council)
  - [x] Custom Agent Architecture (blank, user picks agents)

### Acceptance Criteria

1. User can create a complete GOAL.md in under 2 minutes using templates
2. User can create a GOAL.md from scratch using wizard without needing to know GOAL.md syntax
3. Model selector works and persists selections correctly
4. Live preview updates in real-time during wizard
5. All functionality works with pure HTMX + PicoCSS (no custom JavaScript except Idiomorph)
