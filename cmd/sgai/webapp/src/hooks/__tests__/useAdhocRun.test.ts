import { describe, it, expect, beforeEach, mock } from "bun:test";
import { renderHook, act } from "@testing-library/react";
import { useAdhocRun } from "@/hooks/useAdhocRun";

const mockAdhoc = mock(() => Promise.resolve({ output: "result", running: false }));
const mockAdhocStatus = mock(() => Promise.resolve({ output: "", running: false }));
const mockAdhocStop = mock(() => Promise.resolve({ output: "Stopped.", running: false }));
const mockModelsList = mock(() => Promise.resolve({ models: [{ id: "model-1" }], defaultModel: "model-1" }));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
      adhoc: mockAdhoc,
      adhocStatus: mockAdhocStatus,
      adhocStop: mockAdhocStop,
    },
    models: {
      list: mockModelsList,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

beforeEach(() => {
  localStorage.clear();
  mockAdhoc.mockClear();
  mockAdhocStatus.mockClear();
  mockAdhocStop.mockClear();
  mockModelsList.mockClear();
  mockAdhoc.mockImplementation(() => Promise.resolve({ output: "result", running: false }));
  mockAdhocStatus.mockImplementation(() => Promise.resolve({ output: "", running: false }));
  mockAdhocStop.mockImplementation(() => Promise.resolve({ output: "Stopped.", running: false }));
  mockModelsList.mockImplementation(() => Promise.resolve({ models: [{ id: "model-1" }], defaultModel: "model-1" }));
});

describe("useAdhocRun", () => {
  describe("initial state", () => {
    it("returns initial state values", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      expect(result.current.output).toBe("");
      expect(result.current.isRunning).toBe(false);
      expect(result.current.runError).toBeNull();
      expect(result.current.prompt).toBe("");
      expect(result.current.selectedModel).toBe("");
    });

    it("returns empty prompt history by default", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      expect(result.current.promptHistory).toEqual([]);
    });
  });

  describe("setPrompt", () => {
    it("updates prompt value", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("new prompt");
      });

      expect(result.current.prompt).toBe("new prompt");
    });

    it("persists prompt to localStorage", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("saved prompt");
      });

      expect(localStorage.getItem("adhoc-prompt-test-ws")).toBe('"saved prompt"');
    });
  });

  describe("setSelectedModel", () => {
    it("updates selected model", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setSelectedModel("model-2");
      });

      expect(result.current.selectedModel).toBe("model-2");
    });

    it("persists model to localStorage", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setSelectedModel("model-2");
      });

      expect(localStorage.getItem("adhoc-model-test-ws")).toBe('"model-2"');
    });
  });

  describe("clearHistory", () => {
    it("clears prompt history", () => {
      localStorage.setItem("adhoc-history-test-ws", JSON.stringify(["old prompt"]));

      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.clearHistory();
      });

      expect(result.current.promptHistory).toEqual([]);
      expect(JSON.parse(localStorage.getItem("adhoc-history-test-ws") || "[]")).toEqual([]);
    });
  });

  describe("selectFromHistory", () => {
    it("sets prompt to selected history entry", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.selectFromHistory("old prompt");
      });

      expect(result.current.prompt).toBe("old prompt");
    });
  });

  describe("handleSubmit", () => {
    it("prevents default form submission", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      const preventDefault = mock(() => {});
      act(() => {
        result.current.handleSubmit({ preventDefault } as any);
      });

      expect(preventDefault).toHaveBeenCalled();
    });
  });

  describe("handleKeyDown", () => {
    it("starts run on Ctrl+Enter", () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("test prompt");
        result.current.setSelectedModel("model-1");
      });

      const preventDefault = mock(() => {});
      act(() => {
        result.current.handleKeyDown({
          key: "Enter",
          ctrlKey: true,
          metaKey: false,
          preventDefault,
        } as any);
      });

      expect(preventDefault).toHaveBeenCalled();
    });
  });

  describe("startRun", () => {
    it("does not run when prompt is empty", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setSelectedModel("model-1");
      });

      await act(async () => {
        result.current.startRun();
      });

      expect(mockAdhoc).not.toHaveBeenCalled();
    });

    it("does not run when model is empty", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("test");
      });

      await act(async () => {
        result.current.startRun();
      });

      expect(mockAdhoc).not.toHaveBeenCalled();
    });

    it("calls adhoc API with prompt and model", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("test prompt");
        result.current.setSelectedModel("model-1");
      });

      await act(async () => {
        result.current.startRun();
      });

      expect(mockAdhoc).toHaveBeenCalledWith("test-ws", "test prompt", "model-1");
    });

    it("uses overrides when provided", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("ignored");
        result.current.setSelectedModel("ignored-model");
      });

      await act(async () => {
        result.current.startRun("override prompt", "override-model");
      });

      expect(mockAdhoc).toHaveBeenCalledWith("test-ws", "override prompt", "override-model");
    });

    it("sets output after successful run", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("test");
        result.current.setSelectedModel("model-1");
      });

      await act(async () => {
        result.current.startRun();
      });

      expect(result.current.output).toBe("result");
    });

    it("sets runError when API fails", async () => {
      mockAdhoc.mockImplementation(() => Promise.reject(new Error("Failed")));

      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("test");
        result.current.setSelectedModel("model-1");
      });

      await act(async () => {
        result.current.startRun();
      });

      expect(result.current.runError).toBe("Failed to execute ad-hoc prompt");
      expect(result.current.isRunning).toBe(false);
    });

    it("adds prompt to history on run", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      act(() => {
        result.current.setPrompt("test prompt");
        result.current.setSelectedModel("model-1");
      });

      await act(async () => {
        result.current.startRun();
      });

      expect(result.current.promptHistory).toContain("test prompt");
    });
  });

  describe("models fetching", () => {
    it("fetches models when skipModelsFetch is false", async () => {
      renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: false })
      );

      await new Promise((r) => setTimeout(r, 100));
      expect(mockModelsList).toHaveBeenCalledWith("test-ws");
    });

    it("skips fetching models when skipModelsFetch is true", async () => {
      renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      await new Promise((r) => setTimeout(r, 100));
      expect(mockModelsList).not.toHaveBeenCalled();
    });
  });

  describe("stopRun", () => {
    it("does nothing when not running", async () => {
      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      await act(async () => {
        result.current.stopRun();
      });

      expect(mockAdhocStop).not.toHaveBeenCalled();
    });
  });

  describe("localStorage persistence", () => {
    it("restores prompt from localStorage", () => {
      localStorage.setItem("adhoc-prompt-test-ws", '"saved prompt"');

      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      expect(result.current.prompt).toBe("saved prompt");
    });

    it("restores model from localStorage", () => {
      localStorage.setItem("adhoc-model-test-ws", '"saved-model"');

      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      expect(result.current.selectedModel).toBe("saved-model");
    });

    it("restores history from localStorage", () => {
      localStorage.setItem("adhoc-history-test-ws", '["prompt1","prompt2"]');

      const { result } = renderHook(() =>
        useAdhocRun({ workspaceName: "test-ws", skipModelsFetch: true })
      );

      expect(result.current.promptHistory).toEqual(["prompt1", "prompt2"]);
    });
  });
});
