import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { EventsTab } from "./EventsTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiEventsResponse } from "@/types";

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

const eventsResponse: ApiEventsResponse = {
  events: [
    {
      timestamp: "2026-02-08T10:00:00Z",
      formattedTime: "10:00 AM",
      agent: "coordinator",
      description: "Started workflow",
      showDateDivider: true,
      dateDivider: "Feb 8, 2026",
    },
    {
      timestamp: "2026-02-08T10:05:00Z",
      formattedTime: "10:05 AM",
      agent: "backend-developer",
      description: "Implementing API endpoints",
      showDateDivider: false,
      dateDivider: "",
    },
  ],
  currentAgent: "backend-developer",
  currentModel: "anthropic/claude-opus-4-6",
  svgHash: "abc123",
  needsInput: false,
  humanMessage: "",
  goalContent: "# Test Goal",
};

function renderEventsTab(goalContent?: string) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <EventsTab workspaceName="test-project" goalContent={goalContent} />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("EventsTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderEventsTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders events timeline when data loads", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(eventsResponse)));
    renderEventsTab();

    await waitFor(() => {
      expect(screen.getByText("Started workflow")).toBeDefined();
    });

    expect(screen.getByText("Implementing API endpoints")).toBeDefined();
    expect(screen.getByText("Feb 8, 2026")).toBeDefined();
  });

  it("renders GOAL.md section when goal content is provided", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(eventsResponse)));
    renderEventsTab("# Test Goal");

    await waitFor(() => {
      expect(screen.getByText("GOAL.md")).toBeDefined();
    });

    const summary = screen.getByText("GOAL.md");
    const details = summary.closest("details");
    expect(details).not.toBeNull();
    expect(details?.hasAttribute("open")).toBe(false);
    expect(screen.getByText("Test Goal")).toBeDefined();
  });

  it("uses workflow svg endpoint for workflow image", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(eventsResponse)));
    renderEventsTab();

    await waitFor(() => {
      expect(screen.getAllByAltText("Workflow graph").length).toBeGreaterThan(0);
    });

    const img = screen.getAllByAltText("Workflow graph")[0] as HTMLImageElement;
    expect(img.src).toContain("/api/v1/workspaces/test-project/workflow.svg");
    expect(img.src).toContain("abc123");
  });

  it("renders empty state when no events", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify({
      events: [],
      currentAgent: "",
      currentModel: "",
      svgHash: "",
      needsInput: false,
      humanMessage: "",
      goalContent: "",
    })));
    renderEventsTab();

    await waitFor(() => {
      expect(screen.getByText("No events recorded yet")).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderEventsTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load events/i)).toBeDefined();
    });
  });

  it("calls events API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(eventsResponse)));
    renderEventsTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project/events");
  });
});
