import { cleanup, render, screen } from "@testing-library/react";
import { describe, it, expect, afterEach, mock } from "bun:test";
import { MemoryRouter } from "react-router";
import { CommitsTab } from "./CommitsTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceEntry } from "@/types";

const mockFetch = mock(() => Promise.resolve(new Response("{}")));
globalThis.fetch = mockFetch as unknown as typeof fetch;

afterEach(() => {
  cleanup();
  mockFetch.mockReset();
});

function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-project",
    dir: "/home/user/test-project",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    isRoot: false,
    isFork: false,
    status: "stopped",
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
    totalExecTime: "0s",
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
    ...overrides,
  };
}

const sampleCommits = [
  {
    changeId: "nyyqzwxs",
    commitId: "ad8009f9",
    timestamp: "2026-02-08 10:00",
    description: "Initial commit",
    graphChar: "@",
    workspaces: ["main"],
    bookmarks: ["main"],
  },
];

function renderCommitsTab(workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle") {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <CommitsTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("CommitsTab", () => {
  it("renders loading skeleton when fetching with no workspace", () => {
    renderCommitsTab([], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders null when workspace not found and idle", () => {
    const { container } = renderCommitsTab([], "idle");
    expect(container.firstChild).toBeNull();
  });

  it("renders error state when fetch fails and workspace not found", () => {
    renderCommitsTab([], "error");
    expect(screen.getByText(/Failed to load commits/i)).toBeDefined();
  });

  it("renders commit entries from factory state", () => {
    const workspace = makeWorkspace({ commits: sampleCommits });
    renderCommitsTab([workspace]);

    expect(screen.getByText("nyyqzwxs")).toBeDefined();
    expect(screen.getByText("ad8009f9")).toBeDefined();
    expect(screen.getByText("Initial commit")).toBeDefined();
  });

  it("renders empty state when no commits", () => {
    const workspace = makeWorkspace({ commits: [] });
    renderCommitsTab([workspace]);

    expect(screen.getByText(/No commits found/i)).toBeDefined();
  });

  it("renders table headers", () => {
    const workspace = makeWorkspace({ commits: sampleCommits });
    renderCommitsTab([workspace]);

    expect(screen.getByText("Change ID")).toBeDefined();
    expect(screen.getByText("Time")).toBeDefined();
    expect(screen.getByText("Description")).toBeDefined();
  });

  it("does not call individual commits API endpoint", () => {
    const workspace = makeWorkspace({ commits: sampleCommits });
    renderCommitsTab([workspace]);

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/api/v1/workspaces/test-project/commits"))).toBe(false);
  });
});
