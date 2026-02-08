import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { SkillList } from "./SkillList";
import { TooltipProvider } from "@/components/ui/tooltip";

function mockFetch(data: unknown) {
  return spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

function renderWithRouter(workspaceName = "test-workspace") {
  return render(
    <MemoryRouter
      initialEntries={[`/workspaces/${workspaceName}/skills`]}
    >
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/skills" element={<SkillList />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("SkillList", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
  });

  test("renders loading skeleton initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(
      new Promise(() => {}),
    );

    await act(async () => {
      renderWithRouter();
    });

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders skill categories and cards when data loads", async () => {
    fetchSpy = mockFetch({
      categories: [
        {
          name: "General",
          skills: [
            {
              name: "brainstorming",
              fullPath: "product-design/brainstorming",
              description: "Interactive idea refinement",
            },
          ],
        },
        {
          name: "Architecture",
          skills: [
            {
              name: "detecting-emergent-patterns",
              fullPath: "architecture/detecting-emergent-patterns",
              description: "Find breakthrough insights",
            },
          ],
        },
      ],
    });

    await act(async () => {
      renderWithRouter();
    });

    const general = await screen.findByText("General");
    expect(general).toBeTruthy();

    expect(screen.getByText("Architecture")).toBeTruthy();
    expect(screen.getByText("brainstorming")).toBeTruthy();
    expect(screen.getByText("detecting-emergent-patterns")).toBeTruthy();
  });

  test("renders empty state when no categories", async () => {
    fetchSpy = mockFetch({ categories: [] });

    await act(async () => {
      renderWithRouter();
    });

    const emptyMsg = await screen.findByText("No skills found.");
    expect(emptyMsg).toBeTruthy();
  });

  test("renders error message when API fails", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("Network error"),
    );

    await act(async () => {
      renderWithRouter();
    });

    const errorMsg = await screen.findByText(/Failed to load skills/);
    expect(errorMsg).toBeTruthy();
  });

  test("skill cards have links to detail page", async () => {
    fetchSpy = mockFetch({
      categories: [
        {
          name: "General",
          skills: [
            {
              name: "brainstorming",
              fullPath: "product-design/brainstorming",
              description: "Interactive idea refinement",
            },
          ],
        },
      ],
    });

    await act(async () => {
      renderWithRouter();
    });

    await screen.findByText("brainstorming");

    const link = document.querySelector(
      'a[href="/workspaces/test-workspace/skills/product-design/brainstorming"]',
    );
    expect(link).toBeTruthy();
  });
});
