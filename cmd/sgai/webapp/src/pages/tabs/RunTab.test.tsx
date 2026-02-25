import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { RunTab } from "./RunTab";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiModelsResponse } from "@/types";

mock.module("@/lib/factory-state", () => ({
  useFactoryState: () => ({ workspaces: [], fetchStatus: "idle", lastFetchedAt: 1000000 }),
  resetFactoryStateStore: () => {},
}));

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
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

function mockModelsAndStatus() {
  return mockFetch.mockImplementation((input: string | URL | Request) => {
    const url = input.toString();
    if (url.includes("/api/v1/models")) {
      return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
    }
    if (url.includes("/api/v1/workspaces/test-project/adhoc")) {
      return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
    }
    return Promise.resolve(new Response("{}"));
  });
}

function mockModelsWithDefaultAndStatus() {
  return mockFetch.mockImplementation((input: string | URL | Request) => {
    const url = input.toString();
    if (url.includes("/api/v1/models")) {
      return Promise.resolve(new Response(JSON.stringify(modelsResponseWithDefault)));
    }
    if (url.includes("/api/v1/workspaces/test-project/adhoc")) {
      return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
    }
    return Promise.resolve(new Response("{}"));
  });
}

function renderRunTab() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <RunTab workspaceName="test-project" />
      </TooltipProvider>
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
    mockModelsAndStatus();
    renderRunTab();

    await waitFor(() => {
      expect(screen.getByText("Model")).toBeDefined();
    });

    expect(screen.getByText("Prompt")).toBeDefined();
    expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
  });

  it("places model selector and prompt textarea in a vertical layout", async () => {
    mockModelsAndStatus();
    renderRunTab();

    const modelSelect = await screen.findByRole("combobox", { name: "Model" });
    const promptTextarea = screen.getByRole("textbox", { name: "Prompt" });

    const verticalContainer = modelSelect.closest(".flex.flex-col");
    expect(verticalContainer).toBeTruthy();
    expect(verticalContainer?.contains(promptTextarea)).toBe(true);
  });

  it("selects the default model when provided", async () => {
    mockModelsWithDefaultAndStatus();
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
    mockModelsAndStatus();
    renderRunTab();

    await waitFor(() => {
      const calledUrls = mockFetch.mock.calls.map((call) => (call[0] as string));
      expect(calledUrls.some((url) => url.includes("/api/v1/models?workspace=test-project"))).toBe(true);
    });
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

  it("persists prompt to localStorage on change", async () => {
    mockModelsAndStatus();
    renderRunTab();

    const promptInput = await screen.findByRole("textbox", { name: "Prompt" });
    fireEvent.change(promptInput, { target: { value: "my saved prompt" } });

    await waitFor(() => {
      const stored = window.localStorage.getItem("adhoc-prompt-test-project");
      expect(stored).toBe(JSON.stringify("my saved prompt"));
    });
  });

  it("restores prompt from localStorage on mount", async () => {
    window.localStorage.setItem("adhoc-prompt-test-project", JSON.stringify("restored prompt"));
    mockModelsAndStatus();
    renderRunTab();

    const promptInput = await screen.findByRole("textbox", { name: "Prompt" });
    expect((promptInput as HTMLTextAreaElement).value).toBe("restored prompt");
  });

  it("saves prompt to history after running", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = input.toString();
      if (url.includes("/api/v1/models")) {
        return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
      }
      if (url.includes("/api/v1/workspaces/test-project/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ output: "done", running: false })));
      }
      return Promise.resolve(new Response("{}"));
    });
    renderRunTab();

    const modelSelect = await screen.findByRole("combobox", { name: "Model" });
    fireEvent.change(modelSelect, { target: { value: "anthropic/claude-opus-4-6" } });

    const promptInput = screen.getByRole("textbox", { name: "Prompt" });
    fireEvent.change(promptInput, { target: { value: "history test prompt" } });

    const submitButton = screen.getByRole("button", { name: "Submit" });
    await waitFor(() => {
      expect(submitButton.hasAttribute("disabled")).toBe(false);
    });
    fireEvent.click(submitButton);

    await waitFor(() => {
      const history = JSON.parse(window.localStorage.getItem("adhoc-history-test-project") ?? "[]");
      expect(history).toContain("history test prompt");
    });
  });
});
