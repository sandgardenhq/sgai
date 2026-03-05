import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SessionTab, ActionBar } from "../SessionTab";

beforeEach(() => {
  document.body.style.pointerEvents = "auto";
});

const mockSteer = mock(() => Promise.resolve({ success: true, message: "ok" }));
const mockOpenEditorPM = mock(() => Promise.resolve({ opened: true }));
const mockTriggerFactoryRefresh = mock(() => {});

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
      steer: mockSteer,
      openEditorProjectManagement: mockOpenEditorPM,
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

function renderSessionTab(props = {}) {
  const defaultProps = {
    workspaceName: "test-workspace",
    pmContent: undefined as string | undefined,
    hasProjectMgmt: false,
    ...props,
  };

  return render(
    <MemoryRouter>
      <TooltipProvider>
        <SessionTab {...defaultProps} />
      </TooltipProvider>
    </MemoryRouter>
  );
}

afterEach(() => {
  cleanup();
});

describe("SessionTab", () => {
  beforeEach(() => {
    mockWorkspaces = [createMockWorkspace()];
    mockSteer.mockClear();
    mockOpenEditorPM.mockClear();
    mockTriggerFactoryRefresh.mockClear();
  });

  describe("steer next turn", () => {
    it("renders steer section with textarea", async () => {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Steer Next Turn")).toBeTruthy();
        expect(screen.getByPlaceholderText("Enter re-steering instruction...")).toBeTruthy();
      });
    });

    it("shows submit button", async () => {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Submit")).toBeTruthy();
      });
    });

    it("disables submit when textarea is empty", async () => {
      renderSessionTab();

      await waitFor(() => {
        const submitButton = screen.getByText("Submit");
        expect(submitButton.hasAttribute("disabled")).toBe(true);
      });
    });

    it("enables submit when textarea has content", async () => {
      renderSessionTab();

      const textarea = screen.getByPlaceholderText("Enter re-steering instruction...");
      fireEvent.change(textarea, { target: { value: "go faster" } });

      await waitFor(() => {
        const submitButton = screen.getByText("Submit");
        expect(submitButton.hasAttribute("disabled")).toBe(false);
      });
    });

    it("calls steer API on submit", async () => {
      const user = userEvent.setup();
      renderSessionTab();

      const textarea = screen.getByPlaceholderText("Enter re-steering instruction...");
      fireEvent.change(textarea, { target: { value: "go faster" } });

      const submitButton = screen.getByText("Submit");
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockSteer).toHaveBeenCalledWith("test-workspace", "go faster");
      });
    });

    it("shows success message after steer", async () => {
      const user = userEvent.setup();
      renderSessionTab();

      const textarea = screen.getByPlaceholderText("Enter re-steering instruction...");
      fireEvent.change(textarea, { target: { value: "go faster" } });

      const submitButton = screen.getByText("Submit");
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText("Steering instruction sent.")).toBeTruthy();
      });
    });

    it("shows error when steer fails", async () => {
      const user = userEvent.setup();
      mockSteer.mockImplementationOnce(() => Promise.reject(new Error("Steer failed")));

      renderSessionTab();

      const textarea = screen.getByPlaceholderText("Enter re-steering instruction...");
      fireEvent.change(textarea, { target: { value: "go faster" } });

      const submitButton = screen.getByText("Submit");
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText("Steer failed")).toBeTruthy();
      });
    });
  });

  describe("tasks section", () => {
    it("shows tasks section", async () => {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Tasks")).toBeTruthy();
      });
    });

    it("shows empty message when no project todos", async () => {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("No project todos")).toBeTruthy();
      });
    });

    it("shows empty message when no agent todos", async () => {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("No active agent todos")).toBeTruthy();
      });
    });

    it("displays project todos", async () => {
      mockWorkspaces = [createMockWorkspace({
        projectTodos: [
          { id: "1", content: "Fix bug", status: "in_progress", priority: "high" },
          { id: "2", content: "Add feature", status: "pending", priority: "medium" },
        ],
      })];

      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Fix bug")).toBeTruthy();
        expect(screen.getByText("Add feature")).toBeTruthy();
      });
    });

    it("displays agent todos", async () => {
      mockWorkspaces = [createMockWorkspace({
        agentTodos: [
          { id: "1", content: "Write tests", status: "completed", priority: "high" },
        ],
      })];

      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Write tests")).toBeTruthy();
      });
    });
  });

  describe("cost section", () => {
    it("displays cost tracking when cost data is present", async () => {
      mockWorkspaces = [createMockWorkspace({
        cost: {
          totalCost: 1.2345,
          totalTokens: { input: 1000, output: 500, reasoning: 0, cacheRead: 200, cacheWrite: 0 },
          byAgent: [],
        },
      })];

      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Cost Tracking")).toBeTruthy();
        expect(screen.getByText("$1.2345")).toBeTruthy();
      });
    });

    it("shows per-agent costs when available", async () => {
      mockWorkspaces = [createMockWorkspace({
        cost: {
          totalCost: 1.5,
          totalTokens: { input: 1000, output: 500 },
          byAgent: [
            { agent: "coordinator", cost: 0.75, steps: [] },
            { agent: "developer", cost: 0.75, steps: [] },
          ],
        },
      })];

      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("By Agent (2 agents)")).toBeTruthy();
      });
    });
  });

  describe("agent sequence", () => {
    it("shows empty state when no agent sequence", async () => {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("No agent sequence yet")).toBeTruthy();
      });
    });

    it("displays agent sequence when available", async () => {
      mockWorkspaces = [createMockWorkspace({
        agentSequence: [
          { agent: "coordinator", model: "opencode/glm-5", elapsedTime: "1m", isCurrent: true },
          { agent: "developer", model: "opencode/glm-5", elapsedTime: "2m", isCurrent: false },
        ],
      })];

      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("coordinator")).toBeTruthy();
        expect(screen.getByText("developer")).toBeTruthy();
      });
    });
  });

  describe("project management section", () => {
    it("does not show PM section when hasProjectMgmt is false", () => {
      renderSessionTab({ hasProjectMgmt: false });
      expect(screen.queryByText("PROJECT_MANAGEMENT.md")).toBeNull();
    });

    it("shows PM section when hasProjectMgmt is true", () => {
      renderSessionTab({ hasProjectMgmt: true, pmContent: "# PM" });
      expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeTruthy();
    });

    it("shows no content message when PM content is empty", () => {
      renderSessionTab({ hasProjectMgmt: true });
      expect(screen.getByText("No content available")).toBeTruthy();
    });
  });
});

