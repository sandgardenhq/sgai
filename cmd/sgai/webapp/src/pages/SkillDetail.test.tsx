import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { SkillDetail } from "./SkillDetail";

function mockFetch(data: unknown) {
  return spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

function renderWithRouter(
  skillPath = "product-design/brainstorming",
  workspaceName = "test-workspace",
) {
  return render(
    <MemoryRouter
      initialEntries={[`/workspaces/${workspaceName}/skills/${skillPath}`]}
    >
      <Routes>
        <Route
          path="workspaces/:name/skills/*"
          element={<SkillDetail />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("SkillDetail", () => {
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

  test("renders skill content when data loads", async () => {
    fetchSpy = mockFetch({
      name: "brainstorming",
      fullPath: "product-design/brainstorming",
      content: "<h1>Brainstorming</h1><p>Interactive idea refinement</p>",
      rawContent: "# Brainstorming\nInteractive idea refinement",
    });

    await act(async () => {
      renderWithRouter();
    });

    const heading = await screen.findByText("Brainstorming");
    expect(heading).toBeTruthy();
    expect(screen.getByText("Interactive idea refinement")).toBeTruthy();
  });

  test("renders error message when API fails", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("Network error"),
    );

    await act(async () => {
      renderWithRouter();
    });

    const errorMsg = await screen.findByText(/Failed to load skill/);
    expect(errorMsg).toBeTruthy();
  });

  test("renders back navigation link", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(
      new Promise(() => {}),
    );

    await act(async () => {
      renderWithRouter();
    });

    expect(screen.getByText("← Back to Skills")).toBeTruthy();
  });

  test("renders markdown content", async () => {
    fetchSpy = mockFetch({
      name: "test-skill",
      fullPath: "test-skill",
      content: "<h1>Custom Skill</h1>",
      rawContent: "# Custom Skill",
    });

    await act(async () => {
      renderWithRouter("test-skill");
    });

    const heading = await screen.findByRole("heading", { name: "Custom Skill" });
    expect(heading).toBeTruthy();
  });

  test("calls fetch with correct URL including skill path", async () => {
    fetchSpy = mockFetch({
      name: "detecting-emergent-patterns",
      fullPath: "architecture/detecting-emergent-patterns",
      content: "<p>Content</p>",
      rawContent: "Content",
    });

    await act(async () => {
      renderWithRouter("architecture/detecting-emergent-patterns", "my-project");
    });

    await screen.findByText("Content");

    expect(fetchSpy).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/skills/"),
      expect.anything(),
    );
  });
});
