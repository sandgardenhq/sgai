import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ChangesTab } from "./ChangesTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import type { ApiChangesResponse } from "@/types";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  closed = false;
  constructor(url: string) { this.url = url; }
  addEventListener() {}
  removeEventListener() {}
  close() { this.closed = true; }
}

const originalEventSource = globalThis.EventSource;
const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  (globalThis as unknown as Record<string, unknown>).EventSource = MockEventSource;
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
});

const changesResponse: ApiChangesResponse = {
  description: "Add new feature",
  diffLines: [
    { lineNumber: 1, text: "diff --git a/main.go b/main.go", class: "header" },
    { lineNumber: 2, text: "+func newFunction() {", class: "add" },
    { lineNumber: 3, text: "-func oldFunction() {", class: "remove" },
    { lineNumber: 4, text: " func unchanged() {", class: "" },
  ],
};

function renderChangesTab() {
  return render(
    <MemoryRouter>
      <ChangesTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("ChangesTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderChangesTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders diff lines when data loads", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(changesResponse)));
    renderChangesTab();

    await waitFor(() => {
      expect(screen.getByRole("textbox", { name: "Commit Description" })).toBeDefined();
    });

    expect(screen.queryByText("Commit Description")).toBeNull();

    await waitFor(() => {
      const descriptionInput = screen.getByRole("textbox", { name: "Commit Description" }) as HTMLInputElement;
      expect(descriptionInput.value).toBe("Add new feature");
    });
    expect(screen.getByRole("button", { name: "Update" })).toBeDefined();
    expect(screen.getByText("Diff")).toBeDefined();
    expect(screen.getByText("+func newFunction() {")).toBeDefined();
    expect(screen.getByText("-func oldFunction() {")).toBeDefined();
  });

  it("renders diff lines with line numbers", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(changesResponse)));
    renderChangesTab();

    await waitFor(() => {
      expect(screen.getByText("+func newFunction() {")).toBeDefined();
    });

    const addLine = screen.getByText("+func newFunction() {");
    expect(addLine.getAttribute("data-line-number")).toBe("2");
  });

  it("applies diff line styling by line class", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(changesResponse)));
    const { container } = renderChangesTab();
    const view = within(container);

    await waitFor(() => {
      expect(view.getByText("+func newFunction() {")).toBeDefined();
    });

    const addLine = view.getByText("+func newFunction() {");
    const removeLine = view.getByText("-func oldFunction() {");
    const headerLine = view.getByText("diff --git a/main.go b/main.go");

    expect(addLine.className).toContain("border-l-4");
    expect(addLine.className).toContain("border-green-500");
    expect(removeLine.className).toContain("border-red-500");
    expect(headerLine.className).toContain("border-yellow-400");
  });

  it("renders empty diff state", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify({ description: "", diffLines: [] })));
    renderChangesTab();

    await waitFor(() => {
      expect(screen.getByText("No changes to display")).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderChangesTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load changes/i)).toBeDefined();
    });
  });

  it("calls changes API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(changesResponse)));
    renderChangesTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/workspaces/test-project/changes");
  });
});
