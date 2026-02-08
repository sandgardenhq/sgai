import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { ComposePreviewPage } from "./ComposePreviewPage";
import { TooltipProvider } from "@/components/ui/tooltip";

function mockFetch(data: unknown) {
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
    Promise.resolve(
      new Response(JSON.stringify(data), { status: 200, headers: { "Content-Type": "application/json" } }),
    ),
  );
}

function renderWithRouter() {
  return render(
    <MemoryRouter initialEntries={["/compose/preview?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/preview" element={<ComposePreviewPage />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("ComposePreviewPage", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders loading skeleton initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    await act(async () => { renderWithRouter(); });
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders preview content after load", async () => {
    fetchSpy = mockFetch({ content: "# My GOAL.md\n\nBuild a project", etag: '"etag1"' });
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("GOAL.md Preview")).toBeTruthy();
    expect(screen.getByText(/My GOAL.md/)).toBeTruthy();
  });

  test("renders flow error when present", async () => {
    fetchSpy = mockFetch({ content: "# GOAL.md", flowError: "Invalid flow definition", etag: '"etag1"' });
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Invalid flow definition")).toBeTruthy();
  });

  test("renders copy button", async () => {
    fetchSpy = mockFetch({ content: "# GOAL.md", etag: '"etag1"' });
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Copy")).toBeTruthy();
  });

  test("renders back link", async () => {
    fetchSpy = mockFetch({ content: "# GOAL.md", etag: '"etag1"' });
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Back to Composer")).toBeTruthy();
  });
});
