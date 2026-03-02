import { useSyncExternalStore, useCallback } from "react";
import { getDefaultChatStore } from "../lib/chat-store";
import type { ChatStoreSnapshot, ChatMessage, ChatContext } from "../lib/chat-store";

export function useChatStore(): ChatStoreSnapshot {
  const store = getDefaultChatStore();
  return useSyncExternalStore(
    store.subscribe,
    store.getSnapshot,
    store.getServerSnapshot,
  );
}

export function useChatMessages(): ChatMessage[] {
  const store = getDefaultChatStore();

  const getSnapshot = useCallback(
    () => store.getSnapshot().messages,
    [store],
  );

  const getServerSnapshot = useCallback(
    () => [] as ChatMessage[],
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}

export function useChatIsOpen(): boolean {
  const store = getDefaultChatStore();

  const getSnapshot = useCallback(
    () => store.getSnapshot().isOpen,
    [store],
  );

  const getServerSnapshot = useCallback(
    () => false,
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}

export function useChatIsStreaming(): boolean {
  const store = getDefaultChatStore();

  const getSnapshot = useCallback(
    () => store.getSnapshot().isStreaming,
    [store],
  );

  const getServerSnapshot = useCallback(
    () => false,
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}

export function useChatContext(): ChatContext {
  const store = getDefaultChatStore();

  const getSnapshot = useCallback(
    () => store.getSnapshot().context,
    [store],
  );

  const getServerSnapshot = useCallback(
    () => ({ currentPage: "/" }) as ChatContext,
    [],
  );

  return useSyncExternalStore(
    store.subscribe,
    getSnapshot,
    getServerSnapshot,
  );
}

export function useChatActions() {
  const store = getDefaultChatStore();

  return {
    addUserMessage: store.addUserMessage,
    addAssistantMessage: store.addAssistantMessage,
    startStreaming: store.startStreaming,
    appendStreamChunk: store.appendStreamChunk,
    finishStreaming: store.finishStreaming,
    setOpen: store.setOpen,
    toggleOpen: store.toggleOpen,
    setContext: store.setContext,
    clearMessages: store.clearMessages,
  };
}
