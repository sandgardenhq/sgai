import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { SessionTab } from "./SessionTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceEntry } from "@/types";

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

function renderSessionTab(extraProps?: { pmContent?: string; hasProjectMgmt?: boolean }) {
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

  it("renders Start Application button", async () => {
    renderSessionTab();

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Start Application" })).toBeDefined();
    });
  });

  it("calls start API when Start Application button is clicked", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = input.toString();
      if (url.includes("/start")) {
        return Promise.resolve(new Response(JSON.stringify({ running: true, status: "working", message: "Started" })));
      }
      return Promise.resolve(new Response(JSON.stringify({})));
    });

    renderSessionTab();

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Start Application" })).toBeDefined();
    });

    fireEvent.click(screen.getByRole("button", { name: "Start Application" }));

    await waitFor(() => {
      const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
      expect(calledUrls.some((url) => url.includes("/start"))).toBe(true);
    });
  });
});
