import { describe, it, expect, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { LogTab } from "./LogTab";
import type { ApiLogEntry, ApiWorkspaceEntry } from "@/types";

afterEach(() => {
  cleanup();
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

const sampleLogLines: ApiLogEntry[] = [
  { prefix: "[10:00] ", text: "Starting build process..." },
  { prefix: "[10:01] ", text: "Build completed successfully" },
  { prefix: "", text: "All tests passed" },
];

function renderLogTab(workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle") {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
  return render(
    <MemoryRouter>
      <LogTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("LogTab", () => {
  it("renders loading skeleton when fetching with no workspace", () => {
    renderLogTab([], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders log lines from factory state", async () => {
    const workspace = makeWorkspace({ log: sampleLogLines });
    renderLogTab([workspace]);

    await waitFor(() => {
      expect(screen.getByText("Starting build process...")).toBeDefined();
    });

    expect(screen.getByText("Build completed successfully")).toBeDefined();
    expect(screen.getByText("All tests passed")).toBeDefined();
  });

  it("renders empty state when no logs", async () => {
    const workspace = makeWorkspace({ log: [] });
    renderLogTab([workspace]);

    await waitFor(() => {
      expect(screen.getByText("No logs available")).toBeDefined();
    });
  });

  it("renders error state when fetchStatus is error and no workspace", async () => {
    renderLogTab([], "error");

    await waitFor(() => {
      expect(screen.getByText(/Failed to load log/i)).toBeDefined();
    });
  });

  it("renders null when workspace not found and idle", () => {
    const { container } = renderLogTab([], "idle");
    expect(container.firstChild).toBeNull();
  });

  it("renders prefix spans with muted styling", async () => {
    const workspace = makeWorkspace({ log: sampleLogLines });
    renderLogTab([workspace]);

    await waitFor(() => {
      expect(screen.getByText("Starting build process...")).toBeDefined();
    });

    const prefixSpans = document.querySelectorAll(".text-muted-foreground.select-none");
    expect(prefixSpans.length).toBeGreaterThan(0);
    expect(prefixSpans[0].textContent).toContain("[10:00]");
  });
});
