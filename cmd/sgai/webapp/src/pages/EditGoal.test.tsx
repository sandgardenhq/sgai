import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { EditGoal } from "./EditGoal";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockWorkspaceDetail = {
  name: "test-ws",
  dir: "/tmp/test-ws",
  running: false,
  needsInput: false,
  status: "idle",
  badgeClass: "",
  badgeText: "Idle",
  isRoot: true,
  isFork: false,
  pinned: false,
  hasSgai: true,
  hasEditedGoal: true,
  interactiveAuto: false,
  currentAgent: "",
  currentModel: "",
  task: "",
  goalContent: "<h1>My Project</h1>\n<p>Build something great</p>",
  rawGoalContent: "# My Project\n\nBuild something great",
  fullGoalContent: "---\ntitle: My Project\n---\n\n# My Project\n\nBuild something great",
  pmContent: "",
  hasProjectMgmt: false,
  svgHash: "",
  totalExecTime: "",
  latestProgress: "",
  agentSequence: [],
  cost: { totalCost: 0, totalTokens: { input: 0, output: 0, reasoning: 0, cacheRead: 0, cacheWrite: 0 }, byAgent: [] },
};

function mockFetchSequence(responses: unknown[]) {
  let callIndex = 0;
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
    const data = responses[callIndex] ?? responses[responses.length - 1];
    callIndex++;
    return Promise.resolve(
      new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/workspaces/test-ws/goal/edit"]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/goal/edit" element={<EditGoal />} />
          <Route path="workspaces/:name" element={<div>Workspace Detail</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("EditGoal", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders loading skeleton initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    await act(async () => { renderPage(); });
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders heading and textarea after load", async () => {
    fetchSpy = mockFetchSequence([mockWorkspaceDetail]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Edit GOAL.md")).toBeTruthy();
    expect(screen.getByLabelText("GOAL.md Content")).toBeTruthy();
  });

  test("loads GOAL.md content from server", async () => {
    fetchSpy = mockFetchSequence([mockWorkspaceDetail]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const textarea = screen.getByLabelText("GOAL.md Content") as HTMLTextAreaElement;
    expect(textarea.value).toBe("---\ntitle: My Project\n---\n\n# My Project\n\nBuild something great");
  });

  test("falls back to rawGoalContent when full content is missing", async () => {
    fetchSpy = mockFetchSequence([{ ...mockWorkspaceDetail, fullGoalContent: undefined }]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const textarea = screen.getByLabelText("GOAL.md Content") as HTMLTextAreaElement;
    expect(textarea.value).toBe("# My Project\n\nBuild something great");
  });

  test("renders save and cancel buttons", async () => {
    fetchSpy = mockFetchSequence([mockWorkspaceDetail]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Save GOAL.md")).toBeTruthy();
    expect(screen.getByText("Cancel")).toBeTruthy();
  });

  test("renders back link", async () => {
    fetchSpy = mockFetchSequence([mockWorkspaceDetail]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Back to test-ws")).toBeTruthy();
  });

  test("saves and shows success message", async () => {
    fetchSpy = mockFetchSequence([mockWorkspaceDetail, { updated: true, workspace: "test-ws" }]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const button = screen.getByText("Save GOAL.md").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Saved!")).toBeTruthy();
  });

  test("shows error on save failure", async () => {
    let callIndex = 0;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
      callIndex++;
      if (callIndex === 1) {
        return Promise.resolve(
          new Response(JSON.stringify(mockWorkspaceDetail), { status: 200, headers: { "Content-Type": "application/json" } }),
        );
      }
      return Promise.resolve(new Response("content cannot be empty", { status: 400 }));
    });

    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const button = screen.getByText("Save GOAL.md").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("content cannot be empty")).toBeTruthy();
  });

  test("disables save button during submission (R-18)", async () => {
    let callIndex = 0;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
      callIndex++;
      if (callIndex === 1) {
        return Promise.resolve(
          new Response(JSON.stringify(mockWorkspaceDetail), { status: 200, headers: { "Content-Type": "application/json" } }),
        );
      }
      return new Promise(() => {});
    });

    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const button = screen.getByText("Save GOAL.md").closest("button")!;
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Saving...")).toBeTruthy();
  });
});
