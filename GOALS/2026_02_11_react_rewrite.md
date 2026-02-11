---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
  "migration-verifier"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-developer": "anthropic/claude-opus-4-6"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-6"
  "react-developer": "openai/gpt-5.2-codex (xhigh)"
  "react-reviewer": "openai/gpt-5.2-codex (xhigh)"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "openai/gpt-5.2", "openai/gpt-5.2-codex"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
  "migration-verifier": "anthropic/claude-opus-4-6 (max)"
interactive: yes
completionGateScript: make test
---

COORDINATOR-SPECIFIC:
**MANDATORY**: Add to your master plan this instruction:
- you must also use the "migration-verifier" agent to confirm the parity between old and new versions.

CRITICAL Terminology:
- Standalone Repository: a repository that has only _one_ `jj workspace` -- itself.
- Root Repository: a repository that has more than one `jj workspace`, and it is the root (it is the one in which `.jj/repo` is a directory and not a file)
- Forked Repository: a repository that is part of a `jj workspace, and it is not the root (it is the one in which `.jj/repo` is a text file, whose content points to the parent).

- Repository Mode: is when a repository is served by SGAI in a way that it can actually run software.
- Forked Mode: is when a root repository has at least one child, it displays the fork (dashboard-style) mode.
**CRITICAL** when a Root Repository run out of children, it must revert back from Forked Mode to Repository Mode.

CRITICAL: you must refer to the implementation exposed in http://localhost:7070/trees - you have to browse it extensively and trigger behaviors in order to map what is doing.
          if you need to see the source code, you can by looking back in the `jj log` history.

**CRITICAL** interview me to ask clarification questions.
**CRITICAL** regarding PicoCSS tooltips and browser native tooltips: be extremely critical; if something is a tooltip, when translating into React/shadcn it must be also a tooltip - avoid inlinizations.

---

# Human Review

**HUMAN OBSERVATION**: It seems that the common theme is that you didn't implement the changes necessary for event-driven updates in the UI

- [x] The `Edit GOAL` button is not showing up in the Standalone Repositories
- [x] The `Edit GOAL` button is not showing up in the Forked Repositories
- [x] The Progress events box is not updating as the agent makes progress
- [x] The `Edit GOAL` button inside the `Compose GOAL` doesn't show the frontmatter
  - [x] being able to change the frontmatter is mandatory
  - [x] show and accept the RAW markdown definition of the GOAL.md
- [x] It seems the contents of the Internal Tab is not loading as the agents make progress
  - [x] also, I never saw the Agent TODO being populated, are you reading the source correctly?
- [x] The `Respond To Agent` button doesn't show when the agent is asking me questions
  - [x] I only see the button when I do a hard refresh


---

# Merge

- [x] Fix Merge Conflicts
      refer to `jj log -G -r main@origin::@`

# Exhaustive Hands-On Testing - Take 4

Notes:

- [x] Remove any messaging that's not existent in :7070
  - [x] Among all, "Total execution time"? Where is this from?
- [x] Add messaging that's existent in :7070 but missing in :8181
- [x] The spinner that indicates a workspace is running should only show when the workspace is running
- [x] Where is the "Open in OpenCode"
  - [x] Remember it has complex rules regarding of when it shows or not

- [x] Browse and USE both the sandbox (:7070) and the application (:8181) end-to-end:
  - [x] Create a workspace (e.g., try making a tic-tac-toe game)
  - [x] Use self-drive mode
  - [x] Use interactive mode (Start button)
  - [x] Stop and resume sessions
  - [x] Edit the GOAL file
  - [x] Use the GOAL Compose wizard (templates + guided wizard all steps)
  - [x] Fork a workspace, navigate forks
  - [x] Test Respond to Agent flow
  - [x] Test Retrospectives (create, analyze, apply)
  - [x] Test Run tab (ad-hoc prompt)
  - [x] Map all discovered use cases and expand migration documentation if needed

# Exhaustive Parity Checks - Take 4

- [x] You must assert fully parity between old (available in the sandbox available http://localhost:7070) and new versions (you must start it in port :8181)
  - [x] use migration-milestone-checklist
  - [x] ask 'migration-verifier' for help

---

# Exhaustive Hands-On Testing - Take 3

Notes:

- [x] Remove any messaging that's not existent in :7070
  - [x] Among all, "Total execution time"? Where is this from?
- [x] Add messaging that's existent in :7070 but missing in :8181
- [x] The spinner that indicates a workspace is running should only show when the workspace is running
- [x] Where is the "Open in OpenCode"
  - [x] Remember it has complex rules regarding of when it shows or not

- [x] Browse and USE both the sandbox (:7070) and the application (:8181) end-to-end:
  - [x] Create a workspace (e.g., try making a tic-tac-toe game)
  - [x] Use self-drive mode
  - [x] Use interactive mode (Start button)
  - [x] Stop and resume sessions
  - [x] Edit the GOAL file
  - [x] Use the GOAL Compose wizard (templates + guided wizard all steps)
  - [x] Fork a workspace, navigate forks
  - [x] Test Respond to Agent flow
  - [x] Test Retrospectives (create, analyze, apply)
  - [x] Test Run tab (ad-hoc prompt)
  - [x] Map all discovered use cases and expand migration documentation if needed

# Exhaustive Parity Checks - Take 3

- [x] You must assert fully parity between old (available in the sandbox available http://localhost:7070) and new versions (you must start it in port :8181)
  - [x] use migration-milestone-checklist
  - [x] ask 'migration-verifier' for help


---

# Merge

- [x] Fix Merge Conflicts
      refer to `jj log -G -r main@origin::@`

# Exhaustive Hands-On Testing Redo

- [x] Browse and USE both the sandbox (:7070) and the application (:8181) end-to-end:
  - [x] Create a workspace (e.g., try making a tic-tac-toe game)
  - [x] Use self-drive mode
  - [x] Use interactive mode (Start button)
  - [x] Stop and resume sessions
  - [x] Edit the GOAL file
  - [x] Use the GOAL Compose wizard (templates + guided wizard all steps)
  - [x] Fork a workspace, navigate forks
  - [x] Test Respond to Agent flow
  - [x] Test Retrospectives (create, analyze, apply)
  - [x] Test Run tab (ad-hoc prompt)
  - [x] Map all discovered use cases and expand migration documentation if needed

# Exhaustive Parity Checks Redo

- [x] You must assert fully parity between old (available in the sandbox available http://localhost:7070) and new versions (you must start it in port :8181)
  - [x] use migration-milestone-checklist
  - [x] ask 'migration-verifier' for help



---

# Exhaustive Hands-On Testing Redo

- [x] Browse and USE both the sandbox (:7070) and the application (:8181) end-to-end:
  - [x] Create a workspace (e.g., try making a tic-tac-toe game)
  - [x] Use self-drive mode
  - [x] Use interactive mode (Start button)
  - [x] Stop and resume sessions
  - [x] Edit the GOAL file
  - [x] Use the GOAL Compose wizard (templates + guided wizard all steps)
  - [x] Fork a workspace, navigate forks
  - [x] Test Respond to Agent flow
  - [x] Test Retrospectives (create, analyze, apply)
  - [x] Test Run tab (ad-hoc prompt)
  - [x] Map all discovered use cases and expand migration documentation if needed

---

# Exhaustive Parity Checks Redo

- [x] You must assert fully parity between old (available in the sandbox available http://localhost:7070) and new versions (you must start it in port :8181)
  - [x] use migration-milestone-checklist
  - [x] ask 'migration-verifier' for help

---

# Human Verified Bugs

- [x] menus are incomplete, buttons that show in the sandbox (:7070) not present in the upgraded version
- [x] Root Repository in Forked Mode
  - [x] There is a respond button on the button bar that makes no sense, the responses only go to the Forked Repositories
- [x] `/workspaces/$workspace/goal` is printing HTML instead of the actual source
- [x] `/compose/step/2?workspace=$workspace` - when I click the buttons, nothing happens
- [x] `/compose/step/3?workspace=$workspace` - when I click the toggle, nothing happens
- [x] in `Internals` tab, the PROJECT_MANAGEMENT.md box is not always showing
  - [x] it lacks the unfold chevron when it does show
- [x] in polling mode, in the left tree, when a repository is active with work in progress, the little spinning indicator is not showing up
  - [x] does it work with a hard-refresh?
- [x] in polling mode, in the left tree, when a repository has a user question to ask, I don't see the red-dot indicating the factory needs my attention
  - [x] does it work with a hard-refresh?
- [x] the notification "sgai was interrupted while working. Reset state to start fresh." is not showing
  - [x] the Reset button for this notification must be implemented too

---

# Exhaustive Hands-On Testing

- [x] Browse and USE both the sandbox (:7070) and the application (:8181) end-to-end:
  - [x] Create a workspace (e.g., try making a tic-tac-toe game)
  - [x] Use self-drive mode
  - [x] Use interactive mode (Start button)
  - [x] Stop and resume sessions
  - [x] Edit the GOAL file
  - [x] Use the GOAL Compose wizard (templates + guided wizard all steps)
  - [x] Fork a workspace, navigate forks
  - [x] Test Respond to Agent flow
  - [x] Test Retrospectives (create, analyze, apply)
  - [x] Test Run tab (ad-hoc prompt)
  - [x] Map all discovered use cases and expand migration documentation if needed

---

# Exhaustive Parity Checks

- [x] You must assert fully parity between old (available in the sandbox available http://localhost:7070) and new versions (you must start it in port :8181)
  - [x] use migration-milestone-checklist
  - [x] ask 'migration-verifier' for help

----

# Human Verified Bugs

- Overall
  - [x] Light Mode must be the ONLY color scheme
  - Button Bar
    - [x] Start and Self-Drive buttons must only highlight when they are active
  - [x] the alternation between Root Repository vs Forked Repository is not working
    - [x] it seem that the Forked Mode for Root Repository is not implemented
  - [x] there is a `session started` on the top (close to the button bar) that I think it is made up
  - [x] the screen in which I would be able to answer questions to sgai is not loading
        refer to http://localhost:7070/respond?dir=$dirToWorkspace
    - [x] use the same routing scheme than other routes (`/respond?dir=` is an undesired consequence of the previous design)
  - [x] For `[ + ]`, after I type a name and submit, it should trigger the `Compose GOAL` wizard
    - [x] The `Compose GOAL` wizard has templates in the first that when I click it redirects to 404.
  - [x] There's a weird message "pin toggled" that doesn't exist in the original
  - [x] in the left tree, when I try to navigate between repositories, it doesn't navigate to the repository I click
    - [x] it flickers and returns to a root repository that somehow I managed to land to
  - [x] It seems this application doesn't run well on mobile phones
    - [x] the layout must be responsive, verify against http://localhost:7070/trees
  - [x] a weird session start keeps showing up when I start a session
    - [x] drop `actionMessage` - why did you add this to begin with?
  - [x] when I start a session, there is no updates
    - [x] I think the polling is broken, I don't see the timer counting up.
  - [x] The Respond To Agent Button is not showing in the button bar
    - [x] The Respond To Agent Button is not showing in the repository list in the Root Repository page

- Run Tab
  - [x] Unnecessary Title
  - [x] Not choosing the Default Model correctly (it should be the mode from the Coordinator agent)
  - [x] When I type a message and hit enter, it reloads instead of showing the result of the underlying opencode call
  - [x] the output should be displayed inline, instead of taking me to another screen.
  - [x] it is weird -- I type a message, I hit enter, and then it enters in this weird reload loop
  - [x] place the model selector side by side with the submit button

- Retrospective Tab
  - [x] I am not able to start Retrospectives
  - [x] it seems I can't apply retrospectives
  - [x] it seems retrospectives aren't correctly reporting
  - [x] when I click analyze, I don't see the analysis screen like in the original (http://localhost:7070/workspaces/$workspace/retro/$retrospectiveID/analyze)
  - [x] when I click analyze, I see both `not analyzed` and `analyzing...`
  - [x] when I click analyze, it must go to the analysis screen and start it right away, I don't want to do a second click
  - [x] it is weird -- I type a message, I hit enter, and then it enters in this weird reload loop
        in the logs I see:
```
[retro] [coordinator                   :1577] 1095 |     const info = provider.models[modelID]
[retro] [coordinator                   :1577] 1096 |     if (!info) {
[retro] [coordinator                   :1577] 1097 |       const availableModels = Object.keys(provider.models)
[retro] [coordinator                   :1577] 1098 |       const matches = fuzzysort.go(modelID, availableModels, { limit: 3, threshold: -10000 })
[retro] [coordinator                   :1577] 1099 |       const suggestions = matches.map((m) => m.target)
[retro] [coordinator                   :1577] 1100 |       throw new ModelNotFoundError({ providerID, modelID, suggestions })
[retro] [coordinator                   :1577]                    ^
[retro] [coordinator                   :1577] ProviderModelNotFoundError: ProviderModelNotFoundError
[retro] [coordinator                   :1577]  data: {
[retro] [coordinator                   :1577]   providerID: "anthropic",
[retro] [coordinator                   :1577]   modelID: "claude-opus-4-6 (max)",
[retro] [coordinator                   :1577]   suggestions: [],
[retro] [coordinator                   :1577] },
[retro] [coordinator                   :1577]
[retro] [coordinator                   :1577]       at getModel (src/provider/provider.ts:1100:13)
[retro] [coordinator                   :1577]
```
        the model for retrospectives MUST be the same as the model for the coordinator (model and variant)
  - [x] Once I start a analysis and browse away, I MUST be able to get back and watch it; I am not able to browse back


- Diffs Tab
  - [x] Unnecessary `Commit Description` title in the "Commit Description" box , the box can be just the form

- Progress Tab
  - [x] Where is the "GOAL.md" box?
  - [x] "GOAL.md" box doesn't have the link to open the file in the editor anymore
  - [x] GOAL.md box lacks the chevron to indicates it can be collapsed
    - [x] GOAL.md box must be collapsed by default

- Internal Tab
  - [x] Where is the "PROJECT_MANAGEMENT.md" box?
  - [x] "PROJECT_MANAGEMENT.md" box doesn't have the link to open the file in the editor anymore
  - [x] Where is "Steer Next Turn" ?

- [x] Root Repository in Forked Mode
  - [x] the button bar doesn't look the same as in the original version
  - [x] I see `open in sgai` but I don't see `open in editor`
    - [x] when I click `open in sgai`, it doesn't navigate to the target repository
  - [x] The delete button errors with 404 Not Found
  - [x] when I clicked in `Open in Editor` (top button bar) it showed a message "opened in editor" that doesn't exist in the original
  - [x] IT IS VERY VERY SLOW TO LOAD, it seems that the commit list takes too long to load.
    - [x] you probably need to limit the log list to only show from the common commit from the root repository to the head (`@`) of the fork
  - [x] In Forked Mode, the pills with the timer and status (running v stopped) shouldn't show up.

---

# Parity Evaluation

- [x] evaluate parity please, remember that the sandbox at http://localhost:7070/trees is available with the original version for you to test against

---

# Conciliate

- [x] rebase against main@origin (`jj rebase -d main@origin`)
  - [x] fix merge conflicts
  - [x] REVALIDATE ALL AGAINST THE FIXED MERGES

---

# Bugs

**CRITICAL** the reference behavior can be accessed in http://localhost:7070/trees - this is a sandbox that you can use to exercise execution paths.

- [x] the SVG visualizer in Progress tab doesn't work
- [x] all Markdown viewers seem to not be working correctly
- [x] navigating the old version of the software and using it as a reference, browse this React version of the application, find all bugs and fix all bugs:
  - [x] tabs they have incorrect content
  - [x] missing features from the previous version

----


> **Reference:** See [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

## Mandatory Prerequisites Milestone (Milestone -1)

**CRITICAL**: use the overlay directory (`sgai/`) to create new skills and agents (check the source code to understand how the overlay directory `sgai/` works)

### Skills to Create

- [x] Create `react-sse-patterns` skill — SSE with `useSyncExternalStore`, reconnection with exponential backoff, snapshot rehydration, typed event parsing, connection status UI (references R-1, R-2, R-3, R-19)
- [x] Create `react-shadcn-component-mapping` skill — maps PicoCSS patterns to shadcn/ui equivalents for all 44 HTMX templates
- [x] Create `go-json-api-patterns` skill — `/api/v1/*` JSON endpoints in `serve_api.go`, shared business logic extraction, SSE event emission patterns (R-4), idempotent endpoints (R-10)
- [x] Create `migration-milestone-checklist` skill — checklist for completing each vertical slice milestone (API, React pages, templates replaced, STPA criteria, Playwright parity tests)
- [x] Update `react-best-practices` skill — add bun build (not Vite), `useSyncExternalStore` for SSE, `useReducer+Context` for app state, React 19 `use()` + Suspense, sessionStorage for form state, no optimistic updates for critical actions

### Agent Definition Updates

- [x] Update `react-developer` agent — bun build, shadcn/ui, `useSyncExternalStore` for SSE, `useReducer+Context` for app state, API client in `lib/api.ts`, Suspense for data loading
- [x] Update `react-reviewer` agent — add review criteria for SSE store pattern, hook composition, shadcn usage, accessibility, no optimistic updates, sessionStorage persistence, `beforeunload` handlers
- [x] Update `backend-go-developer` agent — add JSON API endpoint conventions for `/api/v1/*`, SSE event publishing patterns, cookie-based UI switcher, shared business logic extraction, SPA catch-all handler
- [x] Create `migration-verifier` agent — Playwright-based feature parity agent comparing HTMX and React versions via dual-cookie test pattern

### AGENTS.md Directives

- [x] Add bun build tool directive for React/TypeScript code
- [x] Add shadcn/ui component requirement for React components
- [x] Add React testing directive (bun test for unit/component, Playwright for E2E)
- [x] Add React state management patterns directive (`useSyncExternalStore` for SSE, `useReducer+Context` for app state)
- [x] Preserve all existing Go directives (code style, `make lint`, tmux/playwright testing, jj version control)
- [x] Add build verification directive (run `bun run build && make build` when modifying `cmd/sgai/webapp/`)

---

## M0: Foundation

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

### Infrastructure

- [x] Scaffold React project in `cmd/sgai/webapp/` (bun init, React 19, TypeScript, Tailwind CSS v4, shadcn/ui)
- [x] Create `build.ts` — Bun.build config for production builds (`bun build ./src/main.tsx --outdir ./dist --splitting --minify`)
- [x] Create `dev.ts` — Bun.serve dev server with file watching and proxy to Go API on `:8181`
- [x] Create `webapp_embed.go` with `//go:embed webapp/dist/*` to embed React build output in Go binary
- [x] Implement cookie-based UI switcher middleware in Go (`sgai-ui` cookie: `htmx` default, `react` for SPA)
- [x] Implement SPA fallback handler — serve `index.html` for all routes when `sgai-ui=react`, excluding `/api/v1/*` and static assets
- [x] Create SSE store module (`lib/sse-store.ts`) using `useSyncExternalStore` pattern with auto-reconnect
- [x] Create `AppStateProvider` with `useReducer` for app state management
- [x] Configure React Router matching current URL structure (same URLs as HTMX version)
- [x] Integrate into Makefile — `build` target runs `bun run build` before `go build`, with build manifest timestamp
- [x] Add Dependabot config for `npm` ecosystem in `cmd/sgai/webapp/`

### API Endpoints

- [x] `GET /api/v1/events/stream` — SSE endpoint replacing HTMX polling, with typed event names (`workspace:update`, `session:update`, `messages:new`, `todos:update`, `log:append`, `changes:update`, `events:new`, `compose:update`)

### STPA Exit Criteria

- [x] SSE Store implements auto-reconnect with exponential backoff (1s → 2s → 4s → max 30s) (R-1, R-3)
- [x] SSE endpoint supports snapshot mode — send full state as first event on initial connect/reconnect (R-19)
- [x] React UI shows connection status indicator ("Reconnecting..." banner when disconnected >2s) (R-2)
- [x] SPA catch-all handler correctly excludes `/api/v1/*` routes — explicit API routing BEFORE catch-all (R-12, R-23)
- [x] Build pipeline enforces bun build → go build ordering with build manifest timestamp (R-13)
- [x] Cookie switch triggers full page reload (R-16)
- [x] Deep link test: direct URL access to `/workspaces/test` returns React shell with router (R-12)
- [x] SSE events published after transaction commit using deferred event publishing pattern (R-20)

### Standard Exit Criteria

- [x] Toggle `sgai-ui=react` cookie → see blank React app shell with router; SSE connects successfully
- [x] HTMX version continues working unchanged with `sgai-ui=htmx` cookie

---

## M1: Entity Browsers

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

### API Endpoints

- [x] `GET /api/v1/agents` — list all agent definitions
- [x] `GET /api/v1/skills` — list all skills
- [x] `GET /api/v1/skills/{name}` — skill detail by name
- [x] `GET /api/v1/snippets` — list all snippets
- [x] `GET /api/v1/snippets/{lang}` — snippets filtered by language

### React Pages

- [x] `AgentList` page component (replaces `agents.html`)
- [x] `SkillList` page component (replaces `skills.html`)
- [x] `SkillDetail` page component (replaces `skill_detail.html`)
- [x] `SnippetList` page component (replaces `snippets.html`)
- [x] `SnippetDetail` page component (replaces `snippet_detail.html`)

### STPA Exit Criteria

- [x] Unmigrated areas show "Not Yet Available" placeholder with one-click switch back to HTMX (R-6)
- [x] Playwright parity tests pass for entity browser flows on both `sgai-ui=htmx` and `sgai-ui=react` cookies (R-7)
- [x] All page components use Suspense boundaries with skeleton fallbacks for initial load (R-17)

### Standard Exit Criteria

- [x] Entity browsers fully functional in React (agents, skills, snippets — list and detail views)
- [x] HTMX versions still work correctly via cookie toggle

### Feature Parity Check

- [x] Playwright tests run identical entity browsing flows (navigate to list, click detail, verify content) with both cookie values, comparing page content and navigation behavior

---

## M2: Main Dashboard + Workspace Tree

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.


### API Endpoints

- [x] `GET /api/v1/workspaces` — list all workspaces
- [x] `GET /api/v1/workspaces/{name}` — workspace detail with session state
- [x] `POST /api/v1/workspaces` — create new workspace

### React Pages

- [x] `Dashboard` page component with workspace tree sidebar (replaces `trees.html`, `trees_content.html`, `trees_root_workspace.html`)
- [x] `WorkspaceDetail` page component (replaces `trees_workspace.html`)
- [x] `EmptyState` page component for when no workspace is selected (replaces `trees_no_workspace.html`)

### STPA Exit Criteria

- [x] No optimistic updates for workspace creation — use loading states, update UI from SSE/API response only (R-11)
- [x] Playwright parity tests pass for dashboard and workspace tree flows on both cookies (R-7)
- [x] SSE `workspace:update` events arrive after state commit, not during mutation (R-5)

### Standard Exit Criteria

- [x] Full workspace tree with real-time updates via SSE
- [x] Workspace selection and deep links work correctly

### Feature Parity Check

- [x] Playwright tests verify workspace tree rendering, workspace selection, workspace creation, and real-time update behavior on both cookie values

---

## M3: Session Tabs

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.


### API Endpoints

- [x] `GET /api/v1/workspaces/{name}/session` — session state and current agent info
- [x] `GET /api/v1/workspaces/{name}/messages` — inter-agent messages
- [x] `GET /api/v1/workspaces/{name}/todos` — todo list
- [x] `GET /api/v1/workspaces/{name}/log` — output log stream
- [x] `GET /api/v1/workspaces/{name}/changes` — JJ diff/changes
- [x] `GET /api/v1/workspaces/{name}/events` — progress events
- [x] `GET /api/v1/workspaces/{name}/forks` — workspace forks
- [x] `GET /api/v1/workspaces/{name}/retrospectives` — retrospective entries

### React Pages

- [x] `SessionTab` page component (replaces `trees_session_content.html`)
- [x] `SpecificationTab` page component (replaces `trees_specification_content.html`)
- [x] `MessagesTab` page component (replaces `trees_messages_content.html`)
- [x] `LogTab` page component (replaces `trees_log_content.html`)
- [x] `RunTab` page component (replaces `trees_run_content.html`)
- [x] `ChangesTab` page component (replaces `trees_changes_content.html`)
- [x] `EventsTab` page component (replaces `trees_events_content.html`)
- [x] `ForksTab` page component (replaces `trees_forks_content.html`)
- [x] `RetrospectivesTab` page component (replaces `trees_retrospectives_content.html`, `trees_retrospectives_apply_select.html`)

### STPA Exit Criteria

- [x] All mutating endpoints (start/stop) are idempotent — e.g., "Start Session" on running session returns current state (R-10)
- [x] Loading states shown for all agent control commands — no optimistic updates (R-11)
- [x] SSE events emitted via structural middleware, not manual publish calls (R-4)
- [x] Playwright parity tests pass for all session tab flows on both cookies (R-7)

### Standard Exit Criteria

- [x] All session tabs render correctly with live updates via SSE

### Feature Parity Check

- [x] Playwright tests compare all tab contents, tab switching, and real-time update behavior on both cookie values

---

## M4: Response System

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

### API Endpoints

- [x] `POST /api/v1/workspaces/{name}/respond` — submit response to agent question
- [x] `GET /api/v1/workspaces/{name}/pending-question` — get current pending question
- [x] `POST /api/v1/workspaces/{name}/start` — start agent session
- [x] `POST /api/v1/workspaces/{name}/stop` — stop agent session
- [x] `POST /api/v1/workspaces/{name}/reset` — reset agent session

### React Pages

- [x] `ResponseMultiChoice` page component (replaces `response_multichoice.html`)
- [x] `ResponseModal` page component (replaces `response_multichoice_modal.html`)
- [x] `ResponseContext` page component (replaces `response_context.html`)

### STPA Exit Criteria

- [x] Response input persisted to sessionStorage on keystroke, cleared on successful submit (R-8)
- [x] `beforeunload` warning when response textarea has unsaved text (R-9)
- [x] Respond endpoint validates question ID freshness — returns "Question expired" if stale (R-21)
- [x] Mutation buttons (start/stop/reset/respond) disabled during in-flight requests (R-18)
- [x] Playwright parity tests pass for response flows on both cookies (R-7)

### Standard Exit Criteria

- [x] Can interact with agents (respond to questions, start/stop/reset sessions) entirely in React

### Feature Parity Check

- [x] Playwright tests verify response submission, start/stop/reset controls, and question display on both cookie values

---

## M5: GOAL Composer Wizard

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

### API Endpoints

- [x] `GET /api/v1/compose` — get current compose state
- [x] `POST /api/v1/compose` — save final GOAL.md
- [x] `GET /api/v1/compose/templates` — list available GOAL templates
- [x] `GET /api/v1/compose/preview` — preview generated GOAL.md
- [x] `POST /api/v1/compose/draft` — save wizard draft

### React Pages

- [x] `ComposeLanding` page component (replaces `compose_landing.html`)
- [x] `WizardStep1` page component (replaces `compose_wizard_step1.html`)
- [x] `WizardStep2` page component (replaces `compose_wizard_step2.html`)
- [x] `WizardStep3` page component (replaces `compose_wizard_step3.html`)
- [x] `WizardStep4` page component (replaces `compose_wizard_step4.html`)
- [x] `WizardFinish` page component (replaces `compose_wizard_finish.html`)
- [x] `ComposePreview` page component (replaces `compose_preview.html`, `compose_preview_partial.html`)

### STPA Exit Criteria

- [x] Wizard state persisted to sessionStorage per step — survives route changes and page reloads (R-14)
- [x] Wizard URL reflects current step for deep linking (e.g., `/compose/step/3`) (R-14)
- [x] Auto-save draft to backend every 30s with "Draft saved" indicator (R-15)
- [x] `beforeunload` warning when wizard has unsaved progress (R-9)
- [x] GOAL.md save uses optimistic locking/etag to prevent concurrent save conflicts (R-24)
- [x] Playwright parity tests pass for full wizard flow (all steps, preview, save) on both cookies (R-7)

### Standard Exit Criteria

- [x] Full GOAL.md creation wizard works in React (all steps, preview, save)

### Feature Parity Check

- [x] Playwright tests run the complete wizard flow (all steps, preview, save) on both cookie values, verifying identical results

---

## M6: Workspace Management + Remaining

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

### API Endpoints

- [x] `POST /api/v1/workspaces/{name}/fork` — fork a workspace
- [x] `POST /api/v1/workspaces/{name}/merge` — merge a fork
- [x] `POST /api/v1/workspaces/{name}/rename` — rename a workspace/fork
- [x] `PUT /api/v1/workspaces/{name}/goal` — update GOAL.md content
- [x] `POST /api/v1/workspaces/{name}/adhoc` — execute ad-hoc prompt
- [x] `POST /api/v1/workspaces/{name}/retrospective/analyze` — run retrospective analysis
- [x] `POST /api/v1/workspaces/{name}/retrospective/apply` — apply retrospective recommendations

### React Pages

- [x] `NewWorkspace` page component
- [x] `NewFork` page component
- [x] `RenameFork` page component
- [x] `EditGoal` page component
- [x] `AdhocOutput` page component
- [x] `RetroAnalyze` page component
- [x] `RetroApply` page component

### STPA Exit Criteria

- [x] All mutation endpoints have client-side deduplication — disable buttons on click, track in-flight requests (R-18)
- [x] Playwright parity tests pass for all remaining flows on both cookies (R-7)

### Standard Exit Criteria

- [x] ALL 44 HTMX templates have React equivalents

### Feature Parity Check

- [x] Playwright tests verify all remaining functionality (fork, merge, rename, edit goal, ad-hoc prompt, retrospective analyze/apply) on both cookie values

---

## M7: Polish + HTMX Removal

> **Reference:** REVIEW [REACT_MIGRATION_PLAN.md](REACT_MIGRATION_PLAN.md) for full architectural details, STPA analysis, and design decisions.

### Deliverables

- [x] Default `sgai-ui` cookie to `react` and test all pages thoroughly
- [x] Remove all HTMX templates (`cmd/sgai/templates/`), PicoCSS, and Idiomorph
- [x] Remove cookie switcher logic from Go middleware
- [x] Remove old HTML handlers from `serve.go` — keep only `/api/v1/*` endpoints
- [x] Performance audit and accessibility review of React SPA

### STPA Exit Criteria

- [x] Full Playwright test suite passes on React-only (no HTMX fallback) (R-7)
- [x] Headless smoke test validates embedded `dist/` mounts correctly in Go binary (R-22)
- [x] All deep link patterns tested via Playwright — bookmarked URLs resolve correctly (R-12)
- [x] SSE connection resilience tested — network drop simulation with successful reconnection (R-1, R-3)

### Standard Exit Criteria

- [x] Single React SPA, no HTMX code remains, all tests pass

### Feature Parity Check

- [x] Final Playwright test suite runs all user flows on React-only, verifying complete feature parity with the former HTMX interface (baseline screenshots/assertions captured before HTMX removal)
