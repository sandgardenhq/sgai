import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router";
import { Dashboard } from "./Dashboard";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspacesResponse } from "@/types";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  closed = false;
  listeners: Map<string, ((event: MessageEvent) => void)[]> = new Map();
  constructor(url: string) { this.url = url; }
  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    const existing = this.listeners.get(type) ?? [];
    existing.push(listener);
    this.listeners.set(type, existing);
  }
  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    const existing = this.listeners.get(type) ?? [];
    this.listeners.set(type, existing.filter((item) => item !== listener));
  }
  close() { this.closed = true; }
  simulateEvent(type: string, data: string) {
    const event = new MessageEvent(type, { data });
    const listeners = this.listeners.get(type) ?? [];
    for (const listener of listeners) {
      listener(event);
    }
  }
}

const originalEventSource = globalThis.EventSource;
const mockFetch = mock(() => Promise.resolve(new Response("{}")));
let mockEventSources: MockEventSource[] = [];

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  mockEventSources = [];
  (globalThis as unknown as { EventSource: typeof MockEventSource }).EventSource =
    class extends MockEventSource {
      constructor(url: string) {
        super(url);
        mockEventSources.push(this);
      }
    } as unknown as typeof EventSource;
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
});

const workspacesResponse: ApiWorkspacesResponse = {
  workspaces: [
    {
      name: "project-alpha",
      dir: "/home/user/project-alpha",
      running: true,
      needsInput: false,
      inProgress: true,
      pinned: false,
      isRoot: true,
      status: "Running",
      hasSgai: true,
      forks: [
        {
          name: "project-alpha-fork1",
          dir: "/home/user/project-alpha-fork1",
          running: false,
          needsInput: true,
          inProgress: true,
          pinned: false,
          isRoot: false,
          status: "Needs Input",
          hasSgai: true,
        },
      ],
    },
    {
      name: "project-beta",
      dir: "/home/user/project-beta",
      running: false,
      needsInput: false,
      inProgress: false,
      pinned: true,
      isRoot: true,
      status: "Stopped",
      hasSgai: true,
    },
  ],
};

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location-display">{location.pathname}</div>;
}

