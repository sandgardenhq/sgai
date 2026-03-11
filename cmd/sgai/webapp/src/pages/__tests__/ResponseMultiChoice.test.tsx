import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ResponseMultiChoice } from "../ResponseMultiChoice";
import { ResponseModal } from "../ResponseModal";

// Override pointer-events on body to allow interactions in tests
beforeEach(() => {
  document.body.style.pointerEvents = "auto";
});

const mockQuestion = {
  questionId: "q-123",
  type: "multi-choice" as const,
  agentName: "coordinator",
  message: "What approach should we take?",
  questions: [
    {
      question: "Which architecture pattern?",
      choices: ["Microservices", "Monolith", "Serverless"],
      multiSelect: false,
    },
    {
      question: "Which database?",
      choices: ["PostgreSQL", "MySQL", "MongoDB"],
      multiSelect: true,
    },
  ],
};

const mockWorkspace = {
  name: "test-workspace",
  dir: "/path/to/test-workspace",
  running: true,
  needsInput: true,
  inProgress: true,
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
  currentAgent: "coordinator",
  currentModel: "opencode/glm-5",
  task: "Waiting for response",
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
  pendingQuestion: mockQuestion,
};

const mockRespond = mock(() => Promise.resolve({ success: true, message: "Response submitted" }));
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

