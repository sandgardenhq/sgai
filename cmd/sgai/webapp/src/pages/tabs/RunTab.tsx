import { useState, useEffect, useCallback, useRef } from "react";
import { Square } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectOption } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api, ApiError } from "@/lib/api";
import type { ApiModelsResponse } from "@/types";

interface RunTabProps {
  workspaceName: string;
  currentModel?: string;
}

function RunTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-10 w-full rounded" />
      <Skeleton className="h-32 w-full rounded" />
      <Skeleton className="h-10 w-32 rounded" />
    </div>
  );
}

export function RunTab({ workspaceName, currentModel }: RunTabProps): JSX.Element | null {
  const [models, setModels] = useState<ApiModelsResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedModel, setSelectedModel] = useState("");
  const [prompt, setPrompt] = useState("");
  const [output, setOutput] = useState("");
  const [runError, setRunError] = useState<string | null>(null);
  const [isRunning, setIsRunning] = useState(false);
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

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setModels(null);
    setSelectedModel("");
    setPrompt("");
    setOutput("");
    setRunError(null);
    setIsRunning(false);
    stopPolling();
    setLoading(true);
    setError(null);

    api.models
      .list(workspaceName)
      .then((response) => {
        if (!cancelled) {
          setModels(response);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName]);

  useEffect(() => {
    if (!models || selectedModel) return;

    const fallbackModel = models.defaultModel ?? currentModel;
    if (fallbackModel && models.models.some((model) => model.id === fallbackModel)) {
      setSelectedModel(fallbackModel);
    }
  }, [models, selectedModel, currentModel]);

  const runAdhoc = useCallback(
    async (promptValue: string, modelValue: string) => {
      const trimmedPrompt = promptValue.trim();
      const trimmedModel = modelValue.trim();
      if (!workspaceName || isRunning || !trimmedPrompt || !trimmedModel) return;

      stopPolling();
      setIsRunning(true);
      setRunError(null);
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
          setRunError(err.message);
        } else {
          setRunError("Failed to execute ad-hoc prompt");
        }
        setIsRunning(false);
      }
    },
    [workspaceName, isRunning, stopPolling],
  );

  const handleStop = useCallback(async () => {
    if (!workspaceName || !isRunning) return;

    try {
      await api.workspaces.adhocStop(workspaceName);
      stopPolling();
      setIsRunning(false);
      setOutput((prev) => (prev ? prev + "\n\nStopped." : "Stopped."));
    } catch (err) {
      if (err instanceof ApiError) {
        setRunError(err.message);
      } else {
        setRunError("Failed to stop ad-hoc prompt");
      }
    }
  }, [workspaceName, isRunning, stopPolling]);

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmedPrompt = prompt.trim();
    if (!workspaceName || !selectedModel || !trimmedPrompt) return;
    runAdhoc(trimmedPrompt, selectedModel);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  if (loading && !models) return <RunTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load models: {error.message}
      </p>
    );
  }

  if (!models) return null;

  return (
    <div className="space-y-4">
      {runError ? (
        <Alert className="border-destructive/50 text-destructive">
          <AlertDescription>{runError}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-[200px_1fr] gap-4">
          <div className="space-y-2">
            <Label htmlFor="adhoc-model">Model</Label>
            <Select
              id="adhoc-model"
              value={selectedModel}
              onChange={(event) => setSelectedModel(event.target.value)}
              disabled={isRunning}
              className="w-full"
            >
              <SelectOption value="" disabled>
                Select a model
              </SelectOption>
              {models.models.map((model) => (
                <SelectOption key={model.id} value={model.id}>
                  {model.name}
                </SelectOption>
              ))}
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="adhoc-prompt">Prompt</Label>
            <Textarea
              id="adhoc-prompt"
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Enter prompt..."
              rows={6}
              className="resize-y"
              disabled={isRunning}
            />
          </div>
        </div>

        <div className="flex gap-2">
          {isRunning ? (
            <Button
              type="button"
              variant="destructive"
              onClick={handleStop}
            >
              <Square className="mr-2 h-4 w-4" />
              Stop
            </Button>
          ) : (
            <Button
              type="submit"
              disabled={!selectedModel || !prompt.trim()}
            >
              Submit
            </Button>
          )}
        </div>
      </form>

      {(isRunning || output) ? (
        <div className="space-y-2">
          <Label>Output</Label>
          <pre
            ref={outputRef}
            className="bg-muted rounded-md p-4 text-sm font-mono overflow-auto max-h-[400px] whitespace-pre-wrap"
          >
            {output}
          </pre>
        </div>
      ) : null}
    </div>
  );
}
