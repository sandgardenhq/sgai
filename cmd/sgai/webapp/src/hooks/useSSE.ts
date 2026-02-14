import { useSyncExternalStore, useCallback, useMemo } from "react";
import { getDefaultSSEStore, getWorkspaceSSEStore } from "../lib/sse-store";
import type { SSEEventType, ConnectionStatus, SSEStoreSnapshot } from "../types";

export function useSSEStore(): SSEStoreSnapshot {
  const store = getDefaultSSEStore();
  return useSyncExternalStore(
    store.subscribe,
    store.getSnapshot,
    store.getServerSnapshot,
  );
}

export function useSSEEvent<T = unknown>(eventType: SSEEventType): T | null {
  const store = getDefaultSSEStore();

  const getSnapshot = useCallback(
    () => store.getSnapshot().events[eventType] as T | null,
    [eventType, store],
  );

  const getServerSnapshot = useCallback(
    () => null as T | null,
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}

export function useWorkspaceSSEEvent<T = unknown>(
  workspaceName: string,
  eventType: SSEEventType,
): T | null {
  const store = useMemo(
    () => getWorkspaceSSEStore(workspaceName),
    [workspaceName],
  );

  const getSnapshot = useCallback(
    () => store.getSnapshot().events[eventType] as T | null,
    [eventType, store],
  );

  const getServerSnapshot = useCallback(
    () => null as T | null,
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}

export function useConnectionStatus(): ConnectionStatus {
  const store = getDefaultSSEStore();

  const getSnapshot = useCallback(
    () => store.getSnapshot().connectionStatus,
    [store],
  );

  const getServerSnapshot = useCallback(
    (): ConnectionStatus => "disconnected",
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}
