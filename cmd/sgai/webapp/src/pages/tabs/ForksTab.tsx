import { useState, useEffect, useRef, useTransition, type MouseEvent } from "react";
import { useNavigate } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ScrollArea } from "@/components/ui/scroll-area";
import { api } from "@/lib/api";
import { useSSEEvent } from "@/hooks/useSSE";
import type { ApiForksResponse, ApiForkEntry, ApiForkCommit } from "@/types";

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

  const handleMerge = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (isActionPending) return;
    setActionError(null);
    startActionTransition(async () => {
      try {
        await api.workspaces.merge(rootName, fork.dir);
        onRefresh();
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to merge fork");
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
            variant="outline"
            className="min-w-[110px]"
            onClick={handleMerge}
            disabled={isActionPending}
          >
            Merge
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

  if (forks.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground italic">
        <p>No forks yet. Create a fork to start work.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {forks.map((fork) => (
        <ForkRow
          key={fork.name}
          fork={fork}
          rootName={workspaceName}
          needsInput={needsInputMap[fork.name] ?? false}
          onRefresh={handleRefresh}
        />
      ))}
    </div>
  );
}
