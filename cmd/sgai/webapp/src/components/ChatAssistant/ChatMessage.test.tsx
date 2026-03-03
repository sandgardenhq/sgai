import { describe, it, expect } from "bun:test";
import { render, screen } from "@testing-library/react";
import { ChatMessage } from "./ChatMessage";
import type { ChatMessage as ChatMessageType } from "@/lib/chat-store";

function createMessage(overrides: Partial<ChatMessageType> = {}): ChatMessageType {
  return {
    id: "test-id",
    role: "user",
    content: "Test message",
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}

describe("ChatMessage", () => {
  it("renders user message with correct styling", () => {
    const message = createMessage({ role: "user", content: "Hello!" });
    render(<ChatMessage message={message} />);

    const text = screen.getByText("Hello!");
    expect(text).toBeTruthy();

    const container = text.closest("div[class*='bg-primary']");
    expect(container).toBeTruthy();
  });

  it("renders assistant message with correct styling", () => {
    const message = createMessage({ role: "assistant", content: "Hi there!" });
    render(<ChatMessage message={message} />);

    const text = screen.getByText("Hi there!");
    expect(text).toBeTruthy();

    const container = text.closest("div[class*='bg-muted']");
    expect(container).toBeTruthy();
  });

  it("shows thinking indicator for empty streaming message", () => {
    const message = createMessage({
      role: "assistant",
      content: "",
      isStreaming: true,
    });
    render(<ChatMessage message={message} />);

    expect(screen.getByText("Thinking...")).toBeTruthy();
  });

  it("renders markdown content for assistant messages", () => {
    const message = createMessage({
      role: "assistant",
      content: "**Bold text**",
    });
    render(<ChatMessage message={message} />);

    const boldElement = screen.getByText("Bold text");
    expect(boldElement).toBeTruthy();
  });

  it("applies animation class when streaming", () => {
    const message = createMessage({
      role: "assistant",
      content: "Streaming...",
      isStreaming: true,
    });
    const { container } = render(<ChatMessage message={message} />);

    const animatedElement = container.querySelector(".animate-pulse");
    expect(animatedElement).toBeTruthy();
  });
});
