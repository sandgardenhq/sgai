import { useState, useEffect, useCallback, useRef } from "react";
import { Square } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectOption } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useAdhocRunner } from "@/hooks/useAdhocRunner";
import { api } from "@/lib/api";
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

  const {
    prompt,
    setPrompt,
    output,
    isRunning,
    error: runError,
    outputRef,
    run,
    stop,
    handleKeyDown,
    reset,
  } = useAdhocRunner({ workspaceName });

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setModels(null);
    setSelectedModel("");
    reset();
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
  }, [workspaceName, reset]);

  useEffect(() => {
    if (!models || selectedModel) return;

    const fallbackModel = models.defaultModel ?? currentModel;
    if (fallbackModel && models.models.some((model) => model.id === fallbackModel)) {
      setSelectedModel(fallbackModel);
    }
  }, [models, selectedModel, currentModel]);

  const handleSubmit = useCallback(
    (event: React.FormEvent) => {
      event.preventDefault();
      const trimmedPrompt = prompt.trim();
      if (!workspaceName || !selectedModel || !trimmedPrompt) return;
      run(trimmedPrompt, selectedModel);
    },
    [workspaceName, selectedModel, prompt, run],
  );

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
              onKeyDown={(e) => handleKeyDown(e, selectedModel)}
              placeholder="Enter prompt..."
              rows={6}
              className="resize-y"
              disabled={isRunning}
            />
            <p className="text-xs text-muted-foreground">
              Press Shift+Enter to submit
            </p>
          </div>
        </div>

        <div className="flex gap-2">
          {isRunning ? (
            <Button
              type="button"
              variant="destructive"
              onClick={stop}
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