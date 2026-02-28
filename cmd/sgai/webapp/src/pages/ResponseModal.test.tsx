import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import React from "react";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ResponseModal } from "./ResponseModal";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiPendingQuestionResponse, ApiWorkspaceEntry } from "@/types";

mock.module("@monaco-editor/react", () => ({
  default: () => null,
}));

mock.module("@/components/MarkdownEditor", () => ({
  MarkdownEditor: (props: { value: string; onChange: (v: string | undefined) => void; disabled?: boolean; placeholder?: string }) =>
    React.createElement("textarea", {
      value: props.value,
      onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => props.onChange(e.target.value),
      disabled: props.disabled,
      placeholder: props.placeholder ?? "Type your custom response here...",
      "data-testid": "markdown-editor",
    }),
}));

const pendingQuestion: ApiPendingQuestionResponse = {
  questionId: "modal-q-123",
  type: "multi-choice",
  agentName: "backend-developer",
  message: "",
  questions: [
    {
      question: "Choose a database:",
      choices: ["PostgreSQL", "MySQL", "SQLite"],
      multiSelect: false,
    },
  ],
};

function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-workspace",
    dir: "/tmp/test-workspace",
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
    currentAgent: "backend-developer",
    currentModel: "claude-opus-4",
    task: "",
    goalContent: "<p>Goal content for modal</p>",
    rawGoalContent: "# Goal content",
    pmContent: "<p>PM content for modal</p>",
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

function renderModal(open = true, onOpenChange = mock(() => {}), onResponseSubmitted = mock(() => {})) {
  return {
    onOpenChange,
    onResponseSubmitted,
    ...render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={open}
            onOpenChange={onOpenChange}
            onResponseSubmitted={onResponseSubmitted}
          />
        </TooltipProvider>
      </MemoryRouter>,
    ),
  };
}

describe("ResponseModal", () => {
  it("renders dialog title when open", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText("Response Required")).toBeDefined();
    });
  });

  it("renders agent badge from factory state", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText(/backend-developer/)).toBeDefined();
    });
  });

  it("renders choices from factory state", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("MySQL").length).toBeGreaterThan(0);
    expect(screen.getAllByText("SQLite").length).toBeGreaterThan(0);
  });

  it("renders Cancel and Send buttons", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getAllByText("Cancel").length).toBeGreaterThan(0);
      expect(screen.getAllByText("Send Response").length).toBeGreaterThan(0);
    });
  });

  it("does not call pending-question API when open (uses factory state)", () => {
    renderModal(true);
    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/pending-question"))).toBe(false);
  });

  it("submits response and calls onResponseSubmitted", async () => {
    const onOpenChange = mock(() => {});
    const onResponseSubmitted = mock(() => {});

    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/respond")) {
        return Promise.resolve(
          new Response(JSON.stringify({ success: true, message: "ok" })),
        );
      }
      return Promise.resolve(new Response("{}"));
    });

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={true}
            onOpenChange={onOpenChange}
            onResponseSubmitted={onResponseSubmitted}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
    });

    const firstRadio = screen.getAllByRole("radio")[0] as HTMLInputElement;
    fireEvent.click(firstRadio);

    const submitBtns = screen.getAllByText("Send Response");
    fireEvent.click(submitBtns[0]);

    await waitFor(() => {
      expect(onResponseSubmitted).toHaveBeenCalled();
    });
  });

  it("disables buttons during submission", async () => {
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

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={true}
            onOpenChange={mock(() => {})}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
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

    const cancelBtns = screen.getAllByText("Cancel") as HTMLButtonElement[];
    expect((cancelBtns[0] as HTMLButtonElement).disabled).toBe(true);

    resolveSubmit?.();
  });

  it("persists modal response state to sessionStorage", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
    });

    const textareas = screen.getAllByPlaceholderText("Type your custom response here...");
    fireEvent.change(textareas[0], { target: { value: "modal notes" } });

    const stored = sessionStorage.getItem("sgai-response-modal-test-workspace");
    expect(stored).not.toBeNull();
    const parsed = JSON.parse(stored!);
    expect(parsed.otherText).toBe("modal notes");
    expect(parsed.questionId).toBe("modal-q-123");
  });

  it("shows error on 409 conflict", async () => {
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/respond")) {
        return Promise.resolve(new Response("question expired", { status: 409 }));
      }
      return Promise.resolve(new Response("{}"));
    });

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={true}
            onOpenChange={mock(() => {})}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
    });

    const textareas = screen.getAllByPlaceholderText("Type your custom response here...");
    fireEvent.change(textareas[0], { target: { value: "my answer" } });

    const submitBtns = screen.getAllByText("Send Response");
    fireEvent.click(submitBtns[0]);

    await waitFor(() => {
      expect(screen.getAllByText(/question has expired/i).length).toBeGreaterThan(0);
    });
  });

  it("renders loading skeleton when fetching and workspace not found", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "fetching" };

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={true}
            onOpenChange={mock(() => {})}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders error state when fetchStatus is error and no workspace", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "error" };

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={true}
            onOpenChange={mock(() => {})}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.getByText(/Failed to load question/i)).toBeDefined();
  });

  it("always renders DialogDescription for accessibility when loading", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "fetching" };

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ResponseModal
            workspaceName="test-workspace"
            open={true}
            onOpenChange={mock(() => {})}
          />
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(screen.getByText("Loading agent question...")).toBeDefined();
  });

  it("renders ResponseContext with workspace data in modal", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("GOAL.md")).toBeDefined();
    expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
  });
});
