import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { SnippetList } from "./SnippetList";
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
      initialEntries={[`/workspaces/${workspaceName}/snippets`]}
    >
      <TooltipProvider>
        <Routes>
          <Route
            path="workspaces/:name/snippets"
            element={<SnippetList />}
          />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("SnippetList", () => {
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

  test("renders snippet languages and cards when data loads", async () => {
    fetchSpy = mockFetch({
      languages: [
        {
          name: "go",
          snippets: [
            {
              name: "HTTP Server",
              fileName: "http-server.go",
              fullPath: "go/http-server.go",
              description: "Basic Go HTTP server setup",
              language: "go",
            },
          ],
        },
        {
          name: "typescript",
          snippets: [
            {
              name: "React Hook",
              fileName: "react-hook.tsx",
              fullPath: "typescript/react-hook.tsx",
              description: "Custom React hook pattern",
              language: "typescript",
            },
          ],
        },
      ],
    });

    await act(async () => {
      renderWithRouter();
    });

    const goLang = await screen.findByText("go");
    expect(goLang).toBeTruthy();

    expect(screen.getByText("typescript")).toBeTruthy();
    expect(screen.getByText("HTTP Server")).toBeTruthy();
    expect(screen.getByText("React Hook")).toBeTruthy();
  });

  test("renders empty state when no languages", async () => {
    fetchSpy = mockFetch({ languages: [] });

    await act(async () => {
      renderWithRouter();
    });

    const emptyMsg = await screen.findByText("No snippets found.");
    expect(emptyMsg).toBeTruthy();
  });

  test("renders error message when API fails", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("Network error"),
    );

    await act(async () => {
      renderWithRouter();
    });

    const errorMsg = await screen.findByText(/Failed to load snippets/);
    expect(errorMsg).toBeTruthy();
  });

  test("snippet cards have links to detail page", async () => {
    fetchSpy = mockFetch({
      languages: [
        {
          name: "go",
          snippets: [
            {
              name: "HTTP Server",
              fileName: "http-server.go",
              fullPath: "go/http-server.go",
              description: "Basic Go HTTP server setup",
              language: "go",
            },
          ],
        },
      ],
    });

    await act(async () => {
      renderWithRouter();
    });

    await screen.findByText("HTTP Server");

    const link = document.querySelector(
      'a[href="/workspaces/test-workspace/snippets/go/http-server.go"]',
    );
    expect(link).toBeTruthy();
  });
});
