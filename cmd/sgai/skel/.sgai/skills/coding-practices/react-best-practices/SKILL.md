---
name: react-best-practices
description: React and Next.js performance optimization guidelines from Vercel Engineering, plus SGAI project-specific patterns (bun build, useSyncExternalStore for SSE, useReducer+Context for app state, React 19 use() + Suspense, sessionStorage form persistence, no optimistic updates for critical actions). This skill should be used when writing, reviewing, or refactoring React/TypeScript code to ensure optimal performance patterns. Triggers on tasks involving React components, data fetching, bundle optimization, performance improvements, SSE integration, or state management.
license: MIT
metadata:
  author: vercel
  version: "2.0.0"
---

# React Best Practices

Comprehensive performance optimization guide for React applications. Contains the Vercel Engineering 57 rules across 8 categories, plus SGAI project-specific patterns for the React migration.

## When to Apply

Reference these guidelines when:
- Writing new React components or pages
- Implementing data fetching (API calls or SSE)
- Reviewing code for performance issues
- Refactoring existing React code
- Optimizing bundle size or load times
- Implementing state management
- Working with form state persistence

---

## SGAI Project-Specific Patterns

> **These patterns are mandatory for all React code in `cmd/sgai/webapp/`.** They take precedence over generic patterns when there is a conflict.

### Build Tool: bun (Native Bundler)

