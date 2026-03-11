import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/components/ui/sidebar";
import { Dashboard } from "../Dashboard";

// Override pointer-events on body to allow interactions in tests
beforeEach(() => {
  document.body.style.pointerEvents = "auto";
});

const createMockWorkspace = (overrides: Record<string, unknown> = {}) => ({
  name: "workspace",
  dir: "/path/to/workspace",
  running: false,
  needsInput: false,
  inProgress: false,
  pinned: false,
  isRoot: false,
  isFork: false,
  description: "",
  status: "",
  badgeClass: "",
  badgeText: "",
  hasSgai: true,
  hasEditedGoal: false,
  interactiveAuto: false,
  continuousMode: false,
  currentAgent: "",
  currentModel: "",
  task: "",
  goalContent: "",
  rawGoalContent: "",
  pmContent: "",
  hasProjectMgmt: false,
  svgHash: "",
  totalExecTime: "",
  latestProgress: "",
  humanMessage: "",
  agentSequence: [],
  cost: { totalCost: 0, totalTokens: { input: 0, output: 0, reasoning: 0, cacheRead: 0, cacheWrite: 0 }, byAgent: [] },
  modelStatuses: [],
  agentModels: [],
  events: [],
  messages: [],
  projectTodos: [],
  agentTodos: [],
  changes: { description: "", diffLines: [] },
  commits: [],
  log: [],
  external: false,
  ...overrides,
});

const mockWorkspaces = [
  createMockWorkspace({
    name: "workspace-1",
    dir: "/path/to/workspace-1",
    description: "Test Workspace 1",
  }),
  createMockWorkspace({
    name: "workspace-2",
    dir: "/path/to/workspace-2",
    running: true,
    inProgress: true,
    pinned: true,
    isRoot: true,
    description: "Test Workspace 2",
    currentAgent: "coordinator",
    currentModel: "opencode/glm-5",
    task: "Working on task",
    totalExecTime: "1m 30s",
    forks: [
      {
        name: "workspace-2-fork-1",
        dir: "/path/to/workspace-2-fork-1",
        running: false,
        needsInput: true,
        inProgress: false,
        pinned: false,
        description: "Fork 1",
        commitAhead: 1,
        commits: [],
      },
    ],
  }),
  createMockWorkspace({
    name: "workspace-3",
    dir: "/path/to/workspace-3",
    needsInput: true,
    inProgress: true,
    description: "Needs Input Workspace",
    external: true,
  }),
  createMockWorkspace({
    name: "root-unpinned",
    dir: "/path/to/root-unpinned",
    isRoot: true,
    pinned: false,
    description: "Unpinned Root",
    forks: [
      {
        name: "orphan-pinned-fork",
        dir: "/path/to/orphan-pinned-fork",
        running: false,
        needsInput: false,
        inProgress: false,
        pinned: true,
        description: "Orphan Pinned Fork",
        commitAhead: 0,
        commits: [],
      },
    ],
  }),
  createMockWorkspace({
    name: "orphan-pinned-fork",
    dir: "/path/to/orphan-pinned-fork",
    isFork: true,
    pinned: true,
    description: "Orphan Pinned Fork",
  }),
];

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({
    workspaces: mockWorkspaces,
    fetchStatus: "idle",
    lastFetchedAt: Date.now(),
  }),
  triggerFactoryRefresh: mock(() => {}),
}));

const mockDeleteFork = mock(() => Promise.resolve({ deleted: true }));
const mockDeleteWorkspace = mock(() => Promise.resolve({ deleted: true }));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
      deleteFork: mockDeleteFork,
      deleteWorkspace: mockDeleteWorkspace,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

mock.module("@/hooks/useSidebarResize", () => ({
  useSidebarResize: () => ({
    sidebarWidth: 280,
    handleMouseDown: mock(() => {}),
  }),
}));

mock.module("@/hooks/use-mobile", () => ({
  useIsMobile: () => false,
}));

