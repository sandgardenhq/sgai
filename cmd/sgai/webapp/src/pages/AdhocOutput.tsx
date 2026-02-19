import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { ArrowLeft, Play, Square } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useAdhocRunner } from "@/hooks/useAdhocRunner";

export function AdhocOutput(): JSX.Element {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const [searchParams] = useSearchParams();
  const [model, setModel] = useState("");
  const autoRunRef = useRef(false);

  const {
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
  } = useAdhocRunner({ workspaceName });

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      run(prompt, model);
    },
    [run, prompt, model],
  );

  useEffect(() => {
    if (autoRunRef.current) return;
    const promptParam = searchParams.get("prompt");
    const modelParam = searchParams.get("model");
    if (!promptParam || !modelParam) return;

    autoRunRef.current = true;
    setPrompt(promptParam);
    setModel(modelParam);
    run(promptParam, modelParam);
  }, [searchParams, setPrompt, run]);

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
        <div className="grid grid-cols-[200px_1fr] gap-4">
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
            <Label htmlFor="adhoc-prompt">Prompt</Label>
            <Textarea
              id="adhoc-prompt"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              onKeyDown={(e) => handleKeyDown(e, model)}
              placeholder="Enter your prompt..."
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