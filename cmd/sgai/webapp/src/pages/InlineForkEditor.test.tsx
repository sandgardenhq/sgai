import { describe, test, expect, afterEach, spyOn, mock } from "bun:test";
import React from "react";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { InlineForkEditor } from "./InlineForkEditor";
import { TooltipProvider } from "@/components/ui/tooltip";

mock.module("@monaco-editor/react", () => ({
  default: () => null,
}));

mock.module("@/components/MarkdownEditor", () => ({
  MarkdownEditor: (props: { value: string; onChange: (v: string | undefined) => void; disabled?: boolean; placeholder?: string }) =>
    React.createElement("textarea", {
      "data-testid": "markdown-editor",
      value: props.value,
      onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => props.onChange(e.target.value),
      disabled: props.disabled,
      placeholder: props.placeholder,
    }),
}));

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

function renderEditor(workspaceName = "root-ws") {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <TooltipProvider>
        <Routes>
          <Route
            path="/"
            element={<InlineForkEditor workspaceName={workspaceName} />}
          />
          <Route
            path="/workspaces/:name/progress"
            element={<div>Fork Progress</div>}
          />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

function mockForkTemplateFetch() {
  return spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    if (url.includes("fork-template")) {
      return Promise.resolve(
        new Response(JSON.stringify({ content: "" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    return Promise.resolve(new Response("Not Found", { status: 404 }));
  });
}

describe("InlineForkEditor", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
  });

  test("renders heading and description", () => {
    fetchSpy = mockForkTemplateFetch();
    renderEditor();
    expect(screen.getByText("New Task")).toBeTruthy();
    expect(screen.getByText(/Write a GOAL.md/)).toBeTruthy();
  });

  test("renders Create Fork button disabled when content is empty", () => {
    fetchSpy = mockForkTemplateFetch();
    renderEditor();
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("enables button when content is entered", async () => {
    fetchSpy = mockForkTemplateFetch();
    renderEditor();
    const editor = screen.getByTestId("markdown-editor");
    await act(async () => {
      fireEvent.change(editor, { target: { value: "My new goal" } });
    });
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(false);
  });

  test("keeps button disabled for empty content", () => {
    fetchSpy = mockForkTemplateFetch();
    renderEditor();
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("keeps button disabled for frontmatter-only content", async () => {
    fetchSpy = mockForkTemplateFetch();
    renderEditor();
    const editor = screen.getByTestId("markdown-editor");
    await act(async () => {
      fireEvent.change(editor, { target: { value: "---\nfoo: bar\n---\n" } });
    });
    const button = screen.getByText("Create Fork").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("prefills editor when fork-template returns content", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      if (url.includes("fork-template")) {
        return Promise.resolve(
          new Response(JSON.stringify({ content: "# Template\n\nPrefilled content" }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(new Response("Not Found", { status: 404 }));
    });
    renderEditor();
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });
    const editor = screen.getByTestId("markdown-editor") as HTMLTextAreaElement;
    expect(editor.value).toBe("# Template\n\nPrefilled content");
  });

  test("submits fork and navigates on success", async () => {
    fetchSpy = mockFetch({
      name: "clever-blue-a1e2",
      dir: "/tmp/clever-blue-a1e2",
      parent: "root-ws",
    });
    renderEditor();

    const editor = screen.getByTestId("markdown-editor");
    await act(async () => {
      fireEvent.change(editor, { target: { value: "Build a new feature" } });
    });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => {
      fireEvent.click(button);
    });
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("Fork Progress")).toBeTruthy();
  });

  test("shows error on API failure", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(
      (_input: string | URL | Request) =>
        Promise.resolve(
          new Response("GOAL.md body cannot be empty", { status: 400 }),
        ),
    );
    renderEditor();

    const editor = screen.getByTestId("markdown-editor");
    await act(async () => {
      fireEvent.change(editor, { target: { value: "Some content" } });
    });

    const button = screen.getByText("Create Fork").closest("button")!;
    await act(async () => {
      fireEvent.click(button);
    });
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("GOAL.md body cannot be empty")).toBeTruthy();
  });
});
