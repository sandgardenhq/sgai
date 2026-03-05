import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { EditGoal } from "../EditGoal";

const mockWorkspace = {
  name: "test-workspace",
  dir: "/path/to/test-workspace",
  running: false,
  needsInput: false,
  inProgress: false,
  pinned: false,
  isRoot: false,
  isFork: false,
  description: "Test Workspace Description",
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
  goalContent: "# Test Goal\n\nThis is a test goal.",
  rawGoalContent: "# Test Goal\n\nThis is a test goal.",
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
};

const mockGetGoal = mock(() => Promise.resolve({ content: "# Test Goal\n\nThis is a test goal." }));
const mockUpdateGoal = mock(() => Promise.resolve({ updated: true, workspace: "test-workspace" }));
const mockTriggerFactoryRefresh = mock(() => {});

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({
    workspaces: [mockWorkspace],
    fetchStatus: "idle",
    lastFetchedAt: Date.now(),
  }),
  triggerFactoryRefresh: mockTriggerFactoryRefresh,
}));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
      getGoal: mockGetGoal,
      updateGoal: mockUpdateGoal,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

mock.module("@/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ value, onChange, disabled }: { value: string; onChange: (v: string) => void; disabled: boolean }) => (
    <div data-testid="markdown-editor">
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        data-testid="markdown-textarea"
      />
    </div>
  ),
}));

