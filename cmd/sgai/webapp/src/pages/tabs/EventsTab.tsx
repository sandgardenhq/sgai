import { useState, useEffect, useTransition, type MouseEvent } from "react";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";
import { api } from "@/lib/api";
import { useSSEEvent } from "@/hooks/useSSE";
import type { ApiEventsResponse, ApiEventEntry, ApiModelStatusEntry } from "@/types";

interface EventsTabProps {
  workspaceName: string;
  goalContent?: string;
}

function EventsTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-24 w-full rounded-xl" />
      <div className="space-y-3">
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-10 w-full rounded" />
        ))}
      </div>
    </div>
  );
}

function WorkflowSection({ eventsData, workspaceName }: { eventsData: ApiEventsResponse; workspaceName: string }) {
  const svgUrl = `/api/v1/workspaces/${encodeURIComponent(workspaceName)}/workflow.svg${eventsData.svgHash ? `?h=${eventsData.svgHash}` : ""}`;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Work Flow</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <img
          src={svgUrl}
          alt="Workflow graph"
          className="max-w-full h-auto"
        />

        {eventsData.modelStatuses && eventsData.modelStatuses.length > 0 && (
          <ModelStatusList statuses={eventsData.modelStatuses} />
        )}

        {eventsData.needsInput && eventsData.humanMessage && (
          <div className="mt-3 p-3 border rounded-lg bg-yellow-50">
            <p className="text-sm font-medium">
              <Badge variant="default">{eventsData.currentAgent}</Badge>
            </p>
            <blockquote className="mt-2 text-sm italic border-l-2 pl-3 text-muted-foreground">
              {eventsData.humanMessage}
            </blockquote>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ModelStatusList({ statuses }: { statuses: ApiModelStatusEntry[] }) {
  return (
    <div className="text-sm">
      <strong>Model Consensus:</strong>
      <ul className="mt-1 space-y-1">
        {statuses.map((ms) => (
          <li key={ms.modelId} className="flex items-center gap-2">
            <span>
              {ms.status === "model-working" ? "◐" : ms.status === "model-done" ? "●" : "✕"}
            </span>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="cursor-help truncate max-w-[200px]">
                  {ms.modelId.split("/").pop() ?? ms.modelId}
                </span>
              </TooltipTrigger>
              <TooltipContent>{ms.modelId}</TooltipContent>
            </Tooltip>
            <span className="text-xs text-muted-foreground">({ms.status})</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function EventTimeline({ events }: { events: ApiEventEntry[] }) {
  if (!events || events.length === 0) {
    return <p className="text-sm italic text-muted-foreground">No events recorded yet</p>;
  }

  return (
    <ScrollArea className="max-h-[calc(100vh-24rem)]">
      <div className="flex flex-col">
        {events.map((event, index) => {
          const eventKey = `${event.timestamp}-${event.agent}-${event.description}`;
          return (
            <div key={eventKey}>
              {event.showDateDivider && (
                <div className="flex items-center gap-3 py-3 ml-[18px]">
                  <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider bg-background px-3 py-1 rounded-full border">
                    {event.dateDivider}
                  </span>
                </div>
              )}
              <div className="flex gap-3 py-1.5">
                <div className="flex flex-col items-center w-3 shrink-0 pt-1.5">
                  <span className="w-2 h-2 rounded-full bg-primary shrink-0 shadow-[0_0_0_3px_rgba(var(--primary),0.2)]" />
                  {index < events.length - 1 && (
                    <span className="w-0.5 flex-1 min-h-[20px] bg-border mt-1" />
                  )}
                </div>
                <div className="flex-1 min-w-0 pb-2">
                  <div className="flex items-center gap-2 flex-wrap mb-0.5">
                    <time className="text-xs text-muted-foreground whitespace-nowrap">
                      {event.formattedTime}
                    </time>
                    <Badge variant="secondary" className="text-[0.65rem] px-2 py-0">
                      {event.agent}
                    </Badge>
                  </div>
                  <span className="text-sm break-words">{event.description}</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </ScrollArea>
  );
}

export function EventsTab({ workspaceName, goalContent }: EventsTabProps) {
  const [data, setData] = useState<ApiEventsResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [goalOpenError, setGoalOpenError] = useState<string | null>(null);
  const [isGoalOpenPending, startGoalOpenTransition] = useTransition();

  const eventsEvent = useSSEEvent("events:new");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setLoading((prev) => !data ? true : prev);
    setError(null);

    api.workspaces
      .events(workspaceName)
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
    if (eventsEvent !== null) {
      setRefreshKey((k) => k + 1);
    }
  }, [eventsEvent]);

  const handleOpenGoal = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!workspaceName || isGoalOpenPending) return;
    setGoalOpenError(null);
    startGoalOpenTransition(async () => {
      try {
        await api.workspaces.openEditorGoal(workspaceName);
      } catch (err) {
        setGoalOpenError(err instanceof Error ? err.message : "Failed to open GOAL.md");
      }
    });
  };

  if (loading && !data) return <EventsTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load events: {error.message}
      </p>
    );
  }

  if (!data) return null;

  return (
    <div className="space-y-4">
      <WorkflowSection eventsData={data} workspaceName={workspaceName} />
      {goalContent && (
        <details className="group">
          <summary className="cursor-pointer font-semibold text-sm mb-2 flex items-center gap-2 list-none [&::-webkit-details-marker]:hidden">
            <ChevronRight
              className="h-4 w-4 text-muted-foreground transition-transform duration-200 group-open:rotate-90"
              aria-hidden="true"
            />
            <span>GOAL.md</span>
            <span className="ml-auto">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="Open GOAL.md in editor"
                aria-label="Open GOAL.md in editor"
                onClick={handleOpenGoal}
                disabled={isGoalOpenPending}
              >
                📝
              </Button>
            </span>
          </summary>
          <MarkdownContent
            content={goalContent}
            className="p-4 border rounded-lg bg-muted/20"
          />
          {goalOpenError && (
            <p className="text-xs text-destructive mt-2" role="alert">
              {goalOpenError}
            </p>
          )}
        </details>
      )}
      <Card>
        <CardContent className="p-4">
          <EventTimeline events={data.events ?? []} />
        </CardContent>
      </Card>
    </div>
  );
}
