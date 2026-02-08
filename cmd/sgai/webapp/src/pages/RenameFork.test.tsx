import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { RenameFork } from "./RenameFork";
import { TooltipProvider } from "@/components/ui/tooltip";

const workspaceDetail = {
  name: "my-fork",
  dir: "/tmp/my-fork",
  running: false,
  needsInput: false,
  status: "stopped",
  badgeClass: "",
  badgeText: "stopped",
  isRoot: false,
  isFork: true,
  pinned: false,
  hasSgai: true,
  hasEditedGoal: false,
  interactiveAuto: false,
  currentAgent: "",
  currentModel: "",
  task: "",
  goalContent: "",
  pmContent: "",
  hasProjectMgmt: false,
  svgHash: "",
  totalExecTime: "",
  latestProgress: "",
  agentSequence: [],
  cost: { totalCost: 0, inputTokens: 0, outputTokens: 0, cacheCreationInputTokens: 0, cacheReadInputTokens: 0 },
};

function mockFetch(detail = workspaceDetail, renameResponse?: Response | Promise<Response>) {
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
    const urlStr = typeof _input === "string" ? _input : _input instanceof URL ? _input.href : _input.url;
    if (urlStr.includes("/api/v1/workspaces/") && !urlStr.includes("/rename")) {
      return Promise.resolve(
        new Response(JSON.stringify(detail), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (urlStr.includes("/rename")) {
      if (renameResponse instanceof Promise) return renameResponse;
      if (renameResponse) return Promise.resolve(renameResponse);
      return Promise.resolve(
        new Response(JSON.stringify({ name: detail.name, oldName: detail.name, dir: detail.dir }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    return Promise.resolve(new Response("{}"));
  });
}

function renderPage(workspace = "my-fork") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspace}/rename`]}>
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
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders page heading", async () => {
    fetchSpy = mockFetch();
    renderPage();

    expect(await screen.findByRole("heading", { name: "Rename Fork" })).toBeTruthy();
  });

  test("renders current workspace name", async () => {
    fetchSpy = mockFetch({ ...workspaceDetail, name: "old-fork-name" });
    renderPage("old-fork-name");

    expect(await screen.findByText("old-fork-name")).toBeTruthy();
  });

  test("renders new name input", async () => {
    fetchSpy = mockFetch();
    renderPage();

    expect(await screen.findByLabelText("New Name")).toBeTruthy();
  });

  test("renders back link", async () => {
    fetchSpy = mockFetch();
    renderPage("my-fork");

    expect(await screen.findByText("Back to my-fork")).toBeTruthy();
  });

  test("disables button when name is empty", async () => {
    fetchSpy = mockFetch();
    renderPage();

    const button = await screen.findByRole("button", { name: /Rename Fork/i });
    expect(button).toBeTruthy();
    expect((button as HTMLButtonElement).disabled).toBe(true);
  });

  test("submits rename and navigates on success", async () => {
    fetchSpy = mockFetch(workspaceDetail, new Response(
      JSON.stringify({ name: "new-name", oldName: "my-fork", dir: "/tmp/new-name" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    ));
    renderPage();

    const input = await screen.findByLabelText("New Name");
    await act(async () => { fireEvent.change(input, { target: { value: "new-name" } }); });

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Workspace Detail")).toBeTruthy();
  });

  test("shows error on conflict", async () => {
    fetchSpy = mockFetch(workspaceDetail, new Response("cannot rename: session is running", { status: 409 }));
    renderPage();

    const input = await screen.findByLabelText("New Name");
    await act(async () => { fireEvent.change(input, { target: { value: "new-name" } }); });

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("cannot rename: session is running")).toBeTruthy();
  });

  test("disables button during submission (R-18)", async () => {
    fetchSpy = mockFetch(workspaceDetail, new Promise(() => {}));
    renderPage();

    const input = await screen.findByLabelText("New Name");
    await act(async () => { fireEvent.change(input, { target: { value: "test" } }); });

    const button = screen.getByRole("button", { name: /Rename Fork/i });
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Renaming...")).toBeTruthy();
  });

  test("shows guard when workspace is not a fork", async () => {
    fetchSpy = mockFetch({ ...workspaceDetail, isFork: false, name: "main-workspace" });
    renderPage("main-workspace");

    expect(await screen.findByText("Only forks can be renamed.")).toBeTruthy();
    expect(screen.queryByLabelText("New Name")).toBeNull();
  });
});
