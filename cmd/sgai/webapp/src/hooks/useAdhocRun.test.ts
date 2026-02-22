import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { renderHook, act, cleanup } from "@testing-library/react";
import { useAdhocRun } from "./useAdhocRun";
import type { ApiModelsResponse } from "@/types";

const mockFetch = mock(() => Promise.resolve(new Response("{}")));

const modelsResponse: ApiModelsResponse = {
  models: [
    { id: "model-a", name: "Model A" },
    { id: "model-b", name: "Model B" },
  ],
  defaultModel: "model-a",
};

beforeEach(() => {
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
});

function defaultMock() {
  mockFetch.mockImplementation((input: string | URL | Request) => {
    const url = input.toString();
    if (url.includes("/api/v1/models")) {
      return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
    }
    if (url.includes("/adhoc")) {
      return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
    }
    return Promise.resolve(new Response("{}"));
  });
}

describe("useAdhocRun", () => {
  it("initializes with empty state", () => {
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    expect(result.current.prompt).toBe("");
    expect(result.current.output).toBe("");
    expect(result.current.isRunning).toBe(false);
    expect(result.current.runError).toBeNull();
    expect(result.current.promptHistory).toEqual([]);
  });

  it("restores prompt from localStorage", () => {
    window.localStorage.setItem("adhoc-prompt-ws1", JSON.stringify("saved"));
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    expect(result.current.prompt).toBe("saved");
  });

  it("restores model from localStorage", () => {
    window.localStorage.setItem("adhoc-model-ws1", JSON.stringify("model-b"));
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    expect(result.current.selectedModel).toBe("model-b");
  });

  it("restores history from localStorage", () => {
    window.localStorage.setItem("adhoc-history-ws1", JSON.stringify(["a", "b"]));
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    expect(result.current.promptHistory).toEqual(["a", "b"]);
  });

  it("persists prompt to localStorage on setPrompt", () => {
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    act(() => {
      result.current.setPrompt("new prompt");
    });
    expect(result.current.prompt).toBe("new prompt");
    expect(JSON.parse(window.localStorage.getItem("adhoc-prompt-ws1")!)).toBe("new prompt");
  });

  it("persists model to localStorage on setSelectedModel", () => {
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    act(() => {
      result.current.setSelectedModel("model-b");
    });
    expect(result.current.selectedModel).toBe("model-b");
    expect(JSON.parse(window.localStorage.getItem("adhoc-model-ws1")!)).toBe("model-b");
  });

  it("selectFromHistory updates prompt", () => {
    window.localStorage.setItem("adhoc-history-ws1", JSON.stringify(["old prompt"]));
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    act(() => {
      result.current.selectFromHistory("old prompt");
    });
    expect(result.current.prompt).toBe("old prompt");
  });

  it("clearHistory empties the history", () => {
    window.localStorage.setItem("adhoc-history-ws1", JSON.stringify(["a", "b"]));
    defaultMock();
    const { result } = renderHook(() =>
      useAdhocRun({ workspaceName: "ws1" }),
    );
    expect(result.current.promptHistory).toEqual(["a", "b"]);
    act(() => {
      result.current.clearHistory();
    });
    expect(result.current.promptHistory).toEqual([]);
    expect(JSON.parse(window.localStorage.getItem("adhoc-history-ws1")!)).toEqual([]);
  });

  it("skips model fetch when skipModelsFetch is true", async () => {
    defaultMock();
    renderHook(() =>
      useAdhocRun({ workspaceName: "ws1", skipModelsFetch: true }),
    );
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const modelCalls = mockFetch.mock.calls.filter((call) =>
      (call[0] as string).includes("/api/v1/models"),
    );
    expect(modelCalls.length).toBe(0);
  });
});
