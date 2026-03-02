type Listener = () => void;

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: string;
  isStreaming?: boolean;
}

export interface ChatContext {
  currentPage: string;
  workspaceName?: string;
}

export interface ChatStoreSnapshot {
  messages: ChatMessage[];
  isOpen: boolean;
  isStreaming: boolean;
  streamingMessageId: string | null;
  context: ChatContext;
}

function createInitialSnapshot(): ChatStoreSnapshot {
  return {
    messages: [],
    isOpen: false,
    isStreaming: false,
    streamingMessageId: null,
    context: { currentPage: "/" },
  };
}

export function createChatStore() {
  let snapshot = createInitialSnapshot();
  const listeners: Set<Listener> = new Set();

  function emitChange() {
    for (const listener of listeners) {
      listener();
    }
  }

  function updateSnapshot(partial: Partial<ChatStoreSnapshot>) {
    snapshot = { ...snapshot, ...partial };
    emitChange();
  }

  function generateId(): string {
    return `msg-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
  }

  function addUserMessage(content: string): ChatMessage {
    const message: ChatMessage = {
      id: generateId(),
      role: "user",
      content,
      timestamp: new Date().toISOString(),
    };
    updateSnapshot({ messages: [...snapshot.messages, message] });
    return message;
  }

  function addAssistantMessage(content: string): ChatMessage {
    const message: ChatMessage = {
      id: generateId(),
      role: "assistant",
      content,
      timestamp: new Date().toISOString(),
    };
    updateSnapshot({ messages: [...snapshot.messages, message] });
    return message;
  }

  function startStreaming(): string {
    const messageId = generateId();
    const message: ChatMessage = {
      id: messageId,
      role: "assistant",
      content: "",
      timestamp: new Date().toISOString(),
      isStreaming: true,
    };
    updateSnapshot({
      messages: [...snapshot.messages, message],
      isStreaming: true,
      streamingMessageId: messageId,
    });
    return messageId;
  }

  function appendStreamChunk(chunk: string): void {
    if (!snapshot.streamingMessageId) return;

    const updatedMessages = snapshot.messages.map((msg) => {
      if (msg.id === snapshot.streamingMessageId) {
        return { ...msg, content: msg.content + chunk };
      }
      return msg;
    });
    updateSnapshot({ messages: updatedMessages });
  }

  function finishStreaming(): void {
    if (!snapshot.streamingMessageId) return;

    const updatedMessages = snapshot.messages.map((msg) => {
      if (msg.id === snapshot.streamingMessageId) {
        return { ...msg, isStreaming: false };
      }
      return msg;
    });
    updateSnapshot({
      messages: updatedMessages,
      isStreaming: false,
      streamingMessageId: null,
    });
  }

  function setOpen(isOpen: boolean): void {
    updateSnapshot({ isOpen });
  }

  function toggleOpen(): void {
    updateSnapshot({ isOpen: !snapshot.isOpen });
  }

  function setContext(context: ChatContext): void {
    updateSnapshot({ context });
  }

  function clearMessages(): void {
    updateSnapshot({ messages: [] });
  }

  function subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }

  function getSnapshot(): ChatStoreSnapshot {
    return snapshot;
  }

  function getServerSnapshot(): ChatStoreSnapshot {
    return createInitialSnapshot();
  }

  return {
    subscribe,
    getSnapshot,
    getServerSnapshot,
    addUserMessage,
    addAssistantMessage,
    startStreaming,
    appendStreamChunk,
    finishStreaming,
    setOpen,
    toggleOpen,
    setContext,
    clearMessages,
  };
}

export type ChatStore = ReturnType<typeof createChatStore>;

let defaultChatStore: ChatStore | null = null;

export function getDefaultChatStore(): ChatStore {
  if (!defaultChatStore) {
    defaultChatStore = createChatStore();
  }
  return defaultChatStore;
}

export function resetDefaultChatStore(): void {
  defaultChatStore = null;
}
