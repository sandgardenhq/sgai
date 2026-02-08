import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { LogTab } from "./LogTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import type { ApiLogResponse } from "@/types";

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

const logResponse: ApiLogResponse = {
  lines: [
    { prefix: "[10:00] ", text: "Starting build process..." },
    { prefix: "[10:01] ", text: "Build completed successfully" },
    { prefix: "", text: "All tests passed" },
  ],
};

function renderLogTab() {
  return render(
    <MemoryRouter>
      <LogTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("LogTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderLogTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders log lines when data loads", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(logResponse)));
    renderLogTab();

    await waitFor(() => {
      expect(screen.getByText("Starting build process...")).toBeDefined();
    });

    expect(screen.getByText("Build completed successfully")).toBeDefined();
    expect(screen.getByText("All tests passed")).toBeDefined();
  });

  it("renders empty state when no logs", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ lines: [] })));
    renderLogTab();

    await waitFor(() => {
      expect(screen.getByText("No logs available")).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderLogTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load log/i)).toBeDefined();
    });
  });

  it("calls log API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(logResponse)));
    renderLogTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project/log");
  });
});
