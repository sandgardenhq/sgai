import { describe, it, expect, afterEach, mock } from "bun:test";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ChangesTab } from "./ChangesTab";
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

const statData = {
  description: "Add new feature",
  diffLines: [
    { lineNumber: 1, text: "cmd/sgai/serve.go     | 10 +++++-----", class: "" },
    { lineNumber: 2, text: "cmd/sgai/serve_api.go | 25 ++++++++++++++++---------", class: "" },
    { lineNumber: 3, text: "3 files changed, 20 insertions(+), 15 deletions(-)", class: "" },
  ],
};

function renderChangesTab(workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle") {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
  return render(
    <MemoryRouter>
      <ChangesTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("ChangesTab", () => {
  it("renders loading skeleton when fetching with no workspace", () => {
    renderChangesTab([], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders null when workspace not found and idle", () => {
    const { container } = renderChangesTab([], "idle");
    expect(container.firstChild).toBeNull();
  });

  it("renders error state when fetch fails and workspace not found", () => {
    renderChangesTab([], "error");
    expect(screen.getByText(/Failed to load changes/i)).toBeDefined();
  });

  it("renders stat lines from factory state", () => {
    const workspace = makeWorkspace({ changes: statData });
    renderChangesTab([workspace]);

    expect(screen.getByText("Diff Stat")).toBeDefined();
    expect(screen.getByText((content) => content.includes("serve.go") && content.includes("+++++"))).toBeDefined();
  });

  it("renders View Full Diff button", () => {
    const workspace = makeWorkspace({ changes: statData });
    renderChangesTab([workspace]);

    expect(screen.getByRole("button", { name: "View Full Diff" })).toBeDefined();
  });

  it("renders stat lines with line number attributes", () => {
    const workspace = makeWorkspace({ changes: statData });
    const { container } = renderChangesTab([workspace]);

    const line = container.querySelector("[data-line-number='1']");
    expect(line).not.toBeNull();
    expect(line?.textContent).toContain("serve.go");
  });

  it("renders empty diff state when no diff lines", () => {
    const workspace = makeWorkspace({ changes: { description: "", diffLines: [] } });
    renderChangesTab([workspace]);

    expect(screen.getByText("No changes to display")).toBeDefined();
  });

  it("does not call individual changes API endpoint", () => {
    const workspace = makeWorkspace({ changes: statData });
    renderChangesTab([workspace]);

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/api/v1/workspaces/test-project/changes"))).toBe(false);
  });
});
