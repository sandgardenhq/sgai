import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor, within, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WorkspaceDetail } from "./WorkspaceDetail";
import { resetDefaultSSEStore, resetAllWorkspaceSSEStores } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceDetailResponse } from "@/types";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  closed = false;
  listeners: Map<string, ((event: MessageEvent) => void)[]> = new Map();
  constructor(url: string) { this.url = url; }
  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    const existing = this.listeners.get(type) ?? [];
    existing.push(listener);
    this.listeners.set(type, existing);
  }
  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    const existing = this.listeners.get(type) ?? [];
    this.listeners.set(type, existing.filter((item) => item !== listener));
  }
  close() { this.closed = true; }
  simulateEvent(type: string, data: string) {
    const event = new MessageEvent(type, { data });
    const listeners = this.listeners.get(type) ?? [];
    for (const listener of listeners) {
      listener(event);
    }
  }
}

const originalEventSource = globalThis.EventSource;
const mockFetch = mock(() => Promise.resolve(new Response("{}")));
let mockEventSources: MockEventSource[] = [];

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  window.sessionStorage.clear();
  mockEventSources = [];
  (globalThis as unknown as { EventSource: typeof MockEventSource }).EventSource =
    class extends MockEventSource {
      constructor(url: string) {
        super(url);
        mockEventSources.push(this);
      }
    } as unknown as typeof EventSource;
  resetDefaultSSEStore();
  resetAllWorkspaceSSEStores();
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  resetAllWorkspaceSSEStores();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
});

const workspaceDetail: ApiWorkspaceDetailResponse = {
  name: "test-project",
  dir: "/home/user/test-project",
  running: true,
  needsInput: false,
  status: "working",
  badgeClass: "running",
  badgeText: "Running",
  isRoot: false,
  isFork: false,
  pinned: false,
  hasSgai: true,
  hasEditedGoal: true,
  interactiveAuto: false,
  continuousMode: false,
  currentAgent: "react-developer",
  currentModel: "anthropic/claude-opus-4-6",
  task: "Implementing Dashboard component",
  goalContent: "<h1>Project Goal</h1>",
  rawGoalContent: "# Project Goal",
  fullGoalContent: "---\n---\n\n# Project Goal",
  pmContent: "<p>PM Content</p>",
  hasProjectMgmt: true,
  svgHash: "abc123",
  totalExecTime: "45m 30s",
  latestProgress: "Working on React migration M2",
  agentSequence: [
    { agent: "coordinator", elapsedTime: "5m", isCurrent: false },
    { agent: "react-developer", elapsedTime: "10m", isCurrent: true },
  ],
  cost: {
    totalCost: 0.05,
    inputTokens: 5000,
    outputTokens: 2000,
    cacheCreationInputTokens: 1000,
    cacheReadInputTokens: 500,
  },
  forks: [],
};

