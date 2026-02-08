import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { RunTab } from "./RunTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import type { ApiModelsResponse } from "@/types";

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

const modelsResponse: ApiModelsResponse = {
  models: [
    { id: "anthropic/claude-opus-4-6", name: "anthropic/claude-opus-4-6" },
    { id: "openai/gpt-4.1", name: "openai/gpt-4.1" },
  ],
};

const modelsResponseWithDefault: ApiModelsResponse = {
  ...modelsResponse,
  defaultModel: "anthropic/claude-opus-4-6",
};

function renderRunTab() {
  return render(
    <MemoryRouter>
      <RunTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("RunTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderRunTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders ad-hoc form when models load", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(modelsResponse)));
    renderRunTab();

    await waitFor(() => {
      expect(screen.getByText("Model")).toBeDefined();
    });

    expect(screen.getByText("Prompt")).toBeDefined();
    expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
  });

  it("places model selector and submit button on the same row", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(modelsResponse)));
    renderRunTab();

    const modelSelect = await screen.findByRole("combobox", { name: "Model" });
    const submitButton = screen.getByRole("button", { name: "Submit" });

    const row = modelSelect.parentElement;
    expect(row).toBeTruthy();
    expect(row?.contains(submitButton)).toBe(true);
  });

  it("selects the default model when provided", async () => {
    mockFetch.mockResolvedValue(
      new Response(JSON.stringify(modelsResponseWithDefault)),
    );
    renderRunTab();

    const modelSelect = await screen.findByRole("combobox", { name: "Model" });
    await waitFor(() => {
      expect((modelSelect as HTMLSelectElement).value).toBe(
        "anthropic/claude-opus-4-6",
      );
    });
  });

  it("renders error state", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));
    renderRunTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load models/i)).toBeDefined();
    });
  });

  it("calls models API on mount", async () => {
    mockFetch.mockResolvedValue(new Response(JSON.stringify(modelsResponse)));
    renderRunTab();

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const calledUrl = (mockFetch.mock.calls[0] as unknown[])[0] as string;
    expect(calledUrl).toContain("/api/v1/models?workspace=test-project");
  });

  it("runs adhoc prompt and renders output inline", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = input.toString();
      if (url.includes("/api/v1/models?workspace=test-project")) {
        return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
      }
      if (url.includes("/api/v1/workspaces/test-project/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ output: "Hello output", running: false })));
      }
      return Promise.resolve(new Response("{}"));
    });
    renderRunTab();

    const modelSelect = await screen.findByRole("combobox", { name: "Model" });
    fireEvent.change(modelSelect, { target: { value: "anthropic/claude-opus-4-6" } });

    const promptInput = screen.getByRole("textbox", { name: "Prompt" });
    fireEvent.change(promptInput, { target: { value: "Hello world" } });

    const submitButton = screen.getByRole("button", { name: "Submit" });
    await waitFor(() => {
      expect(submitButton.hasAttribute("disabled")).toBe(false);
    });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText("Hello output")).toBeDefined();
    });

    const adhocCalls = mockFetch.mock.calls.filter((call) =>
      (call[0] as string).includes("/api/v1/workspaces/test-project/adhoc"),
    );
    expect(adhocCalls.length).toBeGreaterThan(0);
  });
});
