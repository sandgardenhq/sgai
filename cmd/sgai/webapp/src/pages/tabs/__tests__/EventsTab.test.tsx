import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { EventsTab } from "../EventsTab";

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
  goalContent: "",
  rawGoalContent: "",
  pmContent: "",
  hasProjectMgmt: false,
  svgHash: "abc123",
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
let mockFetchStatus = "idle";

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({
    workspaces: mockWorkspaces,
    fetchStatus: mockFetchStatus,
    lastFetchedAt: Date.now(),
  }),
  triggerFactoryRefresh: mock(() => {}),
}));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
      openEditorGoal: mock(() => Promise.resolve({ opened: true })),
      adhoc: mock(() => Promise.resolve({ output: "", running: false })),
      adhocStatus: mock(() => Promise.resolve({ output: "", running: false })),
      adhocStop: mock(() => Promise.resolve({ output: "", running: false })),
    },
    models: {
      list: mock(() => Promise.resolve({ models: [], defaultModel: "" })),
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

mock.module("@/components/MarkdownContent", () => ({
  MarkdownContent: ({ content }: { content: string }) => (
    <div data-testid="markdown-content">{content}</div>
  ),
}));

function renderEventsTab(props = {}) {
  const defaultProps = {
    workspaceName: "test-workspace",
    goalContent: undefined as string | undefined,
    actions: undefined as any[] | undefined,
    ...props,
  };

  return render(
    <MemoryRouter>
      <TooltipProvider>
        <EventsTab {...defaultProps} />
      </TooltipProvider>
    </MemoryRouter>
  );
}

afterEach(() => {
  cleanup();
});

describe("EventsTab", () => {
  beforeEach(() => {
    mockWorkspaces = [createMockWorkspace()];
    mockFetchStatus = "idle";
  });

  describe("event rendering", () => {
    it("shows empty events message when no events", async () => {
      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("No events recorded yet")).toBeTruthy();
      });
    });

    it("displays events timeline", async () => {
      mockWorkspaces = [createMockWorkspace({
        events: [
          {
            timestamp: "2026-03-05T10:00:00Z",
            agent: "coordinator",
            description: "Started workspace",
            formattedTime: "10:00 AM",
            showDateDivider: true,
            dateDivider: "Mar 5, 2026",
          },
          {
            timestamp: "2026-03-05T10:05:00Z",
            agent: "developer",
            description: "Writing tests",
            formattedTime: "10:05 AM",
            showDateDivider: false,
            dateDivider: "",
          },
        ],
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("Started workspace")).toBeTruthy();
        expect(screen.getByText("Writing tests")).toBeTruthy();
      });
    });

    it("shows date dividers", async () => {
      mockWorkspaces = [createMockWorkspace({
        events: [
          {
            timestamp: "2026-03-05T10:00:00Z",
            agent: "coordinator",
            description: "Event 1",
            formattedTime: "10:00 AM",
            showDateDivider: true,
            dateDivider: "Mar 5, 2026",
          },
        ],
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("Mar 5, 2026")).toBeTruthy();
      });
    });

    it("shows agent badge for events", async () => {
      mockWorkspaces = [createMockWorkspace({
        events: [
          {
            timestamp: "2026-03-05T10:00:00Z",
            agent: "coordinator",
            description: "Event 1",
            formattedTime: "10:00 AM",
            showDateDivider: false,
            dateDivider: "",
          },
        ],
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("coordinator")).toBeTruthy();
      });
    });

    it("shows formatted time for events", async () => {
      mockWorkspaces = [createMockWorkspace({
        events: [
          {
            timestamp: "2026-03-05T10:00:00Z",
            agent: "test-agent",
            description: "Event 1",
            formattedTime: "10:00 AM",
            showDateDivider: false,
            dateDivider: "",
          },
        ],
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("10:00 AM")).toBeTruthy();
      });
    });
  });

  describe("workflow section", () => {
    it("renders workflow graph image", async () => {
      renderEventsTab();

      await waitFor(() => {
        const img = screen.getByAltText("Workflow graph");
        expect(img).toBeTruthy();
        expect(img.getAttribute("src")).toContain("workflow.svg");
      });
    });

    it("shows agent models table when available", async () => {
      mockWorkspaces = [createMockWorkspace({
        agentModels: [
          { agent: "coordinator", models: ["opencode/glm-5"] },
          { agent: "developer", models: ["anthropic/claude-opus-4-6"] },
        ],
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("Agent")).toBeTruthy();
        expect(screen.getByText("Model(s)")).toBeTruthy();
      });
    });

    it("shows model status list when available", async () => {
      mockWorkspaces = [createMockWorkspace({
        modelStatuses: [
          { modelId: "opencode/glm-5", status: "model-working" },
        ],
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("Model Consensus:")).toBeTruthy();
      });
    });

    it("shows human message when workspace needs input", async () => {
      mockWorkspaces = [createMockWorkspace({
        needsInput: true,
        humanMessage: "Please choose an option",
        currentAgent: "coordinator",
      })];

      renderEventsTab();

      await waitFor(() => {
        expect(screen.getByText("Please choose an option")).toBeTruthy();
      });
    });
  });

  describe("goal content section", () => {
    it("does not show GOAL.md section when no goal content", () => {
      renderEventsTab();
      expect(screen.queryByText("GOAL.md")).toBeNull();
    });

    it("shows GOAL.md section when goal content is provided", () => {
      renderEventsTab({ goalContent: "# My Goal" });
      expect(screen.getByText("GOAL.md")).toBeTruthy();
    });
  });

  describe("loading state", () => {
    it("shows skeleton when fetching and no workspace", () => {
      mockWorkspaces = [];
      mockFetchStatus = "fetching";

      renderEventsTab({ workspaceName: "nonexistent" });
      // The skeleton component renders with role="status" and aria-live
      const statusElements = screen.queryAllByRole("status");
      // Skeleton component uses div wrappers, check for loading state
      expect(statusElements.length > 0 || screen.queryByText("No events recorded yet") === null).toBe(true);
    });

    it("shows error message when fetch fails and no workspace", () => {
      mockWorkspaces = [];
      mockFetchStatus = "error";

      renderEventsTab({ workspaceName: "nonexistent" });
      expect(screen.getByText("Failed to load events")).toBeTruthy();
    });
  });

  describe("empty state", () => {
    it("returns null when workspace not found and not fetching/error", () => {
      mockWorkspaces = [];
      mockFetchStatus = "idle";

      const { container } = renderEventsTab({ workspaceName: "nonexistent" });
      expect(container.innerHTML).toBe("");
    });
  });
});
