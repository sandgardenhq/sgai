import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router";
import { RetrospectivesTab } from "./RetrospectivesTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiRetrospectivesResponse } from "@/types";

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
  cleanup();
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  (globalThis as unknown as Record<string, unknown>).EventSource = MockEventSource;
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
});

const retroResponse: ApiRetrospectivesResponse = {
  sessions: [
    { name: "2026-02-08-session1", hasImprovements: true, goalSummary: "Test project" },
    { name: "2026-02-07-session2", hasImprovements: false, goalSummary: "Old session" },
  ],
  selectedSession: "2026-02-08-session1",
  details: {
    sessionName: "2026-02-08-session1",
    goalSummary: "Test project",
    goalContent: "<h1>Test Goal</h1>",
    improvements: "<h2>Improvements</h2><p>Better error handling</p>",
    improvementsRaw: "- Better error handling",
    hasImprovements: true,
    isAnalyzing: false,
    isApplying: false,
  },
};

function renderRetrospectivesTab() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <RetrospectivesTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

function LocationDisplay() {
  const location = useLocation();
  return (
    <div data-testid="location-display">
      {location.pathname}{location.search}
    </div>
  );
}

function renderRetrospectivesTabWithLocation() {
  return render(
    <MemoryRouter initialEntries={["/workspaces/test-project/retro"]}>
      <TooltipProvider>
        <RetrospectivesTab workspaceName="test-project" />
        <LocationDisplay />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("RetrospectivesTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderRetrospectivesTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders session list when data loads", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(retroResponse)))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getAllByText("2026-02-08-session1").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("2026-02-07-session2")).toBeDefined();
  });

  it("renders detail view for selected session", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(retroResponse)))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getByText("Analyzed")).toBeDefined();
    });

    expect(screen.getAllByText("GOAL.md").length).toBeGreaterThan(0);
    expect(screen.getByText("IMPROVEMENTS.md")).toBeDefined();
  });

  it("navigates to analyze screen when analyze is clicked", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(retroResponse)))
    );
    renderRetrospectivesTabWithLocation();

    await waitFor(() => {
      expect(screen.getByText("Analyze")).toBeDefined();
    });

    fireEvent.click(screen.getByRole("button", { name: "Analyze" }));

    const expected = `/workspaces/test-project/retro/${encodeURIComponent(retroResponse.details?.sessionName ?? "")}/analyze`;
    await waitFor(() => {
      expect(screen.getByTestId("location-display").textContent).toBe(expected);
    });
  });

  it("shows analyzing badge without not analyzed badge when analysis is running", async () => {
    const analyzingResponse: ApiRetrospectivesResponse = {
      ...retroResponse,
      details: {
        ...retroResponse.details,
        hasImprovements: false,
        isAnalyzing: true,
      },
    };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(analyzingResponse)))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getByText("Analyzing...")).toBeDefined();
    });

    expect(screen.queryByText("Not Analyzed")).toBeNull();
  });

  it("renders apply button when improvements exist", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(retroResponse)))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getByText("Apply")).toBeDefined();
    });

    const applyButton = screen.getByText("Apply").closest("button");
    expect(applyButton?.hasAttribute("disabled")).toBe(false);
  });

  it("disables apply button when no improvements", async () => {
    const noImprovements = {
      ...retroResponse,
      details: { ...retroResponse.details, hasImprovements: false },
    };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(noImprovements)))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getByText("Apply")).toBeDefined();
    });

    const applyButton = screen.getByText("Apply").closest("button");
    expect(applyButton?.hasAttribute("disabled")).toBe(true);
  });

  it("renders empty state when no sessions", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ sessions: [], selectedSession: "" })))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getByText(/No retrospective sessions found/)).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockImplementation(() => Promise.reject(new Error("Network error")));
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(screen.getAllByText(/Failed to load retrospectives/i).length).toBeGreaterThan(0);
    });
  });

  it("calls retrospectives API on mount", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(retroResponse)))
    );
    renderRetrospectivesTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project/retrospectives");
  });
});
