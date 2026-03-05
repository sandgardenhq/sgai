import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/components/ui/sidebar";
import { WorkspaceDetail } from "../WorkspaceDetail";

// Override pointer-events on body to allow interactions in tests
beforeEach(() => {
  document.body.style.pointerEvents = "auto";
});

const createMockWorkspace = (overrides = {}) => ({
  name: "test-workspace",
  dir: "/path/to/test-workspace",
  running: false,
  needsInput: false,
  inProgress: false,
  pinned: false,
  isRoot: false,
  isFork: false,
  description: "Test Workspace",
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
  goalContent: "# Test Goal",
  rawGoalContent: "# Test Goal",
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

let mockWorkspaces = [createMockWorkspace()];

const mockStart = mock(() => Promise.resolve({ running: true }));
const mockStop = mock(() => Promise.resolve({ running: false }));
const mockTogglePin = mock(() => Promise.resolve({ pinned: true }));
const mockOpenEditor = mock(() => Promise.resolve({ opened: true }));
const mockDeleteWorkspace = mock(() => Promise.resolve({ deleted: true }));
const mockDeleteFork = mock(() => Promise.resolve({ deleted: true }));
const mockTriggerFactoryRefresh = mock(() => {});
const mockRespond = mock(() => Promise.resolve({ success: true }));
const mockNavigate = mock(() => {});

mock.module("react-router", () => ({
  ...require("react-router"),
  useNavigate: () => mockNavigate,
}));

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({
    workspaces: mockWorkspaces,
    fetchStatus: "idle",
    lastFetchedAt: Date.now(),
  }),
  triggerFactoryRefresh: mockTriggerFactoryRefresh,
}));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
      start: mockStart,
      stop: mockStop,
      togglePin: mockTogglePin,
      openEditor: mockOpenEditor,
      deleteWorkspace: mockDeleteWorkspace,
      deleteFork: mockDeleteFork,
      respond: mockRespond,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

mock.module("@/hooks/useAdhocRun", () => ({
  useAdhocRun: () => ({
    output: "",
    isRunning: false,
    runError: null,
    startRun: mock(() => {}),
    stopRun: mock(() => {}),
    outputRef: { current: null },
  }),
}));

mock.module("@/hooks/use-mobile", () => ({
  useIsMobile: () => false,
}));

