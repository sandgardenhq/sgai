import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor, within, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WorkspaceDetail } from "./WorkspaceDetail";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceEntry } from "@/types";

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  window.sessionStorage.clear();
});

afterEach(() => {
  cleanup();
});

function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-project",
    dir: "/home/user/test-project",
    running: true,
    needsInput: false,
    inProgress: true,
    pinned: false,
    isRoot: false,
    isFork: false,
    status: "working",
    badgeClass: "running",
    badgeText: "Running",
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
    humanMessage: "",
    agentSequence: [
      { agent: "coordinator", elapsedTime: "5m", isCurrent: false },
      { agent: "react-developer", elapsedTime: "10m", isCurrent: true },
    ],
    cost: {
      totalCost: 0.05,
      totalTokens: { input: 5000, output: 2000, reasoning: 100, cacheRead: 500, cacheWrite: 0 },
      byAgent: [],
    },
    events: [],
    messages: [],
    projectTodos: [],
    agentTodos: [],
    changes: { description: "", diffLines: [] },
    commits: [],
    log: [],
    forks: [],
    ...overrides,
  };
}

function renderWorkspaceDetail(path = "/workspaces/test-project/progress", workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle") {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
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
  it("renders loading skeleton when fetching with no workspace", () => {
    renderWorkspaceDetail("/workspaces/test-project/progress", [], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders workspace header when data loads from factory state", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });
  });

  it("renders status badge", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("running")).toBeDefined();
    });
  });

  it("renders execution timer", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("45m 30s")).toBeDefined();
    });

    const execBadge = screen.getByLabelText("Total execution time");
    expect(execBadge.getAttribute("tabindex")).toBe("0");
  });

  it("renders status line with agent and model", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.getByText("react-developer | claude-opus-4-6")).toBeDefined();
    expect(screen.getByText("Implementing Dashboard component")).toBeDefined();
  });

  it("does not render latest progress banner", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.queryByText(/Working on React migration M2/)).toBeNull();
  });

  it("renders tab navigation", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    expect(await screen.findByRole("link", { name: "Progress" })).toBeDefined();
    expect(screen.getByRole("link", { name: "Log" })).toBeDefined();
    expect(screen.getByRole("link", { name: "Diffs" })).toBeDefined();
    expect(screen.getByRole("link", { name: "Messages" })).toBeDefined();
  });

  it("highlights active tab based on URL", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    const progressLink = await screen.findByRole("link", { name: "Progress" });
    expect(progressLink.getAttribute("aria-current")).toBe("page");
  });

  it("renders entity browser links (Agents, Skills, Snippets)", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    expect(await screen.findByRole("button", { name: "Agents" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Skills" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Snippets" })).toBeDefined();
  });

  it("renders workspace action buttons", async () => {
    const stoppedWorkspace = makeWorkspace({ running: false });
    renderWorkspaceDetail("/workspaces/test-project/progress", [stoppedWorkspace]);

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
    const workspace = makeWorkspace({ running: true });
    const { container } = renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);
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
    const workspace = makeWorkspace({ running: true });
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    expect(await screen.findByRole("button", { name: "Open in OpenCode" })).toBeDefined();
  });

  it("shows fork, open editor, and pin actions for root workspaces with forks", async () => {
    const rootWithForks = makeWorkspace({
      isRoot: true,
      isFork: false,
      running: false,
      forks: [{ name: "test-project-fork", dir: "/tmp/test-project-fork", running: false, commitAhead: 0, commits: [] }],
    });
    renderWorkspaceDetail("/workspaces/test-project/forks", [rootWithForks]);

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
    const rootRunning = makeWorkspace({
      isRoot: true,
      isFork: false,
      running: true,
      forks: [{ name: "test-project-fork", dir: "/tmp/test-project-fork", running: false, commitAhead: 0, commits: [] }],
    });
    renderWorkspaceDetail("/workspaces/test-project/forks", [rootRunning]);

    expect(await screen.findByRole("button", { name: "Open in OpenCode" })).toBeDefined();
  });

  it("renders no-workspace state when hasSgai is false", async () => {
    const noWorkspace = makeWorkspace({ hasSgai: false, isRoot: false });
    renderWorkspaceDetail("/workspaces/test-project/progress", [noWorkspace]);

    await waitFor(() => {
      expect(screen.getByText(/No workspace configured/i)).toBeDefined();
    });
  });

  it("renders error message when fetchStatus is error", async () => {
    renderWorkspaceDetail("/workspaces/test-project/progress", [], "error");

    await waitFor(() => {
      expect(screen.getByText(/Failed to load workspace/i)).toBeDefined();
    });
  });

  it("does not call individual workspace detail API endpoint", async () => {
    const workspace = makeWorkspace();
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url === "/api/v1/workspaces/test-project")).toBe(false);
  });

  it("does not render action messages after start", async () => {
    const stoppedWorkspace = makeWorkspace({ running: false });
    mockFetch.mockImplementation((input) => {
      const url = String(input);
      if (url.includes("/api/v1/workspaces/test-project/start")) {
        return Promise.resolve(new Response(JSON.stringify({ running: true, message: "Session started" })));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail("/workspaces/test-project/progress", [stoppedWorkspace]);

    const startButton = await screen.findByRole("button", { name: "Start" });
    fireEvent.click(startButton);

    await waitFor(() => {
      expect(screen.queryByText("Session started")).toBeNull();
    });
  });

  it("sends auto=false when Start is clicked", async () => {
    const stoppedWorkspace = makeWorkspace({ running: false, interactiveAuto: true });
    mockFetch.mockImplementation((input) => {
      const url = String(input);
      if (url.includes("/api/v1/workspaces/test-project/start")) {
        return Promise.resolve(new Response(JSON.stringify({ running: true, message: "Session started" })));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail("/workspaces/test-project/progress", [stoppedWorkspace]);

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

  it("sends auto=true when Self-Drive is clicked", async () => {
    const stoppedWorkspace = makeWorkspace({ running: false });
    mockFetch.mockImplementation((input) => {
      const url = String(input);
      if (url.includes("/api/v1/workspaces/test-project/start")) {
        return Promise.resolve(new Response(JSON.stringify({ running: true, message: "Session started" })));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail("/workspaces/test-project/progress", [stoppedWorkspace]);

    const selfDriveButton = await screen.findByRole("button", { name: "Self-drive" });
    fireEvent.click(selfDriveButton);

    await waitFor(() => {
      const startCall = mockFetch.mock.calls.find(
        (call) => String(call[0]).includes("/start"),
      );
      expect(startCall).toBeDefined();
      const body = JSON.parse(String((startCall![1] as RequestInit).body));
      expect(body.auto).toBe(true);
    });
  });

  it("hides respond button when there is no pending question", async () => {
    const workspace = makeWorkspace({ needsInput: false });
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    expect(screen.queryByRole("button", { name: "Respond" })).toBeNull();
  });

  it("shows respond button when needsInput is true", async () => {
    const workspace = makeWorkspace({ needsInput: true });
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    const respondButton = await screen.findByRole("button", { name: "Respond" });
    expect(respondButton).toBeDefined();
  });

  it("hides status and timer pills for root workspaces with forks", async () => {
    const rootWithForks = makeWorkspace({
      isRoot: true,
      isFork: false,
      running: false,
      forks: [{ name: "test-project-fork", dir: "/tmp/test-project-fork", running: false, commitAhead: 0, commits: [] }],
    });

    const { container } = renderWorkspaceDetail("/workspaces/test-project/forks", [rootWithForks]);
    await waitFor(() => {
      expect(screen.getByText("test-project")).toBeDefined();
    });

    const header = container.querySelector("header");
    const headerScope = header ? within(header) : null;
    expect(headerScope?.queryByText("45m 30s") ?? null).toBeNull();
    expect(headerScope?.queryByText("stopped") ?? null).toBeNull();
  });

  it("renders rename link for fork workspaces", async () => {
    const forkWorkspace = makeWorkspace({ isFork: true });
    renderWorkspaceDetail("/workspaces/test-project/progress", [forkWorkspace]);

    await waitFor(() => {
      expect(screen.getByText(/test-project ✏️/)).toBeDefined();
    });
  });

  it("renders stopped badge when not running", async () => {
    const stoppedWorkspace = makeWorkspace({ running: false });
    renderWorkspaceDetail("/workspaces/test-project/progress", [stoppedWorkspace]);

    await waitFor(() => {
      expect(screen.getByText("stopped")).toBeDefined();
    });
  });

  it("renders Continuous Self-Drive button when continuousMode is true and not running", async () => {
    const workspace = makeWorkspace({ running: false, continuousMode: true });
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    expect(await screen.findByRole("button", { name: "Continuous Self-Drive" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Self-drive" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Start" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
  });

  it("renders Continuous Self-Drive and Stop buttons when continuousMode is true and running", async () => {
    const workspace = makeWorkspace({ running: true, continuousMode: true });
    const { container } = renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);
    const view = within(container);

    expect(await view.findByRole("button", { name: "Continuous Self-Drive" })).toBeDefined();
    expect(view.getAllByRole("button", { name: "Stop" }).length).toBeGreaterThan(0);
    expect(view.queryByRole("button", { name: "Self-Drive" })).toBeNull();
    expect(view.queryByRole("button", { name: "Start" })).toBeNull();
  });

  it("renders normal buttons when continuousMode is false and not running", async () => {
    const workspace = makeWorkspace({ running: false, continuousMode: false });
    renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);

    expect(await screen.findByRole("button", { name: "Self-drive" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Start" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Continuous Self-Drive" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
  });

  it("renders normal buttons when continuousMode is false and running", async () => {
    const workspace = makeWorkspace({ running: true, continuousMode: false });
    const { container } = renderWorkspaceDetail("/workspaces/test-project/progress", [workspace]);
    const view = within(container);

    expect(await view.findByRole("button", { name: "Self-Drive" })).toBeDefined();
    expect(view.getAllByRole("button", { name: "Stop" }).length).toBeGreaterThan(0);
    expect(view.queryByRole("button", { name: "Continuous Self-Drive" })).toBeNull();
  });

  it("shows interrupted banner and reset action when status is working but session is stopped", async () => {
    const interrupted = makeWorkspace({ running: false, status: "working" });
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = String(input);
      if (url.includes("/reset")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, status: "idle" })));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderWorkspaceDetail("/workspaces/test-project/progress", [interrupted]);

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