function renderWorkspaceDetail(path = "/workspaces/test-project/progress") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <Routes>
          <Route path="/workspaces/:name/*" element={<WorkspaceDetail />} />
          <Route path="/workspaces/:name" element={<WorkspaceDetail />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WorkspaceDetail", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(
      () => new Promise(() => {}),
    );
    renderWorkspaceDetail();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders workspace header when data loads", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });
  });

  it("renders status badge", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("running")).toBeDefined();
    });
  });

  it("renders execution timer", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("45m 30s")).toBeDefined();
    });

    const execBadge = screen.getByLabelText("Total execution time");
    expect(execBadge.getAttribute("tabindex")).toBe("0");
  });

  it("renders status line with agent and model", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.getByText("react-developer | claude-opus-4-6")).toBeDefined();
    expect(screen.getByText("Implementing Dashboard component")).toBeDefined();
  });

  it("does not render latest progress banner", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.queryByText(/Working on React migration M2/)).toBeNull();
  });

  it("renders tab navigation", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("link", { name: "Progress" })).toBeDefined();
    expect(screen.getByRole("link", { name: "Log" })).toBeDefined();
    expect(screen.getByRole("link", { name: "Diffs" })).toBeDefined();
    expect(screen.getByRole("link", { name: "Messages" })).toBeDefined();
  });

  it("highlights active tab based on URL", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail("/workspaces/test-project/progress");

    const progressLink = await screen.findByRole("link", { name: "Progress" });
    expect(progressLink.getAttribute("aria-current")).toBe("page");
  });

  it("renders entity browser links (Agents, Skills, Snippets)", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("button", { name: "Agents" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Skills" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Snippets" })).toBeDefined();
  });

  it("renders workspace action buttons", async () => {
    const stoppedDetail = { ...workspaceDetail, running: false };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(stoppedDetail))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("button", { name: "Self-drive" })).toBeDefined();

    expect(screen.getByRole("button", { name: "Start" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Fork" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Compose GOAL" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Edit GOAL" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Open in Editor" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Pin" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
  });

  it("hides fork and compose actions when running", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    const { container } = renderWorkspaceDetail();
    const view = within(container);

    await waitFor(() => {
      expect(view.getAllByRole("button", { name: "Stop" }).length).toBeGreaterThan(0);
    });

    expect(view.queryByRole("button", { name: "Fork" })).toBeNull();
    expect(view.queryByRole("button", { name: "Compose GOAL" })).toBeNull();
    expect(view.getByRole("button", { name: "Edit GOAL" })).toBeDefined();
    expect(view.queryByRole("button", { name: "Open in Editor" })).toBeNull();
  });

  it("shows open in OpenCode when running", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("button", { name: "Open in OpenCode" })).toBeDefined();
  });

  it("shows fork, open editor, and pin actions for root workspaces with forks", async () => {
    const rootWithForks = {
      ...workspaceDetail,
      isRoot: true,
      isFork: false,
      running: false,
      forks: [{ name: "test-project-fork", dir: "/tmp/test-project-fork", running: false, commitAhead: 0 }],
    };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(rootWithForks))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.getByRole("button", { name: "Fork" })).toBeDefined();
    const openEditorButtons = screen.getAllByRole("button", { name: "Open in Editor" });
    expect(openEditorButtons.length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByRole("button", { name: "Respond" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Self-drive" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Start" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Compose GOAL" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Edit GOAL" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Skills" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Snippets" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Agents" })).toBeNull();
    expect(screen.getByRole("button", { name: "Pin" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Unpin" })).toBeNull();
  });

  it("shows open in OpenCode for running root workspaces with forks", async () => {
    const rootRunning = {
      ...workspaceDetail,
      isRoot: true,
      isFork: false,
      running: true,
      forks: [{ name: "test-project-fork", dir: "/tmp/test-project-fork", running: false, commitAhead: 0 }],
    };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(rootRunning))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("button", { name: "Open in OpenCode" })).toBeDefined();
  });

  it("renders no-workspace state when hasSgai is false", async () => {
    const noWorkspace = { ...workspaceDetail, hasSgai: false, isRoot: false };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(noWorkspace))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText(/No workspace configured/i)).toBeDefined();
    });
  });

  it("renders error message when API fails", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load workspace/i)).toBeDefined();
    });
  });

  it("calls workspace detail API with correct name", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      const detailCalls = mockFetch.mock.calls.filter(
        (call) => (call[0] as string) === "/api/v1/workspaces/test-project",
      );
      expect(detailCalls.length).toBeGreaterThanOrEqual(1);
    });

    const detailCall = mockFetch.mock.calls.find(
      (call) => (call[0] as string) === "/api/v1/workspaces/test-project",
    );
    const calledUrl = (detailCall as unknown[])[0] as string;
    expect(calledUrl).toBe("/api/v1/workspaces/test-project");
  });

  it("refreshes detail on workspace SSE and global SSE events", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      const detailCalls = mockFetch.mock.calls.filter(
        (call) => (call[0] as string) === "/api/v1/workspaces/test-project",
      );
      expect(detailCalls.length).toBe(1);
      expect(mockEventSources.length).toBeGreaterThan(0);
    });

    const globalSource = mockEventSources.find((s) => s.url === "/api/v1/events/stream");
    const workspaceSource = mockEventSources.find((s) => s.url.includes("/workspaces/test-project/events/stream"));
    expect(globalSource).toBeDefined();
    expect(workspaceSource).toBeDefined();

    globalSource!.simulateEvent("workspace:update", JSON.stringify({ workspace: "test-project" }));
    await waitFor(() => {
      const detailCalls = mockFetch.mock.calls.filter(
        (call) => (call[0] as string) === "/api/v1/workspaces/test-project",
      );
      expect(detailCalls.length).toBe(2);
    });
  });

  it("does not render action messages after start", async () => {
    const stoppedDetail = { ...workspaceDetail, running: false };
    mockFetch.mockImplementation((input) => {
      const url = String(input);
      if (url.includes("/api/v1/workspaces/test-project/start")) {
        return Promise.resolve(new Response(JSON.stringify({ running: true, message: "Session started" })));
      }
      if (url.includes("/api/v1/workspaces/test-project")) {
        return Promise.resolve(new Response(JSON.stringify(stoppedDetail)));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail();

    const startButton = await screen.findByRole("button", { name: "Start" });
    fireEvent.click(startButton);

    await waitFor(() => {
      expect(screen.queryByText("Session started")).toBeNull();
    });
  });

  it("sends auto=false when Start is clicked even if interactiveAuto is true", async () => {
    const stoppedAutoDetail = { ...workspaceDetail, running: false, interactiveAuto: true };
    mockFetch.mockImplementation((input, init) => {
      const url = String(input);
      if (url.includes("/api/v1/workspaces/test-project/start")) {
        return Promise.resolve(new Response(JSON.stringify({ running: true, message: "Session started" })));
      }
      if (url.includes("/api/v1/workspaces/test-project")) {
        return Promise.resolve(new Response(JSON.stringify(stoppedAutoDetail)));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail();

    const startButton = await screen.findByRole("button", { name: "Start" });
    fireEvent.click(startButton);

    await waitFor(() => {
      const startCall = mockFetch.mock.calls.find(
        (call) => String(call[0]).includes("/start"),
      );
      expect(startCall).toBeDefined();
      const body = JSON.parse(String((startCall![1] as RequestInit).body));
      expect(body.auto).toBe(false);
    });
  });

  it("hides respond button when there is no pending question", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );

    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.queryByRole("button", { name: "Respond" })).toBeNull();
  });

  it("shows respond button when a pending question exists", async () => {
    const detailWithInput = { ...workspaceDetail, needsInput: true };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(detailWithInput))),
    );

    renderWorkspaceDetail();

    const respondButton = await screen.findByRole("button", { name: "Respond" });
    expect(respondButton).toBeDefined();
  });

  it("hides status and timer pills for root workspaces with forks", async () => {
    const rootWithForks = {
      ...workspaceDetail,
      isRoot: true,
      isFork: false,
      running: false,
      forks: [{ name: "test-project-fork", dir: "/tmp/test-project-fork", running: false, commitAhead: 0 }],
    };
    mockFetch.mockImplementation((url: string | URL | Request) => {
      if (typeof url === "string" && url.includes("/api/v1/models")) {
        return Promise.resolve(new Response(JSON.stringify({
          models: [{ id: "test-model", name: "Test Model" }],
          defaultModel: "test-model",
        })));
      }
      return Promise.resolve(new Response(JSON.stringify(rootWithForks)));
    });

    const { container } = renderWorkspaceDetail();
    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    const header = container.querySelector("header");
    const headerScope = header ? within(header) : null;
    expect(headerScope?.queryByText("45m 30s") ?? null).toBeNull();
    expect(headerScope?.queryByText("stopped") ?? null).toBeNull();
  });

  it("polls for workspace detail while running", async () => {
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(workspaceDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      const detailCalls = mockFetch.mock.calls.filter(
        (call) => (call[0] as string) === "/api/v1/workspaces/test-project",
      );
      expect(detailCalls.length).toBe(1);
    });

    await new Promise((r) => setTimeout(r, 3500));

    await waitFor(() => {
      const detailCalls = mockFetch.mock.calls.filter(
        (call) => (call[0] as string) === "/api/v1/workspaces/test-project",
      );
      expect(detailCalls.length).toBeGreaterThan(1);
    });
  });

  it("renders rename link for fork workspaces", async () => {
    const forkDetail = { ...workspaceDetail, isFork: true };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(forkDetail))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText(/test-project ✏️/)).toBeDefined();
    });
  });

  it("renders stopped badge when not running", async () => {
    const stopped = { ...workspaceDetail, running: false };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(stopped))),
    );
    renderWorkspaceDetail();

    await waitFor(() => {
      expect(screen.getByText("stopped")).toBeDefined();
    });
  });

  it("renders Continuous Self-Drive button when continuousMode is true and not running", async () => {
    const continuousDetail = { ...workspaceDetail, running: false, continuousMode: true };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(continuousDetail))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("button", { name: "Continuous Self-Drive" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Self-drive" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Start" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
  });

  it("renders Continuous Self-Drive and Stop buttons when continuousMode is true and running", async () => {
    const continuousRunning = { ...workspaceDetail, running: true, continuousMode: true };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(continuousRunning))),
    );
    const { container } = renderWorkspaceDetail();
    const view = within(container);

    expect(await view.findByRole("button", { name: "Continuous Self-Drive" })).toBeDefined();
    expect(view.getAllByRole("button", { name: "Stop" }).length).toBeGreaterThan(0);
    expect(view.queryByRole("button", { name: "Self-Drive" })).toBeNull();
    expect(view.queryByRole("button", { name: "Start" })).toBeNull();
  });

  it("renders normal buttons when continuousMode is false and not running", async () => {
    const normalDetail = { ...workspaceDetail, running: false, continuousMode: false };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(normalDetail))),
    );
    renderWorkspaceDetail();

    expect(await screen.findByRole("button", { name: "Self-drive" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Start" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Continuous Self-Drive" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
  });

  it("renders normal buttons when continuousMode is false and running", async () => {
    const normalRunning = { ...workspaceDetail, running: true, continuousMode: false };
    mockFetch.mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify(normalRunning))),
    );
    const { container } = renderWorkspaceDetail();
    const view = within(container);

    expect(await view.findByRole("button", { name: "Self-Drive" })).toBeDefined();
    expect(view.getAllByRole("button", { name: "Stop" }).length).toBeGreaterThan(0);
    expect(view.queryByRole("button", { name: "Continuous Self-Drive" })).toBeNull();
  });

  it("shows interrupted banner and reset action when status is working but session is stopped", async () => {
    const interrupted = { ...workspaceDetail, running: false, status: "working" };
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = String(input);
      if (url.includes("/reset")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, status: "idle" })));
      }
      if (url.includes("/api/v1/workspaces/test-project")) {
        return Promise.resolve(new Response(JSON.stringify(interrupted)));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail("/workspaces/test-project/progress");

    await waitFor(() => {
      expect(screen.getByText("sgai was interrupted while working. Reset state to start fresh.")).toBeDefined();
    });

    const resetButton = screen.getByRole("button", { name: "Reset" });
    fireEvent.click(resetButton);

    await waitFor(() => {
      const resetCalls = mockFetch.mock.calls.filter((call) => String(call[0]).includes("/reset"));
      expect(resetCalls.length).toBe(1);
    });
  });
});
