import { useState, useTransition, type MouseEvent } from "react";
import { ChevronRight, Square, ArrowRight } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip";
import { MarkdownContent } from "@/components/MarkdownContent";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { useAdhocRun } from "@/hooks/useAdhocRun";
import { ActionBar } from "./SessionTab";
import type { ApiEventEntry, ApiActionEntry, ApiActiveAgentEntry } from "@/types";

interface EventsTabProps {
  workspaceName: string;
  goalContent?: string;
  actions?: ApiActionEntry[];
}

function EventsTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-24 w-full rounded-xl" />
      <div className="space-y-3">
        {["event-1", "event-2", "event-3", "event-4", "event-5"].map((key) => (
          <Skeleton key={key} className="h-10 w-full rounded" />
        ))}
      </div>
    </div>
  );
}

function NeedsInputBanner({ needsInput, humanMessage, currentAgent }: {
  needsInput: boolean;
  humanMessage: string;
  currentAgent: string;
}) {
  if (!needsInput || !humanMessage) {
    return null;
  }

  return (
    <Card>
      <CardContent className="p-3 bg-yellow-50">
        <p className="text-sm font-medium">
          <Badge variant="default">{currentAgent}</Badge>
        </p>
        <blockquote className="mt-2 text-sm italic border-l-2 pl-3 text-muted-foreground">
          {humanMessage}
        </blockquote>
      </CardContent>
    </Card>
  );
}

function ActiveAgentSection({ agents }: { agents: ApiActiveAgentEntry[] }) {
  const hasActive = agents && agents.length > 0;

  if (!hasActive) {
    return (
      <p className="text-sm italic text-muted-foreground">
        No OpenCode subagents currently active
      </p>
    );
  }

  const statusVariant = (status: string): "default" | "secondary" | "outline" => {
    switch (status) {
      case "running":
        return "default";
      case "pending":
        return "secondary";
      case "completed":
        return "outline";
      default:
        return "secondary";
    }
  };

  return (
    <Card>
      <CardContent className="p-3">
        <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          Active Subagents
        </div>
        <div className="flex items-start gap-2">
          <Badge variant="outline" className="shrink-0 mt-0.5 text-[0.65rem]">
            coordinator
          </Badge>
          <ArrowRight className="size-3 text-muted-foreground mt-1.5 shrink-0" aria-hidden="true" />
          <div className="flex flex-wrap gap-x-3 gap-y-1.5 min-w-0 flex-1">
            {agents.map((agent) => (
              <div key={agent.id} className="flex items-center gap-1.5 max-w-full" data-testid={`active-agent-${agent.agent}`}>
                <span
                  className={`size-1.5 rounded-full shrink-0 ${
                    agent.status === "running" ? "bg-green-500" : "bg-muted-foreground"
                  }`}
                />
                <Badge variant="default" className="text-[0.65rem] px-1.5 py-0">
                  {agent.agent}
                </Badge>
                <Badge variant={statusVariant(agent.status)} className="text-[0.6rem] px-1.5 py-0">
                  {agent.status}
                </Badge>
                {agent.title && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="text-xs text-muted-foreground truncate max-w-[160px] inline-block" title={agent.title}>
                        {agent.title}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" className="max-w-[300px] break-words">
                      <p className="text-xs">{agent.title}</p>
                      {agent.model && <p className="text-[0.65rem] opacity-70 mt-0.5">{agent.model}</p>}
                      {agent.sessionId && <p className="text-[0.6rem] opacity-50 mt-0.5">Session: {agent.sessionId}</p>}
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function EventTimeline({ events }: { events: ApiEventEntry[] }) {
  if (!events || events.length === 0) {
    return <p className="text-sm italic text-muted-foreground">No events recorded yet</p>;
  }

  return (
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
                <span className="size-2 rounded-full bg-primary shrink-0 shadow-[0_0_0_3px_rgba(var(--primary),0.2)]" />
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
  );
}

export function EventsTab({ workspaceName, goalContent, actions }: EventsTabProps) {
  const [goalOpenError, setGoalOpenError] = useState<string | null>(null);
  const [isGoalOpenPending, startGoalOpenTransition] = useTransition();
  const [actionOutputOpen, setActionOutputOpen] = useState(false);

  const hasActions = Boolean(actions && actions.length > 0);

  const {
    output: actionOutput,
    isRunning: isActionRunning,
    runError: actionRunError,
    startRun: startActionRun,
    stopRun: stopActionRun,
    outputRef: actionOutputRef,
  } = useAdhocRun({ workspaceName, skipModelsFetch: true });

  const { workspaces, fetchStatus } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);

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

  const handleActionClick = (action: ApiActionEntry) => {
    setActionOutputOpen(true);
    startActionRun(action.prompt, action.model);
  };

  if (fetchStatus === "fetching" && !workspace) return <EventsTabSkeleton />;

  if (!workspace) {
    if (fetchStatus === "error") {
      return (
        <p className="text-sm text-destructive">
          Failed to load events
        </p>
      );
    }
    return null;
  }

  return (
    <div className="space-y-4">
      {hasActions && (
        <div className="space-y-3">
          <ActionBar
            actions={actions!}
            isRunning={isActionRunning}
            onActionClick={handleActionClick}
          />
          {actionRunError ? (
            <p className="text-sm text-destructive" role="alert">{actionRunError}</p>
          ) : null}
          {(isActionRunning || actionOutput) ? (
            <details open={actionOutputOpen} onToggle={(e) => setActionOutputOpen((e.target as HTMLDetailsElement).open)}>
              <summary className="cursor-pointer text-sm font-medium flex items-center gap-2">
                <ChevronRight
                  className="size-4 text-muted-foreground transition-transform duration-200 [[open]>&]:rotate-90"
                  aria-hidden="true"
                />
                Output
                {isActionRunning && (
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    onClick={(e) => { e.preventDefault(); stopActionRun(); }}
                    className="ml-auto"
                  >
                    <Square className="mr-1 size-3" />
                    Stop
                  </Button>
                )}
              </summary>
              <pre
                ref={actionOutputRef}
                className="mt-2 bg-muted rounded-md p-4 text-sm font-mono overflow-auto max-h-[400px] whitespace-pre-wrap"
              >
                {actionOutput || (isActionRunning ? "Running..." : "")}
              </pre>
            </details>
          ) : null}
        </div>
      )}
      <NeedsInputBanner
        needsInput={workspace.needsInput}
        humanMessage={workspace.humanMessage}
        currentAgent={workspace.currentAgent}
      />
      <ActiveAgentSection agents={workspace.activeAgents} />
      {goalContent && (
        <details className="group">
          <summary className="cursor-pointer font-semibold text-sm mb-2 flex items-center gap-2 list-none [&::-webkit-details-marker]:hidden">
            <ChevronRight
              className="size-4 text-muted-foreground transition-transform duration-200 group-open:rotate-90"
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
          <EventTimeline events={workspace.events ?? []} />
        </CardContent>
      </Card>
    </div>
  );
}
