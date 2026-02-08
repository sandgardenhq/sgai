import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { NewWorkspace } from "./NewWorkspace";
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

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/workspaces/new"]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/new" element={<NewWorkspace />} />
          <Route path="compose" element={<div>Compose Landing</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("NewWorkspace", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders page heading", () => {
    renderPage();
    expect(screen.getByText("Create New Workspace")).toBeTruthy();
  });

  test("renders name input", () => {
    renderPage();
    expect(screen.getByLabelText("Workspace Name")).toBeTruthy();
  });

  test("renders back link to dashboard", () => {
    renderPage();
    expect(screen.getByText("Back to Dashboard")).toBeTruthy();
  });

  test("renders disabled submit button when name is empty", () => {
    renderPage();
    const button = screen.getByText("Create Workspace").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("enables submit button when name is entered", async () => {
    renderPage();
    const input = screen.getByLabelText("Workspace Name");
    await act(async () => { fireEvent.change(input, { target: { value: "my-project" } }); });
    const button = screen.getByText("Create Workspace").closest("button");
    expect(button?.disabled).toBe(false);
  });

  test("submits form and navigates to compose wizard on success", async () => {
    fetchSpy = mockFetch({ name: "my-project", dir: "/tmp/my-project" }, 200);
    renderPage();

    const input = screen.getByLabelText("Workspace Name");
    await act(async () => { fireEvent.change(input, { target: { value: "my-project" } }); });

    const button = screen.getByText("Create Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Compose Landing")).toBeTruthy();
  });

  test("shows error on failed creation", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(new Response("name already exists", { status: 409 })),
    );
    renderPage();

    const input = screen.getByLabelText("Workspace Name");
    await act(async () => { fireEvent.change(input, { target: { value: "existing" } }); });

    const button = screen.getByText("Create Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("name already exists")).toBeTruthy();
  });

  test("disables button during submission (R-18)", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      new Promise(() => {}),
    );
    renderPage();

    const input = screen.getByLabelText("Workspace Name");
    await act(async () => { fireEvent.change(input, { target: { value: "test" } }); });

    const button = screen.getByText("Create Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Creating...")).toBeTruthy();
    expect(button.disabled).toBe(true);
  });
});
