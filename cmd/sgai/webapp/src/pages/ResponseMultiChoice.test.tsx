import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { ResponseMultiChoice } from "./ResponseMultiChoice";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiPendingQuestionResponse, ApiWorkspaceEntry } from "@/types";

const pendingQuestion: ApiPendingQuestionResponse = {
  questionId: "abc123def456",
  type: "multi-choice",
  agentName: "coordinator",
  message: "",
  questions: [
    {
      question: "Which approach do you prefer?",
      choices: ["Option A", "Option B", "Option C"],
      multiSelect: false,
    },
  ],
};

const multiSelectQuestion: ApiPendingQuestionResponse = {
  questionId: "multi123",
  type: "multi-choice",
  agentName: "react-developer",
  message: "",
  questions: [
    {
      question: "Select features to implement:",
      choices: ["Dark mode", "Internationalization", "Analytics"],
      multiSelect: true,
    },
  ],
};

const multipleQuestions: ApiPendingQuestionResponse = {
  questionId: "multiple456",
  type: "multi-choice",
  agentName: "coordinator",
  message: "",
  questions: [
    {
      question: "First question?",
      choices: ["Yes", "No"],
      multiSelect: false,
    },
    {
      question: "Second question?",
      choices: ["Alpha", "Beta", "Gamma"],
      multiSelect: true,
    },
  ],
};

const markdownMessageQuestion: ApiPendingQuestionResponse = {
  questionId: "md123",
  type: "multi-choice",
  agentName: "coordinator",
  message: "# Markdown Title\n\nSome **bold** text.",
  questions: [],
};

function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-project",
    dir: "/tmp/test-project",
    running: false,
    needsInput: true,
    inProgress: true,
    pinned: false,
    isRoot: true,
    isFork: false,
    status: "waiting",
    badgeClass: "",
    badgeText: "waiting",
    hasSgai: true,
    hasEditedGoal: false,
    interactiveAuto: false,
    continuousMode: false,
    currentAgent: "coordinator",
    currentModel: "claude-opus-4",
    task: "",
    goalContent: "<p>Test goal content</p>",
    rawGoalContent: "# Test Goal",
    pmContent: "<p>Test PM content</p>",
    hasProjectMgmt: true,
    svgHash: "",
    totalExecTime: "",
    latestProgress: "",
    humanMessage: "",
    agentSequence: [],
    cost: { totalCost: 0, totalTokens: { input: 0, output: 0, reasoning: 0, cacheRead: 0, cacheWrite: 0 }, byAgent: [] },
    events: [],
    messages: [],
    projectTodos: [],
    agentTodos: [],
    changes: { description: "", diffLines: [] },
    commits: [],
    log: [],
    summary: "A brief summary of the test project",
    summaryManual: false,
    pendingQuestion: pendingQuestion,
    ...overrides,
  };
}

type MockFactoryState = {
  workspaces: ApiWorkspaceEntry[];
  fetchStatus: "idle" | "fetching" | "error";
};

let mockFactoryState: MockFactoryState = {
  workspaces: [makeWorkspace()],
  fetchStatus: "idle",
};

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => mockFactoryState,
  resetFactoryStateStore: () => {},
}));

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  cleanup();
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  sessionStorage.clear();
  mockFactoryState = {
    workspaces: [makeWorkspace()],
    fetchStatus: "idle",
  };
});

afterEach(() => {
  cleanup();
  sessionStorage.clear();
});

