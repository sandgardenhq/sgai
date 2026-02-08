import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { AgentList } from "./AgentList";

function mockFetch(data: unknown, ok = true) {
  return spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(data), {
      status: ok ? 200 : 500,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

function renderWithRouter(workspaceName = "test-workspace") {
  return render(
    <MemoryRouter
      initialEntries={[`/workspaces/${workspaceName}/agents`]}
    >
      <Routes>
        <Route path="workspaces/:name/agents" element={<AgentList />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("AgentList", () => {
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

  test("renders agent cards when data loads", async () => {
    fetchSpy = mockFetch({
      agents: [
        { name: "coordinator", description: "Coordinates workflow" },
        { name: "react-developer", description: "Builds React UI" },
      ],
    });

    await act(async () => {
      renderWithRouter();
    });

    const coordCard = await screen.findByText("coordinator");
    expect(coordCard).toBeTruthy();

    const reactDevCard = await screen.findByText("react-developer");
    expect(reactDevCard).toBeTruthy();

    expect(screen.getByText("Coordinates workflow")).toBeTruthy();
    expect(screen.getByText("Builds React UI")).toBeTruthy();
  });

  test("renders empty state when no agents", async () => {
    fetchSpy = mockFetch({ agents: [] });

    await act(async () => {
      renderWithRouter();
    });

    const emptyMsg = await screen.findByText("No agents found.");
    expect(emptyMsg).toBeTruthy();
  });

  test("renders navigation with back link and refresh", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(
      new Promise(() => {}),
    );

    await act(async () => {
      renderWithRouter();
    });

    const backLinks = screen.getAllByText("← Back");
    expect(backLinks.length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Refresh").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Agents").length).toBeGreaterThanOrEqual(1);
  });

  test("renders error message when API fails", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("Network error"),
    );

    await act(async () => {
      renderWithRouter();
    });

    const errorMsg = await screen.findByText(/Failed to load agents/);
    expect(errorMsg).toBeTruthy();
  });

  test("calls fetch with correct URL", async () => {
    fetchSpy = mockFetch({ agents: [] });

    await act(async () => {
      renderWithRouter("my-project");
    });

    await screen.findByText("No agents found.");

    expect(fetchSpy).toHaveBeenCalledWith(
      "/api/v1/agents?workspace=my-project",
      expect.anything(),
    );
  });
});
