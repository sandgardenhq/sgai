import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { MessagesTab } from "./MessagesTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiMessageEntry, ApiWorkspaceEntry } from "@/types";

const baseWorkspace: Partial<ApiWorkspaceEntry> = {
  name: "test-project",
  dir: "/projects/test-project",
  running: false,
  needsInput: false,
  inProgress: false,
  pinned: false,
  isRoot: false,
  isFork: false,
  status: "idle",
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
  projectTodos: [],
  agentTodos: [],
  changes: { description: "", diffLines: [] },
  commits: [],
  log: [],
};

const sampleMessages: ApiMessageEntry[] = [
  { id: 1, fromAgent: "coordinator", toAgent: "backend-developer", body: "Please implement the API", subject: "API Implementation", read: true },
  { id: 2, fromAgent: "backend-developer", toAgent: "coordinator", body: "API done", subject: "API Complete", read: false },
];

const markdownMessages: ApiMessageEntry[] = [
  {
    id: 3,
    fromAgent: "coordinator",
    toAgent: "backend-developer",
    body: "## Task\n\nPlease implement the **API** with:\n\n- endpoint `/users`\n- endpoint `/posts`\n\n```go\nfunc main() {}\n```",
    subject: "API Implementation",
    read: true,
  },
];

type FactoryStateOverride = {
  workspaces?: ApiWorkspaceEntry[];
  fetchStatus?: "idle" | "fetching" | "error";
};

let mockFactoryOverride: FactoryStateOverride = {};

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({
    workspaces: mockFactoryOverride.workspaces ?? [{ ...baseWorkspace, messages: sampleMessages }],
    fetchStatus: mockFactoryOverride.fetchStatus ?? "idle",
    lastFetchedAt: Date.now(),
  }),
  resetFactoryStateStore: () => {},
}));

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  mockFactoryOverride = {};
});

afterEach(() => {
  cleanup();
});

function renderMessagesTab() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <MessagesTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("MessagesTab", () => {
  it("renders loading skeleton when fetching and no workspace yet", () => {
    mockFactoryOverride = { workspaces: [], fetchStatus: "fetching" };
    renderMessagesTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders messages from factory state", async () => {
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    });

    expect(screen.getAllByText("backend-developer").length).toBeGreaterThan(0);
  });

  it("shows unread messages with bold styling", async () => {
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    });

    const summaries = document.querySelectorAll("summary");
    const unreadSummary = Array.from(summaries).find((s) => s.className.includes("font-bold"));
    expect(unreadSummary).toBeDefined();
    const readSummary = Array.from(summaries).find((s) => !s.className.includes("font-bold"));
    expect(readSummary).toBeDefined();
  });

  it("renders empty state when no messages", async () => {
    mockFactoryOverride = { workspaces: [{ ...baseWorkspace, messages: [] } as ApiWorkspaceEntry] };
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getByText("No messages")).toBeDefined();
    });
  });

  it("renders error state when fetchStatus is error and no workspace", async () => {
    mockFactoryOverride = { workspaces: [], fetchStatus: "error" };
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load messages/i)).toBeDefined();
    });
  });

  it("renders markdown content in message body", async () => {
    mockFactoryOverride = { workspaces: [{ ...baseWorkspace, messages: markdownMessages } as ApiWorkspaceEntry] };
    renderMessagesTab();

    await waitFor(() => {
      expect(screen.getAllByText("coordinator").length).toBeGreaterThan(0);
    });

    const details = document.querySelector("details");
    expect(details).not.toBeNull();
    details!.setAttribute("open", "");

    await waitFor(() => {
      const heading = document.querySelector("h2");
      expect(heading).not.toBeNull();
      expect(heading!.textContent).toBe("Task");
    });

    const strongTexts = Array.from(document.querySelectorAll("strong")).map((el) => el.textContent);
    expect(strongTexts).toContain("API");

    const listItems = document.querySelectorAll("li");
    expect(listItems.length).toBe(2);

    const codeBlock = document.querySelector("pre");
    expect(codeBlock).not.toBeNull();
  });
});
