import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SessionTab, ActionBar } from "../SessionTab";

beforeEach(() => {
  document.body.style.pointerEvents = "auto";
});

const mockOpenEditorPM = mock(() => Promise.resolve({ opened: true }));

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
  task: "",
  goalContent: "",
  rawGoalContent: "",
  pmContent: "",
  hasProjectMgmt: false,
  totalExecTime: "",
  latestProgress: "",
  humanMessage: "",
  events: [],
  projectTodos: [],
  agentTodos: [],
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
}));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
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
    mockOpenEditorPM.mockClear();
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
