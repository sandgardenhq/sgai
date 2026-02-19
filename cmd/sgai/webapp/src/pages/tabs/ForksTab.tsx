import { useState, useEffect, useRef, useCallback, useTransition, type MouseEvent } from "react";
import { useNavigate } from "react-router";
import { Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectOption } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ScrollArea } from "@/components/ui/scroll-area";
import { api, ApiError } from "@/lib/api";
import { useSSEEvent } from "@/hooks/useSSE";
import type { ApiForksResponse, ApiForkEntry, ApiForkCommit, ApiModelsResponse } from "@/types";

interface ForksTabProps {
  workspaceName: string;
}

function ForksTabSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 3 }, (_, i) => (
        <Skeleton key={i} className="h-24 w-full rounded-xl" />
      ))}
    </div>
  );
}

function ForkCommitList({ commits }: { commits: ApiForkCommit[] }) {
  if (!commits || commits.length === 0) {
    return <p className="text-xs italic text-muted-foreground">No commits to display.</p>;
  }

  return (
    <ol className="space-y-1.5 list-decimal list-inside">
      {commits.map((commit) => (
        <li key={commit.changeId} className="text-xs space-y-0.5">
          <div className="inline-flex items-center gap-2 flex-wrap">
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="font-mono text-muted-foreground cursor-help">
                  {commit.changeId.slice(0, 8)}
                </span>
              </TooltipTrigger>
              <TooltipContent>{commit.changeId} {commit.commitId}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="text-muted-foreground cursor-help">{commit.timestamp}</span>
              </TooltipTrigger>
              <TooltipContent>{commit.timestamp}</TooltipContent>
            </Tooltip>
            {commit.bookmarks && commit.bookmarks.length > 0 && (
              <span className="inline-flex gap-1">
                {commit.bookmarks.map((bm) => (
                  <Badge key={bm} variant="secondary" className="text-[0.6rem] px-1.5 py-0">
                    {bm}
                  </Badge>
                ))}
              </span>
            )}
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="text-sm truncate max-w-md cursor-help ml-4">
                {commit.description}
              </div>
            </TooltipTrigger>
            <TooltipContent className="max-w-sm">{commit.description}</TooltipContent>
          </Tooltip>
        </li>
      ))}
    </ol>
  );
}

function ForkRow({ fork, rootName, needsInput, onRefresh }: { fork: ApiForkEntry; rootName: string; needsInput: boolean; onRefresh: () => void }) {
  const navigate = useNavigate();
  const [actionError, setActionError] = useState<string | null>(null);
  const [isActionPending, startActionTransition] = useTransition();
  const respondVariant = needsInput ? "default" : "outline";

  const handleOpenEditor = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (isActionPending) return;
    setActionError(null);
    startActionTransition(async () => {
      try {
        await api.workspaces.openEditor(fork.name);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to open editor");
      }
    });
  };

  const handleDelete = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (isActionPending) return;
    const confirmed = window.confirm(`Delete fork ${fork.name}? This cannot be undone.`);
    if (!confirmed) return;
    setActionError(null);
    startActionTransition(async () => {
      try {
        await api.workspaces.deleteFork(rootName, fork.dir);
        onRefresh();
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to delete fork");
      }
    });
  };

  return (
    <div className="border rounded-lg overflow-hidden">
      <div className="flex items-center gap-4 p-4 bg-muted/20">
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="font-medium text-sm truncate max-w-[200px] cursor-help">
              {fork.name}
            </span>
          </TooltipTrigger>
          <TooltipContent>{fork.name}</TooltipContent>
        </Tooltip>

        <span className="text-xs text-muted-foreground">
          {fork.commitAhead} commits ahead
        </span>

        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant={respondVariant}
            className="min-w-[110px]"
            onClick={() => navigate(`/workspaces/${encodeURIComponent(fork.name)}/respond`)}
            disabled={isActionPending || !needsInput}
          >
            Respond
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="min-w-[110px]"
            onClick={handleOpenEditor}
            disabled={isActionPending}
          >
            Open in Editor
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="min-w-[110px]"
            onClick={() => navigate(`/workspaces/${encodeURIComponent(fork.name)}/progress`)}
            disabled={isActionPending}
          >
            Open in sgai
          </Button>
          <Button
            type="button"
            size="sm"
            variant="destructive"
            className="min-w-[110px]"
            onClick={handleDelete}
            disabled={isActionPending}
          >
            Delete
          </Button>
        </div>
      </div>

      {actionError && (
        <div className="px-4 pb-3">
          <p className="text-xs text-destructive" role="alert">{actionError}</p>
        </div>
      )}

      {fork.commits && fork.commits.length > 0 && (
        <div className="p-4 border-t">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-medium">Commits</span>
            <span className="text-xs text-muted-foreground">{fork.commitAhead} ahead</span>
          </div>
          <ScrollArea className="max-h-[200px]">
            <ForkCommitList commits={fork.commits} />
          </ScrollArea>
        </div>
      )}
    </div>
  );
}

