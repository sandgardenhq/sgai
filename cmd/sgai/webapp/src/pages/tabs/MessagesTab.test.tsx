import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { MessagesTab } from "./MessagesTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiMessagesResponse } from "@/types";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  closed = false;
  constructor(url: string) { this.url = url; }
  addEventListener() {}
  removeEventListener() {}
  close() { this.closed = true; }
}

const originalEventSource = globalThis.EventSource;
const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  (globalThis as unknown as Record<string, unknown>).EventSource = MockEventSource;
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
});

const messagesResponse: ApiMessagesResponse = {
  messages: [
    { id: 1, fromAgent: "coordinator", toAgent: "backend-developer", body: "Please implement the API", subject: "API Implementation", read: true },
    { id: 2, fromAgent: "backend-developer", toAgent: "coordinator", body: "API done", subject: "API Complete", read: false },
  ],
};

const markdownMessagesResponse: ApiMessagesResponse = {
  messages: [
    {
      id: 3,
      fromAgent: "coordinator",
      toAgent: "backend-developer",
      body: "## Task\n\nPlease implement the **API** with:\n\n- endpoint `/users`\n- endpoint `/posts`\n\n```go\nfunc main() {}\n```",
      subject: "API Implementation",
      read: true,
    },
  ],
};

function renderMessagesTab() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <MessagesTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("MessagesTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderMessagesTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders messages when data loads", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(messagesResponse)));
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("backend-developer").length).toBeGreaterThan(0);
  });

  it("shows unread messages with bold styling", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(messagesResponse)));
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    });

    const summaries = document.querySelectorAll("summary");
    const unreadSummary = Array.from(summaries).find((s) => s.className.includes("font-bold"));
    expect(unreadSummary).toBeDefined();
    const readSummary = Array.from(summaries).find((s) => !s.className.includes("font-bold"));
    expect(readSummary).toBeDefined();
  });

  it("renders empty state when no messages", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ messages: [] })));
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getByText("No messages")).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load messages/i)).toBeDefined();
    });
  });

  it("calls messages API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(messagesResponse)));
    renderMessagesTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project/messages");
  });

  it("renders markdown content in message body", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(markdownMessagesResponse)));
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    });

    const details = document.querySelector("details");
    expect(details).not.toBeNull();
    details!.setAttribute("open", "");

    await waitFor(() => {
      const heading = document.querySelector("h2");
      expect(heading).not.toBeNull();
      expect(heading!.textContent).toBe("Task");
    });

    const bold = document.querySelector("strong");
    const strongTexts = Array.from(document.querySelectorAll("strong")).map((el) => el.textContent);
    expect(strongTexts).toContain("API");

    const listItems = document.querySelectorAll("li");
    expect(listItems.length).toBe(2);

    const codeBlock = document.querySelector("pre");
    expect(codeBlock).not.toBeNull();
  });
});
