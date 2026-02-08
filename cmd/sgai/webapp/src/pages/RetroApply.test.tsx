import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, cleanup, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { RetroApply } from "./RetroApply";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockRetroData = {
  sessions: [{ name: "2026-02-01.abc1", hasImprovements: true, goalSummary: "Build an app" }],
  selectedSession: "2026-02-01.abc1",
  details: {
    sessionName: "2026-02-01.abc1",
    goalSummary: "Build an app",
    goalContent: "# GOAL.md",
    improvements: "<h2>Improvements</h2><ul><li>Add error handling</li><li>Improve test coverage</li><li>Refactor main module</li></ul>",
    improvementsRaw: "- Add error handling\n- Improve test coverage\n- Refactor main module",
    hasImprovements: true,
    isAnalyzing: false,
    isApplying: false,
  },
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
    <MemoryRouter initialEntries={["/workspaces/test-ws/retrospective/apply?session=2026-02-01.abc1"]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/retrospective/apply" element={<RetroApply />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

async function renderPageAndWait() {
  renderPage();
  await screen.findByText("Apply Retrospective Recommendations");
}

describe("RetroApply", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders loading skeleton initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    renderPage();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders heading after load", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    renderPage();
    expect(await screen.findByText("Apply Retrospective Recommendations")).toBeTruthy();
  });

  test("renders missing params alert when session is absent", async () => {
    render(
      <MemoryRouter initialEntries={["/workspaces/test-ws/retrospective/apply"]}>
        <TooltipProvider>
          <Routes>
            <Route path="workspaces/:name/retrospective/apply" element={<RetroApply />} />
          </Routes>
        </TooltipProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Missing workspace or session")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Back to Retrospectives" })).toBeTruthy();
  });

  test("renders session name", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await renderPageAndWait();
    expect(screen.getByText("2026-02-01.abc1")).toBeTruthy();
  });

  test("renders parsed suggestions as checkboxes", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await renderPageAndWait();
    await screen.findByText("Add error handling");

    expect(screen.getByText("Improve test coverage")).toBeTruthy();
    expect(screen.getByText("Refactor main module")).toBeTruthy();

    const checkboxes = await screen.findAllByRole("checkbox");
    expect(checkboxes.length).toBe(3);
  });

  test("all suggestions are selected by default", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await renderPageAndWait();
    const checkboxes = await screen.findAllByRole("checkbox");
    for (const cb of checkboxes) {
      expect(cb.getAttribute("aria-checked")).toBe("true");
    }
  });

  test("renders back link to retrospectives", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await renderPageAndWait();
    expect(await screen.findByRole("link", { name: "Back to Retrospectives" })).toBeTruthy();
  });

  test("renders apply button", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await renderPageAndWait();
    expect(screen.getByRole("button", { name: "Apply Selected Recommendations" })).toBeTruthy();
  });

  test("can toggle suggestions off", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await renderPageAndWait();
    const [firstCheckbox] = await screen.findAllByRole("checkbox");
    fireEvent.click(firstCheckbox);

    await waitFor(() => {
      expect(firstCheckbox.getAttribute("aria-checked")).toBe("false");
    });
  });

  test("applies selected suggestions on click", async () => {
    fetchSpy = mockFetchSequence([
      mockRetroData,
      { running: true, session: "2026-02-01.abc1", message: "apply started" },
    ]);
    await renderPageAndWait();

    const button = screen.getByRole("button", { name: "Apply Selected Recommendations" });
    fireEvent.click(button);

    expect(await screen.findByText("Apply Started")).toBeTruthy();
    expect(button.hasAttribute("disabled")).toBe(true);
  });

  test("disables button during apply (R-18)", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(
        new Response(JSON.stringify(mockRetroData), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    await renderPageAndWait();

    fetchSpy.mockImplementation((_input: string | URL | Request) => new Promise(() => {}));

    const button = screen.getByRole("button", { name: "Apply Selected Recommendations" });
    fireEvent.click(button);

    expect(await screen.findByText("Applying...")).toBeTruthy();
    expect(button.hasAttribute("disabled")).toBe(true);
  });

  test("renders no suggestions message when improvements empty", async () => {
    const emptyRetro = {
      ...mockRetroData,
      details: {
        ...mockRetroData.details,
        improvements: "",
        improvementsRaw: "",
        hasImprovements: false,
      },
    };
    fetchSpy = mockFetchSequence([emptyRetro]);
    renderPage();
    expect(await screen.findByText("No improvement suggestions found. Run analysis first.")).toBeTruthy();
  });
});
