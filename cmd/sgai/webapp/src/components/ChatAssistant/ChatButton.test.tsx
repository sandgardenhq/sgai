import { describe, it, expect, beforeEach } from "bun:test";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { ChatButton } from "./ChatButton";
import { resetDefaultChatStore, getDefaultChatStore } from "@/lib/chat-store";

describe("ChatButton", () => {
  beforeEach(() => {
    cleanup();
    resetDefaultChatStore();
  });

  it("renders a button with chat icon", () => {
    render(<ChatButton />);

    const button = screen.getByRole("button", { name: /open chat assistant/i });
    expect(button).toBeTruthy();
  });

  it("toggles chat open state when clicked", () => {
    render(<ChatButton />);
    const store = getDefaultChatStore();

    expect(store.getSnapshot().isOpen).toBe(false);

    const button = screen.getByRole("button");
    fireEvent.click(button);

    expect(store.getSnapshot().isOpen).toBe(true);

    fireEvent.click(button);
    expect(store.getSnapshot().isOpen).toBe(false);
  });

  it("has fixed positioning", () => {
    const { container } = render(<ChatButton />);

    const button = container.querySelector("button");
    expect(button?.className).toContain("fixed");
    expect(button?.className).toContain("bottom-4");
    expect(button?.className).toContain("right-4");
  });

  it("updates aria-label based on open state", () => {
    render(<ChatButton />);
    const store = getDefaultChatStore();

    let button = screen.getByRole("button", { name: /open chat assistant/i });
    expect(button).toBeTruthy();

    fireEvent.click(button);

    button = screen.getByRole("button", { name: /close chat assistant/i });
    expect(button).toBeTruthy();
  });
});