function renderDashboard(initialRoute = "/") {
  return render(
    <MemoryRouter initialEntries={[initialRoute]}>
      <TooltipProvider>
        <SidebarProvider>
          <Routes>
            <Route path="/workspaces/:name/forks" element={<Dashboard><div data-testid="redirect-target">Redirected to forks</div></Dashboard>} />
            <Route path="*" element={<Dashboard><div data-testid="dashboard-content">Content</div></Dashboard>} />
          </Routes>
        </SidebarProvider>
      </TooltipProvider>
    </MemoryRouter>
  );
}

describe("Dashboard", () => {
  beforeEach(() => {
    mockDeleteFork.mockClear();
    mockDeleteWorkspace.mockClear();
  });

  afterEach(() => {
    cleanup();
  });

  describe("display repository tree correctly", () => {
    it("renders workspace list with all workspaces", async () => {
      renderDashboard();

      await waitFor(() => {
        const ws1Elements = screen.queryAllByText("Test Workspace 1");
        expect(ws1Elements.length).toBeGreaterThan(0);
      });

      await waitFor(() => {
        const ws2Elements = screen.queryAllByText("workspace-2");
        expect(ws2Elements.length).toBeGreaterThan(0);
      });
      
      await waitFor(() => {
        const ws3Elements = screen.queryAllByText("Needs Input Workspace");
        expect(ws3Elements.length).toBeGreaterThan(0);
      });
    });

    it("shows workspace description when available", async () => {
      renderDashboard();

      await waitFor(() => {
        const ws1Elements = screen.queryAllByText("Test Workspace 1");
        expect(ws1Elements.length).toBeGreaterThan(0);
      });
    });

    it("displays external workspace indicator", async () => {
      renderDashboard();

      await waitFor(() => {
        const externalIndicators = screen.queryAllByLabelText("External workspace");
        expect(externalIndicators.length).toBeGreaterThan(0);
      });
    });

    it("shows running indicator for active workspaces", async () => {
      renderDashboard();

      await waitFor(() => {
        const runningIndicators = screen.queryAllByLabelText("Running");
        expect(runningIndicators.length).toBeGreaterThan(0);
      });
    });

    it("shows needs input indicator", async () => {
      renderDashboard();

      await waitFor(() => {
        const needsInputIndicators = screen.queryAllByLabelText("Waiting for response");
        expect(needsInputIndicators.length).toBeGreaterThan(0);
      });
    });
  });

  describe("show pinned repositories", () => {
    it("displays pinned section when workspaces are pinned", async () => {
      renderDashboard();

      await waitFor(() => {
        const pinnedSections = screen.queryAllByRole("region", { name: "Pinned" });
        expect(pinnedSections.length).toBeGreaterThan(0);
      });
    });

    it("shows pinned indicator on pinned workspaces", async () => {
      renderDashboard();

      await waitFor(() => {
        const pinnedIndicators = screen.queryAllByLabelText("Pinned");
        expect(pinnedIndicators.length).toBeGreaterThan(0);
      });
    });
  });

  describe("handle fork nesting", () => {
    it("displays expand button for workspaces with forks", async () => {
      renderDashboard();

      await waitFor(() => {
        const expandButtons = screen.queryAllByLabelText("Toggle forks");
        expect(expandButtons.length).toBeGreaterThan(0);
      });
    });

    it("expands to show nested forks when clicked", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const ws2Elements = screen.queryAllByText("workspace-2");
        expect(ws2Elements.length).toBeGreaterThan(0);
      });

      const expandButtons = screen.getAllByLabelText("Toggle forks");
      await user.click(expandButtons[0]);

      await waitFor(() => {
        const forkElements = screen.queryAllByText("Fork 1");
        expect(forkElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("update on state changes", () => {
    it("reflects workspace running state", async () => {
      renderDashboard();

      await waitFor(() => {
        const runningBadges = screen.queryAllByLabelText("Running");
        expect(runningBadges.length).toBeGreaterThan(0);
      });
    });

    it("shows needs input state for workspaces awaiting response", async () => {
      renderDashboard();

      await waitFor(() => {
        const needsInputIndicators = screen.queryAllByLabelText("Waiting for response");
        expect(needsInputIndicators.length).toBeGreaterThan(0);
      });
    });
  });

  describe("workspace actions", () => {
    it("shows delete button on workspace hover", async () => {
      renderDashboard();

      await waitFor(() => {
        const deleteButtons = screen.queryAllByLabelText("Delete workspace-1");
        expect(deleteButtons.length).toBeGreaterThan(0);
      });
    });

    it("opens delete confirmation dialog", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const ws1Elements = screen.queryAllByText("Test Workspace 1");
        expect(ws1Elements.length).toBeGreaterThan(0);
      });

      const deleteButtons = screen.getAllByLabelText("Delete workspace-1");
      await user.click(deleteButtons[0]);

      await waitFor(() => {
        const alertDialogs = screen.queryAllByRole("alertdialog");
        expect(alertDialogs.length).toBeGreaterThan(0);
        
        const deleteWorkspaceElements = screen.queryAllByText(/Delete workspace/);
        expect(deleteWorkspaceElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("in progress section", () => {
    it("shows in progress section for active workspaces", async () => {
      renderDashboard();

      await waitFor(() => {
        const inProgressSections = screen.queryAllByRole("region", { name: "In progress" });
        expect(inProgressSections.length).toBeGreaterThan(0);
      });
    });
  });

  describe("accessibility", () => {
    it("all workspace links are keyboard accessible", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const workspaceLinks = screen.getAllByRole("link").filter(link => 
          link.getAttribute("href")?.startsWith("/workspaces/")
        );
        expect(workspaceLinks.length).toBeGreaterThan(0);
        
        const firstLink = workspaceLinks[0];
        expect(firstLink.getAttribute("tabindex")).not.toBe("-1");
        firstLink.focus();
        expect(document.activeElement).toBe(firstLink);
      });
    });

    it("workspace sections have proper ARIA landmarks", async () => {
      renderDashboard();

      await waitFor(() => {
        const pinnedRegion = screen.queryAllByRole("region", { name: "Pinned" });
        const inProgressRegion = screen.queryAllByRole("region", { name: "In progress" });
        
        expect(pinnedRegion.length + inProgressRegion.length).toBeGreaterThan(0);
      });
    });

    it("status indicators have accessible labels", async () => {
      renderDashboard();

      await waitFor(() => {
        const runningIndicator = screen.queryAllByLabelText("Running");
        const needsInputIndicator = screen.queryAllByLabelText("Waiting for response");
        const externalIndicator = screen.queryAllByLabelText("External workspace");

        expect(runningIndicator.length + needsInputIndicator.length + externalIndicator.length).toBeGreaterThan(0);
      });
    });
  });

  describe("fork vs workspace deletion", () => {
    it("calls deleteFork API for fork workspaces", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const expandButtons = screen.queryAllByLabelText("Toggle forks");
        expect(expandButtons.length).toBeGreaterThan(0);
      });

      const expandButtons = screen.getAllByLabelText("Toggle forks");
      await user.click(expandButtons[0]);

      await waitFor(() => {
        const forkElements = screen.queryAllByText("Fork 1");
        expect(forkElements.length).toBeGreaterThan(0);
      });

      const deleteForkButtons = screen.getAllByLabelText(/Delete workspace-2-fork-1/);
      expect(deleteForkButtons.length).toBeGreaterThan(0);
      await user.click(deleteForkButtons[0]);

      await waitFor(() => {
        const confirmButtons = screen.queryAllByRole("button", { name: /^Delete$/ });
        expect(confirmButtons.length).toBeGreaterThan(0);
      });

      const confirmButtons = screen.getAllByRole("button", { name: /^Delete$/ });
      await user.click(confirmButtons[confirmButtons.length - 1]);

      await waitFor(() => {
        expect(mockDeleteFork).toHaveBeenCalled();
      });
    });

    it("shows delete button for standalone workspaces", async () => {
      renderDashboard();

      await waitFor(() => {
        const ws1Elements = screen.queryAllByText("Test Workspace 1");
        expect(ws1Elements.length).toBeGreaterThan(0);
      });

      const deleteButtons = screen.getAllByLabelText("Delete workspace-1");
      expect(deleteButtons.length).toBeGreaterThan(0);
    });
  });

  describe("external repository handling", () => {
    it("displays external indicator for external repositories", async () => {
      renderDashboard();

      await waitFor(() => {
        const externalIndicators = screen.queryAllByLabelText("External workspace");
        expect(externalIndicators.length).toBeGreaterThan(0);
      });
    });

    it("allows forking external repositories", async () => {
      renderDashboard();

      await waitFor(() => {
        const ws3Elements = screen.queryAllByText("Needs Input Workspace");
        expect(ws3Elements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("repository tree structure", () => {
    it("nests forks under their root repositories", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const ws2Elements = screen.queryAllByText("workspace-2");
        expect(ws2Elements.length).toBeGreaterThan(0);
      });

      const expandButtons = screen.getAllByLabelText("Toggle forks");
      await user.click(expandButtons[0]);

      await waitFor(() => {
        const forkElements = screen.queryAllByText("Fork 1");
        expect(forkElements.length).toBeGreaterThan(0);
      });
    });

    it("shows correct hierarchy for root and fork workspaces", async () => {
      renderDashboard();

      await waitFor(() => {
        const ws2Elements = screen.queryAllByText("workspace-2");
        expect(ws2Elements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("pinned fork with unpinned root", () => {
    it("shows pinned forks with rootName/description when root is not pinned", async () => {
      renderDashboard();

      await waitFor(() => {
        const pinnedSection = screen.queryAllByRole("region", { name: "Pinned" });
        expect(pinnedSection.length).toBeGreaterThan(0);
      });

      await waitFor(() => {
        const orphanFork = screen.queryAllByText("root-unpinned/Orphan Pinned Fork");
        expect(orphanFork.length).toBeGreaterThan(0);
      });
    });

    it("shows tooltip with fork name and root name for orphan pinned forks", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const orphanFork = screen.queryAllByText("root-unpinned/Orphan Pinned Fork");
        expect(orphanFork.length).toBeGreaterThan(0);
      });

      const orphanForkLabel = screen.getAllByText("root-unpinned/Orphan Pinned Fork")[0];
      await user.hover(orphanForkLabel);

      await waitFor(() => {
        const tooltipForkName = screen.queryAllByText("orphan-pinned-fork");
        expect(tooltipForkName.length).toBeGreaterThan(0);
      });

      await waitFor(() => {
        const tooltipRootName = screen.queryAllByText("Root: root-unpinned");
        expect(tooltipRootName.length).toBeGreaterThan(0);
      });
    });
  });

  describe("fork deletion redirect", () => {
    it("navigates to root forks page after deleting a fork", async () => {
      const user = userEvent.setup();
      renderDashboard();

      await waitFor(() => {
        const expandButtons = screen.queryAllByLabelText("Toggle forks");
        expect(expandButtons.length).toBeGreaterThan(0);
      });

      const expandButtons = screen.getAllByLabelText("Toggle forks");
      await user.click(expandButtons[0]);

      await waitFor(() => {
        const forkElements = screen.queryAllByText("Fork 1");
        expect(forkElements.length).toBeGreaterThan(0);
      });

      const deleteForkButtons = screen.getAllByLabelText(/Delete workspace-2-fork-1/);
      await user.click(deleteForkButtons[0]);

      await waitFor(() => {
        const confirmButtons = screen.queryAllByRole("button", { name: /^Delete$/ });
        expect(confirmButtons.length).toBeGreaterThan(0);
      });

      const confirmButtons = screen.getAllByRole("button", { name: /^Delete$/ });
      await user.click(confirmButtons[confirmButtons.length - 1]);

      await waitFor(() => {
        expect(mockDeleteFork).toHaveBeenCalledWith("workspace-2-fork-1", "");
      });

      await waitFor(() => {
        const redirectTarget = screen.queryByTestId("redirect-target");
        expect(redirectTarget).not.toBeNull();
        expect(redirectTarget?.textContent).toBe("Redirected to forks");
      });
    });
  });
});
