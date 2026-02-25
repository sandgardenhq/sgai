import { describe, test, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { RenameFork } from "./RenameFork";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiWorkspaceEntry } from "@/types";

const baseWorkspace: ApiWorkspaceEntry = {
  name: "my-fork",
  dir: "/tmp/my-fork",
  running: false,
  needsInput: false,
  inProgress: false,
  pinned: false,
  isRoot: false,
  isFork: true,
  status: "stopped",
  badgeClass: "",
  badgeText: "stopped",
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
  totalExecTime: "",
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

type MockFactoryState = {
  workspaces: ApiWorkspaceEntry[];
  fetchStatus: "idle" | "fetching" | "error";
};

let mockFactoryState: MockFactoryState = {
  workspaces: [baseWorkspace],
  fetchStatus: "idle",
};

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => mockFactoryState,
  resetFactoryStateStore: () => {},
}));

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  cleanup();
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  mockFactoryState = {
    workspaces: [{ ...baseWorkspace }],
    fetchStatus: "idle",
  };
});

afterEach(() => {
  cleanup();
});

function renderPage(workspaceName = "my-fork") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspaceName}/rename`]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/rename" element={<RenameFork />} />
          <Route path="workspaces/:name" element={<div>Workspace Detail</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("RenameFork", () => {
  test("renders page heading", () => {
    renderPage();
    expect(screen.getByRole("heading", { name: "Rename Fork" })).toBeTruthy();
  });

  test("renders current workspace name", () => {
    mockFactoryState = {
      workspaces: [{ ...baseWorkspace, name: "old-fork-name" }],
      fetchStatus: "idle",
    };
    renderPage("old-fork-name");

    expect(screen.getByText("old-fork-name")).toBeTruthy();
  });

  test("renders new name input", () => {
    renderPage();
    expect(screen.getByLabelText("New Name")).toBeTruthy();
  });

  test("renders back link", () => {
    renderPage("my-fork");
    expect(screen.getByText("Back to my-fork")).toBeTruthy();
  });

  test("renders loading skeleton when fetching with no workspace", () => {
    mockFactoryState = { workspaces: [], fetchStatus: "fetching" };
    renderPage();

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("disables button when name is empty", () => {
    renderPage();

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    expect(button).toBeTruthy();
    expect((button as HTMLButtonElement).disabled).toBe(true);
  });

  test("submits rename and navigates on success", async () => {
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/rename")) {
        return Promise.resolve(
          new Response(JSON.stringify({ name: "new-name", oldName: "my-fork", dir: "/tmp/new-name" }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(new Response("{}"));
    });

    renderPage();

    const input = screen.getByLabelText("New Name");
    await act(async () => { fireEvent.change(input, { target: { value: "new-name" } }); });

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Workspace Detail")).toBeTruthy();
  });

  test("shows error on conflict", async () => {
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/rename")) {
        return Promise.resolve(new Response("cannot rename: session is running", { status: 409 }));
      }
      return Promise.resolve(new Response("{}"));
    });

    renderPage();

    const input = screen.getByLabelText("New Name");
    await act(async () => { fireEvent.change(input, { target: { value: "new-name" } }); });

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("cannot rename: session is running")).toBeTruthy();
  });

  test("disables button during submission", async () => {
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlStr = typeof url === "string" ? url : url instanceof URL ? url.href : url.url;
      if (urlStr.includes("/rename")) {
        return new Promise(() => {});
      }
      return Promise.resolve(new Response("{}"));
    });

    renderPage();

    const input = screen.getByLabelText("New Name");
    await act(async () => { fireEvent.change(input, { target: { value: "test" } }); });

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Renaming...")).toBeTruthy();
  });

  test("shows guard when workspace is not a fork", () => {
    mockFactoryState = {
      workspaces: [{ ...baseWorkspace, isFork: false, name: "main-workspace" }],
      fetchStatus: "idle",
    };
    renderPage("main-workspace");

    expect(screen.getByText("Only forks can be renamed.")).toBeTruthy();
    expect(screen.queryByLabelText("New Name")).toBeNull();
  });

  test("does not call individual workspace detail API endpoint", () => {
    renderPage();

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    const hasWorkspaceDetailCall = calledUrls.some(
      (url) => url.match(/\/api\/v1\/workspaces\/[^/]+$/) !== null,
    );
    expect(hasWorkspaceDetailCall).toBe(false);
  });
});
