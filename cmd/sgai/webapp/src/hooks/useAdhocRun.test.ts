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

  describe("workspace switching resets running state", () => {
    it("clears output when workspaceName changes to one with no active run", async () => {
      mockFetch.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
        const url = input.toString();
        if (url.includes("/api/v1/models")) {
          return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
        }
        if (url.includes("/adhoc")) {
          if (init?.method === "POST") {
            return Promise.resolve(new Response(JSON.stringify({ running: false, output: "run output", message: "" })));
          }
          if (url.includes("/ws1/")) {
            return Promise.resolve(new Response(JSON.stringify({ running: false, output: "ws1 output", message: "" })));
          }
          return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
        }
        return Promise.resolve(new Response("{}"));
      });

      const { result, rerender } = renderHook(({ ws }: { ws: string }) =>
        useAdhocRun({ workspaceName: ws, skipModelsFetch: true }),
        { initialProps: { ws: "ws1" } },
      );

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      act(() => {
        result.current.setSelectedModel("model-a");
      });

      await act(async () => {
        result.current.startRun("test prompt", "model-a");
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.output).toBe("run output");

      rerender({ ws: "ws2" });

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.output).toBe("");
      expect(result.current.isRunning).toBe(false);
      expect(result.current.runError).toBeNull();
    });

    it("resets isRunning to false when workspaceName changes to one with no active run", async () => {
      let postCallCount = 0;
      mockFetch.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
        const url = input.toString();
        if (url.includes("/api/v1/models")) {
          return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
        }
        if (url.includes("/adhoc")) {
          if (init?.method === "POST") {
            postCallCount++;
            return Promise.resolve(new Response(JSON.stringify({ running: true, output: "", message: "" })));
          }
          if (url.includes("/ws1/")) {
            return Promise.resolve(new Response(JSON.stringify({ running: true, output: "", message: "" })));
          }
          return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
        }
        return Promise.resolve(new Response("{}"));
      });

      const { result, rerender } = renderHook(({ ws }: { ws: string }) =>
        useAdhocRun({ workspaceName: ws, skipModelsFetch: true }),
        { initialProps: { ws: "ws1" } },
      );

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      act(() => {
        result.current.setSelectedModel("model-a");
      });

      await act(async () => {
        result.current.startRun("test prompt", "model-a");
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.isRunning).toBe(true);

      rerender({ ws: "ws2" });

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.isRunning).toBe(false);
    });

    it("clears runError when workspaceName changes", async () => {
      mockFetch.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
        const url = input.toString();
        if (url.includes("/api/v1/models")) {
          return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
        }
        if (url.includes("/adhoc")) {
          if (init?.method === "POST") {
            return Promise.resolve(new Response("Internal Server Error", { status: 500 }));
          }
          return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
        }
        return Promise.resolve(new Response("{}"));
      });

      const { result, rerender } = renderHook(({ ws }: { ws: string }) =>
        useAdhocRun({ workspaceName: ws, skipModelsFetch: true }),
        { initialProps: { ws: "ws1" } },
      );

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      act(() => {
        result.current.setSelectedModel("model-a");
      });

      await act(async () => {
        result.current.startRun("test prompt", "model-a");
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.runError).not.toBeNull();

      rerender({ ws: "ws2" });

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.runError).toBeNull();
    });

    it("does not carry output from ws1 to ws2 after switching", async () => {
      mockFetch.mockImplementation((input: string | URL | Request, init?: RequestInit) => {
        const url = input.toString();
        if (url.includes("/api/v1/models")) {
          return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
        }
        if (url.includes("/adhoc")) {
          if (init?.method === "POST") {
            return Promise.resolve(new Response(JSON.stringify({ running: false, output: "ws1 run output", message: "" })));
          }
          if (url.includes("/ws2/")) {
            return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
          }
          return Promise.resolve(new Response(JSON.stringify({ running: false, output: "ws1 status output", message: "" })));
        }
        return Promise.resolve(new Response("{}"));
      });

      const { result, rerender } = renderHook(({ ws }: { ws: string }) =>
        useAdhocRun({ workspaceName: ws, skipModelsFetch: true }),
        { initialProps: { ws: "ws1" } },
      );

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      act(() => {
        result.current.setSelectedModel("model-a");
      });

      await act(async () => {
        result.current.startRun("test prompt", "model-a");
        await new Promise((r) => setTimeout(r, 50));
      });

      const ws1Output = result.current.output;
      expect(ws1Output).toBeTruthy();

      rerender({ ws: "ws2" });

      await act(async () => {
        await new Promise((r) => setTimeout(r, 50));
      });

      expect(result.current.output).toBe("");
      expect(result.current.isRunning).toBe(false);
    });
  });
});
