import { useState, useEffect, useCallback, useReducer, useRef } from "react";
import { api, ApiError } from "@/lib/api";
import type { ApiModelsResponse } from "@/types";

const MAX_HISTORY_ENTRIES = 20;
const POLL_INTERVAL_MS = 2000;

function storageKey(workspaceName: string, suffix: string): string {
  return `adhoc-${suffix}-${workspaceName}:v1`;
}

function readLocalStorage<T>(key: string, fallback: T): T {
  try {
    const raw = window.localStorage.getItem(key);
    if (raw === null) return fallback;
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function writeLocalStorage<T>(key: string, value: T): void {
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // quota exceeded or unavailable — silently ignore
  }
}

export interface UseAdhocRunOptions {
  workspaceName: string;
  currentModel?: string;
  /** When true, skip fetching model list (AdhocOutput uses a text input) */
  skipModelsFetch?: boolean;
}

export interface UseAdhocRunResult {
  models: ApiModelsResponse | null;
  modelsLoading: boolean;
  modelsError: Error | null;
  selectedModel: string;
  setSelectedModel: (model: string) => void;
  prompt: string;
  setPrompt: (prompt: string) => void;
  output: string;
  isRunning: boolean;
  runError: string | null;
  startRun: (promptOverride?: string, modelOverride?: string) => void;
  stopRun: () => void;
  handleSubmit: (event: React.FormEvent) => void;
  handleKeyDown: (event: React.KeyboardEvent) => void;
  outputRef: React.RefObject<HTMLPreElement | null>;
  promptHistory: string[];
  selectFromHistory: (entry: string) => void;
  clearHistory: () => void;
}

export function useAdhocRun({
  workspaceName,
  currentModel,
  skipModelsFetch = false,
}: UseAdhocRunOptions): UseAdhocRunResult {
  const [{ models, modelsLoading, modelsError }, updateModelsState] = useReducer(
    (
      state: { models: ApiModelsResponse | null; modelsLoading: boolean; modelsError: Error | null },
      update: Partial<{ models: ApiModelsResponse | null; modelsLoading: boolean; modelsError: Error | null }>,
    ) => ({ ...state, ...update }),
    { models: null, modelsLoading: !skipModelsFetch, modelsError: null },
  );

  const [selectedModel, setSelectedModelState] = useState(() =>
    readLocalStorage(storageKey(workspaceName, "model"), ""),
  );
  const [prompt, setPromptState] = useState(() =>
    readLocalStorage(storageKey(workspaceName, "prompt"), ""),
  );
  const [{ output, isRunning, runError }, updateRunState] = useReducer(
    (
      state: { output: string; isRunning: boolean; runError: string | null },
      update: Partial<{ output: string; isRunning: boolean; runError: string | null }>,
    ) => ({ ...state, ...update }),
    { output: "", isRunning: false, runError: null },
  );
  const [promptHistory, setPromptHistory] = useState<string[]>(() =>
    readLocalStorage<string[]>(storageKey(workspaceName, "history"), []),
  );

  const outputRef = useRef<HTMLPreElement | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const mountedRef = useRef(true);

  // -- persist helpers --

  const setSelectedModel = useCallback(
    (model: string) => {
      setSelectedModelState(model);
      writeLocalStorage(storageKey(workspaceName, "model"), model);
    },
    [workspaceName],
  );

  const setPrompt = useCallback(
    (value: string) => {
      setPromptState(value);
      writeLocalStorage(storageKey(workspaceName, "prompt"), value);
    },
    [workspaceName],
  );

  const addToHistory = useCallback(
    (entry: string) => {
      setPromptHistory((prev) => {
        const deduped = prev.filter((p) => p !== entry);
        const next = [entry, ...deduped].slice(0, MAX_HISTORY_ENTRIES);
        writeLocalStorage(storageKey(workspaceName, "history"), next);
        return next;
      });
    },
    [workspaceName],
  );

  const clearHistory = useCallback(() => {
    setPromptHistory([]);
    writeLocalStorage(storageKey(workspaceName, "history"), []);
  }, [workspaceName]);

  const selectFromHistory = useCallback(
    (entry: string) => {
      setPrompt(entry);
    },
    [setPrompt],
  );

  // -- polling --

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      stopPolling();
    };
  }, [stopPolling]);

  // reset running state when workspace changes
  useEffect(() => {
    stopPolling();
    updateRunState({ isRunning: false, output: "", runError: null });
  }, [workspaceName, stopPolling]);

  // auto-scroll output
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [output]);

  // -- fetch models --

  useEffect(() => {
    if (skipModelsFetch || !workspaceName) return;

    let cancelled = false;
    updateModelsState({ models: null, modelsLoading: true, modelsError: null });

    api.models
      .list(workspaceName)
      .then((response) => {
        if (!cancelled) {
          updateModelsState({ models: response, modelsLoading: false });
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          updateModelsState({ modelsError: err instanceof Error ? err : new Error(String(err)), modelsLoading: false });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName, skipModelsFetch]);

  // -- auto-select default model --

  useEffect(() => {
    if (!models || selectedModel) return;
    const fallbackModel = models.defaultModel ?? currentModel;
    if (fallbackModel && models.models.some((m) => m.id === fallbackModel)) {
      setSelectedModel(fallbackModel);
    }
  }, [models, selectedModel, currentModel, setSelectedModel]);

  const startStatusPolling = useCallback(() => {
    pollTimerRef.current = setInterval(async () => {
      try {
        const poll = await api.workspaces.adhocStatus(workspaceName);
        if (!mountedRef.current) return;
        if (poll.output) updateRunState({ output: poll.output });
        if (!poll.running) { stopPolling(); updateRunState({ isRunning: false }); }
      } catch {
        stopPolling();
        updateRunState({ isRunning: false });
      }
    }, POLL_INTERVAL_MS);
  }, [workspaceName, stopPolling]);

  useEffect(() => {
    if (!workspaceName) return;
    let cancelled = false;
    api.workspaces.adhocStatus(workspaceName).then((status) => {
      if (cancelled) return;
      if (status.output) updateRunState({ output: status.output });
      if (status.running) { updateRunState({ isRunning: true }); startStatusPolling(); }
    }).catch(() => {});
    return () => { cancelled = true; };
  }, [workspaceName, startStatusPolling]);

  const startRun = useCallback(
    async (promptOverride?: string, modelOverride?: string) => {
      const trimmedPrompt = (promptOverride ?? prompt).trim();
      const trimmedModel = (modelOverride ?? selectedModel).trim();
      if (!workspaceName || isRunning || !trimmedPrompt || !trimmedModel) return;

      stopPolling();
      updateRunState({ isRunning: true, runError: null, output: "" });
      addToHistory(trimmedPrompt);

      try {
        const result = await api.workspaces.adhoc(workspaceName, trimmedPrompt, trimmedModel);
        if (result.output) updateRunState({ output: result.output });
        if (!result.running) { updateRunState({ isRunning: false }); return; }
        startStatusPolling();
      } catch (err) {
        if (err instanceof ApiError) {
          updateRunState({ runError: err.message });
        } else {
          updateRunState({ runError: "Failed to execute ad-hoc prompt" });
        }
        updateRunState({ isRunning: false });
      }
    },
    [workspaceName, isRunning, prompt, selectedModel, stopPolling, addToHistory, startStatusPolling],
  );

  const stopRun = useCallback(async () => {
    if (!workspaceName || !isRunning) return;

    try {
      await api.workspaces.adhocStop(workspaceName);
      stopPolling();
      updateRunState({ isRunning: false, output: output ? output + "\n\nStopped." : "Stopped." });
    } catch (err) {
      if (err instanceof ApiError) {
        updateRunState({ runError: err.message });
      } else {
        updateRunState({ runError: "Failed to stop ad-hoc prompt" });
      }
    }
  }, [workspaceName, isRunning, stopPolling, output]);

  const handleSubmit = useCallback(
    (event: React.FormEvent) => {
      event.preventDefault();
      startRun();
    },
    [startRun],
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        startRun();
      }
    },
    [startRun],
  );

  return {
    models,
    modelsLoading,
    modelsError,
    selectedModel,
    setSelectedModel,
    prompt,
    setPrompt,
    output,
    isRunning,
    runError,
    startRun,
    stopRun,
    handleSubmit,
    handleKeyDown,
    outputRef,
    promptHistory,
    selectFromHistory,
    clearHistory,
  };
}
