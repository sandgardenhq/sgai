---
name: migration-milestone-checklist
description: "Checklist for completing each vertical slice milestone during the React migration. Use when finishing a milestone (M0-M7), verifying exit criteria, running parity tests, or signing off on milestone completion. Triggers on milestone completion, exit criteria verification, parity test, sign-off, or milestone review tasks."
---

# Migration Milestone Checklist

## Overview

Every milestone (M0-M7) in the React migration is a vertical slice that must be fully complete before moving on. This skill provides the generic checklist that applies to every milestone, plus per-milestone specifics.

**STPA Reference:** R-7 (Playwright feature parity tests per milestone).

## When to Use

- Use when completing any milestone (M0-M7)
- Use when verifying exit criteria before sign-off
- Use when running the dual-cookie Playwright parity test pattern
- Use when reviewing whether a milestone is truly done
- Don't use for in-progress development — use when you think you're done

## Generic Milestone Completion Checklist

Every milestone must satisfy ALL of the following before it can be marked complete:

### 1. API Endpoints

- [ ] All endpoints listed in GOAL.md for this milestone are implemented in `serve_api.go`
- [ ] Each endpoint follows the JSON handler pattern (see `go-json-api-patterns` skill)
- [ ] Error responses use consistent `{"error": "...", "code": "..."}` format
- [ ] Shared business logic extracted (not reimplemented)
- [ ] Go API tests written and passing (`go test ./...`)

### 2. React Page Components

- [ ] All React pages listed in GOAL.md for this milestone are implemented
- [ ] Pages use shadcn/ui components (see `react-shadcn-component-mapping` skill)
- [ ] Pages use `use()` + Suspense for initial data loading
- [ ] Pages use `useSyncExternalStore` for SSE live updates where applicable
- [ ] Component unit/integration tests written and passing (`bun test`)

### 3. HTMX Templates Replaced

- [ ] All templates listed for this milestone have React equivalents
- [ ] Template-to-component mapping verified against REACT_MIGRATION_PLAN.md appendix
- [ ] HTMX templates remain untouched (still working for `sgai-ui=htmx`)

### 4. STPA Exit Criteria

- [ ] All STPA-derived exit criteria from GOAL.md for this milestone are verified
- [ ] Each criterion has a concrete verification (test, manual check, or Playwright assertion)
- [ ] No STPA criteria skipped or deferred

### 5. Playwright Parity Tests (R-7)

- [ ] Dual-cookie Playwright tests written for this milestone's flows
- [ ] Tests run identical flows with `sgai-ui=htmx` and `sgai-ui=react` cookies
- [ ] Both cookie values produce equivalent results
- [ ] Tests cover: page loads, navigation, form submissions, real-time updates, deep links

### 6. Standard Exit Criteria

- [ ] All standard exit criteria from GOAL.md for this milestone are met
- [ ] HTMX version continues working unchanged with `sgai-ui=htmx` cookie

### 7. Build Verification

- [ ] `bun run build` succeeds in `cmd/sgai/webapp/`
- [ ] `make build` succeeds (Go binary compiles with embedded React dist)
- [ ] `make lint` passes
- [ ] `make test` passes (includes both `bun test` and `go test`)

### 8. Feature Parity Check

- [ ] Feature parity verification from GOAL.md for this milestone is confirmed
- [ ] No functionality lost in React version compared to HTMX version

## Dual-Cookie Playwright Test Pattern

The standard pattern for feature parity testing:

```typescript
import { test, expect } from '@playwright/test';

for (const uiMode of ['htmx', 'react'] as const) {
  test.describe(`Feature X (${uiMode})`, () => {
    test.beforeEach(async ({ context }) => {
      await context.addCookies([{
        name: 'sgai-ui',
        value: uiMode,
        url: 'http://127.0.0.1:8181'
      }]);
    });

    test('performs action correctly', async ({ page }) => {
      await page.goto('http://127.0.0.1:8181/path');
      // ... identical test steps for both UIs
      // Assert: expected behavior
    });
  });
}
```

**Key points:**
- Tests run at `127.0.0.1:8181` (as per project conventions)
- Same test steps for both cookie values
- Assertions verify equivalent behavior (not identical DOM)
- Each milestone adds parity tests for its specific flows

## STPA Exit Criteria Verification

To verify STPA exit criteria:

1. **Read the STPA criteria** from GOAL.md for the current milestone
2. **For each criterion**, determine verification method:
   - **Automated test** — Playwright test or unit test that asserts the property
   - **Manual verification** — Developer performs the action and confirms behavior
   - **Code review** — Reviewer confirms implementation matches criterion
3. **Document the verification** — Include in the coordinator message which test file or manual step verifies each STPA criterion (e.g., "R-6 verified by entity-browsers.spec.ts line 42" or "R-17 verified by manual check: Suspense skeleton shown on slow network")
4. **No criteria skipped** — Every STPA criterion must have at least one verification