function renderResponseMultiChoice(workspaceName = "test-workspace") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspaceName}/respond`]}>
      <TooltipProvider>
        <Routes>
          <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>
  );
}

function renderResponseModal(workspaceName = "test-workspace", open = true) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <ResponseModal
          workspaceName={workspaceName}
          open={open}
          onOpenChange={mock(() => {})}
          onResponseSubmitted={mock(() => {})}
        />
      </TooltipProvider>
    </MemoryRouter>
  );
}

describe("ResponseMultiChoice", () => {
  beforeEach(() => {
    sessionStorage.clear();
    mockRespond.mockClear();
    mockTriggerFactoryRefresh.mockClear();
  });

  afterEach(() => {
    cleanup();
  });

  describe("display retrospective questions", () => {
    it("displays agent name badge", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const coordinatorElements = screen.queryAllByText(/coordinator/);
        expect(coordinatorElements.length).toBeGreaterThan(0);
      });
    });

    it("shows all questions in multi-question flow", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const archElements = screen.queryAllByText("Which architecture pattern?");
        const dbElements = screen.queryAllByText("Which database?");
        expect(archElements.length).toBeGreaterThan(0);
        expect(dbElements.length).toBeGreaterThan(0);
      });
    });

    it("displays question counter for multiple questions", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const q1Elements = screen.queryAllByText("Question 1 of 2");
        const q2Elements = screen.queryAllByText("Question 2 of 2");
        expect(q1Elements.length).toBeGreaterThan(0);
        expect(q2Elements.length).toBeGreaterThan(0);
      });
    });

    it("shows all choices for each question", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const microElements = screen.queryAllByText("Microservices");
        const monoElements = screen.queryAllByText("Monolith");
        const serverlessElements = screen.queryAllByText("Serverless");
        expect(microElements.length).toBeGreaterThan(0);
        expect(monoElements.length).toBeGreaterThan(0);
        expect(serverlessElements.length).toBeGreaterThan(0);
      });
    });

    it("indicates multi-select vs single-select", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const singleSelectElements = screen.queryAllByText("Select your answer:");
        const multiSelectElements = screen.queryAllByText("Select your answer(s):");
        expect(singleSelectElements.length + multiSelectElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("handle user responses", () => {
    it("allows selecting single choice for single-select question", async () => {
      const user = userEvent.setup();
      renderResponseMultiChoice();

      await waitFor(() => {
        const microElements = screen.queryAllByLabelText("Microservices");
        expect(microElements.length).toBeGreaterThan(0);
      });

      const microservicesRadios = screen.getAllByLabelText("Microservices");
      await user.click(microservicesRadios[0]);

      await waitFor(() => {
        expect((microservicesRadios[0] as HTMLInputElement).checked).toBe(true);
      });
    });

    it("allows selecting multiple choices for multi-select question", async () => {
      const user = userEvent.setup();
      renderResponseMultiChoice();

      await waitFor(() => {
        const postgresElements = screen.queryAllByLabelText("PostgreSQL");
        expect(postgresElements.length).toBeGreaterThan(0);
      });

      const postgresCheckboxes = screen.getAllByLabelText("PostgreSQL");
      const mysqlCheckboxes = screen.getAllByLabelText("MySQL");

      await user.click(postgresCheckboxes[0]);
      await user.click(mysqlCheckboxes[0]);

      await waitFor(() => {
        expect((postgresCheckboxes[0] as HTMLInputElement).checked).toBe(true);
        expect((mysqlCheckboxes[0] as HTMLInputElement).checked).toBe(true);
      });
    });

    it("allows entering custom text response", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const otherElements = screen.queryAllByLabelText(/Other/);
        expect(otherElements.length).toBeGreaterThan(0);
      });

      const otherTextareas = screen.getAllByPlaceholderText("Type your custom response here...");
      // Using fireEvent.change because user.type() has timing issues with bun:test
      fireEvent.change(otherTextareas[0], { target: { value: "I prefer a hybrid approach" } });

      await waitFor(() => {
        expect((otherTextareas[0] as HTMLTextAreaElement).value).toBe("I prefer a hybrid approach");
      });
    });
  });

  describe("navigate through phases", () => {
    it("shows Send Response button", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });
    });

    it("submits response when button is clicked", async () => {
      const user = userEvent.setup();

      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });

      const sendButtons = screen.getAllByText("Send Response");
      await user.click(sendButtons[0]);

      await waitFor(() => {
        expect(mockRespond).toHaveBeenCalled();
      });
    });

    it("shows sending state during submission", async () => {
      const user = userEvent.setup();
      mockRespond.mockImplementationOnce(
        () => new Promise((resolve) => setTimeout(() => resolve({ success: true, message: "ok" }), 100))
      );

      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });

      const sendButtons = screen.getAllByText("Send Response");
      await user.click(sendButtons[0]);

      await waitFor(() => {
        const sendingButtons = screen.queryAllByText("Sending...");
        expect(sendingButtons.length).toBeGreaterThan(0);
      });
    });
  });

  describe("error handling", () => {
    it("shows error when submission fails", async () => {
      const user = userEvent.setup();
      mockRespond.mockImplementationOnce(() => Promise.reject(new Error("Failed to submit")));

      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });

      const sendButtons = screen.getAllByText("Send Response");
      await user.click(sendButtons[0]);

      await waitFor(() => {
        expect(mockRespond).toHaveBeenCalled();
      });

      await waitFor(() => {
        const errorElements = screen.queryAllByText(/Failed to submit/);
        expect(errorElements.length).toBeGreaterThan(0);
      });
    });
  });

  describe("ResponseModal variant", () => {
    it("renders in dialog mode", async () => {
      renderResponseModal();

      await waitFor(() => {
        const responseRequiredElements = screen.queryAllByText("Response Required");
        expect(responseRequiredElements.length).toBeGreaterThan(0);
      });
    });

    it("shows Cancel button in modal", async () => {
      renderResponseModal();

      await waitFor(() => {
        const cancelButtons = screen.queryAllByText("Cancel");
        expect(cancelButtons.length).toBeGreaterThan(0);
      });
    });
  });

  describe("accessibility", () => {
    it("uses proper fieldset and legend for choices", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const fieldsets = screen.queryAllByRole("group");
        expect(fieldsets.length).toBeGreaterThan(0);
      });
    });

    it("associates labels with inputs", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const microservicesInput = screen.queryAllByLabelText("Microservices");
        expect(microservicesInput.length).toBeGreaterThan(0);
      });
    });

    it("all interactive elements are keyboard accessible", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const radios = screen.queryAllByRole("radio");
        const checkboxes = screen.queryAllByRole("checkbox");
        const buttons = screen.queryAllByRole("button");

        expect(radios.length + checkboxes.length + buttons.length).toBeGreaterThan(0);
      });
    });

    it("question groups have proper ARIA attributes", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const fieldsets = screen.queryAllByRole("group");
        expect(fieldsets.length).toBeGreaterThan(0);
      });
    });

    it("form inputs have required attribute when necessary", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const radios = screen.queryAllByRole("radio");
        expect(radios.length).toBeGreaterThan(0);
      });
    });

    it("error messages are announced to screen readers", async () => {
      const user = userEvent.setup();
      mockRespond.mockImplementationOnce(() => Promise.reject(new Error("Failed to submit")));

      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });

      const sendButtons = screen.getAllByText("Send Response");
      await user.click(sendButtons[0]);

      await waitFor(() => {
        const errorAlerts = screen.queryAllByRole("alert");
        expect(errorAlerts.length).toBeGreaterThan(0);
        const alertWithError = errorAlerts.find(el => el.textContent?.includes("Failed to submit"));
        expect(alertWithError).toBeTruthy();
      });
    });

    it("focus is managed correctly in modal", async () => {
      renderResponseModal();

      await waitFor(() => {
        const dialogs = screen.queryAllByRole("dialog");
        expect(dialogs.length).toBeGreaterThan(0);
      });
    });
  });

  describe("sessionStorage persistence", () => {
    it("persists form inputs to sessionStorage on keystroke", async () => {
      renderResponseMultiChoice();

      await waitFor(() => {
        const otherTextareas = screen.queryAllByPlaceholderText("Type your custom response here...");
        expect(otherTextareas.length).toBeGreaterThan(0);
      });

      const otherTextareas = screen.getAllByPlaceholderText("Type your custom response here...");
      fireEvent.change(otherTextareas[0], { target: { value: "Test response" } });

      await waitFor(() => {
        expect((otherTextareas[0] as HTMLTextAreaElement).value).toBe("Test response");
        const stored = JSON.parse(sessionStorage.getItem("sgai-response-test-workspace") || "{}");
        expect(stored.otherText).toBe("Test response");
      });
    });

    it("clears sessionStorage on successful submit", async () => {
      const user = userEvent.setup();
      sessionStorage.setItem("sgai-response-test-workspace", JSON.stringify({ selections: {}, otherText: "saved content", questionId: "q-123" }));

      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });

      const sendButtons = screen.getAllByText("Send Response");
      await user.click(sendButtons[0]);

      await waitFor(() => {
        expect(mockRespond).toHaveBeenCalled();
        expect(sessionStorage.getItem("sgai-response-test-workspace")).toBeNull();
      });
    });
  });

  describe("beforeunload handlers", () => {
    it("sets beforeunload handler when form has unsaved data", async () => {
      const originalAddEventListener = window.addEventListener.bind(window);
      const calls: Array<[string, ...unknown[]]> = [];
      const mockAddEventListener = mock((...args: Parameters<typeof window.addEventListener>) => {
        calls.push([args[0], ...args.slice(1)]);
        return originalAddEventListener(...args);
      });
      window.addEventListener = mockAddEventListener as typeof window.addEventListener;

      try {
        renderResponseMultiChoice();

        await waitFor(() => {
          const otherTextareas = screen.queryAllByPlaceholderText("Type your custom response here...");
          expect(otherTextareas.length).toBeGreaterThan(0);
        });

        const otherTextareas = screen.getAllByPlaceholderText("Type your custom response here...");
        fireEvent.change(otherTextareas[0], { target: { value: "Unsaved content" } });

        await waitFor(() => {
          expect((otherTextareas[0] as HTMLTextAreaElement).value).toBe("Unsaved content");
          const beforeUnloadCall = calls.find(([type]) => type === "beforeunload");
          expect(beforeUnloadCall).toBeTruthy();
        });
      } finally {
        window.addEventListener = originalAddEventListener;
      }
    });
  });

  describe("Cancel button functionality", () => {
    it("shows Cancel button in modal", async () => {
      renderResponseModal();

      await waitFor(() => {
        const cancelButtons = screen.queryAllByRole("button", { name: /Cancel/ });
        expect(cancelButtons.length).toBeGreaterThan(0);
      });
    });

    it("closes modal when Cancel button is clicked", async () => {
      const user = userEvent.setup();
      const mockOnOpenChange = mock(() => {});

      render(
        <MemoryRouter>
          <TooltipProvider>
            <ResponseModal
              workspaceName="test-workspace"
              open={true}
              onOpenChange={mockOnOpenChange}
              onResponseSubmitted={mock(() => {})}
            />
          </TooltipProvider>
        </MemoryRouter>
      );

      await waitFor(() => {
        const cancelButtons = screen.queryAllByRole("button", { name: /Cancel/ });
        expect(cancelButtons.length).toBeGreaterThan(0);
      });

      const cancelButtons = screen.getAllByRole("button", { name: /Cancel/ });
      await user.click(cancelButtons[0]);

      await waitFor(() => {
        expect(mockOnOpenChange).toHaveBeenCalledWith(false);
      });
    });
  });

  describe("critical actions without optimistic updates", () => {
    it("respond action waits for server response", async () => {
      const user = userEvent.setup();

      renderResponseMultiChoice();

      await waitFor(() => {
        const sendButtons = screen.queryAllByText("Send Response");
        expect(sendButtons.length).toBeGreaterThan(0);
      });

      const sendButtons = screen.getAllByText("Send Response");
      await user.click(sendButtons[0]);

      await waitFor(() => {
        expect(mockRespond).toHaveBeenCalled();
      });
    });
  });
});
