import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { ResponseMultiChoice } from "./ResponseMultiChoice";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiPendingQuestionResponse, ApiWorkspaceDetailResponse } from "@/types";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  closed = false;
  constructor(url: string) { this.url = url; }
  addEventListener() {}
  removeEventListener() {}
  close() { this.closed = true; }
}

const originalEventSource = globalThis.EventSource;
const mockFetch = mock(() => Promise.resolve(new Response("{}")));

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

const workspaceDetail: ApiWorkspaceDetailResponse = {
  name: "test-project",
  dir: "/tmp/test-project",
  running: false,
  needsInput: true,
  status: "waiting",
  badgeClass: "",
  badgeText: "waiting",
  isRoot: true,
  isFork: false,
  pinned: false,
  hasSgai: true,
  hasEditedGoal: false,
  interactiveAuto: false,
  currentAgent: "coordinator",
  currentModel: "claude-opus-4",
  task: "",
  goalContent: "<p>Test goal content</p>",
  pmContent: "<p>Test PM content</p>",
  hasProjectMgmt: true,
  svgHash: "",
  totalExecTime: "",
  latestProgress: "",
  agentSequence: [],
  cost: { totalCost: 0, inputTokens: 0, outputTokens: 0, cacheCreationInputTokens: 0, cacheReadInputTokens: 0 },
};

beforeEach(() => {
  cleanup();
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  (globalThis as unknown as Record<string, unknown>).EventSource = MockEventSource;
  sessionStorage.clear();
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
  sessionStorage.clear();
});

function createFetchHandler(question: ApiPendingQuestionResponse | null = pendingQuestion) {
  return (url: string | URL | Request) => {
    const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
    if (urlStr.includes("/pending-question")) {
      if (question) {
        return Promise.resolve(new Response(JSON.stringify(question)));
      }
      return Promise.resolve(new Response(null, { status: 204 }));
    }
    if (urlStr.includes("/api/v1/workspaces/") && !urlStr.includes("/respond")) {
      return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
    }
    return Promise.resolve(new Response("{}"));
  };
}

function renderResponse(question: ApiPendingQuestionResponse | null = pendingQuestion) {
  mockFetch.mockImplementation(createFetchHandler(question));

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
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
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

  it("renders question and choices when loaded", async () => {
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
    renderResponse(multiSelectQuestion);

    await waitFor(() => {
      expect(screen.getAllByText("Dark mode").length).toBeGreaterThan(0);
    });

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes.length).toBeGreaterThanOrEqual(3);
  });

  it("renders multiple question blocks", async () => {
    renderResponse(multipleQuestions);

    await waitFor(() => {
      expect(screen.getAllByText("First question?").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("Second question?").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Question 1 of 2").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Question 2 of 2").length).toBeGreaterThan(0);
  });

  it("renders markdown message as formatted content", async () => {
    renderResponse(markdownMessageQuestion);

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
    renderResponse(multiSelectQuestion);

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

  it("submits response successfully", async () => {
    let submitCalled = false;
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/pending-question")) {
        return Promise.resolve(new Response(JSON.stringify(pendingQuestion)));
      }
      if (urlStr.includes("/respond")) {
        submitCalled = true;
        return Promise.resolve(
          new Response(JSON.stringify({ success: true, message: "response submitted" })),
        );
      }
      if (urlStr.includes("/api/v1/workspaces/")) {
        return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
      }
      return Promise.resolve(new Response("{}"));
    });

    render(
      <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
            <Route path="/workspaces/:name/progress" element={<div>Progress Page</div>} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

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
      if (urlStr.includes("/pending-question")) {
        return Promise.resolve(new Response(JSON.stringify(pendingQuestion)));
      }
      if (urlStr.includes("/respond")) {
        return Promise.resolve(new Response("question expired", { status: 409 }));
      }
      if (urlStr.includes("/api/v1/workspaces/")) {
        return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
      }
      return Promise.resolve(new Response("{}"));
    });

    render(
      <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
            <Route path="/workspaces/:name/progress" element={<div>Progress Page</div>} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

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

  it("disables submit button during submission (R-18)", async () => {
    let resolveSubmit: (() => void) | null = null;
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/pending-question")) {
        return Promise.resolve(new Response(JSON.stringify(pendingQuestion)));
      }
      if (urlStr.includes("/respond")) {
        return new Promise<Response>((resolve) => {
          resolveSubmit = () => resolve(new Response(JSON.stringify({ success: true, message: "ok" })));
        });
      }
      if (urlStr.includes("/api/v1/workspaces/")) {
        return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
      }
      return Promise.resolve(new Response("{}"));
    });

    render(
      <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
            <Route path="/workspaces/:name/progress" element={<div>Progress Page</div>} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

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

  it("persists response state to sessionStorage (R-8)", async () => {
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

  it("renders error when API fails", async () => {
    mockFetch.mockImplementation(() =>
      Promise.reject(new Error("Network error")),
    );

    render(
      <MemoryRouter initialEntries={["/workspaces/test-project/respond"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/workspaces/:name/respond" element={<ResponseMultiChoice />} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText(/Failed to load question/i)).toBeDefined();
    });
  });

  it("calls pending-question API on mount", async () => {
    renderResponse();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled();
    });

    const calls = mockFetch.mock.calls;
    const pendingQuestionCall = calls.find((call) => {
      const urlStr = (call as unknown[])[0] as string;
      return urlStr.includes("/pending-question");
    });
    expect(pendingQuestionCall).toBeDefined();
  });

  it("renders back link", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getByText("← Back")).toBeDefined();
    });
  });

  it("shows workspace required for legacy workspace query param", async () => {
    mockFetch.mockImplementation(createFetchHandler());

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

    await waitFor(() => {
      expect(screen.getByText("Workspace required")).toBeDefined();
    });
  });

  it("shows workspace required for legacy dir query param", async () => {
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.endsWith("/api/v1/workspaces")) {
        return Promise.resolve(new Response(JSON.stringify({
          workspaces: [{ name: "test-project", dir: "/tmp/test-project", running: false, needsInput: true, inProgress: true, pinned: false, isRoot: true, status: "waiting", hasSgai: true }],
        })));
      }
      if (urlStr.includes("/pending-question")) {
        return Promise.resolve(new Response(JSON.stringify(pendingQuestion)));
      }
      if (urlStr.includes("/api/v1/workspaces/") && !urlStr.includes("/respond")) {
        return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
      }
      return Promise.resolve(new Response("{}"));
    });

    render(
      <MemoryRouter initialEntries={["/respond?dir=%2Ftmp%2Ftest-project"]}>
        <TooltipProvider>
          <Routes>
            <Route path="/respond" element={<ResponseMultiChoice />} />
            <Route path="/workspaces/:name/progress" element={<div>Progress Page</div>} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText("Workspace required")).toBeDefined();
    });
  });

  it("renders ResponseContext with workspace data (feature parity)", async () => {
    renderResponse();

    await waitFor(() => {
      expect(screen.getAllByText("Response Required").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("GOAL.md")).toBeDefined();
    expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
  });
});