## Per-Milestone Specifics

### M0: Foundation

**Focus:** Infrastructure scaffolding, no user-visible features.

- SSE store with auto-reconnect and exponential backoff (R-1, R-3)
- SSE snapshot rehydration on connect/reconnect (R-19)
- Connection status banner after >2s disconnect (R-2)
- Cookie-based UI switcher with full page reload on switch (R-16)
- SPA catch-all with `/api/v1/*` exclusion (R-12, R-23)
- Build pipeline: bun build → go build with manifest timestamp (R-13)
- Deep link test: direct URL access returns React shell (R-12)
- SSE events published after transaction commit (R-20)

**Parity check:** N/A (foundation only, no migrated features)

### M1: Entity Browsers

**Focus:** Read-only entity lists (agents, skills, snippets).

- Unmigrated areas show "Not Yet Available" with one-click HTMX switch (R-6)
- Suspense boundaries with skeleton fallbacks (R-17)
- Parity tests for entity browsing flows

### M2: Dashboard + Workspace Tree

**Focus:** Core landing experience with real-time updates.

- No optimistic updates for workspace creation (R-11)
- SSE `workspace:update` events after state commit (R-5)
- Parity tests for workspace tree, selection, creation

### M3: Session Tabs

**Focus:** All session information tabs with live updates.

- Idempotent mutating endpoints — start/stop on running/stopped returns current state (R-10)
- Loading states for all agent control commands (R-11)
- SSE events emitted via structural middleware (R-4)
- Parity tests for all tab flows

### M4: Response System

**Focus:** Agent interaction — respond, start, stop, reset.

- sessionStorage persistence for response input on keystroke, clear on submit (R-8)
- `beforeunload` warning when response has unsaved text (R-9)
- Question ID freshness validation on respond endpoint (R-21)
- Mutation buttons disabled during in-flight requests (R-18)
- Parity tests for response flows

### M5: GOAL Composer Wizard

**Focus:** Multi-step wizard with state persistence.

- Wizard state in sessionStorage per step, survives route changes/reloads (R-14)
- Wizard URL reflects step for deep linking (`/compose/step/3`) (R-14)
- Auto-save draft to backend every 30s with "Draft saved" indicator (R-15)
- `beforeunload` warning for unsaved wizard progress (R-9)
- GOAL.md save uses optimistic locking/etag (R-24)
- Parity tests for full wizard flow

### M6: Workspace Management + Remaining

**Focus:** All remaining templates — fork, merge, rename, ad-hoc, retrospectives.

- Client-side mutation deduplication — disable buttons, track in-flight (R-18)
- ALL 44 HTMX templates have React equivalents
- Parity tests for all remaining flows

### M7: Polish + HTMX Removal

**Focus:** React becomes sole interface.

- Full Playwright test suite passes on React-only (R-7)
- Headless smoke test validates embedded `dist/` mounts correctly (R-22)
- All deep link patterns tested via Playwright (R-12)
- SSE connection resilience tested — network drop simulation (R-1, R-3)
- Remove HTMX templates, PicoCSS, Idiomorph, cookie switcher

## Sign-Off Process

1. **Developer** completes all checklist items above
2. **Developer** runs full test suite (`make test`)
3. **Developer** sends message to coordinator: `"GOAL COMPLETE: [milestone description]"`
4. **Coordinator** verifies completion and marks GOAL.md checkboxes
5. **Project critic council** (optional) reviews for completeness

## Rules

1. **No milestone is complete with unchecked items** — Every applicable item on the generic checklist must be satisfied. For milestones where a section doesn't apply (e.g., M0 has no Feature Parity Check), explicitly mark it "N/A — [reason]" in the coordinator message. Unmarked items are assumed incomplete.

2. **STPA criteria are not optional** — They exist because the STPA analysis identified real risks. Skipping them reintroduces those risks.

3. **Parity tests must run on both cookies** — Testing only the React version doesn't verify feature parity. The dual-cookie pattern is mandatory (R-7).

4. **HTMX must keep working** — Every milestone leaves the HTMX version fully functional. Breaking HTMX is a migration regression (L-4).

5. **Build must pass** — `bun run build && make build && make lint && make test` must all succeed.

## Red Flags - STOP

- "We can skip the parity tests for this milestone"
- "The STPA criteria don't apply to this case"
- "HTMX works fine, we don't need to test it"
- "We'll add the tests later"
- "The build is broken but the feature works in dev"
- "My unit tests pass, I don't need to run the full `make test`"
- "The coordinator can verify the STPA criteria during review"
- "This is 90% done, we can finish the edge cases in the next milestone"

## Checklist

Before marking a milestone complete:

- [ ] All 8 generic checklist sections verified
- [ ] Per-milestone specifics verified
- [ ] `make test` passes
- [ ] Coordinator notified with "GOAL COMPLETE: ..." message
