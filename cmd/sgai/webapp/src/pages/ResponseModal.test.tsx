import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ResponseModal } from "./ResponseModal";
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

const workspaceDetail: ApiWorkspaceDetailResponse = {
  name: "test-workspace",
  dir: "/tmp/test-workspace",
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
  currentAgent: "backend-developer",
  currentModel: "claude-opus-4",
  task: "",
  goalContent: "<p>Goal content for modal</p>",
  pmContent: "<p>PM content for modal</p>",
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

function createFetchHandler() {
  return (url: string | URL | Request) => {
    const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
    if (urlStr.includes("/pending-question")) {
      return Promise.resolve(new Response(JSON.stringify(pendingQuestion)));
    }
    if (urlStr.includes("/api/v1/workspaces/") && !urlStr.includes("/respond")) {
      return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
    }
    return Promise.resolve(new Response("{}"));
  };
}

function renderModal(open = true, onOpenChange = mock(() => {}), onResponseSubmitted = mock(() => {})) {
  mockFetch.mockImplementation(createFetchHandler());

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

  it("renders agent badge", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getByText(/backend-developer/)).toBeDefined();
    });
  });

  it("renders choices", async () => {
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

  it("does not fetch when closed", () => {
    renderModal(false);
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("fetches pending question when opened", async () => {
    renderModal(true);

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

  it("submits response and calls onResponseSubmitted", async () => {
    const onOpenChange = mock(() => {});
    const onResponseSubmitted = mock(() => {});

    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/pending-question")) {
        return Promise.resolve(new Response(JSON.stringify(pendingQuestion)));
      }
      if (urlStr.includes("/respond")) {
        return Promise.resolve(
          new Response(JSON.stringify({ success: true, message: "ok" })),
        );
      }
      if (urlStr.includes("/api/v1/workspaces/")) {
        return Promise.resolve(new Response(JSON.stringify(workspaceDetail)));
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

    const radio = document.querySelector('#modal-choice-0-0') as HTMLInputElement;
    fireEvent.click(radio);

    const submitBtns = screen.getAllByText("Send Response");
    fireEvent.click(submitBtns[0]);

    await waitFor(() => {
      expect(onResponseSubmitted).toHaveBeenCalled();
    });
  });

  it("disables buttons during submission (R-18)", async () => {
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

  it("persists modal response state to sessionStorage (R-8)", async () => {
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

  it("shows error on 409 conflict (R-21)", async () => {
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

  it("renders loading skeleton while fetching", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));

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

  it("renders error state on fetch failure", async () => {
    mockFetch.mockImplementation(() =>
      Promise.reject(new Error("Connection refused")),
    );

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
      expect(screen.getByText(/Failed to load question/i)).toBeDefined();
    });
  });

  it("always renders DialogDescription for accessibility", async () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));

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

  it("renders ResponseContext with workspace data in modal (feature parity)", async () => {
    renderModal();

    await waitFor(() => {
      expect(screen.getAllByText("PostgreSQL").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("GOAL.md")).toBeDefined();
    expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
  });
});
