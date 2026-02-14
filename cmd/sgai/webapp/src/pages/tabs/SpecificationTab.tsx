import { useState, useEffect } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { MarkdownContent } from "@/components/MarkdownContent";
import { api } from "@/lib/api";
import { useWorkspaceSSEEvent } from "@/hooks/useSSE";
import type { ApiWorkspaceDetailResponse } from "@/types";

interface SpecificationTabProps {
  workspaceName: string;
}

function SpecificationTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-6 w-32" />
      <Skeleton className="h-48 w-full rounded-xl" />
      <Skeleton className="h-6 w-48" />
      <Skeleton className="h-32 w-full rounded-xl" />
    </div>
  );
}

export function SpecificationTab({ workspaceName }: SpecificationTabProps) {
  const [detail, setDetail] = useState<ApiWorkspaceDetailResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  const sessionUpdateEvent = useWorkspaceSSEEvent(workspaceName, "session:update");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setLoading((prev) => !detail ? true : prev);
    setError(null);

    api.workspaces
      .get(workspaceName)
      .then((response) => {
        if (!cancelled) {
          setDetail(response);
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
    if (sessionUpdateEvent !== null) {
      setRefreshKey((k) => k + 1);
    }
  }, [sessionUpdateEvent]);

  if (loading && !detail) return <SpecificationTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load specification: {error.message}
      </p>
    );
  }

  if (!detail) return null;

  return (
    <div className="space-y-4">
      {detail.goalContent ? (
        <details open>
          <summary className="cursor-pointer font-semibold text-sm mb-2">
            GOAL.md
          </summary>
          <MarkdownContent
            content={detail.goalContent}
            className="p-4 border rounded-lg bg-muted/20"
          />
        </details>
      ) : (
        <div className="text-center py-8 text-muted-foreground italic">
          <p>No GOAL.md file found</p>
        </div>
      )}

      {detail.hasProjectMgmt && detail.pmContent && (
        <details open>
          <summary className="cursor-pointer font-semibold text-sm mb-2">
            PROJECT_MANAGEMENT.md
          </summary>
          <MarkdownContent
            content={detail.pmContent}
            className="p-4 border rounded-lg bg-muted/20"
          />
        </details>
      )}
    </div>
  );
}
