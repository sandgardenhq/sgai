import { useSyncExternalStore } from "react";
import type { ApiWorkspaceEntry } from "../types";

export type { ApiWorkspaceEntry };

export type FetchStatus = "idle" | "fetching" | "error";

export interface FactoryStateSnapshot {
  workspaces: ApiWorkspaceEntry[];
  fetchStatus: FetchStatus;
  lastFetchedAt: number | null;
}

interface FactoryStateResponse {
  workspaces: ApiWorkspaceEntry[] | null;
}

type Listener = () => void;

const POLL_INTERVAL_MS = 3000;
const SLOW_POLL_INTERVAL_MS = 10000;
const SSE_CONNECT_TIMEOUT_MS = 5000;

function createFactoryStateStore() {
  let snapshot: FactoryStateSnapshot = {
    workspaces: [],
    fetchStatus: "idle",
    lastFetchedAt: null,
  };

  const listeners: Set<Listener> = new Set();
  let pollTimerId: ReturnType<typeof setTimeout> | null = null;
  let sseSource: EventSource | null = null;
  let sseTimeoutId: ReturnType<typeof setTimeout> | null = null;
  let sseConnected = false;
  let isFetching = false;
  let isDestroyed = false;
  let isStarted = false;

  function emitChange() {
    for (const listener of listeners) {
      listener();
    }
  }

  function updateSnapshot(partial: Partial<FactoryStateSnapshot>) {
    snapshot = { ...snapshot, ...partial };
    emitChange();
  }

  async function fetchState() {
    if (isFetching || isDestroyed) return;
    isFetching = true;
    updateSnapshot({ fetchStatus: "fetching" });

    try {
      const response = await fetch("/api/v1/state");
      if (isDestroyed) return;
      if (!response.ok) {
        updateSnapshot({ fetchStatus: "error" });
        return;
      }
      const data = (await response.json()) as FactoryStateResponse;
      if (isDestroyed) return;
      updateSnapshot({
        workspaces: data.workspaces ?? [],
        fetchStatus: "idle",
        lastFetchedAt: Date.now(),
      });
    } catch {
      if (!isDestroyed) {
        updateSnapshot({ fetchStatus: "error" });
      }
    } finally {
      isFetching = false;
    }
  }

  function schedulePoll(intervalMs: number) {
    if (pollTimerId !== null) {
      clearTimeout(pollTimerId);
      pollTimerId = null;
    }
    if (isDestroyed) return;
    pollTimerId = setTimeout(() => {
      pollTimerId = null;
      const hidden = typeof document !== "undefined" && document.visibilityState === "hidden";
      void fetchState();
      schedulePoll(hidden ? SLOW_POLL_INTERVAL_MS : POLL_INTERVAL_MS);
    }, intervalMs);
  }

  function handleVisibilityChange() {
    if (isDestroyed) return;
    if (document.visibilityState === "visible") {
      void fetchState();
      schedulePoll(POLL_INTERVAL_MS);
    } else {
      schedulePoll(SLOW_POLL_INTERVAL_MS);
    }
  }

  function connectSSESignal() {
    if (sseSource !== null || isDestroyed) return;

    sseSource = new EventSource("/api/v1/signal");

    sseTimeoutId = setTimeout(() => {
      sseTimeoutId = null;
      if (!sseConnected && sseSource !== null) {
        sseSource.close();
        sseSource = null;
      }
    }, SSE_CONNECT_TIMEOUT_MS);

    sseSource.onopen = () => {
      sseConnected = true;
      if (sseTimeoutId !== null) {
        clearTimeout(sseTimeoutId);
        sseTimeoutId = null;
      }
    };

    sseSource.onerror = () => {
      if (sseSource !== null) {
        sseSource.close();
        sseSource = null;
      }
      if (sseTimeoutId !== null) {
        clearTimeout(sseTimeoutId);
        sseTimeoutId = null;
      }
      sseConnected = false;
    };

    sseSource.addEventListener("reload", () => {
      if (!isDestroyed) {
        void fetchState();
      }
    });
  }

  function start() {
    if (isStarted || isDestroyed) return;
    isStarted = true;

    void fetchState();
    schedulePoll(POLL_INTERVAL_MS);
    connectSSESignal();

    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", handleVisibilityChange);
    }
  }

  function stop() {
    isDestroyed = true;
    isStarted = false;

    if (pollTimerId !== null) {
      clearTimeout(pollTimerId);
      pollTimerId = null;
    }

    if (sseTimeoutId !== null) {
      clearTimeout(sseTimeoutId);
      sseTimeoutId = null;
    }

    if (sseSource !== null) {
      sseSource.close();
      sseSource = null;
    }

    if (typeof document !== "undefined") {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    }
  }

  function subscribe(listener: Listener): () => void {
    listeners.add(listener);
    if (!isStarted && !isDestroyed) {
      start();
    }
    return () => {
      listeners.delete(listener);
    };
  }

  function getSnapshot(): FactoryStateSnapshot {
    return snapshot;
  }

  function getServerSnapshot(): FactoryStateSnapshot {
    return {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: null,
    };
  }

  return {
    subscribe,
    getSnapshot,
    getServerSnapshot,
    stop,
  };
}

type FactoryStateStore = ReturnType<typeof createFactoryStateStore>;

let storeInstance: FactoryStateStore | null = null;

function getFactoryStateStore(): FactoryStateStore {
  if (!storeInstance) {
    storeInstance = createFactoryStateStore();
  }
  return storeInstance;
}

export function resetFactoryStateStore(): void {
  if (storeInstance) {
    storeInstance.stop();
    storeInstance = null;
  }
}

export function useFactoryState(): FactoryStateSnapshot {
  const store = getFactoryStateStore();
  return useSyncExternalStore(
    store.subscribe,
    store.getSnapshot,
    store.getServerSnapshot,
  );
}