function renderDashboard(initialRoute = "/") {
  return render(
    <MemoryRouter initialEntries={[initialRoute]}>
      <TooltipProvider>
        <Routes>
          <Route
            path="/*"
            element={
              <Dashboard>
                <div data-testid="dashboard-content">Content</div>
                <LocationDisplay />
              </Dashboard>
            }
          />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("Dashboard", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(
      () => new Promise(() => {}),
    );
    renderDashboard();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders workspace tree when data loads", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(workspacesResponse)),
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
  });

  it("renders forks under root workspace", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(workspacesResponse)),
    );
    renderDashboard("/workspaces/project-alpha/progress");

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("project-alpha-fork1").length).toBeGreaterThan(0);
  });

  it("renders in-progress section for running workspaces", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(workspacesResponse)),
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    const inProgressSection = document.querySelector(".mb-3.pb-2.border-b");
    expect(inProgressSection).not.toBeNull();

    const links = inProgressSection!.querySelectorAll("a");
    const linkNames = Array.from(links).map((a) => a.textContent?.trim());
    expect(linkNames).toContain("project-alpha");
    expect(linkNames).toContain("project-alpha-fork1");
  });

  it("renders workspace indicators (pinned, running, needs input)", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(workspacesResponse)),
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    const runningIndicators = document.querySelectorAll('[title="Running"]');
    expect(runningIndicators.length).toBeGreaterThan(0);

    const pinnedIndicators = document.querySelectorAll("span");
    const hasPinned = Array.from(pinnedIndicators).some((el) => el.textContent === "📌");
    expect(hasPinned).toBe(true);

    const needsInputIndicators = document.querySelectorAll('[title="Waiting for response"]');
    expect(needsInputIndicators.length).toBeGreaterThan(0);
  });

  it("does not show running indicator for in-progress only workspaces", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(workspacesResponse)),
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    const inProgressIndicators = document.querySelectorAll('[title="In progress"]');
    expect(inProgressIndicators.length).toBe(0);
  });

  it("renders empty state when no workspaces", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [] })),
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText(/No workspaces found/i)).toBeDefined();
    });
  });

  it("renders error message when API fails", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load workspaces/i)).toBeDefined();
    });
  });

  it("renders new workspace button", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [] })),
    );
    renderDashboard();

    await waitFor(() => {
      const buttons = screen.getAllByText("[ + ]");
      expect(buttons.length).toBeGreaterThan(0);
    });
  });

  it("renders content in main area", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [] })),
    );
    renderDashboard();

    await waitFor(() => {
      const contents = screen.getAllByTestId("dashboard-content");
      expect(contents.length).toBeGreaterThan(0);
    });
  });

  it("uses responsive layout classes for sidebar and content", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [] })),
    );
    const { container } = renderDashboard();

    await waitFor(() => {
      const sidebarWrapper = container.querySelector("[data-slot='sidebar-wrapper']");
      expect(sidebarWrapper?.className).toContain("flex");
      const sidebar = container.querySelector("[data-sidebar='sidebar']");
      expect(sidebar).toBeTruthy();
      const contentArea = container.querySelector("main");
      expect(contentArea?.className).toContain("overflow-auto");
    });
  });

  it("calls workspace list API on mount", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [] })),
    );
    renderDashboard();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toBe("/api/v1/workspaces");
  });

  it("refreshes workspace list on workspace updates", async () => {
    const initialResponse: ApiWorkspacesResponse = {
      workspaces: workspacesResponse.workspaces.map((workspace) => ({
        ...workspace,
        running: false,
        inProgress: false,
      })),
    };
    const updatedResponse: ApiWorkspacesResponse = {
      workspaces: workspacesResponse.workspaces.map((workspace) => ({
        ...workspace,
        running: workspace.name === "project-alpha",
        inProgress: workspace.name === "project-alpha",
      })),
    };

    mockFetch
      .mockResolvedValueOnce(new Response(JSON.stringify(initialResponse)))
      .mockResolvedValueOnce(new Response(JSON.stringify(updatedResponse)));

    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
      expect(mockEventSources.length).toBeGreaterThan(0);
    });

    expect(mockFetch).toHaveBeenCalledTimes(1);

    mockEventSources[0].simulateEvent("workspace:update", JSON.stringify({ workspace: "project-alpha" }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(2);
      const runningIndicators = document.querySelectorAll('[title="Running"]');
      expect(runningIndicators.length).toBeGreaterThan(0);
    });
  });

  it("navigates to first needsInput workspace when inbox icon is clicked", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(workspacesResponse)),
    );
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    const inboxButtons = screen.getAllByRole("button", {
      name: /workspace.*waiting.*response/i,
    });
    expect(inboxButtons.length).toBeGreaterThan(0);

    fireEvent.click(inboxButtons[0]);

    await waitFor(() => {
      const locationEl = screen.getByTestId("location-display");
      expect(locationEl.textContent).toBe(
        "/workspaces/project-alpha-fork1/respond"
      );
    });
  });

  it("does not show skeleton during SSE-triggered refresh (stale-while-revalidate)", async () => {
    let resolveSecondFetch: ((value: Response) => void) | null = null;
    const secondFetchPromise = new Promise<Response>((resolve) => {
      resolveSecondFetch = resolve;
    });

    mockFetch
      .mockResolvedValueOnce(new Response(JSON.stringify(workspacesResponse)))
      .mockImplementationOnce(() => secondFetchPromise);

    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
      expect(mockEventSources.length).toBeGreaterThan(0);
    });

    mockEventSources[0].simulateEvent("workspace:update", JSON.stringify({ workspace: "project-alpha" }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBe(0);
    expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);

    resolveSecondFetch!(new Response(JSON.stringify(workspacesResponse)));
  });
});