function InlineRunBox({ workspaceName }: { workspaceName: string }) {
  const [models, setModels] = useState<ApiModelsResponse | null>(null);
  const [modelsError, setModelsError] = useState<Error | null>(null);
  const [modelsLoading, setModelsLoading] = useState(true);
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
    setModelsLoading(true);
    setModelsError(null);

    api.models
      .list(workspaceName)
      .then((response) => {
        if (!cancelled) {
          setModels(response);
          setModelsLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setModelsError(err instanceof Error ? err : new Error(String(err)));
          setModelsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName, stopPolling]);

  useEffect(() => {
    if (!models || selectedModel) return;

    const fallbackModel = models.defaultModel;
    if (fallbackModel && models.models.some((model) => model.id === fallbackModel)) {
      setSelectedModel(fallbackModel);
    }
  }, [models, selectedModel]);

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

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmedPrompt = prompt.trim();
    if (!workspaceName || !selectedModel || !trimmedPrompt) return;
    runAdhoc(trimmedPrompt, selectedModel);
  };

  if (modelsLoading && !models) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-10 w-full rounded" />
        <Skeleton className="h-32 w-full rounded" />
        <Skeleton className="h-10 w-32 rounded" />
      </div>
    );
  }

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
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="forks-adhoc-model">Model</Label>
            <div className="flex items-center gap-2">
              <Select
                id="forks-adhoc-model"
                value={selectedModel}
                onChange={(event) => setSelectedModel(event.target.value)}
                disabled={isRunning}
                className="flex-1"
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
              <Button
                type="submit"
                className="shrink-0"
                disabled={isRunning || !selectedModel || !prompt.trim()}
              >
                {isRunning ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Running...
                  </>
                ) : (
                  "Submit"
                )}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="forks-adhoc-prompt">Prompt</Label>
            <Textarea
              id="forks-adhoc-prompt"
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
              placeholder="Enter prompt..."
              rows={6}
              className="resize-y"
              disabled={isRunning}
            />
          </div>
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

export function ForksTab({ workspaceName }: ForksTabProps) {
  const [data, setData] = useState<ApiForksResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [needsInputMap, setNeedsInputMap] = useState<Record<string, boolean>>({});
  const hasLoadedRef = useRef(false);

  const workspaceEvent = useSSEEvent("workspace:update");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    if (!hasLoadedRef.current) {
      setLoading(true);
    }
    setError(null);

    Promise.all([
      api.workspaces.forks(workspaceName),
      api.workspaces.list(),
    ])
      .then(([response, listResponse]) => {
        if (!cancelled) {
          setData(response);
          const nextNeedsInput: Record<string, boolean> = {};
          for (const ws of listResponse.workspaces ?? []) {
            nextNeedsInput[ws.name] = ws.needsInput;
            if (ws.forks) {
              for (const fork of ws.forks) {
                nextNeedsInput[fork.name] = fork.needsInput;
              }
            }
          }
          setNeedsInputMap(nextNeedsInput);
          setLoading(false);
          hasLoadedRef.current = true;
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
          setLoading(false);
          hasLoadedRef.current = true;
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName, refreshKey]);

  useEffect(() => {
    if (workspaceEvent !== null) {
      setRefreshKey((k) => k + 1);
    }
  }, [workspaceEvent]);

  if (loading && !data) return <ForksTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load forks: {error.message}
      </p>
    );
  }

  if (!data) return null;

  const forks = data.forks ?? [];
  const handleRefresh = () => setRefreshKey((k) => k + 1);

  return (
    <div className="space-y-4">
      {forks.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground italic">
          <p>No forks yet. Create a fork to start work.</p>
        </div>
      ) : (
        forks.map((fork) => (
          <ForkRow
            key={fork.name}
            fork={fork}
            rootName={workspaceName}
            needsInput={needsInputMap[fork.name] ?? false}
            onRefresh={handleRefresh}
          />
        ))
      )}

      <Separator className="my-6" />

      <div>
        <h3 className="text-lg font-semibold mb-4">Ad-hoc Prompt</h3>
        <InlineRunBox workspaceName={workspaceName} />
      </div>
    </div>
  );
}
