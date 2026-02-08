import type {
  SSEEventType,
  SSEEvent,
  ConnectionStatus,
  SSEStoreSnapshot,
} from "../types";

type Listener = () => void;

const SSE_EVENT_TYPES: SSEEventType[] = [
  "workspace:update",
  "session:update",
  "messages:new",
  "todos:update",
  "log:append",
  "changes:update",
  "events:new",
  "compose:update",
];

const INITIAL_RETRY_DELAY_MS = 1000;
const MAX_RETRY_DELAY_MS = 30000;
const BACKOFF_MULTIPLIER = 2;

function createInitialSnapshot(): SSEStoreSnapshot {
  const events = {} as Record<SSEEventType, unknown>;
  for (const type of SSE_EVENT_TYPES) {
    events[type] = null;
  }
  return {
    connectionStatus: "disconnected",
    lastEvent: null,
    events,
  };
}

export function createSSEStore(url: string) {
  let snapshot = createInitialSnapshot();
  const listeners: Set<Listener> = new Set();
  let eventSource: EventSource | null = null;
  let retryDelay = INITIAL_RETRY_DELAY_MS;
  let retryTimeoutId: ReturnType<typeof setTimeout> | null = null;
  let destroyed = false;

  function emitChange() {
    for (const listener of listeners) {
      listener();
    }
  }

  function updateSnapshot(partial: Partial<SSEStoreSnapshot>) {
    snapshot = { ...snapshot, ...partial };
    emitChange();
  }

  function updateConnectionStatus(status: ConnectionStatus) {
    updateSnapshot({ connectionStatus: status });
  }

  function handleEvent(eventType: SSEEventType, data: unknown) {
    const event: SSEEvent = {
      type: eventType,
      data,
      timestamp: new Date().toISOString(),
    };
    const updatedEvents = { ...snapshot.events, [eventType]: data };
    snapshot = {
      ...snapshot,
      lastEvent: event,
      events: updatedEvents,
    };
    emitChange();
  }

  function connect() {
    if (destroyed) return;

    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    updateConnectionStatus("reconnecting");

    eventSource = new EventSource(url);

    eventSource.onopen = () => {
      retryDelay = INITIAL_RETRY_DELAY_MS;
      updateConnectionStatus("connected");
    };

    eventSource.onerror = () => {
      if (destroyed) return;

      if (eventSource) {
        eventSource.close();
        eventSource = null;
      }

      updateConnectionStatus("disconnected");
      scheduleReconnect();
    };

    for (const eventType of SSE_EVENT_TYPES) {
      eventSource.addEventListener(eventType, (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data);
          handleEvent(eventType, data);
        } catch {
          handleEvent(eventType, event.data);
        }
      });
    }
  }

  function scheduleReconnect() {
    if (destroyed) return;

    if (retryTimeoutId !== null) {
      clearTimeout(retryTimeoutId);
    }

    retryTimeoutId = setTimeout(() => {
      retryTimeoutId = null;
      connect();
    }, retryDelay);

    retryDelay = Math.min(retryDelay * BACKOFF_MULTIPLIER, MAX_RETRY_DELAY_MS);
  }

  function subscribe(listener: Listener): () => void {
    listeners.add(listener);

    if (!eventSource && !destroyed) {
      connect();
    }

    return () => {
      listeners.delete(listener);

      if (listeners.size === 0 && eventSource) {
        eventSource.close();
        eventSource = null;
        if (retryTimeoutId !== null) {
          clearTimeout(retryTimeoutId);
          retryTimeoutId = null;
        }
        updateConnectionStatus("disconnected");
      }
    };
  }

  function getSnapshot(): SSEStoreSnapshot {
    return snapshot;
  }

  function getServerSnapshot(): SSEStoreSnapshot {
    return createInitialSnapshot();
  }

  function destroy() {
    destroyed = true;
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }
    if (retryTimeoutId !== null) {
      clearTimeout(retryTimeoutId);
      retryTimeoutId = null;
    }
    listeners.clear();
    snapshot = createInitialSnapshot();
  }

  return {
    subscribe,
    getSnapshot,
    getServerSnapshot,
    destroy,
  };
}

export type SSEStore = ReturnType<typeof createSSEStore>;

let defaultStore: SSEStore | null = null;

export function getDefaultSSEStore(): SSEStore {
  if (!defaultStore) {
    defaultStore = createSSEStore("/api/v1/events/stream");
  }
  return defaultStore;
}

export function resetDefaultSSEStore(): void {
  if (defaultStore) {
    defaultStore.destroy();
    defaultStore = null;
  }
}
