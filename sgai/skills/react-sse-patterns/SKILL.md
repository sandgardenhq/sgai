---
name: react-sse-patterns
description: SSE with useSyncExternalStore, reconnection with exponential backoff, snapshot rehydration, typed event parsing, connection status UI. Use when implementing SSE data stores, real-time update hooks, or connection resilience in the React SPA. Triggers on SSE, EventSource, useSyncExternalStore, real-time updates, reconnection, or live data tasks.
---

# React SSE Patterns

## Overview

Guide for implementing Server-Sent Events (SSE) in the SGAI React SPA using React's `useSyncExternalStore` pattern. Covers the external store module, auto-reconnect with exponential backoff, snapshot rehydration on reconnect, typed event parsing, connection status UI, and domain hooks that combine initial fetch with live SSE updates.

**STPA References:** R-1 (auto-reconnect), R-2 (connection status banner), R-3 (unbounded reconnection), R-19 (snapshot rehydration).

## When to Use

- Use when building or modifying `lib/sse-store.ts`
- Use when creating domain hooks that subscribe to SSE events (e.g., `useWorkspaces`, `useSession`)
- Use when implementing connection status indicators
- Use when debugging SSE reconnection or stale data issues
- Don't use for non-SSE data fetching patterns (use `react-best-practices` instead)

## Architecture

```
SSE Store (lib/sse-store.ts)          NOT in React Context
  ├── EventSource connection          Singleton, module-level
  ├── subscribe() / getSnapshot()     useSyncExternalStore API
  ├── Auto-reconnect                  Exponential backoff 1s → 30s
  └── Typed event parsing             workspace:update, session:update, etc.

Domain Hooks (hooks/)
  ├── useWorkspaces()                 Initial fetch + SSE live updates
  ├── useSession(name)                Initial fetch + SSE live updates
  ├── useMessages(name)               Initial fetch + SSE live updates
  └── ...                             Pattern: fetch + subscribe
```

**Key constraint:** The SSE store is an **external store**, NOT a React Context. This follows React's official recommendation for external data sources. Components subscribe via `useSyncExternalStore(store.subscribe, store.getSnapshot)`.

## Process

### Step 1: SSE Store Module (`lib/sse-store.ts`)

The SSE store is a standalone TypeScript module that manages the `EventSource` connection at module level.

```typescript
// lib/sse-store.ts

// --- Types ---

type SSEEventType =
  | 'workspace:update'
  | 'session:update'
  | 'messages:new'
  | 'todos:update'
  | 'log:append'
  | 'changes:update'
  | 'events:new'
  | 'compose:update';

interface SSEEvent<T = unknown> {
  type: SSEEventType;
  data: T;
  timestamp: number;
}

type ConnectionStatus = 'connected' | 'disconnected' | 'reconnecting';

interface SSEStoreState {
  connectionStatus: ConnectionStatus;
  lastEventTimestamp: number;
  events: Map<SSEEventType, SSEEvent>;
}

// --- Module-level state (NOT React state) ---

let state: SSEStoreState = {
  connectionStatus: 'disconnected',
  lastEventTimestamp: 0,
  events: new Map(),
};

let eventSource: EventSource | null = null;
let reconnectAttempts = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
const listeners = new Set<() => void>();

// --- Subscribe / getSnapshot for useSyncExternalStore ---

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot(): SSEStoreState {
  return state;
}

function emitChange(): void {
  // Create new state reference to trigger React re-render
  state = { ...state };
  for (const listener of listeners) {
    listener();
  }
}

// --- Connection Management ---

const MAX_BACKOFF_MS = 30_000;
const BASE_BACKOFF_MS = 1_000;

function getBackoffDelay(): number {
  const delay = Math.min(
    BASE_BACKOFF_MS * Math.pow(2, reconnectAttempts),
    MAX_BACKOFF_MS
  );
  return delay;
}

function connect(): void {
  if (eventSource) {
    eventSource.close();
  }

  eventSource = new EventSource('/api/v1/events/stream');

  eventSource.onopen = () => {
    reconnectAttempts = 0;
    state = { ...state, connectionStatus: 'connected' };
    emitChange();
  };

  eventSource.onerror = () => {
    eventSource?.close();
    eventSource = null;
    state = { ...state, connectionStatus: 'reconnecting' };
    emitChange();
    scheduleReconnect();
  };

  // Register typed event listeners
  const eventTypes: SSEEventType[] = [
    'workspace:update',
    'session:update',
    'messages:new',
    'todos:update',
    'log:append',
    'changes:update',
    'events:new',
    'compose:update',
  ];

  for (const eventType of eventTypes) {
    eventSource.addEventListener(eventType, (e: MessageEvent) => {
      const event: SSEEvent = {
        type: eventType,
        data: JSON.parse(e.data),
        timestamp: Date.now(),
      };
      state = {
        ...state,
        lastEventTimestamp: event.timestamp,
        events: new Map(state.events).set(eventType, event),
      };
      emitChange();
    });
  }
}

function scheduleReconnect(): void {
  // R-3: Unbounded reconnection - never give up
  if (reconnectTimer) clearTimeout(reconnectTimer);
  const delay = getBackoffDelay();
  reconnectAttempts++;
  reconnectTimer = setTimeout(() => {
    connect();
  }, delay);
}

function disconnect(): void {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  eventSource?.close();
  eventSource = null;
  state = { ...state, connectionStatus: 'disconnected' };
  emitChange();
}

// --- Public API ---

export const sseStore = {
  subscribe,
  getSnapshot,
  connect,
  disconnect,
};
```

