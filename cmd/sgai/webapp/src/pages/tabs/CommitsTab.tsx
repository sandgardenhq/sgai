import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import { useWorkspaceSSEEvent } from "@/hooks/useSSE";
import type { ApiCommitEntry, ApiCommitsResponse } from "@/types";

interface CommitsTabProps {
  workspaceName: string;
}

function CommitsTabSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-8 w-32" />
      <Skeleton className="h-20 w-full rounded-xl" />
      <Skeleton className="h-20 w-full rounded-xl" />
    </div>
  );
}

function CommitRow({ entry }: { entry: ApiCommitEntry }) {
  return (
    <div className="flex gap-3 py-3 border-b last:border-b-0">
      <span className="font-mono text-muted-foreground w-4 shrink-0">{entry.graphChar}</span>
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono font-semibold text-foreground truncate max-w-[120px]">
                {entry.changeId}
              </span>
            </TooltipTrigger>
            <TooltipContent>{entry.changeId}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono truncate max-w-[120px]">
                {entry.commitId}
              </span>
            </TooltipTrigger>
            <TooltipContent>{entry.commitId}</TooltipContent>
          </Tooltip>
          <span className="whitespace-nowrap">{entry.timestamp}</span>
          {entry.bookmarks?.map((bookmark) => (
            <Badge key={bookmark} variant="secondary" className="text-[0.65rem]">
              {bookmark}
            </Badge>
          ))}
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <p className="text-sm mt-1 truncate max-w-full">
              {entry.description || "(no description)"}
            </p>
          </TooltipTrigger>
          <TooltipContent>{entry.description || "(no description)"}</TooltipContent>
        </Tooltip>
        {entry.workspaces && entry.workspaces.length > 0 && (
          <div className="mt-1 text-xs text-muted-foreground">
            Workspaces: {entry.workspaces.join(", ")}
          </div>
        )}
      </div>
    </div>
  );
}

export function CommitsTab({ workspaceName }: CommitsTabProps) {
  const [data, setData] = useState<ApiCommitsResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  const changesEvent = useWorkspaceSSEEvent(workspaceName, "changes:update");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setLoading((prev) => !data ? true : prev);
    setError(null);

    api.workspaces
      .commits(workspaceName)
      .then((response) => {
        if (!cancelled) {
          setData(response);
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
  }, [workspaceName, refreshKey]);

  useEffect(() => {
    if (changesEvent !== null) {
      setRefreshKey((k) => k + 1);
    }
  }, [changesEvent]);

  if (loading && !data) return <CommitsTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load commits: {error.message}
      </p>
    );
  }

  if (!data) return null;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Commits</CardTitle>
      </CardHeader>
      <CardContent>
        {data.commits.length === 0 ? (
          <p className="text-sm italic text-muted-foreground">No commits found</p>
        ) : (
          <div className="divide-y divide-border/50">
            {data.commits.map((entry, index) => (
              <CommitRow key={`${entry.changeId}-${index}`} entry={entry} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