**Use `bun build` (bun's native bundler), NOT Vite.**

| Command | Purpose |
|---------|---------|
| `bun build ./src/main.tsx --outdir ./dist --splitting --minify` | Production build |
| `bun run dev.ts` | Dev server (Bun.serve() with watch + proxy) |
| `bun test` | Run tests (vitest-compatible API) |

**Build verification:** After modifying `cmd/sgai/webapp/`, always run:
```bash
cd cmd/sgai/webapp && bun run build && cd ../../.. && make build
```

### SSE Data Fetching: `useSyncExternalStore`

**Use `useSyncExternalStore` for Server-Sent Events. Do NOT use React Context for SSE.**

The SSE store (`lib/sse-store.ts`) is an **external store** at module level. Components subscribe via `useSyncExternalStore(store.subscribe, store.getSnapshot)`.

```typescript
// CORRECT: External store with useSyncExternalStore
import { useSyncExternalStore } from 'react';
import { sseStore } from '../lib/sse-store';

function useSSEEvent(eventType: SSEEventType) {
  const state = useSyncExternalStore(sseStore.subscribe, sseStore.getSnapshot);
  return state.events.get(eventType);
}
```

```typescript
// WRONG: Do NOT use React Context for SSE
const SSEContext = createContext<EventSource | null>(null);
```

**Why:** React's official recommendation for external data sources is `useSyncExternalStore`. Context would cause unnecessary re-renders for all consumers on every SSE event.

See the `react-sse-patterns` skill for detailed SSE implementation guidance.

### App State: `useReducer` + Context

**Use `useReducer` + Context for app state management. Do NOT use Redux, Zustand, or other state management libraries.**

```typescript
// CORRECT: useReducer + Context
const AppStateContext = createContext<AppState | null>(null);
const AppDispatchContext = createContext<Dispatch<AppAction> | null>(null);

function AppStateProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(appReducer, initialState);
  return (
    <AppStateContext.Provider value={state}>
      <AppDispatchContext.Provider value={dispatch}>
        {children}
      </AppDispatchContext.Provider>
    </AppStateContext.Provider>
  );
}
```

**Actions:** `workspace/select`, `ui/togglePanel`, `ui/setTab`. SSE events dispatch actions into this reducer.

### Data Loading: React 19 `use()` + Suspense

**Use React 19's `use()` hook with `<Suspense>` for initial data fetching.**

```typescript
// CORRECT: use() + Suspense
import { use, Suspense } from 'react';

const workspacesPromise = api.getWorkspaces();

function WorkspaceList() {
  const workspaces = use(workspacesPromise);
  return <div>{workspaces.map(renderWorkspace)}</div>;
}

// Usage:
<Suspense fallback={<Skeleton />}>
  <WorkspaceList />
</Suspense>
```

**Pattern for domain hooks:** Combine `use()` for initial data + `useSyncExternalStore` for SSE live updates.

```typescript
export function useWorkspaces() {
  const initial = use(workspacesPromise);
  const sse = useSyncExternalStore(sseStore.subscribe, sseStore.getSnapshot);
  const live = sse.events.get('workspace:update');
  return live ? live.data : initial;
}
```

### Form State Persistence: sessionStorage

**Persist form state to sessionStorage on keystroke. Clear on successful submit.**

```typescript
function usePersistedFormState(key: string, initialValue: string) {
  const [value, setValue] = useState(() => {
    const stored = sessionStorage.getItem(key);
    return stored ?? initialValue;
  });

  const handleChange = (newValue: string) => {
    setValue(newValue);
    sessionStorage.setItem(key, newValue);
  };

  const clear = () => {
    setValue(initialValue);
    sessionStorage.removeItem(key);
  };

  return [value, handleChange, clear] as const;
}
```

**STPA references:** R-8 (sessionStorage persistence), R-14 (wizard state persistence).

### Critical Actions: No Optimistic Updates

**Do NOT use optimistic updates for critical workflow actions (start, stop, respond, reset, create workspace, save GOAL.md).**

```typescript
// CORRECT: Loading state, update from API/SSE response
const [isPending, startTransition] = useTransition();

function handleStart() {
  startTransition(async () => {
    const result = await api.startSession(name);
    // UI updates from SSE event, not optimistic
  });
}

<Button disabled={isPending} onClick={handleStart}>
  {isPending ? <Loader2 className="animate-spin" /> : 'Start Session'}
</Button>
```

```typescript
// WRONG: Optimistic update for critical action
function handleStart() {
  dispatch({ type: 'session/started' }); // DON'T update UI before API confirms
  api.startSession(name);
}
```

**Why (R-11):** Critical actions have side effects that can't be undone. Show loading states, update UI only from server confirmation (API response or SSE event).

### `beforeunload` Warning for Unsaved Data

**Add `beforeunload` handler when forms have unsaved data (R-9).**

```typescript
function useBeforeUnload(hasUnsavedData: boolean) {
  useEffect(() => {
    if (!hasUnsavedData) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [hasUnsavedData]);
}

// Usage:
const [response, setResponse, clearResponse] = usePersistedFormState('response-input', '');
useBeforeUnload(response.length > 0);
```

### Mutation Button Deduplication

**Disable mutation buttons during in-flight requests (R-18).**

```typescript
function MutationButton({ onClick, children }: Props) {
  const [isPending, startTransition] = useTransition();

  return (
    <Button
      disabled={isPending}
      onClick={() => startTransition(async () => { await onClick(); })}
    >
      {isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
      {children}
    </Button>
  );
}
```

### shadcn/ui Components

**React components must use shadcn/ui components where possible.** Do not create custom implementations when a shadcn component exists.

Reference: [shadcn/ui docs](https://ui.shadcn.com/docs)

See the `react-shadcn-component-mapping` skill for PicoCSS-to-shadcn mapping.

---

## Vercel Engineering Rule Categories by Priority

| Priority | Category | Impact | Prefix |
|----------|----------|--------|--------|
| 1 | Eliminating Waterfalls | CRITICAL | `async-` |
| 2 | Bundle Size Optimization | CRITICAL | `bundle-` |
| 3 | Server-Side Performance | HIGH | `server-` |
| 4 | Client-Side Data Fetching | MEDIUM-HIGH | `client-` |
| 5 | Re-render Optimization | MEDIUM | `rerender-` |
| 6 | Rendering Performance | MEDIUM | `rendering-` |
| 7 | JavaScript Performance | LOW-MEDIUM | `js-` |
| 8 | Advanced Patterns | LOW | `advanced-` |

## Quick Reference

### 1. Eliminating Waterfalls (CRITICAL)

- `async-defer-await` - Move await into branches where actually used
- `async-parallel` - Use Promise.all() for independent operations
- `async-dependencies` - Use better-all for partial dependencies
- `async-api-routes` - Start promises early, await late in API routes
- `async-suspense-boundaries` - Use Suspense to stream content

### 2. Bundle Size Optimization (CRITICAL)

- `bundle-barrel-imports` - Import directly, avoid barrel files
- `bundle-dynamic-imports` - Use next/dynamic for heavy components
- `bundle-defer-third-party` - Load analytics/logging after hydration
- `bundle-conditional` - Load modules only when feature is activated
- `bundle-preload` - Preload on hover/focus for perceived speed

### 3. Server-Side Performance (HIGH)

- `server-auth-actions` - Authenticate server actions like API routes
- `server-cache-react` - Use React.cache() for per-request deduplication
- `server-cache-lru` - Use LRU cache for cross-request caching
- `server-dedup-props` - Avoid duplicate serialization in RSC props
- `server-serialization` - Minimize data passed to client components
- `server-parallel-fetching` - Restructure components to parallelize fetches
- `server-after-nonblocking` - Use after() for non-blocking operations

### 4. Client-Side Data Fetching (MEDIUM-HIGH)

- `client-swr-dedup` - Use SWR for automatic request deduplication
- `client-event-listeners` - Deduplicate global event listeners
- `client-passive-event-listeners` - Use passive listeners for scroll
- `client-localstorage-schema` - Version and minimize localStorage data

### 5. Re-render Optimization (MEDIUM)

- `rerender-defer-reads` - Don't subscribe to state only used in callbacks
- `rerender-memo` - Extract expensive work into memoized components
- `rerender-memo-with-default-value` - Hoist default non-primitive props
- `rerender-dependencies` - Use primitive dependencies in effects
- `rerender-derived-state` - Subscribe to derived booleans, not raw values
- `rerender-derived-state-no-effect` - Derive state during render, not effects
- `rerender-functional-setstate` - Use functional setState for stable callbacks
- `rerender-lazy-state-init` - Pass function to useState for expensive values
- `rerender-simple-expression-in-memo` - Avoid memo for simple primitives
- `rerender-move-effect-to-event` - Put interaction logic in event handlers
- `rerender-transitions` - Use startTransition for non-urgent updates
- `rerender-use-ref-transient-values` - Use refs for transient frequent values

### 6. Rendering Performance (MEDIUM)

- `rendering-animate-svg-wrapper` - Animate div wrapper, not SVG element
- `rendering-content-visibility` - Use content-visibility for long lists
- `rendering-hoist-jsx` - Extract static JSX outside components
- `rendering-svg-precision` - Reduce SVG coordinate precision
- `rendering-hydration-no-flicker` - Use inline script for client-only data
- `rendering-hydration-suppress-warning` - Suppress expected mismatches
- `rendering-activity` - Use Activity component for show/hide
- `rendering-conditional-render` - Use ternary, not && for conditionals
- `rendering-usetransition-loading` - Prefer useTransition for loading state

### 7. JavaScript Performance (LOW-MEDIUM)

- `js-batch-dom-css` - Group CSS changes via classes or cssText
- `js-index-maps` - Build Map for repeated lookups
- `js-cache-property-access` - Cache object properties in loops
- `js-cache-function-results` - Cache function results in module-level Map
- `js-cache-storage` - Cache localStorage/sessionStorage reads
- `js-combine-iterations` - Combine multiple filter/map into one loop
- `js-length-check-first` - Check array length before expensive comparison
- `js-early-exit` - Return early from functions
- `js-hoist-regexp` - Hoist RegExp creation outside loops
- `js-min-max-loop` - Use loop for min/max instead of sort
- `js-set-map-lookups` - Use Set/Map for O(1) lookups
- `js-tosorted-immutable` - Use toSorted() for immutability

### 8. Advanced Patterns (LOW)

- `advanced-event-handler-refs` - Store event handlers in refs
- `advanced-init-once` - Initialize app once per app load
- `advanced-use-latest` - useLatest for stable callback refs

## How to Use

Read individual rule files for detailed explanations and code examples:

```
rules/async-parallel.md
rules/bundle-barrel-imports.md
```

Each rule file contains:
- Brief explanation of why it matters
- Incorrect code example with explanation
- Correct code example with explanation
- Additional context and references

## Full Compiled Document

For the complete guide with all rules expanded: `AGENTS.md`