### Step 2: Exponential Backoff (R-1, R-3)

Reconnection uses exponential backoff: `1s → 2s → 4s → 8s → 16s → 30s (max)`.

**Rules:**
- Base delay: 1 second
- Multiplier: 2x per attempt
- Maximum delay: 30 seconds
- **Never give up** — reconnection attempts are unbounded (R-3)
- Reset attempt counter to 0 on successful connection

```
Attempt 0: 1s
Attempt 1: 2s
Attempt 2: 4s
Attempt 3: 8s
Attempt 4: 16s
Attempt 5+: 30s (capped)
```

### Step 3: Snapshot Rehydration (R-19)

On initial connect or reconnect, the SSE endpoint sends a full state snapshot as the first event. This prevents stale data after reconnection.

**Server-side contract:** `GET /api/v1/events/stream` sends a `snapshot` event as the first message containing the complete current state. After that, incremental events follow.

**Client-side handling:**

```typescript
eventSource.addEventListener('snapshot', (e: MessageEvent) => {
  const snapshot = JSON.parse(e.data);
  // Replace entire state from snapshot
  const newEvents = new Map<SSEEventType, SSEEvent>();
  for (const [type, data] of Object.entries(snapshot)) {
    newEvents.set(type as SSEEventType, {
      type: type as SSEEventType,
      data,
      timestamp: Date.now(),
    });
  }
  state = {
    ...state,
    events: newEvents,
    lastEventTimestamp: Date.now(),
  };
  emitChange();
});
```

### Step 4: Connection Status UI (R-2)

Show a "Reconnecting..." banner when disconnected for more than 2 seconds.

```typescript
// hooks/useConnectionStatus.ts
import { useSyncExternalStore } from 'react';
import { sseStore } from '../lib/sse-store';

export function useConnectionStatus(): ConnectionStatus {
  const state = useSyncExternalStore(
    sseStore.subscribe,
    sseStore.getSnapshot
  );
  return state.connectionStatus;
}

// components/ConnectionBanner.tsx
export function ConnectionBanner() {
  const status = useConnectionStatus();
  const [showBanner, setShowBanner] = useState(false);

  useEffect(() => {
    if (status === 'reconnecting' || status === 'disconnected') {
      const timer = setTimeout(() => setShowBanner(true), 2000);
      return () => clearTimeout(timer);
    }
    setShowBanner(false);
  }, [status]);

  if (!showBanner) return null;

  return (
    <Alert variant="warning">
      Reconnecting to server...
    </Alert>
  );
}
```

### Step 5: Typed Event Parsing

All SSE events have typed names. Parse and validate on receive:

