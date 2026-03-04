import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { NewFork } from "./NewFork";
import { TooltipProvider } from "@/components/ui/tooltip";
import { setupMarkdownEditorMock, mockFetchJson, mockForkTemplateFetch } from "@/test-utils";

setupMarkdownEditorMock();

function renderPage(workspace = "root-ws") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspace}/fork/new`]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/fork/new" element={<NewFork />} />
          <Route path="workspaces/:name/goal/edit" element={<div>Edit Goal</div>} />
          <Route path="workspaces/:name/progress" element={<div>Workspace Progress</div>} />
          <Route path="workspaces/:name" element={<div>Workspace Detail</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("NewFork", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders page heading", () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage();
    expect(screen.getByText("Fork Workspace")).toBeTruthy();
  });

  test("renders parent workspace name in description", () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage("my-root");
    expect(screen.getByText("my-root")).toBeTruthy();
  });

  test("renders goal content editor", () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage();
    expect(screen.getByTestId("markdown-editor")).toBeTruthy();
  });

  test("renders back link to parent workspace", () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage("my-root");
    expect(screen.getByText("Back to my-root")).toBeTruthy();
  });

  test("disables button when goal content is empty", () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage();
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("enables button when goal content entered", async () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage();
    const editor = screen.getByTestId("markdown-editor");
    await act(async () => { fireEvent.change(editor, { target: { value: "My goal content" } }); });
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(false);
  });

  test("submits fork with goal content and navigates on success", async () => {
    fetchSpy = mockFetchJson({ name: "happy-blue-a1e2", dir: "/tmp/happy-blue-a1e2", parent: "root-ws" });
    renderPage();

    const editor = screen.getByTestId("markdown-editor");
    await act(async () => { fireEvent.change(editor, { target: { value: "Build a feature" } }); });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Workspace Progress")).toBeTruthy();
  });

  test("shows error on API failure", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(new Response("goal content cannot be empty", { status: 400 })),
    );
    renderPage();

    const editor = screen.getByTestId("markdown-editor");
    await act(async () => { fireEvent.change(editor, { target: { value: "Some content" } }); });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("goal content cannot be empty")).toBeTruthy();
  });

  test("shows validation error when submitting empty content", async () => {
    fetchSpy = mockForkTemplateFetch();
    renderPage();

    const form = document.querySelector("form")!;
    await act(async () => { fireEvent.submit(form); });

    expect(screen.getByText("Please write a goal description")).toBeTruthy();
  });

  test("disables button during submission (R-18)", async () => {
    let resolvePending!: (value: Response) => void;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      new Promise<Response>((resolve) => { resolvePending = resolve; }),
    );
    renderPage();

    const editor = screen.getByTestId("markdown-editor");
    await act(async () => { fireEvent.change(editor, { target: { value: "Test goal" } }); });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Creating Fork...")).toBeTruthy();
    expect(button.disabled).toBe(true);

    await act(async () => {
      resolvePending(new Response(JSON.stringify({ name: "x", dir: "/tmp/x", parent: "root-ws" }), {
        status: 200, headers: { "Content-Type": "application/json" },
      }));
      await new Promise((r) => setTimeout(r, 10));
    });
  });
});
