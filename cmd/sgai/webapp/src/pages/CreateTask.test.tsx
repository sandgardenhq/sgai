import { describe, it, expect, beforeEach, afterEach, mock, afterAll } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { CreateTask } from "./CreateTask";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { FactoryStateSnapshot } from "@/lib/factory-state";
import { useFactoryState as _realUseFactoryState } from "@/lib/factory-state";
import { createMockWorkspace, createMockFork } from "@/test/factories";

const savedExports = {
  useFactoryState: _realUseFactoryState,
  resetFactoryStateStore: () => {},
};

let mockFactoryState: FactoryStateSnapshot = {
  workspaces: [],
  fetchStatus: "idle",
  lastFetchedAt: null,
};

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => mockFactoryState,
  resetFactoryStateStore: () => {},
}));

afterAll(() => {
  mock.module("@/lib/factory-state", () => savedExports);
});

const SAMPLE_RAW_GOAL = `---
flow: |
  "backend-go-developer"
models:
  "coordinator": "anthropic/claude-opus-4-6"
---

Build the thing`;

const SAMPLE_FRONTMATTER = `---
flow: |
  "backend-go-developer"
models:
  "coordinator": "anthropic/claude-opus-4-6"
---

`;

const workspaceWithGoal: FactoryStateSnapshot["workspaces"] = [
  createMockWorkspace({
    name: "root-project",
    dir: "/home/user/root-project",
    isRoot: true,
    status: "Stopped",
    hasSgai: true,
    rawGoalContent: SAMPLE_RAW_GOAL,
    forks: [
      createMockFork({
        name: "happy-blue-3a2e",
        dir: "/home/user/happy-blue-3a2e",
        goalDescription: "Build the thing",
      }),
    ],
  }),
];

function renderCreateTask(wsName = "root-project") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${wsName}/create-task`]}>
      <TooltipProvider>
        <Routes>
          <Route
            path="/workspaces/:name/create-task"
            element={<CreateTask />}
          />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("CreateTask", () => {
  beforeEach(() => {
    sessionStorage.clear();
    mockFactoryState = {
      workspaces: workspaceWithGoal,
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the page title and description", async () => {
    renderCreateTask();

    await waitFor(() => {
      expect(screen.getByText("New Task")).toBeDefined();
      expect(screen.getByText(/Write a GOAL.md/i)).toBeDefined();
    });
  });

  it("renders the Create button", async () => {
    renderCreateTask();

    await waitFor(() => {
      const createBtn = screen.getByRole("button", { name: /Create/i });
      expect(createBtn).toBeDefined();
    });
  });

  it("renders the markdown editor", async () => {
    renderCreateTask();

    await waitFor(() => {
      const editor = document.querySelector("[data-testid='markdown-editor']");
      expect(editor).not.toBeNull();
    });
  });

  it("pre-populates with frontmatter from root workspace GOAL.md", async () => {
    renderCreateTask();

    await waitFor(() => {
      const editor = document.querySelector("[data-testid='markdown-editor']");
      expect(editor).not.toBeNull();
      const editorValue = editor?.getAttribute("data-value") ?? "";
      expect(editorValue).toContain("backend-go-developer");
      expect(editorValue).toContain("anthropic/claude-opus-4-6");
    });
  });

  it("uses default frontmatter when workspace has no rawGoalContent", async () => {
    mockFactoryState = {
      workspaces: [
        createMockWorkspace({
          ...workspaceWithGoal[0],
          rawGoalContent: "",
          goalContent: "",
        }),
      ],
      fetchStatus: "idle",
      lastFetchedAt: Date.now(),
    };
    renderCreateTask();

    await waitFor(() => {
      const editor = document.querySelector("[data-testid='markdown-editor']");
      expect(editor).not.toBeNull();
      const editorValue = editor?.getAttribute("data-value") ?? "";
      expect(editorValue).toContain("react-developer");
      expect(editorValue).toContain("skill-writer");
    });
  });
});
