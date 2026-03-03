import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { AttachExternal } from "./AttachExternal";
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
    <MemoryRouter initialEntries={["/workspaces/attach"]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/attach" element={<AttachExternal />} />
          <Route path="workspaces/:name/goal/edit" element={<div>Edit Goal</div>} />
          <Route path="compose" element={<div>Compose Landing</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("AttachExternal", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
  });

  test("renders page heading", () => {
    renderPage();
    expect(screen.getByText("Attach External Workspace")).toBeTruthy();
  });

  test("renders path input", () => {
    renderPage();
    expect(screen.getByLabelText("Directory Path")).toBeTruthy();
  });

  test("renders back link to dashboard", () => {
    renderPage();
    expect(screen.getByText("Back to Dashboard")).toBeTruthy();
  });

  test("renders disabled submit button when path is empty", () => {
    renderPage();
    const button = screen.getByText("Attach Workspace").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("enables submit button when path is entered", async () => {
    fetchSpy = mockFetch({ entries: [] });
    renderPage();
    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/home/user/project" } }); });
    const button = screen.getByText("Attach Workspace").closest("button");
    expect(button?.disabled).toBe(false);
  });

  test("submits form and navigates to Edit Goal when hasGoal is true", async () => {
    fetchSpy = mockFetch({ name: "my-project", dir: "/home/user/my-project", hasGoal: true }, 200);
    renderPage();

    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/home/user/my-project" } }); });

    const button = screen.getByText("Attach Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Edit Goal")).toBeTruthy();
  });

  test("submits form and navigates to Compose when hasGoal is false", async () => {
    fetchSpy = mockFetch({ name: "my-project", dir: "/home/user/my-project", hasGoal: false }, 200);
    renderPage();

    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/home/user/my-project" } }); });

    const button = screen.getByText("Attach Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Compose Landing")).toBeTruthy();
  });

  test("shows error on failed attach", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(new Response("directory not found", { status: 404 })),
    );
    renderPage();

    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/nonexistent/path" } }); });

    const button = screen.getByText("Attach Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("directory not found")).toBeTruthy();
  });

  test("disables button during submission", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      new Promise(() => {}),
    );
    renderPage();

    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/home/user/project" } }); });

    const button = screen.getByText("Attach Workspace").closest("button")!;
    await act(async () => { fireEvent.click(button); });

    expect(screen.getByText("Attaching...")).toBeTruthy();
    expect(button.disabled).toBe(true);
  });

  test("shows autocomplete suggestions when browse-directories returns entries", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("browse-directories")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              entries: [
                { name: "projects", path: "/home/user/projects" },
                { name: "workspace", path: "/home/user/workspace" },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          ),
        );
      }
      return Promise.resolve(new Response("{}", { status: 200 }));
    });

    renderPage();
    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/home/user" } }); });
    await act(async () => { await new Promise((r) => setTimeout(r, 400)); });

    await waitFor(() => {
      expect(screen.getByText("projects")).toBeTruthy();
      expect(screen.getByText("workspace")).toBeTruthy();
    });
  });

  test("selects suggestion and updates path input", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("browse-directories")) {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              entries: [{ name: "projects", path: "/home/user/projects" }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          ),
        );
      }
      return Promise.resolve(new Response("{}", { status: 200 }));
    });

    renderPage();
    const input = screen.getByLabelText("Directory Path");
    await act(async () => { fireEvent.change(input, { target: { value: "/home/user" } }); });
    await act(async () => { await new Promise((r) => setTimeout(r, 400)); });

    await waitFor(() => {
      expect(screen.getByText("projects")).toBeTruthy();
    });

    const suggestion = screen.getByText("projects").closest("button")!;
    await act(async () => { fireEvent.click(suggestion); });

    const updatedInput = screen.getByLabelText("Directory Path") as HTMLInputElement;
    expect(updatedInput.value).toBe("/home/user/projects");
  });
});
