import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { SessionTab } from "./SessionTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiActionEntry, ApiWorkspaceEntry } from "@/types";

const mockWorkspace: ApiWorkspaceEntry = {
  name: "test-project",
  dir: "/projects/test-project",
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
  hasEditedGoal: false,
  interactiveAuto: false,
  continuousMode: false,
  currentAgent: "backend-developer",
  currentModel: "anthropic/claude-opus-4-6",
  task: "Writing tests",
  goalContent: "# Goal",
  rawGoalContent: "# Goal",
  pmContent: "",
  hasProjectMgmt: false,
  svgHash: "abc123",
  totalExecTime: "5m30s",
  latestProgress: "Tests complete",
  humanMessage: "",
  agentSequence: [
    { agent: "coordinator", elapsedTime: "30s", isCurrent: false },
    { agent: "backend-developer", elapsedTime: "5m", isCurrent: true },
  ],
  cost: {
    totalCost: 1.2345,
    totalTokens: { input: 5000, output: 2000, reasoning: 100, cacheRead: 3000, cacheWrite: 0 },
    byAgent: [
      {
        agent: "backend-developer",
        cost: 0.8,
        tokens: { input: 3000, output: 1500, reasoning: 50, cacheRead: 2000, cacheWrite: 0 },
        steps: [
          { stepId: "step-1", agent: "backend-developer", cost: 0.4, tokens: { input: 1500, output: 750, reasoning: 25, cacheRead: 1000, cacheWrite: 0 }, timestamp: "2026-02-08T10:00:00Z" },
        ],
      },
    ],
  },
  events: [],
  messages: [],
  projectTodos: [
    { id: "1", content: "Review internals", status: "pending", priority: "high" },
  ],
  agentTodos: [
    { id: "2", content: "Fix Run tab", status: "in_progress", priority: "medium" },
  ],
  changes: { description: "", diffLines: [] },
  commits: [],
  log: [],
};

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({ workspaces: [mockWorkspace], fetchStatus: "idle", lastFetchedAt: Date.now() }),
  resetFactoryStateStore: () => {},
}));

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
});

afterEach(() => {
  cleanup();
});

const testActions: ApiActionEntry[] = [
  { name: "Create PR", model: "anthropic/claude-opus-4-6 (max)", prompt: "using GH make a PR", description: "Create a draft pull request" },
  { name: "Run Tests", model: "anthropic/claude-opus-4-6", prompt: "run all tests", description: "Run the test suite" },
];

