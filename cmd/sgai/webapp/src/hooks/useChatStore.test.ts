import { describe, it, expect, beforeEach } from "bun:test";
import { renderHook, act } from "@testing-library/react";
import {
  useChatStore,
  useChatMessages,
  useChatIsOpen,
  useChatIsStreaming,
  useChatActions,
} from "./useChatStore";
import { resetDefaultChatStore, getDefaultChatStore } from "../lib/chat-store";

describe("useChatStore hooks", () => {
  beforeEach(() => {
    resetDefaultChatStore();
  });

  describe("useChatStore", () => {
    it("returns the full store snapshot", () => {
      const { result } = renderHook(() => useChatStore());

      expect(result.current).toHaveProperty("messages");
      expect(result.current).toHaveProperty("isOpen");
      expect(result.current).toHaveProperty("isStreaming");
      expect(result.current).toHaveProperty("context");
    });

    it("updates when store changes", () => {
      const { result } = renderHook(() => useChatStore());
      const store = getDefaultChatStore();

      expect(result.current.messages).toHaveLength(0);

      act(() => {
        store.addUserMessage("Test");
      });

      expect(result.current.messages).toHaveLength(1);
    });
  });

  describe("useChatMessages", () => {
    it("returns messages array", () => {
      const { result } = renderHook(() => useChatMessages());
      expect(result.current).toEqual([]);
    });

    it("updates when messages change", () => {
      const { result } = renderHook(() => useChatMessages());
      const store = getDefaultChatStore();

      act(() => {
        store.addUserMessage("Hello");
      });

      expect(result.current).toHaveLength(1);
      expect(result.current[0].content).toBe("Hello");
    });
  });

  describe("useChatIsOpen", () => {
    it("returns isOpen state", () => {
      const { result } = renderHook(() => useChatIsOpen());
      expect(result.current).toBe(false);
    });

    it("updates when open state changes", () => {
      const { result } = renderHook(() => useChatIsOpen());
      const store = getDefaultChatStore();

      act(() => {
        store.setOpen(true);
      });

      expect(result.current).toBe(true);
    });
  });

  describe("useChatIsStreaming", () => {
    it("returns isStreaming state", () => {
      const { result } = renderHook(() => useChatIsStreaming());
      expect(result.current).toBe(false);
    });

    it("updates when streaming state changes", () => {
      const { result } = renderHook(() => useChatIsStreaming());
      const store = getDefaultChatStore();

      act(() => {
        store.startStreaming();
      });

      expect(result.current).toBe(true);

      act(() => {
        store.finishStreaming();
      });

      expect(result.current).toBe(false);
    });
  });

  describe("useChatActions", () => {
    it("returns action functions", () => {
      const { result } = renderHook(() => useChatActions());

      expect(typeof result.current.addUserMessage).toBe("function");
      expect(typeof result.current.addAssistantMessage).toBe("function");
      expect(typeof result.current.startStreaming).toBe("function");
      expect(typeof result.current.appendStreamChunk).toBe("function");
      expect(typeof result.current.finishStreaming).toBe("function");
      expect(typeof result.current.setOpen).toBe("function");
      expect(typeof result.current.toggleOpen).toBe("function");
      expect(typeof result.current.setContext).toBe("function");
      expect(typeof result.current.clearMessages).toBe("function");
    });

    it("actions work correctly", () => {
      const { result: actionsResult } = renderHook(() => useChatActions());
      const { result: messagesResult } = renderHook(() => useChatMessages());

      act(() => {
        actionsResult.current.addUserMessage("Test message");
      });

      expect(messagesResult.current).toHaveLength(1);
      expect(messagesResult.current[0].content).toBe("Test message");
    });
  });
});
