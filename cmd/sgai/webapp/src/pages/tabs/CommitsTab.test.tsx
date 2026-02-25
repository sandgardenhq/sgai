import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { MemoryRouter } from "react-router";
import { CommitsTab } from "./CommitsTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiCommitsResponse } from "@/types";

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

const commitsResponse: ApiCommitsResponse = {
  commits: [
    {
      changeId: "nyyqzwxs",
      commitId: "ad8009f9",
      timestamp: "2026-02-08 10:00",
      description: "Initial commit",
      graphChar: "@",
      workspaces: ["main"],
      bookmarks: ["main"],
    },
  ],
};

function renderCommitsTab() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <CommitsTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("CommitsTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderCommitsTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders commit entries when data loads", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(commitsResponse)));
    renderCommitsTab();

    await waitFor(() => {
      expect(screen.getByText("nyyqzwxs")).toBeDefined();
    });

    expect(screen.getByText("ad8009f9")).toBeDefined();
    expect(screen.getByText("Initial commit")).toBeDefined();
  });

  it("renders empty state when no commits", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ commits: [] })));
    renderCommitsTab();

    await waitFor(() => {
      expect(screen.getByText(/No commits found/i)).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderCommitsTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load commits/i)).toBeDefined();
    });
  });

  it("calls commits API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(commitsResponse)));
    renderCommitsTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project/commits");
  });

  it("renders table headers", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(commitsResponse)));
    renderCommitsTab();

    await waitFor(() => {
      expect(screen.getByText("Change ID")).toBeDefined();
    });

    expect(screen.getByText("Time")).toBeDefined();
    expect(screen.getByText("Description")).toBeDefined();
  });
});
