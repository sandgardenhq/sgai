import { useState, useEffect, useRef, useTransition } from "react";
import { useNavigate } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MarkdownContent } from "@/components/MarkdownContent";
import { cn } from "@/lib/utils";
import { api, ApiError } from "@/lib/api";
import type { ApiRetrospectivesResponse, ApiRetroSession, ApiRetroDetail } from "@/types";

interface RetrospectivesTabProps {
  workspaceName: string;
}

function RetrospectivesTabSkeleton() {
  return (
    <div className="flex gap-4 h-[calc(100vh-20rem)]">
      <div className="w-[200px] space-y-2">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-8 w-full rounded" />
        ))}
      </div>
      <div className="flex-1">
        <Skeleton className="h-48 w-full rounded-xl" />
      </div>
    </div>
  );
}

function SessionList({
  sessions,
  selectedSession,
  onSelectSession,
}: {
  sessions: ApiRetroSession[];
  selectedSession: string;
  onSelectSession: (name: string) => void;
}) {
  if (!sessions || sessions.length === 0) {
    return <p className="text-sm italic text-muted-foreground p-2">No retrospective sessions found</p>;
  }

  return (
    <ScrollArea className="h-full">
      <div className="space-y-1">
        {sessions.map((session) => (
          <Tooltip key={session.name}>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={() => onSelectSession(session.name)}
                className={cn(
                  "w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-2 transition-colors",
                  selectedSession === session.name
                    ? "bg-primary/10 text-primary font-medium"
                    : "hover:bg-muted text-foreground"
                )}
              >
                <span
                  className={cn(
                    "w-2 h-2 rounded-full shrink-0",
                    session.hasImprovements ? "bg-green-500" : "bg-muted-foreground/30"
                  )}
                />
                <span className="truncate">{session.name}</span>
              </button>
            </TooltipTrigger>
            <TooltipContent>{session.name}</TooltipContent>
          </Tooltip>
        ))}
      </div>
    </ScrollArea>
  );
}

function RetroDetailView({
  detail,
  workspaceName,
  onAnalyze,
  onDelete,
  isAnalyzeDisabled,
  isDeleteDisabled,
  actionError,
}: {
  detail: ApiRetroDetail;
  workspaceName: string;
  onAnalyze: () => void;
  onDelete: () => void;
  isAnalyzeDisabled: boolean;
  isDeleteDisabled: boolean;
  actionError: string | null;
}) {
  const navigate = useNavigate();
  const isApplyDisabled = isAnalyzeDisabled || isDeleteDisabled || !detail.hasImprovements;
  const handleApply = () => {
    if (isApplyDisabled) return;
    const target = `/workspaces/${encodeURIComponent(workspaceName)}/retrospective/apply?session=${encodeURIComponent(detail.sessionName)}`;
    navigate(target);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h4 className="text-lg font-semibold m-0">{detail.sessionName}</h4>
        {!detail.isAnalyzing && (
          <Badge variant={detail.hasImprovements ? "default" : "secondary"}>
            {detail.hasImprovements ? "Analyzed" : "Not Analyzed"}
          </Badge>
        )}
        {detail.isAnalyzing && <Badge variant="secondary">Analyzing...</Badge>}
        {detail.isApplying && (
          <Badge variant="secondary">Applying...</Badge>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="default"
          onClick={onAnalyze}
          disabled={isAnalyzeDisabled}
        >
          Analyze
        </Button>
        <Button
          type="button"
          variant="secondary"
          onClick={handleApply}
          disabled={isApplyDisabled}
        >
          Apply
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={onDelete}
          disabled={isDeleteDisabled}
        >
          Delete
        </Button>
      </div>

      {actionError && (
        <p className="text-sm text-destructive" role="alert">
          {actionError}
        </p>
      )}

      {detail.goalContent ? (
        <div>
          <h5 className="text-sm font-semibold mb-2">GOAL.md</h5>
          <MarkdownContent
            content={detail.goalContent}
            className="p-4 border rounded-lg bg-muted/20"
          />
        </div>
      ) : detail.goalSummary ? (
        <div>
          <h5 className="text-sm font-semibold mb-2">Goal Summary</h5>
          <p className="text-sm p-4 border rounded-lg bg-muted/20">{detail.goalSummary}</p>
        </div>
      ) : (
        <p className="text-sm italic text-muted-foreground">No GOAL.md found for this session</p>
      )}

      {detail.improvements && (
        <div>
          <h5 className="text-sm font-semibold mb-2">IMPROVEMENTS.md</h5>
          <MarkdownContent
            content={detail.improvements}
            className="p-4 border rounded-lg bg-muted/20"
          />
        </div>
      )}
    </div>
  );
}

export function RetrospectivesTab({ workspaceName }: RetrospectivesTabProps) {
  const [data, setData] = useState<ApiRetrospectivesResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedSession, setSelectedSession] = useState<string>("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const [isAnalyzePending, startAnalyzeTransition] = useTransition();
  const [isDeletePending, startDeleteTransition] = useTransition();
  const navigate = useNavigate();
  const hasLoadedRef = useRef(false);

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    if (!hasLoadedRef.current) {
      setLoading(true);
    }
    setError(null);

    api.workspaces
      .retrospectives(workspaceName, selectedSession || undefined)
      .then((response) => {
        if (!cancelled) {
          setData(response);
          if (!selectedSession && response.selectedSession) {
            setSelectedSession(response.selectedSession);
          }
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
  }, [workspaceName, selectedSession, refreshKey]);

  if (loading && !data) return <RetrospectivesTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load retrospectives: {error.message}
      </p>
    );
  }

  if (!data) return null;

  const handleAnalyze = () => {
    if (!workspaceName || !data?.details) return;

    setActionError(null);
    const target = `/workspaces/${encodeURIComponent(workspaceName)}/retro/${encodeURIComponent(data.details.sessionName)}/analyze`;
    startAnalyzeTransition(() => {
      navigate(target);
    });
  };

  const handleDelete = () => {
    if (!workspaceName || !data?.details) return;

    setActionError(null);
    startDeleteTransition(async () => {
      try {
        await api.workspaces.retroDelete(workspaceName, data.details.sessionName);
        setSelectedSession("");
        setRefreshKey((k) => k + 1);
      } catch (err) {
        if (err instanceof ApiError) {
          setActionError(err.message);
        } else {
          setActionError("Failed to delete retrospective");
        }
      }
    });
  };

  const isAnalyzeDisabled = Boolean(
    data.details?.isAnalyzing || data.details?.isApplying || isAnalyzePending || isDeletePending,
  );
  const isDeleteDisabled = Boolean(
    data.details?.isAnalyzing || data.details?.isApplying || isAnalyzePending || isDeletePending,
  );

  return (
    <div className="flex gap-4 h-[calc(100vh-20rem)] min-h-[300px]">
      <aside className="w-[200px] border-r pr-2 shrink-0">
        <SessionList
          sessions={data.sessions ?? []}
          selectedSession={selectedSession}
          onSelectSession={setSelectedSession}
        />
      </aside>
      <main className="flex-1 overflow-auto">
        {data.details ? (
          <RetroDetailView
            detail={data.details}
            workspaceName={workspaceName}
            onAnalyze={handleAnalyze}
            onDelete={handleDelete}
            isAnalyzeDisabled={isAnalyzeDisabled}
            isDeleteDisabled={isDeleteDisabled}
            actionError={actionError}
          />
        ) : (
          <p className="text-sm italic text-muted-foreground">
            Select a session to view details
          </p>
        )}
      </main>
    </div>
  );
}
