import { describe, it, expect, afterEach, mock } from "bun:test";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { EventsTab } from "./EventsTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceEntry } from "@/types";

const mockFetch = mock(() => Promise.resolve(new Response("{}")));
globalThis.fetch = mockFetch as unknown as typeof fetch;

afterEach(() => {
  cleanup();
  mockFetch.mockReset();
});

function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-project",
    dir: "/home/user/test-project",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    isRoot: false,
    isFork: false,
    status: "stopped",
    badgeClass: "",
    badgeText: "",
    hasSgai: true,
    hasEditedGoal: false,
    interactiveAuto: false,
    continuousMode: false,
    currentAgent: "coordinator",
    currentModel: "anthropic/claude-opus-4-6",
    task: "",
    goalContent: "",
    rawGoalContent: "",
    pmContent: "",
    hasProjectMgmt: false,
    svgHash: "abc123",
    totalExecTime: "0s",
    latestProgress: "",
    humanMessage: "",
    agentSequence: [],
    cost: { totalCost: 0, totalTokens: { input: 0, output: 0, reasoning: 0, cacheRead: 0, cacheWrite: 0 }, byAgent: [] },
    events: [],
    messages: [],
    projectTodos: [],
    agentTodos: [],
    changes: { description: "", diffLines: [] },
    commits: [],
    log: [],
    ...overrides,
  };
}

const sampleEvents = [
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
];

function renderEventsTab(workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle", goalContent?: string) {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <EventsTab workspaceName="test-project" goalContent={goalContent} />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("EventsTab", () => {
  it("renders loading skeleton when fetching with no workspace", () => {
    renderEventsTab([], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders null when workspace not found and idle", () => {
    const { container } = renderEventsTab([], "idle");
    expect(container.firstChild).toBeNull();
  });

  it("renders error state when fetch fails and workspace not found", () => {
    renderEventsTab([], "error");
    expect(screen.getByText(/Failed to load events/i)).toBeDefined();
  });

  it("renders events timeline from factory state", () => {
    const workspace = makeWorkspace({ events: sampleEvents });
    renderEventsTab([workspace]);

    expect(screen.getByText("Started workflow")).toBeDefined();
    expect(screen.getByText("Implementing API endpoints")).toBeDefined();
    expect(screen.getByText("Feb 8, 2026")).toBeDefined();
  });

  it("renders GOAL.md section when goalContent prop is provided", () => {
    const workspace = makeWorkspace({ events: sampleEvents });
    renderEventsTab([workspace], "idle", "# Test Goal");

    expect(screen.getByText("GOAL.md")).toBeDefined();
    const summary = screen.getByText("GOAL.md");
    const details = summary.closest("details");
    expect(details).not.toBeNull();
    expect(screen.getByText("Test Goal")).toBeDefined();
  });

  it("uses workflow svg endpoint with svgHash from workspace", () => {
    const workspace = makeWorkspace({ events: sampleEvents, svgHash: "abc123" });
    renderEventsTab([workspace]);

    const img = screen.getAllByAltText("Workflow graph")[0] as HTMLImageElement;
    expect(img.src).toContain("/api/v1/workspaces/test-project/workflow.svg");
    expect(img.src).toContain("abc123");
  });

  it("renders empty state when no events", () => {
    const workspace = makeWorkspace({ events: [] });
    renderEventsTab([workspace]);

    expect(screen.getByText("No events recorded yet")).toBeDefined();
  });

  it("does not call individual events API endpoint", () => {
    const workspace = makeWorkspace({ events: sampleEvents });
    renderEventsTab([workspace]);

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/api/v1/workspaces/test-project/events"))).toBe(false);
  });

  it("renders human message when needsInput and humanMessage present", () => {
    const workspace = makeWorkspace({
      events: [],
      needsInput: true,
      humanMessage: "What should I do next?",
      currentAgent: "coordinator",
    });
    renderEventsTab([workspace]);

    expect(screen.getByText("What should I do next?")).toBeDefined();
  });
});