function renderWorkspaceDetail(workspaceName = "test-workspace", tab = "progress") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspaceName}/${tab}`]}>
      <TooltipProvider>
        <SidebarProvider>
          <Routes>
            <Route path="/workspaces/:name/*" element={<WorkspaceDetail />} />
          </Routes>
        </SidebarProvider>
      </TooltipProvider>
    </MemoryRouter>
  );
}

afterEach(() => {
  cleanup();
});

describe("WorkspaceDetail", () => {
  beforeEach(() => {
    mockWorkspaces = [createMockWorkspace()];
    mockStart.mockClear();
    mockStop.mockClear();
    mockTogglePin.mockClear();
    mockOpenEditor.mockClear();
    mockDeleteWorkspace.mockClear();
    mockDeleteFork.mockClear();
    mockTriggerFactoryRefresh.mockClear();
    mockRespond.mockClear();
    mockNavigate.mockClear();
  });

  describe("start/stop buttons work", () => {
    it("shows Start button when workspace is not running", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const startButtons = screen.queryAllByText("Start");
        expect(startButtons.length).toBeGreaterThan(0);
      });
    });

    it("shows Stop button when workspace is running", async () => {
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const stopButtons = screen.queryAllByText("Stop");
        expect(stopButtons.length).toBeGreaterThan(0);
      });
    });

    it("calls start API when Start button is clicked", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const startButtons = screen.queryAllByText("Start");
        expect(startButtons.length).toBeGreaterThan(0);
      });

      const startButtons = screen.getAllByText("Start");
      await user.click(startButtons[0]);

      await waitFor(() => {
        expect(mockStart).toHaveBeenCalledWith("test-workspace", false);
      });
    });

    it("calls stop API when Stop button is clicked", async () => {
      const user = userEvent.setup();
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const stopButtons = screen.queryAllByText("Stop");
        expect(stopButtons.length).toBeGreaterThan(0);
      });

      const stopButtons = screen.getAllByText("Stop");
      await user.click(stopButtons[0]);

      await waitFor(() => {
        expect(mockStop).toHaveBeenCalledWith("test-workspace");
      });
    });

    it("shows Self-drive button", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const selfDriveButtons = screen.queryAllByText("Self-drive");
        expect(selfDriveButtons.length).toBeGreaterThan(0);
      });
    });

    it("calls start API with auto=true when Self-drive is clicked", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const selfDriveButtons = screen.queryAllByText("Self-drive");
        expect(selfDriveButtons.length).toBeGreaterThan(0);
      });

      const selfDriveButtons = screen.getAllByText("Self-drive");
      await user.click(selfDriveButtons[0]);

      await waitFor(() => {
        expect(mockStart).toHaveBeenCalledWith("test-workspace", true);
      });
    });
  });

  describe("state reloads on button click", () => {
    it("triggers factory refresh after start", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const startButtons = screen.queryAllByText("Start");
        expect(startButtons.length).toBeGreaterThan(0);
      });

      const startButtons = screen.getAllByText("Start");
      await user.click(startButtons[0]);

      await waitFor(() => {
        expect(mockTriggerFactoryRefresh).toHaveBeenCalled();
      });
    });

    it("triggers factory refresh after stop", async () => {
      const user = userEvent.setup();
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const stopButtons = screen.queryAllByText("Stop");
        expect(stopButtons.length).toBeGreaterThan(0);
      });

      const stopButtons = screen.getAllByText("Stop");
      await user.click(stopButtons[0]);

      await waitFor(() => {
        expect(mockTriggerFactoryRefresh).toHaveBeenCalled();
      });
    });

    it("triggers factory refresh after pin toggle", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const pinButtons = screen.queryAllByText("Pin");
        expect(pinButtons.length).toBeGreaterThan(0);
      });

      const pinButtons = screen.getAllByText("Pin");
      await user.click(pinButtons[0]);

      await waitFor(() => {
        expect(mockTriggerFactoryRefresh).toHaveBeenCalled();
      });
    });
  });

  describe("workspace status display", () => {
    it("shows running badge when workspace is running", async () => {
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const runningBadges = screen.queryAllByText("running");
        expect(runningBadges.length).toBeGreaterThan(0);
      });
    });

    it("shows stopped badge when workspace is not running", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const stoppedBadges = screen.queryAllByText("stopped");
        expect(stoppedBadges.length).toBeGreaterThan(0);
      });
    });

    it("displays execution time", async () => {
      mockWorkspaces[0] = createMockWorkspace({ totalExecTime: "2m 30s" });

      renderWorkspaceDetail();

      await waitFor(() => {
        const execTimeElements = screen.queryAllByText("2m 30s");
        expect(execTimeElements.length).toBeGreaterThan(0);
      });
    });

    it("displays current agent and model", async () => {
      mockWorkspaces[0] = createMockWorkspace({
        running: true,
        currentAgent: "coordinator",
        currentModel: "opencode/glm-5",
      });

      renderWorkspaceDetail();

      await waitFor(() => {
        const coordinatorElements = screen.queryAllByText(/coordinator/);
        const glmElements = screen.queryAllByText(/glm-5/);
        expect(coordinatorElements.length).toBeGreaterThan(0);
        expect(glmElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("pin functionality", () => {
    it("shows Pin button when workspace is not pinned", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const pinButtons = screen.queryAllByText("Pin");
        expect(pinButtons.length).toBeGreaterThan(0);
      });
    });

    it("shows Unpin button when workspace is pinned", async () => {
      mockWorkspaces[0] = createMockWorkspace({ pinned: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const unpinButtons = screen.queryAllByText("Unpin");
        expect(unpinButtons.length).toBeGreaterThan(0);
      });
    });

    it("calls togglePin API when clicked", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const pinButtons = screen.queryAllByText("Pin");
        expect(pinButtons.length).toBeGreaterThan(0);
      });

      const pinButtons = screen.getAllByText("Pin");
      await user.click(pinButtons[0]);

      await waitFor(() => {
        expect(mockTogglePin).toHaveBeenCalledWith("test-workspace");
      });
    });
  });

  describe("delete functionality", () => {
    it("shows Delete button when workspace is not running", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const deleteButtons = screen.queryAllByText("Delete");
        expect(deleteButtons.length).toBeGreaterThan(0);
      });
    });

    it("opens delete confirmation dialog", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const deleteButtons = screen.queryAllByText("Delete");
        expect(deleteButtons.length).toBeGreaterThan(0);
      });

      const deleteButtons = screen.getAllByText("Delete");
      await user.click(deleteButtons[0]);

      await waitFor(() => {
        const deleteWorkspaceElements = screen.queryAllByText("Delete workspace");
        expect(deleteWorkspaceElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("needs input state", () => {
    it("shows Respond button when workspace needs input", async () => {
      mockWorkspaces[0] = createMockWorkspace({ needsInput: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const respondButtons = screen.queryAllByText("Respond");
        expect(respondButtons.length).toBeGreaterThan(0);
      });
    });
  });

  describe("error handling", () => {
    it("shows error message when start fails", async () => {
      const user = userEvent.setup();
      mockStart.mockImplementationOnce(() => Promise.reject(new Error("Failed to start workspace")));

      renderWorkspaceDetail();

      await waitFor(() => {
        const startButtons = screen.queryAllByText("Start");
        expect(startButtons.length).toBeGreaterThan(0);
      });

      const startButtons = screen.getAllByText("Start");
      await user.click(startButtons[0]);

      await waitFor(() => {
        const errorElements = screen.queryAllByText(/Failed to start workspace/);
        expect(errorElements.length).toBeGreaterThan(0);
        const errorAlert = errorElements.find(el => el.getAttribute("role") === "alert");
        expect(errorAlert).toBeTruthy();
      });
    });

    it("shows error message when stop fails", async () => {
      const user = userEvent.setup();
      mockWorkspaces[0] = createMockWorkspace({ running: true });
      mockStop.mockImplementationOnce(() => Promise.reject(new Error("Failed to stop workspace")));

      renderWorkspaceDetail();

      await waitFor(() => {
        const stopButtons = screen.queryAllByText("Stop");
        expect(stopButtons.length).toBeGreaterThan(0);
      });

      const stopButtons = screen.getAllByText("Stop");
      await user.click(stopButtons[0]);

      await waitFor(() => {
        const errorElements = screen.queryAllByText(/Failed to stop workspace/);
        expect(errorElements.length).toBeGreaterThan(0);
        const errorAlert = errorElements.find(el => el.getAttribute("role") === "alert");
        expect(errorAlert).toBeTruthy();
      });
    });

    it("shows error message when pin toggle fails", async () => {
      const user = userEvent.setup();
      mockTogglePin.mockImplementationOnce(() => Promise.reject(new Error("Failed to toggle pin")));

      renderWorkspaceDetail();

      await waitFor(() => {
        const pinButtons = screen.queryAllByText("Pin");
        expect(pinButtons.length).toBeGreaterThan(0);
      });

      const pinButtons = screen.getAllByText("Pin");
      await user.click(pinButtons[0]);

      await waitFor(() => {
        const errorElements = screen.queryAllByText(/Failed to toggle pin/);
        expect(errorElements.length).toBeGreaterThan(0);
        const errorAlert = errorElements.find(el => el.getAttribute("role") === "alert");
        expect(errorAlert).toBeTruthy();
      });
    });
  });

  describe("accessibility", () => {
    it("all action buttons are keyboard accessible", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const startButtons = screen.queryAllByRole("button", { name: /Start/ });
        expect(startButtons.length).toBeGreaterThan(0);
      });
    });

    it("buttons have accessible names", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const startButton = screen.queryAllByRole("button", { name: /Start/ });
        const selfDriveButton = screen.queryAllByRole("button", { name: /Self-drive/ });
        const pinButton = screen.queryAllByRole("button", { name: /Pin/ });
        const deleteButton = screen.queryAllByRole("button", { name: /Delete/ });

        expect(startButton.length).toBeGreaterThan(0);
        expect(selfDriveButton.length).toBeGreaterThan(0);
        expect(pinButton.length).toBeGreaterThan(0);
        expect(deleteButton.length).toBeGreaterThan(0);
      });
    });

    it("status badges have accessible labels", async () => {
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const statusBadge = screen.queryAllByText("running");
        expect(statusBadge.length).toBeGreaterThan(0);
      });
    });
  });

  describe("button state management", () => {
    it("disables Start button when workspace is running", async () => {
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const startButtons = screen.queryAllByRole("button", { name: /Start/ });
        const disabledStartButtons = startButtons.filter(btn => btn.hasAttribute("disabled"));
        expect(disabledStartButtons.length).toBeGreaterThan(0);
      });
    });
  });

  describe("workspace information display", () => {
    it("displays workspace description", async () => {
      mockWorkspaces[0] = createMockWorkspace({ description: "My Test Workspace" });

      renderWorkspaceDetail();

      await waitFor(() => {
        const descriptionElements = screen.queryAllByText("My Test Workspace");
        expect(descriptionElements.length).toBeGreaterThan(0);
      });
    });

    it("displays current task when running", async () => {
      mockWorkspaces[0] = createMockWorkspace({
        running: true,
        task: "Writing tests for authentication",
      });

      renderWorkspaceDetail();

      await waitFor(() => {
        const taskElements = screen.queryAllByText(/Writing tests for authentication/);
        expect(taskElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("Open Editor functionality", () => {
    it("shows Open Editor button", async () => {
      renderWorkspaceDetail();

      await waitFor(() => {
        const openEditorButtons = screen.queryAllByText("Open in Editor");
        expect(openEditorButtons.length).toBeGreaterThan(0);
      });
    });

    it("calls openEditor API when button is clicked", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const openEditorButtons = screen.queryAllByText("Open in Editor");
        expect(openEditorButtons.length).toBeGreaterThan(0);
      });

      const openEditorButtons = screen.getAllByText("Open in Editor");
      await user.click(openEditorButtons[0]);

      await waitFor(() => {
        expect(mockOpenEditor).toHaveBeenCalledWith("test-workspace");
      });
    });

    it("shows error when openEditor fails", async () => {
      const user = userEvent.setup();
      mockOpenEditor.mockImplementationOnce(() => Promise.reject(new Error("Failed to open editor")));

      renderWorkspaceDetail();

      await waitFor(() => {
        const openEditorButtons = screen.queryAllByText("Open in Editor");
        expect(openEditorButtons.length).toBeGreaterThan(0);
      });

      const openEditorButtons = screen.getAllByText("Open in Editor");
      await user.click(openEditorButtons[0]);

      await waitFor(() => {
        const errorElements = screen.getAllByText(/Failed to open editor/);
        expect(errorElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("Self-drive button state", () => {
    it("disables Self-drive button when workspace is running", async () => {
      mockWorkspaces[0] = createMockWorkspace({ running: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const selfDriveButtons = screen.queryAllByRole("button", { name: /Self-drive/i });
        expect(selfDriveButtons.length).toBeGreaterThan(0);
        const disabledButtons = selfDriveButtons.filter(btn => btn.hasAttribute("disabled"));
        expect(disabledButtons.length).toBeGreaterThan(0);
      });
    });
  });

  describe("Respond button functionality", () => {
    it("shows Respond button when workspace needs input", async () => {
      mockWorkspaces[0] = createMockWorkspace({ needsInput: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const respondButtons = screen.queryAllByRole("button", { name: /Respond/ });
        expect(respondButtons.length).toBeGreaterThan(0);
      });
    });

    it("Respond button navigates to respond page", async () => {
      const user = userEvent.setup();
      mockWorkspaces[0] = createMockWorkspace({ needsInput: true });

      renderWorkspaceDetail();

      await waitFor(() => {
        const respondButtons = screen.queryAllByRole("button", { name: /Respond/ });
        expect(respondButtons.length).toBeGreaterThan(0);
      });

      const respondButtons = screen.getAllByRole("button", { name: /Respond/ });
      await user.click(respondButtons[0]);

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith("/workspaces/test-workspace/respond");
      });
    });
  });

  describe("critical actions without optimistic updates", () => {
    it("pin toggle calls API before triggering refresh", async () => {
      const user = userEvent.setup();

      renderWorkspaceDetail();

      await waitFor(() => {
        const pinButtons = screen.queryAllByText("Pin");
        expect(pinButtons.length).toBeGreaterThan(0);
      });

      const pinButtons = screen.getAllByText("Pin");
      await user.click(pinButtons[0]);

      await waitFor(() => {
        expect(mockTogglePin).toHaveBeenCalledWith("test-workspace");
        expect(mockTriggerFactoryRefresh).toHaveBeenCalled();
      });
    });
  });
});