describe("ActionBar", () => {
  it("renders nothing when actions array is empty", () => {
    const { container } = render(
      <TooltipProvider>
        <ActionBar actions={[]} isRunning={false} onActionClick={() => {}} />
      </TooltipProvider>
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders action buttons", () => {
    const actions = [
      { name: "Run Tests", model: "model-1", prompt: "run tests", description: "Run test suite" },
    ];

    render(
      <TooltipProvider>
        <ActionBar actions={actions} isRunning={false} onActionClick={() => {}} />
      </TooltipProvider>
    );

    expect(screen.getByText("Run Tests")).toBeTruthy();
  });

  it("disables buttons when running", () => {
    const actions = [
      { name: "Run Tests", model: "model-1", prompt: "run tests", description: "Run test suite" },
    ];

    render(
      <TooltipProvider>
        <ActionBar actions={actions} isRunning={true} onActionClick={() => {}} />
      </TooltipProvider>
    );

    const button = screen.getByText("Run Tests");
    expect(button.closest("button")?.hasAttribute("disabled")).toBe(true);
  });

  it("calls onActionClick when button is clicked", async () => {
    const user = userEvent.setup();
    const onActionClick = mock(() => {});
    const actions = [
      { name: "Run Tests", model: "model-1", prompt: "run tests", description: "Run test suite" },
    ];

    render(
      <TooltipProvider>
        <ActionBar actions={actions} isRunning={false} onActionClick={onActionClick} />
      </TooltipProvider>
    );

    await user.click(screen.getByText("Run Tests"));
    expect(onActionClick).toHaveBeenCalledWith(actions[0]);
  });
});
