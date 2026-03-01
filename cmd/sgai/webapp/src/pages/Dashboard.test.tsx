import { describe, it, expect, beforeEach, afterEach, mock, afterAll } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent, act } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router";
import { Dashboard } from "./Dashboard";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { FactoryStateSnapshot } from "@/lib/factory-state";
import { useFactoryState as _realUseFactoryState } from "@/lib/factory-state";
import { createMockWorkspace, createMockFork } from "@/test/factories";

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
  createMockWorkspace({
    name: "project-alpha",
    dir: "/home/user/project-alpha",
    running: true,
    needsInput: false,
    inProgress: true,
    isRoot: true,
    status: "Running",
    hasSgai: true,
    forks: [
      createMockFork({
        name: "project-alpha-fork1",
        dir: "/home/user/project-alpha-fork1",
        needsInput: true,
        inProgress: true,
        goalDescription: "Implement user authentication with OAuth2",
      }),
    ],
  }),
  createMockWorkspace({
    name: "project-alpha-fork1",
    dir: "/home/user/project-alpha-fork1",
    needsInput: true,
    inProgress: true,
    isRoot: false,
    isFork: true,
    status: "Needs Input",
    hasSgai: true,
    goalDescription: "Implement user authentication with OAuth2",
  }),
  createMockWorkspace({
    name: "project-beta",
    dir: "/home/user/project-beta",
    pinned: true,
    isRoot: true,
    status: "Stopped",
    hasSgai: true,
  }),
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

  it("renders promoted forks and standalone workspaces when data loads", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("Implement user authentication with OAuth2").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
  });

  it("shows goalDescription for promoted forks instead of directory name", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("Implement user authentication with OAuth2").length).toBeGreaterThan(0);
    });
  });

  it("shows New task button for root repos in forked mode", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("New task").length).toBeGreaterThan(0);
    });
  });

  it("renders forks as top-level items in forked mode", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      const sidebar = document.querySelector("[data-sidebar='sidebar']");
      expect(sidebar).toBeTruthy();

      const links = sidebar!.querySelectorAll("a[href*='project-alpha-fork1']");
      expect(links.length).toBeGreaterThan(0);
    });
  });

  it("falls back to fork name when goalDescription is absent", async () => {
    const dataWithoutGoalDesc: FactoryStateSnapshot["workspaces"] = [
      createMockWorkspace({
        name: "project-gamma",
        dir: "/home/user/project-gamma",
        isRoot: true,
        status: "Stopped",
        hasSgai: true,
        forks: [
          createMockFork({
            name: "happy-blue-3a2e",
            dir: "/home/user/happy-blue-3a2e",
          }),
        ],
      }),
    ];
    mockFactoryState = {
      workspaces: dataWithoutGoalDesc,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("happy-blue-3a2e").length).toBeGreaterThan(0);
    });
  });

  it("renders in-progress section for running workspaces", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      const inProgressSection = document.querySelector(".mb-3.pb-2.border-b");
      expect(inProgressSection).not.toBeNull();
    });
  });

  it("renders workspace indicators (pinned, running, needs input)", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
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
      expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
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
      expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
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
      expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
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
      expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
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
    expect(screen.getAllByText("project-beta").length).toBeGreaterThan(0);
  });

  it("renders standalone workspaces normally (without forks)", async () => {
    const standaloneData: FactoryStateSnapshot["workspaces"] = [
      createMockWorkspace({
        name: "standalone-repo",
        dir: "/home/user/standalone-repo",
        isRoot: true,
        status: "Stopped",
        hasSgai: true,
        summary: "A standalone project",
      }),
    ];
    mockFactoryState = {
      workspaces: standaloneData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      expect(screen.getAllByText("standalone-repo").length).toBeGreaterThan(0);
      expect(screen.getAllByText("A standalone project").length).toBeGreaterThan(0);
    });
  });

  it("shows goalDescription in in-progress section for forked workspaces", async () => {
    mockFactoryState = {
      workspaces: workspacesData,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderDashboard();

    await waitFor(() => {
      const inProgressSection = document.querySelector(".mb-3.pb-2.border-b");
      expect(inProgressSection).not.toBeNull();

      const links = inProgressSection!.querySelectorAll("a");
      const linkTexts = Array.from(links).map((a) => a.textContent?.trim());
      const hasGoalDesc = linkTexts.some((t) => t?.includes("Implement user authentication with OAuth2"));
      expect(hasGoalDesc).toBe(true);
    });
  });
});