| Event Name | Payload Type | Description |
|------------|-------------|-------------|
| `workspace:update` | `WorkspaceData` | Workspace list/detail changed |
| `session:update` | `SessionData` | Session status, current agent, workflow state |
| `messages:new` | `MessageData` | New inter-agent message |
| `todos:update` | `TodoData` | Todo list changed |
| `log:append` | `LogData` | New output log lines |
| `changes:update` | `ChangesData` | JJ diff changed |
| `events:new` | `EventData` | New progress event |
| `compose:update` | `ComposeData` | GOAL.md composer state |

Define TypeScript interfaces for each payload in `types/sse.ts`.

### Step 6: Domain Hooks (Initial Fetch + SSE)

Domain hooks combine an initial API fetch with SSE live updates:

```typescript
// hooks/useWorkspaces.ts
import { use, useSyncExternalStore } from 'react';
import { sseStore } from '../lib/sse-store';
import { api } from '../lib/api';

const workspacesPromise = api.getWorkspaces();

export function useWorkspaces() {
  // Initial data via React 19 use() + Suspense
  const initialData = use(workspacesPromise);

  // Live updates via SSE
  const sseState = useSyncExternalStore(
    sseStore.subscribe,
    sseStore.getSnapshot
  );

  const liveData = sseState.events.get('workspace:update');

  // SSE data takes precedence when available
  return liveData ? liveData.data : initialData;
}
```

**Pattern:** Every domain hook follows this structure:
1. `use(fetchPromise)` for initial data (wrapped in Suspense)
2. `useSyncExternalStore` for live SSE updates
3. Return SSE data when available, fall back to initial fetch

## Rules

1. **SSE store is NOT React Context** — It's a module-level external store. Components subscribe via `useSyncExternalStore`. This is React's recommended pattern for external data sources.

2. **Never give up reconnecting** — Reconnection is unbounded. The store retries forever with exponential backoff capped at 30s. Users should always see fresh data eventually.

3. **Snapshot first, incremental after** — On every connect/reconnect, the server sends full state. The client replaces its entire cache from the snapshot, then applies incremental events.

4. **Show connection status after 2s** — Don't flash banners on brief disconnects. Only show "Reconnecting..." if disconnected for >2 seconds.

5. **Domain hooks combine fetch + SSE** — Initial page data comes from `use()` + Suspense. SSE keeps it fresh. Never use only one or the other for data that changes.

6. **SSE events published after commit** — The Go backend emits SSE events AFTER transaction commit (R-20), never during mutation. This prevents the React UI from seeing uncommitted state.

## Examples

### Good: External store with useSyncExternalStore

```typescript
// Correct: module-level store, not Context
const state = useSyncExternalStore(sseStore.subscribe, sseStore.getSnapshot);
```

### Bad: SSE in React Context

```typescript
// WRONG: Don't put SSE in Context
const SSEContext = createContext<EventSource | null>(null);
function SSEProvider({ children }) {
  const [es] = useState(() => new EventSource('/api/v1/events/stream'));
  return <SSEContext.Provider value={es}>{children}</SSEContext.Provider>;
}
```

### Good: Domain hook with fetch + SSE

```typescript
export function useSession(name: string) {
  const initial = use(api.getSession(name));
  const sse = useSyncExternalStore(sseStore.subscribe, sseStore.getSnapshot);
  const live = sse.events.get('session:update');
  return live?.data?.name === name ? live.data : initial;
}
```

### Bad: Only SSE, no initial fetch

```typescript
// WRONG: No data until first SSE event arrives
export function useSession(name: string) {
  const sse = useSyncExternalStore(sseStore.subscribe, sseStore.getSnapshot);
  return sse.events.get('session:update')?.data;
}
```

## Checklist

Before completing SSE work, verify:

- [ ] SSE store uses `useSyncExternalStore`, not React Context
- [ ] Exponential backoff: 1s → 2s → 4s → max 30s
- [ ] Reconnection is unbounded (never gives up)
- [ ] Snapshot rehydration on connect/reconnect
- [ ] Connection status banner appears after >2s disconnect
- [ ] All 8 event types have TypeScript interfaces
- [ ] Domain hooks combine `use()` + `useSyncExternalStore`
- [ ] SSE events are emitted server-side after transaction commit
