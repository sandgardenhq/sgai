import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { RetroAnalyze } from "./RetroAnalyze";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockRetroData = {
  sessions: [{ name: "2026-02-01.abc1", hasImprovements: false, goalSummary: "Build an app" }],
  selectedSession: "2026-02-01.abc1",
  details: {
    sessionName: "2026-02-01.abc1",
    goalSummary: "Build an app",
    goalContent: "# GOAL.md",
    improvements: "",
    hasImprovements: false,
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
    <MemoryRouter initialEntries={["/workspaces/test-ws/retrospective/analyze?session=2026-02-01.abc1"]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/retrospective/analyze" element={<RetroAnalyze />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("RetroAnalyze", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders loading skeleton initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    await act(async () => { renderPage(); });
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders heading after load", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Run Retrospective Analysis")).toBeTruthy();
  });

  test("renders session name", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("2026-02-01.abc1")).toBeTruthy();
  });

  test("renders goal summary", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Build an app")).toBeTruthy();
  });

  test("auto-starts analysis after load", async () => {
    fetchSpy = mockFetchSequence([
      mockRetroData,
      { running: true, session: "2026-02-01.abc1", message: "analysis started" },
    ]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Analysis Running")).toBeTruthy();
    expect(screen.getByText("Analyzing...")).toBeTruthy();
  });

  test("does not auto-start when retrospective already analyzed", async () => {
    const analyzedData = {
      ...mockRetroData,
      details: {
        ...mockRetroData.details,
        improvements: "Some improvements",
        hasImprovements: true,
        isAnalyzing: false,
      },
    };
    fetchSpy = mockFetchSequence([analyzedData]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(fetchSpy.mock.calls.length).toBe(1);
    expect(screen.getByText("Analysis Started")).toBeTruthy();
  });

  test("does not auto-retry analysis after error", async () => {
    let callIndex = 0;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
      callIndex += 1;
      if (callIndex === 1) {
        return Promise.resolve(
          new Response(JSON.stringify(mockRetroData), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(new Response("boom", { status: 500 }));
    });

    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(callIndex).toBe(2);
    expect(screen.getByText("boom")).toBeTruthy();
  });

  test("renders back link to retrospectives", async () => {
    fetchSpy = mockFetchSequence([mockRetroData]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Back to Retrospectives")).toBeTruthy();
    const link = screen.getByText("Back to Retrospectives").closest("a");
    expect(link?.getAttribute("href")).toBe("/workspaces/test-ws/retro");
  });

  test("disables button during analysis (R-18)", async () => {
    let callCount = 0;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
      callCount += 1;
      if (callCount === 1) {
        return Promise.resolve(
          new Response(JSON.stringify(mockRetroData), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return new Promise(() => {});
    });

    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const button = screen.getByText("Analyzing...").closest("button")!;
    expect(button.disabled).toBe(true);
  });

  test("shows already running state when isAnalyzing is true", async () => {
    const analyzingData = {
      ...mockRetroData,
      details: { ...mockRetroData.details, isAnalyzing: true },
    };
    fetchSpy = mockFetchSequence([analyzingData]);
    await act(async () => { renderPage(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Analysis Running")).toBeTruthy();
  });
});
