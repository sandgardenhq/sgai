import { useEffect, useRef } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { ArrowLeft, Play, Square } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { PromptHistory } from "@/components/PromptHistory";
import { useAdhocRun } from "@/hooks/useAdhocRun";

export function AdhocOutput(): JSX.Element {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const [searchParams] = useSearchParams();
  const autoRunRef = useRef(false);

  const {
    selectedModel: model,
    setSelectedModel: setModel,
    prompt,
    setPrompt,
    output,
    isRunning,
    runError: error,
    startRun,
    stopRun,
    handleSubmit,
    handleKeyDown,
    outputRef,
    promptHistory,
    selectFromHistory,
    clearHistory,
  } = useAdhocRun({ workspaceName, skipModelsFetch: true });

  useEffect(() => {
    if (autoRunRef.current) return;
    const promptParam = searchParams.get("prompt");
    const modelParam = searchParams.get("model");
    if (!promptParam || !modelParam) return;

    autoRunRef.current = true;
    setPrompt(promptParam);
    setModel(modelParam);
    startRun(promptParam, modelParam);
  }, [searchParams, startRun, setPrompt, setModel]);

  return (
    <div className="max-w-3xl mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to {workspaceName}
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Ad-hoc Prompt</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Execute an ad-hoc prompt against <span className="font-medium text-foreground">{workspaceName}</span>.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4 mb-6">
        <div className="flex flex-col gap-4">
          <div className="space-y-2">
            <Label htmlFor="adhoc-model">Model</Label>
            <Input
              id="adhoc-model"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              placeholder="e.g., anthropic/claude-opus-4-6"
              disabled={isRunning}
            />
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
              onChange={(e) => setPrompt(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Enter your prompt..."
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
              disabled={!prompt.trim() || !model.trim()}
            >
              <Play className="mr-2 h-4 w-4" />
              Execute Prompt
            </Button>
          )}
        </div>
      </form>

      {output ? (
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