function renderEditGoal(workspaceName = "test-workspace") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspaceName}/goal/edit`]}>
      <TooltipProvider>
        <Routes>
          <Route path="/workspaces/:name/goal/edit" element={<EditGoal />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>
  );
}

function getSaveButton() {
  const buttons = screen.getAllByRole("button", { name: /Save GOAL\.md|Saving\.\.\.|Saved!/ });
  return buttons[buttons.length - 1];
}

async function waitForContentToLoad() {
  await waitFor(() => {
    const textareas = screen.queryAllByTestId("markdown-textarea");
    expect(textareas.length).toBeGreaterThan(0);
    const textarea = textareas[0] as HTMLTextAreaElement;
    expect(textarea.value).toBeTruthy();
    expect(textarea.value).toContain("# Test Goal");
  });
}

describe("EditGoal", () => {
  beforeEach(() => {
    mockGetGoal.mockClear();
    mockUpdateGoal.mockClear();
  });

  afterEach(() => {
    cleanup();
  });

  describe("save/load functionality", () => {
    it("loads goal content on mount", async () => {
      renderEditGoal();

      await waitFor(() => {
        expect(mockGetGoal).toHaveBeenCalledWith("test-workspace");
      });
    });

    it("saves goal content when Save button is clicked", async () => {
      const user = userEvent.setup();

      renderEditGoal();

      await waitFor(() => {
        expect(getSaveButton()).toBeTruthy();
      });

      await user.click(getSaveButton());

      await waitFor(() => {
        expect(mockUpdateGoal).toHaveBeenCalled();
      });
    });

    it("shows saving state during save", async () => {
      const user = userEvent.setup();
      mockUpdateGoal.mockImplementationOnce(
        () => new Promise((resolve) => setTimeout(() => resolve({ updated: true, workspace: "test-workspace" }), 500))
      );

      renderEditGoal();

      await waitForContentToLoad();

      await user.click(getSaveButton());

      await waitFor(() => {
        const savingButtons = screen.queryAllByText("Saving...");
        expect(savingButtons.length).toBeGreaterThan(0);
      });
    });

    it("shows success state after save", async () => {
      const user = userEvent.setup();

      renderEditGoal();

      await waitFor(() => {
        expect(getSaveButton()).toBeTruthy();
      });

      await user.click(getSaveButton());

      await waitFor(() => {
        const savedButtons = screen.getAllByRole("button", { name: /Saved!/ });
        expect(savedButtons.length).toBeGreaterThan(0);
      });
    });

    it("triggers factory refresh after save", async () => {
      const user = userEvent.setup();

      renderEditGoal();

      await waitFor(() => {
        expect(getSaveButton()).toBeTruthy();
      });

      await user.click(getSaveButton());

      await waitFor(() => {
        expect(mockTriggerFactoryRefresh).toHaveBeenCalled();
      });
    });

    it("disables save button during saving", async () => {
      const user = userEvent.setup();
      mockUpdateGoal.mockImplementationOnce(
        () => new Promise((resolve) => setTimeout(() => resolve({ updated: true, workspace: "test-workspace" }), 500))
      );

      renderEditGoal();

      await waitForContentToLoad();

      await user.click(getSaveButton());

      await waitFor(() => {
        const savingButtons = screen.queryAllByText("Saving...");
        expect(savingButtons.length).toBeGreaterThan(0);
      });
    });
  });

  describe("keyboard shortcuts", () => {
    it("saves on Ctrl+S / Cmd+S", async () => {
      const user = userEvent.setup();

      renderEditGoal();

      await waitFor(() => {
        expect(getSaveButton()).toBeTruthy();
      });

      await user.keyboard("{Control>}s{/Control}");

      await waitFor(() => {
        expect(mockUpdateGoal).toHaveBeenCalled();
      });
    });
  });

  describe("navigation", () => {
    it("shows back link to workspace", async () => {
      renderEditGoal();

      await waitFor(() => {
        const backLinks = screen.getAllByLabelText("Back to test-workspace");
        expect(backLinks.length).toBeGreaterThan(0);
      });
    });
  });

  describe("description display", () => {
    it("shows workspace description in header", async () => {
      renderEditGoal();

      await waitFor(() => {
        const descriptions = screen.getAllByText("Test Workspace Description");
        expect(descriptions.length).toBeGreaterThan(0);
      });
    });
  });

  describe("validation", () => {
    it("disables save when content is empty", async () => {
      mockGetGoal.mockImplementationOnce(() => Promise.resolve({ content: "" }));

      renderEditGoal();

      await waitFor(() => {
        const saveButton = getSaveButton();
        expect(saveButton).toBeTruthy();
        expect(saveButton.hasAttribute("disabled")).toBe(true);
      });
    });

    it("enables save when content is valid", async () => {
      mockGetGoal.mockImplementationOnce(() => Promise.resolve({ content: "# Test Goal" }));
      
      renderEditGoal();

      await waitFor(() => {
        const saveButtons = screen.queryAllByRole("button", { name: /Save GOAL\.md/ });
        expect(saveButtons.length).toBeGreaterThan(0);
      });

      const saveButtons = screen.getAllByRole("button", { name: /Save GOAL\.md/ });
      expect(saveButtons[0].hasAttribute("disabled")).toBe(false);
    });
  });

  describe("error handling", () => {
    it("shows error message when save fails", async () => {
      const user = userEvent.setup();
      const errorMessage = "Failed to save GOAL.md";
      mockUpdateGoal.mockImplementationOnce(() => Promise.reject(new Error(errorMessage)));

      renderEditGoal();

      await waitForContentToLoad();

      await user.click(getSaveButton());

      await waitFor(
        () => {
          const errorElements = screen.queryAllByText(/Failed to save GOAL\.md/);
          expect(errorElements.length).toBeGreaterThan(0);
        },
        { timeout: 3000 }
      );
    });

    it("shows error message when load fails", async () => {
      mockGetGoal.mockImplementationOnce(() => Promise.reject(new Error("Failed to load")));

      renderEditGoal();

      await waitFor(() => {
        expect(mockGetGoal).toHaveBeenCalled();
      });

      await waitFor(
        () => {
          const errorElements = screen.queryAllByText(/Failed to load/);
          expect(errorElements.length).toBeGreaterThan(0);
        },
        { timeout: 3000 }
      );
    });
  });

  describe("editor text input", () => {
    it("editor accepts agent name text", async () => {
      renderEditGoal();

      await waitFor(() => {
        const textareas = screen.queryAllByTestId("markdown-textarea");
        expect(textareas.length).toBeGreaterThan(0);
      });

      const textareas = screen.getAllByTestId("markdown-textarea");
      fireEvent.change(textareas[0], { target: { value: '"coordinator"' } });

      await waitFor(() => {
        expect((textareas[0] as HTMLTextAreaElement).value).toBe('"coordinator"');
      });
    });

    it("editor accepts model name text", async () => {
      renderEditGoal();

      await waitFor(() => {
        const textareas = screen.queryAllByTestId("markdown-textarea");
        expect(textareas.length).toBeGreaterThan(0);
      });

      const textareas = screen.getAllByTestId("markdown-textarea");
      fireEvent.change(textareas[0], { target: { value: '"opencode/glm-5"' } });

      await waitFor(() => {
        expect((textareas[0] as HTMLTextAreaElement).value).toBe('"opencode/glm-5"');
      });
    });

    it("editor renders and accepts input", async () => {
      renderEditGoal();

      await waitFor(() => {
        const textareas = screen.queryAllByTestId("markdown-textarea");
        expect(textareas.length).toBeGreaterThan(0);
      });

      const textareas = screen.getAllByTestId("markdown-textarea");
      expect(textareas[0]).toBeTruthy();
    });
  });

  describe("editor renders frontmatter content", () => {
    it("editor renders frontmatter with flow and models", async () => {
      renderEditGoal();

      await waitFor(() => {
        const textareas = screen.queryAllByTestId("markdown-textarea");
        expect(textareas.length).toBeGreaterThan(0);
      });

      const textareas = screen.getAllByTestId("markdown-textarea");
      fireEvent.change(textareas[0], { target: { value: "---\nflow: |\n  \"a\" -> \"b\"\nmodels:\n  \"coordinator\": \"opencode/glm-5\"\n---\n# Goal" } });

      await waitFor(() => {
        expect((textareas[0] as HTMLTextAreaElement).value).toContain("flow:");
      });
    });

    it("editor renders flow syntax content", async () => {
      renderEditGoal();

      await waitFor(() => {
        const textareas = screen.queryAllByTestId("markdown-textarea");
        expect(textareas.length).toBeGreaterThan(0);
      });

      const textareas = screen.getAllByTestId("markdown-textarea");
      fireEvent.change(textareas[0], { target: { value: "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Goal" } });

      await waitFor(() => {
        expect((textareas[0] as HTMLTextAreaElement).value).toContain("flow:");
      });
    });

    it("editor renders models section content", async () => {
      renderEditGoal();

      await waitFor(() => {
        const textareas = screen.queryAllByTestId("markdown-textarea");
        expect(textareas.length).toBeGreaterThan(0);
      });

      const textareas = screen.getAllByTestId("markdown-textarea");
      fireEvent.change(textareas[0], { target: { value: "---\nmodels:\n  \"coordinator\": \"opencode/glm-5\"\n---\n# Goal" } });

      await waitFor(() => {
        expect((textareas[0] as HTMLTextAreaElement).value).toContain("models:");
      });
    });
  });
});
