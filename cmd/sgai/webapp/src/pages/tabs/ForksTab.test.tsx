import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ForksTab } from "./ForksTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceEntry, ApiModelsResponse } from "@/types";

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  cleanup();
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
});

const modelsResponse: ApiModelsResponse = {
  models: [
    { id: "anthropic/claude-sonnet-4-20250514", name: "Claude Sonnet" },
    { id: "anthropic/claude-opus-4-20250514", name: "Claude Opus" },
  ],
  defaultModel: "anthropic/claude-sonnet-4-20250514",
};

const sampleForks = [
  {
    name: "project-alpha-fork1",
    dir: "/home/user/project-alpha-fork1",
    running: true,
    needsInput: false,
    inProgress: true,
    pinned: false,
    commitAhead: 3,
    commits: [
      { changeId: "abc12345", commitId: "def67890", timestamp: "2026-02-08 10:00", bookmarks: ["main"], description: "Initial commit" },
      { changeId: "xyz99999", commitId: "uvw11111", timestamp: "2026-02-08 11:00", bookmarks: [], description: "Add feature" },
    ],
  },
  {
    name: "project-alpha-fork2",
    dir: "/home/user/project-alpha-fork2",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    commitAhead: 0,
    commits: [],
  },
];

function makeWorkspace(overrides: Partial<ApiWorkspaceEntry> = {}): ApiWorkspaceEntry {
  return {
    name: "test-project",
    dir: "/home/user/test-project",
    running: false,
    needsInput: false,
    inProgress: false,
    pinned: false,
    isRoot: true,
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
    forks: sampleForks,
    ...overrides,
  };
}

function makeForkWorkspace(name: string, needsInput: boolean): ApiWorkspaceEntry {
  return {
    name,
    dir: `/home/user/${name}`,
    running: needsInput,
    needsInput,
    inProgress: needsInput,
    pinned: false,
    isRoot: false,
    isFork: true,
    status: needsInput ? "Needs Input" : "stopped",
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
  };
}

function setupModelsApiMock() {
  mockFetch.mockImplementation((input: string | URL | Request) => {
    const url = String(input);
    if (url.includes("/api/v1/models")) {
      return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
    }
    if (url.includes("/adhoc")) {
      return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
    }
    return Promise.resolve(new Response("{}"));
  });
}

function renderForksTab(workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle") {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <ForksTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("ForksTab", () => {
  it("renders loading skeleton when fetching with no workspace", () => {
    renderForksTab([], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders null when workspace not found and idle", () => {
    const { container } = renderForksTab([], "idle");
    expect(container.firstChild).toBeNull();
  });

  it("renders error state when fetch fails and workspace not found", () => {
    renderForksTab([], "error");
    expect(screen.getByText(/Failed to load forks/i)).toBeDefined();
  });

  it("renders forks from factory state", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    expect(screen.getByText("project-alpha-fork1")).toBeDefined();
    expect(screen.getByText("project-alpha-fork2")).toBeDefined();
  });

  it("renders commit details for forks with commits after expanding", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    const expandButtons = screen.getAllByRole("button", { name: "Expand commits" });
    const enabledExpandButton = expandButtons.find((btn) => !btn.hasAttribute("disabled"));
    expect(enabledExpandButton).toBeDefined();
    fireEvent.click(enabledExpandButton!);

    expect(screen.getByText("Initial commit")).toBeDefined();
    expect(screen.getByText("Add feature")).toBeDefined();
  });

  it("renders respond button enabled for fork that needs input", () => {
    setupModelsApiMock();
    const rootWorkspace = makeWorkspace();
    const fork1WithInput = makeForkWorkspace("project-alpha-fork1", true);
    const fork2WithoutInput = makeForkWorkspace("project-alpha-fork2", false);
    renderForksTab([rootWorkspace, fork1WithInput, fork2WithoutInput]);

    const respondButtons = screen.getAllByRole("button", { name: "Respond" });
    expect(respondButtons.length).toBeGreaterThan(0);
    const enabledButtons = respondButtons.filter((btn) => !btn.hasAttribute("disabled"));
    const disabledButtons = respondButtons.filter((btn) => btn.hasAttribute("disabled"));
    expect(enabledButtons.length).toBeGreaterThan(0);
    expect(disabledButtons.length).toBeGreaterThan(0);
  });

  it("renders empty state when no forks", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace({ forks: [] });
    renderForksTab([workspace]);

    expect(screen.getByText(/No forks yet/)).toBeDefined();
  });

  it("does not call individual forks API endpoint", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/api/v1/workspaces/test-project/forks"))).toBe(false);
  });

  it("renders run box with model selector", async () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    await waitFor(() => {
      expect(screen.getByLabelText("Model")).toBeDefined();
    });

    expect(screen.getByText("Ad-hoc Prompt")).toBeDefined();
    expect(screen.getByLabelText("Prompt")).toBeDefined();
  });

  it("renders submit button in run box", async () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
    });
  });

  it("renders run box below empty forks state", async () => {
    setupModelsApiMock();
    const workspace = makeWorkspace({ forks: [] });
    renderForksTab([workspace]);

    expect(screen.getByText(/No forks yet/)).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText("Ad-hoc Prompt")).toBeDefined();
    });
  });

  it("populates model selector with available models", async () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    await waitFor(() => {
      expect(screen.getByText("Claude Sonnet")).toBeDefined();
    });

    expect(screen.getByText("Claude Opus")).toBeDefined();
  });

  it("renders icon action buttons for each fork row", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    const editorButtons = screen.getAllByRole("button", { name: "Open in Editor" });
    expect(editorButtons.length).toBe(2);

    const openSgaiButtons = screen.getAllByRole("button", { name: "Open in sgai" });
    expect(openSgaiButtons.length).toBe(2);

    const deleteButtons = screen.getAllByRole("button", { name: "Delete fork" });
    expect(deleteButtons.length).toBe(2);
  });

  it("renders sgai.json action buttons on each fork row", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    const actions = [
      { name: "Create PR", model: "some-model", prompt: "create pr", description: "Create a pull request" },
      { name: "Sync", model: "some-model", prompt: "sync", description: "Sync with upstream" },
    ];

    mock.module("@/lib/factory-state", () => ({
      useFactoryState: () => ({ workspaces: [workspace], fetchStatus: "idle", lastFetchedAt: Date.now() }),
      resetFactoryStateStore: () => {},
    }));

    render(
      <MemoryRouter>
        <TooltipProvider>
          <ForksTab workspaceName="test-project" actions={actions} onActionClick={() => {}} />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const createPRButtons = screen.getAllByRole("button", { name: "Create PR" });
    expect(createPRButtons.length).toBe(2);

    const syncButtons = screen.getAllByRole("button", { name: "Sync" });
    expect(syncButtons.length).toBe(2);
  });

  it("chevron expand button is disabled when fork has no commits", () => {
    setupModelsApiMock();
    const workspace = makeWorkspace();
    renderForksTab([workspace]);

    const expandButtons = screen.getAllByRole("button", { name: /expand commits/i });
    const disabledExpand = expandButtons.filter((btn) => btn.hasAttribute("disabled"));
    expect(disabledExpand.length).toBeGreaterThan(0);
  });
});
