import { describe, it, expect, beforeEach, afterEach, mock, afterAll } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent, act } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router";
import { Dashboard } from "./Dashboard";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { FactoryStateSnapshot } from "@/lib/factory-state";
import { useFactoryState as _realUseFactoryState } from "@/lib/factory-state";

const savedExports = {
  useFactoryState: _realUseFactoryState,
  resetFactoryStateStore: () => {},
};

let mockFactoryState: FactoryStateSnapshot = {
  workspaces: [],
  fetchStatus: "fetching",
  lastFetchedAt: null,
};

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => mockFactoryState,
  resetFactoryStateStore: () => {},
}));

afterAll(() => {
  mock.module("@/lib/factory-state", () => savedExports);
});

const workspacesData: FactoryStateSnapshot["workspaces"] = [
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
        commitAhead: 0,
        commits: [],
      },
    ],
  },
  {
    name: "project-alpha-fork1",
    dir: "/home/user/project-alpha-fork1",
    running: false,
    needsInput: true,
    inProgress: true,
    pinned: false,
    isRoot: false,
    isFork: true,
    status: "Needs Input",
    hasSgai: true,
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
];

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
  beforeEach(() => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "fetching",
      lastFetchedAt: null,
    };
  });

  afterEach(() => {
    cleanup();
  });

  it("renders loading skeleton when fetching and no data", () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "fetching",
      lastFetchedAt: null,
    };
    renderDashboard();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders workspace tree when data loads", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
  });

  it("renders forks under root workspace", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard("/workspaces/project-alpha/progress");

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("project-alpha-fork1").length).toBeGreaterThan(0);
  });

  it("renders in-progress section for running workspaces", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
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
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
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
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    const inProgressIndicators = document.querySelectorAll('[title="In progress"]');
    expect(inProgressIndicators.length).toBe(0);
  });

  it("renders empty state when no workspaces", async () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText(/No workspaces found/i)).toBeDefined();
    });
  });

  it("renders error message when fetch fails", async () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "error",
      lastFetchedAt: null,
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load workspaces/i)).toBeDefined();
    });
  });

  it("renders new workspace button", async () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      const buttons = screen.getAllByText("[ + ]");
      expect(buttons.length).toBeGreaterThan(0);
    });
  });

  it("renders content in main area", async () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      const contents = screen.getAllByTestId("dashboard-content");
      expect(contents.length).toBeGreaterThan(0);
    });
  });

  it("uses responsive layout classes for sidebar and content", async () => {
    mockFactoryState = {
      workspaces: [],
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
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

  it("updates workspace list when factory state changes", async () => {
    mockFactoryState = {
      workspaces: workspacesData.map((w) => ({ ...w, running: false, inProgress: false })),
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    const { rerender } = renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    act(() => {
      mockFactoryState = {
        workspaces: workspacesData,
        fetchStatus: "idle",
        lastFetchedAt: Date.now() + 1,
      };
    });
    rerender(
      <MemoryRouter initialEntries={["/"]}>
        <TooltipProvider>
          <Routes>
            <Route
              path="/*"
              element={
                <Dashboard>
                  <div data-testid="dashboard-content">Content</div>
                </Dashboard>
              }
            />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      const runningIndicators = document.querySelectorAll('[title="Running"]');
      expect(runningIndicators.length).toBeGreaterThan(0);
    });
  });

  it("navigates to first needsInput workspace when inbox icon is clicked", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
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

  it("does not show skeleton during factory state refresh (stale-while-revalidate)", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    const { rerender } = renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
    });

    act(() => {
      mockFactoryState = {
        workspaces: workspacesData,
        fetchStatus: "fetching",
        lastFetchedAt: Date.now(),
      };
    });
    rerender(
      <MemoryRouter initialEntries={["/"]}>
        <TooltipProvider>
          <Routes>
            <Route
              path="/*"
              element={
                <Dashboard>
                  <div data-testid="dashboard-content">Content</div>
                </Dashboard>
              }
            />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBe(0);
    expect(screen.getAllByText("project-alpha").length).toBeGreaterThan(0);
  });
});