function renderSessionTab(extraProps?: { pmContent?: string; hasProjectMgmt?: boolean; actions?: ApiActionEntry[] }) {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <SessionTab workspaceName="test-project" {...extraProps} />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("SessionTab", () => {
  it("renders cost tracking from factory state", async () => {
    renderSessionTab();

    await waitFor(() => {
      expect(screen.getByText("Cost Tracking")).toBeDefined();
    });

    expect(screen.getByText("$1.2345")).toBeDefined();
  });

  it("renders agent sequence from factory state", async () => {
    renderSessionTab();

    await waitFor(() => {
      expect(screen.getByText("Agent Sequence")).toBeDefined();
    });

    expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    expect(screen.getAllByText("backend-developer").length).toBeGreaterThan(0);
  });

  it("renders steer next turn form", async () => {
    renderSessionTab();

    await waitFor(() => {
      expect(screen.getAllByText("Steer Next Turn").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByPlaceholderText(/re-steering instruction/i).length).toBeGreaterThan(0);
    expect(screen.getAllByRole("button", { name: "Submit" }).length).toBeGreaterThan(0);
  });

  it("renders tasks section with project and agent todos from factory state", async () => {
    renderSessionTab();

    await waitFor(() => {
      expect(screen.getAllByText("Project TODO").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("Agent TODO").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Review internals").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Fix Run tab").length).toBeGreaterThan(0);
  });

  it("renders project management section when content is available", async () => {
    renderSessionTab({ hasProjectMgmt: true, pmContent: "# PM Content" });

    await waitFor(() => {
      expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
    });

    const pmLabel = screen.getByText("PROJECT_MANAGEMENT.md");
    const details = pmLabel.closest("details");
    expect(details).not.toBeNull();
    expect(details?.hasAttribute("open")).toBe(false);
    const summaryEl = details?.querySelector("summary");
    const chevron = summaryEl?.querySelector("svg");
    expect(chevron).toBeTruthy();

    expect(screen.getByText("PM Content")).toBeDefined();
  });

  it("does not warn about duplicate model status keys", async () => {
    const originalError = console.error;
    const errorSpy = mock(() => {});
    console.error = errorSpy as unknown as typeof console.error;

    try {
      renderSessionTab();

      await waitFor(() => {
        expect(screen.getByText("Cost Tracking")).toBeDefined();
      });

      const hasDuplicateKeyWarning = errorSpy.mock.calls.some((call) =>
        String(call[0]).includes("Encountered two children with the same key"),
      );
      expect(hasDuplicateKeyWarning).toBe(false);
    } finally {
      console.error = originalError;
    }
  });

  it("renders action buttons when actions are provided", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = input.toString();
      if (url.includes("/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "" })));
      }
      return Promise.resolve(new Response(JSON.stringify({})));
    });
    renderSessionTab({ actions: testActions });

    await waitFor(() => {
      expect(screen.getByRole("toolbar", { name: "Action buttons" })).toBeDefined();
    });

    expect(screen.getByRole("button", { name: "Create PR" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Run Tests" })).toBeDefined();
  });

  it("does not render action bar when actions is empty", async () => {
    renderSessionTab({ actions: [] });

    await waitFor(() => {
      expect(screen.getByText("Steer Next Turn")).toBeDefined();
    });

    expect(screen.queryByRole("toolbar", { name: "Action buttons" })).toBeNull();
  });

  it("does not render action bar when actions is undefined", async () => {
    renderSessionTab();

    await waitFor(() => {
      expect(screen.getByText("Steer Next Turn")).toBeDefined();
    });

    expect(screen.queryByRole("toolbar", { name: "Action buttons" })).toBeNull();
  });

  it("shows description in tooltip content when action has description", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = input.toString();
      if (url.includes("/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "" })));
      }
      return Promise.resolve(new Response(JSON.stringify({})));
    });
    renderSessionTab({ actions: testActions });

    await waitFor(() => {
      expect(screen.getByRole("toolbar", { name: "Action buttons" })).toBeDefined();
    });

    const createPrButton = screen.getByRole("button", { name: "Create PR" });
    fireEvent.focus(createPrButton);

    await waitFor(() => {
      const tooltipContent = document.querySelector("[data-slot='tooltip-content']");
      expect(tooltipContent).not.toBeNull();
      expect(tooltipContent?.textContent).toContain("Create a draft pull request");
    });
  });

  it("shows model in tooltip content when action has no description", async () => {
    const actionsWithoutDescription: ApiActionEntry[] = [
      { name: "Deploy", model: "anthropic/claude-opus-4-6", prompt: "deploy to prod" },
    ];
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = input.toString();
      if (url.includes("/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "" })));
      }
      return Promise.resolve(new Response(JSON.stringify({})));
    });
    renderSessionTab({ actions: actionsWithoutDescription });

    await waitFor(() => {
      expect(screen.getByRole("toolbar", { name: "Action buttons" })).toBeDefined();
    });

    const deployButton = screen.getByRole("button", { name: "Deploy" });
    fireEvent.focus(deployButton);

    await waitFor(() => {
      const tooltipContent = document.querySelector("[data-slot='tooltip-content']");
      expect(tooltipContent).not.toBeNull();
      expect(tooltipContent?.textContent).toContain("anthropic/claude-opus-4-6");
    });
  });

  it("triggers adhoc run when action button is clicked", async () => {
    mockFetch.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/adhoc/stop")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "Stopped." })));
      }
      if (url.includes("/adhoc") && init?.method === "POST") {
        return Promise.resolve(new Response(JSON.stringify({ running: true, output: "Running action..." })));
      }
      if (url.includes("/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "" })));
      }
      return Promise.resolve(new Response(JSON.stringify({})));
    });
    renderSessionTab({ actions: testActions });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Create PR" })).toBeDefined();
    });

    fireEvent.click(screen.getByRole("button", { name: "Create PR" }));

    await waitFor(() => {
      const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
      expect(calledUrls.some((url) => url.includes("/adhoc"))).toBe(true);
    });
  });
});
