import { describe, it, expect, beforeEach } from "bun:test";
import { createChatStore, resetDefaultChatStore, getDefaultChatStore } from "./chat-store";

describe("createChatStore", () => {
  let store: ReturnType<typeof createChatStore>;

  beforeEach(() => {
    store = createChatStore();
  });

  describe("initial state", () => {
    it("starts with empty messages", () => {
      expect(store.getSnapshot().messages).toEqual([]);
    });

    it("starts with isOpen false", () => {
      expect(store.getSnapshot().isOpen).toBe(false);
    });

    it("starts with isStreaming false", () => {
      expect(store.getSnapshot().isStreaming).toBe(false);
    });

    it("starts with default context", () => {
      expect(store.getSnapshot().context).toEqual({ currentPage: "/" });
    });
  });

  describe("addUserMessage", () => {
    it("adds a user message to the store", () => {
      const message = store.addUserMessage("Hello");

      expect(message.role).toBe("user");
      expect(message.content).toBe("Hello");
      expect(store.getSnapshot().messages).toHaveLength(1);
      expect(store.getSnapshot().messages[0]).toEqual(message);
    });

    it("generates unique IDs for messages", () => {
      const msg1 = store.addUserMessage("First");
      const msg2 = store.addUserMessage("Second");

      expect(msg1.id).not.toBe(msg2.id);
    });
  });

  describe("addAssistantMessage", () => {
    it("adds an assistant message to the store", () => {
      const message = store.addAssistantMessage("Hello, how can I help?");

      expect(message.role).toBe("assistant");
      expect(message.content).toBe("Hello, how can I help?");
      expect(store.getSnapshot().messages).toHaveLength(1);
    });
  });

  describe("streaming", () => {
    it("startStreaming creates an empty assistant message", () => {
      const messageId = store.startStreaming();

      expect(store.getSnapshot().isStreaming).toBe(true);
      expect(store.getSnapshot().streamingMessageId).toBe(messageId);
      expect(store.getSnapshot().messages).toHaveLength(1);
      expect(store.getSnapshot().messages[0].content).toBe("");
      expect(store.getSnapshot().messages[0].isStreaming).toBe(true);
    });

    it("appendStreamChunk adds content to streaming message", () => {
      store.startStreaming();
      store.appendStreamChunk("Hello");
      store.appendStreamChunk(" world");

      expect(store.getSnapshot().messages[0].content).toBe("Hello world");
    });

    it("finishStreaming completes the streaming message", () => {
      store.startStreaming();
      store.appendStreamChunk("Hello");
      store.finishStreaming();

      expect(store.getSnapshot().isStreaming).toBe(false);
      expect(store.getSnapshot().streamingMessageId).toBeNull();
      expect(store.getSnapshot().messages[0].isStreaming).toBe(false);
    });
  });

  describe("open state", () => {
    it("setOpen changes isOpen state", () => {
      store.setOpen(true);
      expect(store.getSnapshot().isOpen).toBe(true);

      store.setOpen(false);
      expect(store.getSnapshot().isOpen).toBe(false);
    });

    it("toggleOpen toggles isOpen state", () => {
      expect(store.getSnapshot().isOpen).toBe(false);

      store.toggleOpen();
      expect(store.getSnapshot().isOpen).toBe(true);

      store.toggleOpen();
      expect(store.getSnapshot().isOpen).toBe(false);
    });
  });

  describe("context", () => {
    it("setContext updates the context", () => {
      store.setContext({ currentPage: "/workspaces/test", workspaceName: "test" });

      expect(store.getSnapshot().context).toEqual({
        currentPage: "/workspaces/test",
        workspaceName: "test",
      });
    });
  });

  describe("clearMessages", () => {
    it("removes all messages", () => {
      store.addUserMessage("First");
      store.addAssistantMessage("Second");
      expect(store.getSnapshot().messages).toHaveLength(2);

      store.clearMessages();
      expect(store.getSnapshot().messages).toEqual([]);
    });
  });

  describe("subscribe", () => {
    it("notifies listeners on state change", () => {
      let callCount = 0;
      const unsubscribe = store.subscribe(() => {
        callCount++;
      });

      store.addUserMessage("Test");
      expect(callCount).toBe(1);

      store.setOpen(true);
      expect(callCount).toBe(2);

      unsubscribe();
      store.addUserMessage("Another");
      expect(callCount).toBe(2);
    });
  });
});

describe("getDefaultChatStore", () => {
  beforeEach(() => {
    resetDefaultChatStore();
  });

  it("returns the same instance on multiple calls", () => {
    const store1 = getDefaultChatStore();
    const store2 = getDefaultChatStore();

    expect(store1).toBe(store2);
  });

  it("resetDefaultChatStore creates a new instance", () => {
    const store1 = getDefaultChatStore();
    store1.addUserMessage("Test");

    resetDefaultChatStore();
    const store2 = getDefaultChatStore();

    expect(store2.getSnapshot().messages).toEqual([]);
  });
});
