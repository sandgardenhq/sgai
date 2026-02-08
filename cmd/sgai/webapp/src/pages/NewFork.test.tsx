import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { NewFork } from "./NewFork";
import { TooltipProvider } from "@/components/ui/tooltip";

function mockFetch(data: unknown, status = 200) {
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
    Promise.resolve(
      new Response(JSON.stringify(data), {
        status,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
}

function renderPage(workspace = "root-ws") {
  return render(
    <MemoryRouter initialEntries={[`/workspaces/${workspace}/fork/new`]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/fork/new" element={<NewFork />} />
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
    renderPage();
    expect(screen.getByText("Fork Workspace")).toBeTruthy();
  });

  test("renders parent workspace name in description", () => {
    renderPage("my-root");
    expect(screen.getByText("my-root")).toBeTruthy();
  });

  test("renders fork name input", () => {
    renderPage();
    expect(screen.getByLabelText("Fork Name")).toBeTruthy();
  });

  test("renders back link to parent workspace", () => {
    renderPage("my-root");
    expect(screen.getByText("Back to my-root")).toBeTruthy();
  });

  test("disables button when fork name is empty", () => {
    renderPage();
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("enables button when fork name entered", async () => {
    renderPage();
    const input = screen.getByLabelText("Fork Name");
    await act(async () => { fireEvent.change(input, { target: { value: "my-fork" } }); });
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(false);
  });

  test("submits fork and navigates on success", async () => {
    fetchSpy = mockFetch({ name: "my-fork", dir: "/tmp/my-fork", parent: "root-ws" });
    renderPage();

    const input = screen.getByLabelText("Fork Name");
    await act(async () => { fireEvent.change(input, { target: { value: "my-fork" } }); });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Workspace Detail")).toBeTruthy();
  });

  test("shows error on conflict", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(new Response("a directory with this name already exists", { status: 409 })),
    );
    renderPage();

    const input = screen.getByLabelText("Fork Name");
    await act(async () => { fireEvent.change(input, { target: { value: "existing" } }); });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("a directory with this name already exists")).toBeTruthy();
  });

  test("disables button during submission (R-18)", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      new Promise(() => {}),
    );
    renderPage();

    const input = screen.getByLabelText("Fork Name");
    await act(async () => { fireEvent.change(input, { target: { value: "test" } }); });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Creating Fork...")).toBeTruthy();
    expect(button.disabled).toBe(true);
  });
});