function renderResponse(workspace: ApiWorkspaceEntry = makeWorkspace()) {
  mockFactoryState = { workspaces: [workspace], fetchStatus: "idle" };

  return render(
    <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
      <TooltipProvider>
        <Routes>
          <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
          <Route path="/workspaces/:name/progress" element={<div>Progress Page</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("ResponseMultiChoice", () => {
  it("renders loading skeleton when fetching and no workspace found", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "fetching" };
    render(
      <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );
    expect(screen.getByRole("status", { name: /loading response/i })).toBeDefined();
  });

  it("renders question and choices from factory state", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText(/coordinator/).length).toBeGreaterThan(0);
    expect(screen.getAllByText("Which approach do you prefer?").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Option A").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Option B").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Option C").length).toBeGreaterThan(0);
  });

  it("renders radio buttons for single-select questions", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Option A").length).toBeGreaterThan(0);
    });

    const radios = screen.getAllByRole("radio");
    expect(radios.length).toBeGreaterThanOrEqual(3);
  });

  it("renders checkboxes for multi-select questions", async () => {
    renderResponse(makeWorkspace({ pendingQuestion: multiSelectQuestion }));

    await waitFor(() => {
      expect(screen.getAllByText("Dark mode").length).toBeGreaterThan(0);
    });

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes.length).toBeGreaterThanOrEqual(3);
  });

  it("renders multiple question blocks", async () => {
    renderResponse(makeWorkspace({ pendingQuestion: multipleQuestions }));

    await waitFor(() => {
      expect(screen.getAllByText("First question?").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("Second question?").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Question 1 of 2").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Question 2 of 2").length).toBeGreaterThan(0);
  });

  it("renders markdown message as formatted content", async () => {
    renderResponse(makeWorkspace({ pendingQuestion: markdownMessageQuestion }));

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Markdown Title" })).toBeDefined();
    });
  });

  it("renders other textarea", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    const textareas = screen.getAllByPlaceholderText("Type your custom response here...");
    expect(textareas.length).toBeGreaterThan(0);
  });

  it("renders Send Response button", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Send Response").length).toBeGreaterThan(0);
    });
  });

  it("selects a radio choice on click", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Option A").length).toBeGreaterThan(0);
    });

    const radioA = screen.getByRole("radio", { name: "Option A" }) as HTMLInputElement;
    fireEvent.click(radioA);
    expect(radioA.checked).toBe(true);
  });

  it("toggles checkbox choices", async () => {
    renderResponse(makeWorkspace({ pendingQuestion: multiSelectQuestion }));

    await waitFor(() => {
      expect(screen.getAllByText("Dark mode").length).toBeGreaterThan(0);
    });

    const checkbox0 = screen.getByRole("checkbox", { name: "Dark mode" }) as HTMLInputElement;
    const checkbox1 = screen.getByRole("checkbox", { name: "Internationalization" }) as HTMLInputElement;

    fireEvent.click(checkbox0);
    expect(checkbox0.checked).toBe(true);

    fireEvent.click(checkbox1);
    expect(checkbox1.checked).toBe(true);

    fireEvent.click(checkbox0);
    expect(checkbox0.checked).toBe(false);
  });

  it("submits response successfully via API", async () => {
    let submitCalled = false;
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/respond")) {
        submitCalled = true;
        return Promise.resolve(
          new Response(JSON.stringify({ success: true, message: "response submitted" })),
        );
      }
      return Promise.resolve(new Response("{}"));
    });

    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Option A").length).toBeGreaterThan(0);
    });

    const radioA = screen.getByRole("radio", { name: "Option A" }) as HTMLInputElement;
    fireEvent.click(radioA);

    const submitBtn = screen.getByRole("button", { name: "Send Response" });
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(submitCalled).toBe(true);
    });
  });

  it("shows error on failed submission", async () => {
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/respond")) {
        return Promise.resolve(new Response("question expired", { status: 409 }));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Option A").length).toBeGreaterThan(0);
    });

    const textareas = screen.getAllByPlaceholderText("Type your custom response here...");
    fireEvent.change(textareas[0], { target: { value: "my answer" } });

    const submitBtns = screen.getAllByText("Send Response");
    fireEvent.click(submitBtns[0]);

    await waitFor(() => {
      expect(screen.getAllByText(/question has expired/i).length).toBeGreaterThan(0);
    });
  });

  it("disables submit button during submission", async () => {
    let resolveSubmit: (() => void) | null = null;
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/respond")) {
        return new Promise<Response>((resolve) => {
          resolveSubmit = () => resolve(new Response(JSON.stringify({ success: true, message: "ok" })));
        });
      }
      return Promise.resolve(new Response("{}"));
    });

    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Send Response").length).toBeGreaterThan(0);
    });

    const textareas = screen.getAllByPlaceholderText("Type your custom response here...");
    fireEvent.change(textareas[0], { target: { value: "answer" } });

    const submitBtns = screen.getAllByText("Send Response");
    fireEvent.click(submitBtns[0]);

    await waitFor(() => {
      expect(screen.getAllByText("Sending...").length).toBeGreaterThan(0);
    });

    const sendingBtns = screen.getAllByText("Sending...") as HTMLButtonElement[];
    expect((sendingBtns[0] as HTMLButtonElement).disabled).toBe(true);

    resolveSubmit?.();
  });

  it("persists response state to sessionStorage", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Option A").length).toBeGreaterThan(0);
    });

    const textareas = screen.getAllByPlaceholderText("Type your custom response here...");
    fireEvent.change(textareas[0], { target: { value: "some notes" } });

    const stored = sessionStorage.getItem("sgai-response-test-project");
    expect(stored).not.toBeNull();
    const parsed = JSON.parse(stored!);
    expect(parsed.otherText).toBe("some notes");
    expect(parsed.questionId).toBe("abc123def456");
  });

  it("renders error state when fetchStatus is error and workspace not found", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "error" };
    render(
      <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.getByText(/Failed to load question/i)).toBeDefined();
  });

  it("does not call individual pending-question or workspace detail API endpoints", () => {
    renderResponse();

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/pending-question"))).toBe(false);
    expect(calledUrls.some((url) => url.match(/\/api\/v1\/workspaces\/[^/]+$/))).toBe(false);
  });

  it("renders back link", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getByText("← Back")).toBeDefined();
    });
  });

  it("shows workspace required when no workspace name in path", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "idle" };
    render(
      <MemoryRouter initialEntries={["/respond?workspace=test-project"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/respond" element={<ResponseMultiChoice />} />
            <Route path="/workspaces/:name/progress" element={<div>Progress Page</div>} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.getByText("Workspace required")).toBeDefined();
  });

  it("renders ResponseContext with workspace goal and PM content", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("GOAL.md")).toBeDefined();
    expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
  });

  it("displays workspace name in the card header", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    expect(screen.getByTestId("workspace-name")).toBeDefined();
    expect(screen.getByTestId("workspace-name").textContent).toBe("test-project");
  });

  it("displays workspace summary when present", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    expect(screen.getByTestId("workspace-summary")).toBeDefined();
    expect(screen.getByTestId("workspace-summary").textContent).toBe(
      "A brief summary of the test project",
    );
  });

  it("hides workspace summary when not present", async () => {
    renderResponse(makeWorkspace({ summary: undefined }));

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    expect(screen.getByTestId("workspace-name")).toBeDefined();
    expect(screen.queryByTestId("workspace-summary")).toBeNull();
  });
});
