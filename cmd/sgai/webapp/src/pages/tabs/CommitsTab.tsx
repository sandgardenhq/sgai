import { useEffect, useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import { useWorkspaceSSEEvent } from "@/hooks/useSSE";
import type { ApiCommitEntry, ApiCommitsResponse } from "@/types";

interface CommitsTabProps {
  workspaceName: string;
}

function CommitsTabSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-8 w-full" />
      <Skeleton className="h-6 w-full" />
      <Skeleton className="h-6 w-full" />
      <Skeleton className="h-6 w-full" />
    </div>
  );
}

function TruncatedCell({ value, maxWidth }: { value: string; maxWidth: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`font-mono truncate block ${maxWidth}`}>{value}</span>
      </TooltipTrigger>
      <TooltipContent>{value}</TooltipContent>
    </Tooltip>
  );
}

function CommitTableRow({ entry }: { entry: ApiCommitEntry }) {
  return (
    <TableRow>
      <TableCell>
        <TruncatedCell value={entry.changeId} maxWidth="max-w-[100px]" />
      </TableCell>
      <TableCell className="hidden md:table-cell">
        <TruncatedCell value={entry.commitId} maxWidth="max-w-[100px]" />
      </TableCell>
      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
        {entry.timestamp}
      </TableCell>
      <TableCell className="hidden md:table-cell">
        <div className="flex flex-wrap gap-1">
          {entry.bookmarks?.map((bookmark) => (
            <Badge key={bookmark} variant="secondary" className="text-[0.65rem]">
              {bookmark}
            </Badge>
          ))}
        </div>
      </TableCell>
      <TableCell>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="block truncate max-w-[200px] md:max-w-[400px] text-sm">
              {entry.description || "(no description)"}
            </span>
          </TooltipTrigger>
          <TooltipContent>{entry.description || "(no description)"}</TooltipContent>
        </Tooltip>
      </TableCell>
    </TableRow>
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

  if (data.commits.length === 0) {
    return <p className="text-sm italic text-muted-foreground">No commits found</p>;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Change ID</TableHead>
          <TableHead className="hidden md:table-cell">Commit ID</TableHead>
          <TableHead>Time</TableHead>
          <TableHead className="hidden md:table-cell">Bookmarks</TableHead>
          <TableHead>Description</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {data.commits.map((entry) => (
          <CommitTableRow key={entry.changeId} entry={entry} />
        ))}
      </TableBody>
    </Table>
  );
}
