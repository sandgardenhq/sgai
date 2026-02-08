import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { SpecificationTab } from "./SpecificationTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import type { ApiWorkspaceDetailResponse } from "@/types";

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

const workspaceDetailResponse: ApiWorkspaceDetailResponse = {
  name: "test-project",
  dir: "/home/user/test-project",
  running: true,
  needsInput: false,
  status: "running",
  badgeClass: "running",
  badgeText: "Running",
  isRoot: true,
  isFork: false,
  pinned: false,
  hasSgai: true,
  hasEditedGoal: false,
  interactiveAuto: false,
  currentAgent: "coordinator",
  currentModel: "anthropic/claude-opus-4-6",
  task: "",
  goalContent: "# Test Goal\n\nBuild amazing things",
  pmContent: "## Project Status\n\nIn progress",
  hasProjectMgmt: true,
  svgHash: "abc123",
  totalExecTime: "5m",
  latestProgress: "Tests passing",
  agentSequence: [],
  cost: { totalCost: 0, inputTokens: 0, outputTokens: 0, cacheCreationInputTokens: 0, cacheReadInputTokens: 0 },
};

function renderSpecificationTab() {
  return render(
    <MemoryRouter>
      <SpecificationTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("SpecificationTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderSpecificationTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders GOAL.md content when data loads", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(workspaceDetailResponse)));
    renderSpecificationTab();

    await waitFor(() => {
      expect(screen.getByText("GOAL.md")).toBeDefined();
    });

    expect(screen.getByRole("heading", { name: "Test Goal" })).toBeDefined();
  });

  it("renders PROJECT_MANAGEMENT.md when available", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(workspaceDetailResponse)));
    const { container } = renderSpecificationTab();
    const view = within(container);

    await waitFor(() => {
      expect(view.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
    });

    expect(view.getAllByText("Project Status").length).toBeGreaterThan(0);
    expect(view.getAllByText("In progress").length).toBeGreaterThan(0);
  });

  it("renders empty state when no GOAL.md", async () => {
    const noGoalResponse = { ...workspaceDetailResponse, goalContent: "", hasProjectMgmt: false, pmContent: "" };
    mockFetch.mockResolvedValue(new Response(JSON.stringify(noGoalResponse)));
    renderSpecificationTab();

    await waitFor(() => {
      expect(screen.getByText("No GOAL.md file found")).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderSpecificationTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load specification/i)).toBeDefined();
    });
  });

  it("calls workspace detail API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(workspaceDetailResponse)));
    renderSpecificationTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project");
  });
});
