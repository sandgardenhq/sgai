import { useState, useTransition, useMemo, useCallback, type MouseEvent } from "react";
import { useNavigate } from "react-router";
import { ChevronRight, ChevronDown, Mail, SquarePen, ExternalLink, Trash2, Square } from "lucide-react";
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
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { PromptHistory } from "@/components/PromptHistory";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { useAdhocRun } from "@/hooks/useAdhocRun";
import type { ApiForkEntry, ApiForkCommit, ApiActionEntry } from "@/types";

interface ForksTabProps {
  workspaceName: string;
  actions?: ApiActionEntry[];
  onActionClick?: (action: ApiActionEntry, forkName: string) => void;
}

function ForksTabSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 3 }, (_, i) => (
        <Skeleton key={i} className="h-10 w-full rounded" />
      ))}
    </div>
  );
}

function StatusDot({ running, needsInput }: { running: boolean; needsInput: boolean }) {
  let colorClass = "bg-gray-400";
  let label = "idle";
  if (running) {
    colorClass = "bg-green-500";
    label = "running";
  } else if (needsInput) {
    colorClass = "bg-amber-400";
    label = "needs input";
  }
  return (
    <span
      className={`inline-block w-2 h-2 rounded-full shrink-0 ${colorClass}`}
      aria-label={label}
      title={label}
    />
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

interface CompactForkRowProps {
  fork: ApiForkEntry;
  rootName: string;
  needsInput: boolean;
  actions?: ApiActionEntry[];
  onActionClick?: (action: ApiActionEntry, forkName: string) => void;
}

function CompactForkRow({ fork, rootName, needsInput, actions, onActionClick }: CompactForkRowProps) {
  const navigate = useNavigate();
  const [isOpen, setIsOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [isActionPending, startActionTransition] = useTransition();

  const handleOpenEditor = useCallback((event: MouseEvent<HTMLButtonElement>) => {
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
  }, [fork.name, isActionPending]);

  const handleDelete = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (isActionPending) return;
    const confirmed = window.confirm(`Delete fork ${fork.name}? This cannot be undone.`);
    if (!confirmed) return;
    setActionError(null);
    startActionTransition(async () => {
      try {
        await api.workspaces.deleteFork(rootName, fork.dir);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to delete fork");
      }
    });
  }, [fork.name, fork.dir, rootName, isActionPending]);

  const handleRespond = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    navigate(`/workspaces/${encodeURIComponent(fork.name)}/respond`);
  }, [fork.name, navigate]);

  const handleOpenInSgai = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    navigate(`/workspaces/${encodeURIComponent(fork.name)}/progress`);
  }, [fork.name, navigate]);

  const hasCommits = fork.commits && fork.commits.length > 0;

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className="border rounded-md overflow-hidden">
        <div className="flex items-center gap-2 px-3 py-2 bg-muted/10 hover:bg-muted/20 transition-colors">
          <CollapsibleTrigger asChild>
            <button
              type="button"
              className="shrink-0 p-0.5 rounded hover:bg-muted transition-colors"
              aria-label={isOpen ? "Collapse commits" : "Expand commits"}
              disabled={!hasCommits}
            >
              {isOpen
                ? <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                : <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
              }
            </button>
          </CollapsibleTrigger>

          <StatusDot running={fork.running} needsInput={needsInput} />

          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-medium text-sm truncate max-w-[180px] cursor-default shrink-0">
                {fork.name}
              </span>
            </TooltipTrigger>
            <TooltipContent>{fork.name}</TooltipContent>
          </Tooltip>

          {fork.summary ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="text-xs text-muted-foreground truncate flex-1 min-w-0 cursor-default">
                  {fork.summary}
                </span>
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">{fork.summary}</TooltipContent>
            </Tooltip>
          ) : (
            <span className="flex-1 min-w-0" />
          )}

          <div className="flex items-center gap-1 shrink-0">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  size="icon"
                  variant={needsInput ? "default" : "ghost"}
                  className="h-7 w-7"
                  onClick={handleRespond}
                  disabled={isActionPending || !needsInput}
                  aria-label="Respond"
                >
                  <Mail className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Respond</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  onClick={handleOpenEditor}
                  disabled={isActionPending}
                  aria-label="Open in Editor"
                >
                  <SquarePen className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Open in Editor</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  onClick={handleOpenInSgai}
                  disabled={isActionPending}
                  aria-label="Open in sgai"
                >
                  <ExternalLink className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Open in sgai</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7 text-destructive hover:text-destructive"
                  onClick={handleDelete}
                  disabled={isActionPending}
                  aria-label="Delete fork"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Delete fork</TooltipContent>
            </Tooltip>
          </div>

          {actions && actions.length > 0 && (
            <div className="flex items-center gap-1 shrink-0 border-l pl-2 ml-1">
              {actions.map((action) => (
                <Tooltip key={`${action.name}-${action.model}`}>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      className="h-7 text-xs px-2"
                      disabled={isActionPending}
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        onActionClick?.(action, fork.name);
                      }}
                    >
                      {action.name}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{action.description || action.model}</TooltipContent>
                </Tooltip>
              ))}
            </div>
          )}
        </div>

        {actionError && (
          <div className="px-3 py-1.5 border-t bg-destructive/5">
            <p className="text-xs text-destructive" role="alert">{actionError}</p>
          </div>
        )}

        <CollapsibleContent>
          {hasCommits && (
            <div className="px-4 py-3 border-t bg-muted/5">
              <ScrollArea className="max-h-[150px]">
                <ForkCommitList commits={fork.commits} />
              </ScrollArea>
            </div>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function InlineRunBox({ workspaceName }: { workspaceName: string }) {
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
  } = useAdhocRun({ workspaceName });

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
        <div className="flex flex-col gap-4">
          <div className="space-y-2">
            <Label htmlFor="forks-adhoc-model">Model</Label>
            <Select
              id="forks-adhoc-model"
              value={selectedModel}
              onChange={(event) => setSelectedModel(event.target.value)}
              disabled={isRunning}
              className="w-full"
            >
              <SelectOption value="" disabled>
                Select a model
              </SelectOption>
              {models?.models?.map((model) => (
                <SelectOption key={model.id} value={model.id}>
                  {model.name}
                </SelectOption>
              ))}
            </Select>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="forks-adhoc-prompt">Prompt</Label>
              <PromptHistory
                history={promptHistory}
                onSelect={selectFromHistory}
                onClear={clearHistory}
                disabled={isRunning}
              />
            </div>
            <Textarea
              id="forks-adhoc-prompt"
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

export function ForksTab({ workspaceName, actions, onActionClick }: ForksTabProps) {
  const { workspaces: allWorkspaces, fetchStatus } = useFactoryState();

  const workspace = allWorkspaces.find((ws) => ws.name === workspaceName);

  const needsInputMap = useMemo(() => {
    const map: Record<string, boolean> = {};
    for (const ws of allWorkspaces) {
      map[ws.name] = ws.needsInput;
    }
    return map;
  }, [allWorkspaces]);

  if (fetchStatus === "fetching" && !workspace) return <ForksTabSkeleton />;

  if (!workspace) {
    if (fetchStatus === "error") {
      return (
        <p className="text-sm text-destructive">
          Failed to load forks
        </p>
      );
    }
    return null;
  }

  const forks = workspace.forks ?? [];

  return (
    <div className="space-y-4">
      {forks.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground italic">
          <p>No forks yet. Create a fork to start work.</p>
        </div>
      ) : (
        <div className="space-y-1.5">
          {forks.map((fork) => (
            <CompactForkRow
              key={fork.name}
              fork={fork}
              rootName={workspaceName}
              needsInput={needsInputMap[fork.name] ?? false}
              actions={actions}
              onActionClick={onActionClick}
            />
          ))}
        </div>
      )}

      <Separator className="my-6" />

      <div>
        <h3 className="text-lg font-semibold mb-4">Ad-hoc Prompt</h3>
        <InlineRunBox workspaceName={workspaceName} />
      </div>
    </div>
  );
}
