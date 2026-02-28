import { describe, test, expect, afterEach, beforeEach, spyOn, mock } from "bun:test";
import React from "react";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { AdhocOutput } from "./AdhocOutput";
import { TooltipProvider } from "@/components/ui/tooltip";

mock.module("@monaco-editor/react", () => ({
  default: () => null,
}));

mock.module("@/components/MarkdownEditor", () => ({
  MarkdownEditor: (props: { value: string; onChange: (v: string | undefined) => void; disabled?: boolean; placeholder?: string }) =>
    React.createElement("textarea", {
      "aria-label": "Prompt",
      value: props.value,
      onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => props.onChange(e.target.value),
      disabled: props.disabled,
      placeholder: props.placeholder,
      "data-testid": "markdown-editor",
    }),
}));

function renderPage(workspace = "test-ws", entry?: string) {
  const initialEntry = entry ?? `/workspaces/${workspace}/adhoc`;
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <TooltipProvider>
        <Routes>
          <Route path="workspaces/:name/adhoc" element={<AdhocOutput />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("AdhocOutput", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); });

  test("renders page heading", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage();
    expect(screen.getByText("Ad-hoc Prompt")).toBeTruthy();
  });

  test("renders workspace name in description", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage("my-project");
    expect(screen.getByText("my-project")).toBeTruthy();
  });

  test("renders model input", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage();
    expect(screen.getByLabelText("Model")).toBeTruthy();
  });

  test("renders prompt editor", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage();
    expect(screen.getByLabelText("Prompt")).toBeTruthy();
  });

  test("renders back link", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage("test-ws");
    expect(screen.getByText("Back to test-ws")).toBeTruthy();
  });

  test("auto-runs when model and prompt are provided in the URL", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(
        new Response(JSON.stringify({ output: "Auto output", running: false }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    renderPage("test-ws", "/workspaces/test-ws/adhoc?model=claude&prompt=hello%20world");

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect((screen.getByLabelText("Model") as HTMLInputElement).value).toBe("claude");
    expect((screen.getByLabelText("Prompt") as HTMLTextAreaElement).value).toBe("hello world");
    expect(screen.getByText("Auto output")).toBeTruthy();
  });

  test("disables execute button when fields are empty", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage();
    const button = screen.getByText("Execute Prompt").closest("button");
    expect(button?.disabled).toBe(true);
  });

  test("enables button when both fields have values", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage();
    const modelInput = screen.getByLabelText("Model");
    const promptInput = screen.getByLabelText("Prompt");

    await act(async () => {
      fireEvent.change(modelInput, { target: { value: "claude" } });
      fireEvent.change(promptInput, { target: { value: "hello" } });
    });

    const button = screen.getByText("Execute Prompt").closest("button");
    expect(button?.disabled).toBe(false);
  });

  test("shows running state during execution (R-18)", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request, init?: RequestInit) => {
      if (init?.method === "POST") {
        return Promise.resolve(
          new Response(JSON.stringify({ running: true, output: "", message: "ad-hoc prompt started" }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(
        new Response(JSON.stringify({ running: false, output: "", message: "" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    });
    renderPage();

    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const modelInput = screen.getByLabelText("Model");
    const promptInput = screen.getByLabelText("Prompt");

    await act(async () => {
      fireEvent.change(modelInput, { target: { value: "claude" } });
      fireEvent.change(promptInput, { target: { value: "hello" } });
    });

    const button = screen.getByText("Execute Prompt").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByRole("button", { name: /stop/i })).toBeTruthy();
  });

  test("shows error on failure", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) =>
      Promise.resolve(new Response("prompt and model are required", { status: 400 })),
    );
    renderPage();

    const modelInput = screen.getByLabelText("Model");
    const promptInput = screen.getByLabelText("Prompt");

    await act(async () => {
      fireEvent.change(modelInput, { target: { value: "claude" } });
      fireEvent.change(promptInput, { target: { value: "hello" } });
    });

    const button = screen.getByText("Execute Prompt").closest("button")!;
    await act(async () => { fireEvent.click(button); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("prompt and model are required")).toBeTruthy();
  });

  test("uses vertical layout for model and prompt", () => {
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation(() =>
      Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" }), { status: 200, headers: { "Content-Type": "application/json" } })),
    );
    renderPage();

    const modelInput = screen.getByLabelText("Model");
    const promptInput = screen.getByLabelText("Prompt");

    const verticalContainer = modelInput.closest(".flex.flex-col");
    expect(verticalContainer).toBeTruthy();
    expect(verticalContainer?.contains(promptInput)).toBe(true);
  });
});
