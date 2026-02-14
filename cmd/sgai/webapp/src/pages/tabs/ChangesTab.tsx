import { useState, useEffect, useTransition } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { useWorkspaceSSEEvent } from "@/hooks/useSSE";
import { cn } from "@/lib/utils";
import type { ApiChangesResponse, ApiDiffLine } from "@/types";

interface ChangesTabProps {
  workspaceName: string;
}

function ChangesTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-12 w-full rounded" />
      <Skeleton className="h-64 w-full rounded-xl" />
    </div>
  );
}

function diffLineColor(line: ApiDiffLine): string {
  switch (line.class) {
    case "add":
      return "border-l-4 border-green-500 text-green-700 bg-green-50";
    case "remove":
      return "border-l-4 border-red-500 text-red-700 bg-red-50";
    case "header":
      return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50 font-semibold";
    case "range":
      return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50";
    default:
      break;
  }

  const trimmed = line.text.trimStart();
  if (trimmed.startsWith("@@")) {
    return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50 font-semibold";
  }
  if (trimmed.startsWith("+++ ") || trimmed.startsWith("--- ")) {
    return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50";
  }
  if (trimmed.startsWith("+") && !trimmed.startsWith("+++")) {
    return "border-l-4 border-green-500 text-green-700 bg-green-50";
  }
  if (trimmed.startsWith("-") && !trimmed.startsWith("---")) {
    return "border-l-4 border-red-500 text-red-700 bg-red-50";
  }
  return "border-l-4 border-transparent";
}

function DiffLineRow({ line }: { line: ApiDiffLine }) {
  return (
    <div
      className={cn("font-mono text-xs leading-5 whitespace-pre-wrap break-all px-2", diffLineColor(line))}
      data-line-number={line.lineNumber}
    >
      {line.text}
    </div>
  );
}

export function ChangesTab({ workspaceName }: ChangesTabProps) {
  const [data, setData] = useState<ApiChangesResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [description, setDescription] = useState("");
  const [updateError, setUpdateError] = useState<string | null>(null);
  const [updateSuccess, setUpdateSuccess] = useState(false);
  const [isUpdating, startTransition] = useTransition();

  const changesEvent = useWorkspaceSSEEvent(workspaceName, "changes:update");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setLoading((prev) => !data ? true : prev);
    setError(null);

    api.workspaces
      .changes(workspaceName)
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

  useEffect(() => {
    if (data) {
      setDescription(data.description ?? "");
    }
  }, [data]);

  if (loading && !data) return <ChangesTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load changes: {error.message}
      </p>
    );
  }

  if (!data) return null;

  const diffLines = data.diffLines ?? [];

  const handleUpdate = (event: React.FormEvent) => {
    event.preventDefault();
    if (!description.trim() || isUpdating) return;

    setUpdateError(null);
    setUpdateSuccess(false);
    startTransition(async () => {
      try {
        const response = await api.workspaces.updateDescription(workspaceName, description.trim());
        setDescription(response.description);
        setUpdateSuccess(true);
      } catch (err) {
        setUpdateError(err instanceof Error ? err.message : "Failed to update description");
      }
    });
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardContent>
          <form onSubmit={handleUpdate} className="space-y-3">
            <div className="space-y-2">
              <Input
                id="commit-description"
                value={description}
                onChange={(event) => {
                  setDescription(event.target.value);
                  setUpdateSuccess(false);
                }}
                placeholder="Enter commit message"
                aria-label="Commit Description"
                disabled={isUpdating}
              />
            </div>
            {updateError && (
              <p className="text-sm text-destructive">{updateError}</p>
            )}
            {updateSuccess && !updateError && (
              <p className="text-sm text-primary">Description updated.</p>
            )}
            <Button type="submit" disabled={isUpdating || !description.trim()}>
              Update
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Diff</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {diffLines.length > 0 ? (
            <ScrollArea className="max-h-[calc(100vh-20rem)]">
              <div className="divide-y divide-border/30">
                {diffLines.map((line) => (
                  <DiffLineRow key={line.lineNumber} line={line} />
                ))}
              </div>
            </ScrollArea>
          ) : (
            <p className="text-sm italic text-muted-foreground px-6 pb-4">No changes to display</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
