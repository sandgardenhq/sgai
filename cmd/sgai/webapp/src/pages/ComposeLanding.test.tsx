import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { ComposeLanding } from "./ComposeLanding";
import { TooltipProvider } from "@/components/ui/tooltip";

function mockFetch(data: unknown) {
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
    Promise.resolve(
      new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
}

function renderWithRouter(workspace = "test-workspace") {
  return render(
    <MemoryRouter initialEntries={[`/compose?workspace=${workspace}`]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose" element={<ComposeLanding />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("ComposeLanding", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
  });

  test("renders loading skeletons initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));

    await act(async () => {
      renderWithRouter();
    });

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders page heading", async () => {
    fetchSpy = mockFetch({ templates: [] });

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(screen.getByText("Create New GOAL.md")).toBeTruthy();
  });

  test("renders template cards when templates load", async () => {
    fetchSpy = mockFetch({
      templates: [
        { id: "basic", name: "Basic Project", description: "A simple project", icon: "📦", agents: [], flow: "", interactive: "yes" },
        { id: "fullstack", name: "Full Stack", description: "Frontend + Backend", icon: "🌐", agents: [], flow: "", interactive: "yes" },
      ],
    });

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(screen.getByText("Basic Project")).toBeTruthy();
    expect(screen.getByText("Full Stack")).toBeTruthy();
  });

  test("renders guided wizard section", async () => {
    fetchSpy = mockFetch({ templates: [] });

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(screen.getByText("Guided Wizard")).toBeTruthy();
    expect(screen.getByText("Start Guided Wizard")).toBeTruthy();
  });

  test("renders edit directly section", async () => {
    fetchSpy = mockFetch({ templates: [] });

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(screen.getByText("Edit GOAL.md Directly")).toBeTruthy();
  });

  test("wizard link includes workspace param", async () => {
    fetchSpy = mockFetch({ templates: [] });

    await act(async () => {
      renderWithRouter("my-project");
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    const link = screen.getByText("Start Guided Wizard").closest("a");
    expect(link?.getAttribute("href")).toContain("/compose/step/1");
    expect(link?.getAttribute("href")).toContain("workspace=my-project");
  });

  test("template card links include workspace param", async () => {
    fetchSpy = mockFetch({
      templates: [
        { id: "basic", name: "Basic", description: "desc", icon: "📦", agents: [], flow: "", interactive: "yes" },
      ],
    });

    await act(async () => {
      renderWithRouter("my-project");
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    const link = screen.getByText("Basic").closest("a");
    const href = link?.getAttribute("href") ?? "";
    expect(href).toContain("/compose/template/basic");
    expect(href).toContain("workspace=my-project");
  });

  test("edit GOAL link targets goal/edit route", async () => {
    fetchSpy = mockFetch({ templates: [] });

    await act(async () => {
      renderWithRouter("my-project");
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    const link = screen.getByRole("link", { name: "Edit GOAL.md" });
    expect(link.getAttribute("href")).toBe("/workspaces/my-project/goal/edit");
  });

  test("back link navigates to workspace", async () => {
    fetchSpy = mockFetch({ templates: [] });

    await act(async () => {
      renderWithRouter("my-project");
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    const backLink = screen.getByText(/Back to my-project/);
    expect(backLink).toBeTruthy();
  });
});
