import { Square } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectOption } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { PromptHistory } from "@/components/PromptHistory";
import { useAdhocRun } from "@/hooks/useAdhocRun";

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
  const {
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
    handleSubmit,
    handleKeyDown,
    stopRun,
    outputRef,
    promptHistory,
    selectFromHistory,
    clearHistory,
  } = useAdhocRun({ workspaceName, currentModel });

  if (modelsLoading && !models) return <RunTabSkeleton />;

  if (modelsError) {
    return (
      <p className="text-sm text-destructive">
        Failed to load models: {modelsError.message}
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
        <div className="flex flex-col gap-4">
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
            <div className="flex items-center justify-between">
              <Label htmlFor="adhoc-prompt">Prompt</Label>
              <PromptHistory
                history={promptHistory}
                onSelect={selectFromHistory}
                onClear={clearHistory}
                disabled={isRunning}
              />
            </div>
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
              onClick={stopRun}
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
