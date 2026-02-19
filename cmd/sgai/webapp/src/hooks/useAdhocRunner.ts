import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError } from "@/lib/api";

interface UseAdhocRunnerOptions {
  workspaceName: string;
}

interface UseAdhocRunnerReturn {
  prompt: string;
  setPrompt: (prompt: string) => void;
  output: string;
  isRunning: boolean;
  error: string | null;
  outputRef: React.RefObject<HTMLPreElement>;
  run: (prompt: string, model: string) => Promise<void>;
  stop: () => Promise<void>;
  handleKeyDown: (e: React.KeyboardEvent, model: string) => void;
  reset: () => void;
}

export function useAdhocRunner({ workspaceName }: UseAdhocRunnerOptions): UseAdhocRunnerReturn {
  const [prompt, setPrompt] = useState("");
  const [output, setOutput] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const outputRef = useRef<HTMLPreElement>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [output]);

  const reset = useCallback(() => {
    stopPolling();
    setPrompt("");
    setOutput("");
    setError(null);
    setIsRunning(false);
  }, [stopPolling]);

  const run = useCallback(
    async (promptValue: string, modelValue: string) => {
      const trimmedPrompt = promptValue.trim();
      const trimmedModel = modelValue.trim();
      if (!workspaceName || isRunning || !trimmedPrompt || !trimmedModel) return;

      stopPolling();
      setIsRunning(true);
      setError(null);
      setOutput("");

      try {
        const result = await api.workspaces.adhoc(workspaceName, trimmedPrompt, trimmedModel);
        if (result.output) {
          setOutput(result.output);
        }
        if (!result.running) {
          setIsRunning(false);
          return;
        }

        pollTimerRef.current = setInterval(async () => {
          try {
            const poll = await api.workspaces.adhocStatus(workspaceName);
            if (poll.output) {
              setOutput(poll.output);
            }
            if (!poll.running) {
              stopPolling();
              setIsRunning(false);
            }
          } catch {
            stopPolling();
            setIsRunning(false);
          }
        }, 2000);
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Failed to execute ad-hoc prompt");
        }
        setIsRunning(false);
      }
    },
    [workspaceName, isRunning, stopPolling],
  );

  const stop = useCallback(async () => {
    if (!workspaceName || !isRunning) return;

    try {
      await api.workspaces.adhocStop(workspaceName);
      stopPolling();
      setIsRunning(false);
      setOutput((prev) => (prev ? prev + "\n\nStopped." : "Stopped."));
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to stop ad-hoc prompt");
      }
    }
  }, [workspaceName, isRunning, stopPolling]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent, model: string) => {
      if (e.key === "Enter" && e.shiftKey) {
        e.preventDefault();
        const trimmedPrompt = prompt.trim();
        const trimmedModel = model.trim();
        if (!workspaceName || !trimmedModel || !trimmedPrompt) return;
        run(trimmedPrompt, trimmedModel);
      }
    },
    [prompt, workspaceName, run],
  );

  return {
    prompt,
    setPrompt,
    output,
    isRunning,
    error,
    outputRef,
    run,
    stop,
    handleKeyDown,
    reset,
  };
}