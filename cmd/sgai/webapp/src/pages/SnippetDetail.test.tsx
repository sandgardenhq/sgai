import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, act, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { SnippetDetail } from "./SnippetDetail";

function mockFetch(data: unknown) {
  return spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
}

function renderWithRouter(
  lang = "go",
  fileName = "http-server.go",
  workspaceName = "test-workspace",
) {
  return render(
    <MemoryRouter
      initialEntries={[
        `/workspaces/${workspaceName}/snippets/${lang}/${fileName}`,
      ]}
    >
      <Routes>
        <Route
          path="workspaces/:name/snippets/:lang/:fileName"
          element={<SnippetDetail />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("SnippetDetail", () => {
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

  test("renders snippet content when data loads", async () => {
    fetchSpy = mockFetch({
      name: "HTTP Server",
      fileName: "http-server.go",
      language: "go",
      description: "Basic Go HTTP server setup",
      whenToUse: "When setting up a new Go HTTP server",
      content:
        'package main\n\nfunc main() {\n  http.ListenAndServe(":8080", nil)\n}',
    });

    await act(async () => {
      renderWithRouter();
    });

    const name = await screen.findByText("HTTP Server");
    expect(name).toBeTruthy();

    expect(screen.getByText("Basic Go HTTP server setup")).toBeTruthy();
    expect(
      screen.getByText("When setting up a new Go HTTP server"),
    ).toBeTruthy();
    expect(screen.getByText("When to use:")).toBeTruthy();
  });

  test("renders code block with snippet content", async () => {
    fetchSpy = mockFetch({
      name: "HTTP Server",
      fileName: "http-server.go",
      language: "go",
      description: "Basic Go HTTP server setup",
      whenToUse: "",
      content: "package main",
    });

    await act(async () => {
      renderWithRouter();
    });

    await screen.findByText("HTTP Server");

    const codeElement = document.querySelector("pre code");
    expect(codeElement).toBeTruthy();
    expect(codeElement?.textContent).toContain("package main");
  });

  test("renders error message when API fails", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("Network error"),
    );

    await act(async () => {
      renderWithRouter();
    });

    const errorMsg = await screen.findByText(/Failed to load snippet/);
    expect(errorMsg).toBeTruthy();
  });

  test("renders back navigation link", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(
      new Promise(() => {}),
    );

    await act(async () => {
      renderWithRouter();
    });

    expect(screen.getByText("← Back to Snippets")).toBeTruthy();
  });

  test("calls fetch with correct URL", async () => {
    fetchSpy = mockFetch({
      name: "React Hook",
      fileName: "react-hook.tsx",
      language: "typescript",
      description: "Custom React hook pattern",
      whenToUse: "",
      content: "const hook = () => {};",
    });

    await act(async () => {
      renderWithRouter("typescript", "react-hook.tsx", "my-project");
    });

    await screen.findByText("React Hook");

    expect(fetchSpy).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/snippets/typescript/react-hook.tsx"),
      expect.anything(),
    );
  });

  test("renders without whenToUse section when empty", async () => {
    fetchSpy = mockFetch({
      name: "Simple Snippet",
      fileName: "simple.go",
      language: "go",
      description: "A simple snippet",
      whenToUse: "",
      content: "// simple code",
    });

    await act(async () => {
      renderWithRouter();
    });

    await screen.findByText("Simple Snippet");

    const whenToUse = screen.queryByText("When to use:");
    expect(whenToUse).toBeNull();
  });
});
