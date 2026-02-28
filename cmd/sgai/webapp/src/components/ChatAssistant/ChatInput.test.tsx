import { describe, it, expect, mock, beforeEach } from "bun:test";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { ChatInput } from "./ChatInput";

describe("ChatInput", () => {
  const mockOnSend = mock(() => {});

  beforeEach(() => {
    cleanup();
    mockOnSend.mockClear();
  });

  it("renders input and send button", () => {
    render(<ChatInput onSend={mockOnSend} />);

    expect(screen.getByRole("textbox")).toBeTruthy();
    expect(screen.getByRole("button", { name: /send/i })).toBeTruthy();
  });

  it("uses custom placeholder", () => {
    render(<ChatInput onSend={mockOnSend} placeholder="Type here..." />);

    const input = screen.getByPlaceholderText("Type here...");
    expect(input).toBeTruthy();
  });

  it("calls onSend when form is submitted", () => {
    render(<ChatInput onSend={mockOnSend} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "Hello world" } });
    fireEvent.submit(input.closest("form")!);

    expect(mockOnSend).toHaveBeenCalledWith("Hello world");
  });

  it("calls onSend when Enter is pressed", () => {
    render(<ChatInput onSend={mockOnSend} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "Test message" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(mockOnSend).toHaveBeenCalledWith("Test message");
  });

  it("does not send empty messages", () => {
    render(<ChatInput onSend={mockOnSend} />);

    const button = screen.getByRole("button", { name: /send/i });
    fireEvent.click(button);

    expect(mockOnSend).not.toHaveBeenCalled();
  });

  it("does not send whitespace-only messages", () => {
    render(<ChatInput onSend={mockOnSend} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "   " } });

    const button = screen.getByRole("button", { name: /send/i });
    fireEvent.click(button);

    expect(mockOnSend).not.toHaveBeenCalled();
  });

  it("clears input after sending", () => {
    render(<ChatInput onSend={mockOnSend} />);

    const input = screen.getByRole("textbox") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "Hello" } });
    fireEvent.submit(input.closest("form")!);

    expect(input.value).toBe("");
  });

  it("disables input and button when disabled prop is true", () => {
    render(<ChatInput onSend={mockOnSend} disabled={true} />);

    const input = screen.getByRole("textbox") as HTMLInputElement;
    const button = screen.getByRole("button", { name: /send/i }) as HTMLButtonElement;

    expect(input.disabled).toBe(true);
    expect(button.disabled).toBe(true);
  });

  it("trims message before sending", () => {
    render(<ChatInput onSend={mockOnSend} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "  Hello world  " } });
    fireEvent.submit(input.closest("form")!);

    expect(mockOnSend).toHaveBeenCalledWith("Hello world");
  });
});
