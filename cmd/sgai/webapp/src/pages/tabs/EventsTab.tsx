import { useState, useTransition, type MouseEvent } from "react";
import { ChevronRight, Square } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { useAdhocRun } from "@/hooks/useAdhocRun";
import { ActionBar } from "./SessionTab";
import type { ApiEventEntry, ApiModelStatusEntry, ApiAgentModelEntry, ApiActionEntry } from "@/types";

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
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-10 w-full rounded" />
        ))}
      </div>
    </div>
  );
}

function WorkflowSection({ svgHash, agentModels, modelStatuses, needsInput, humanMessage, currentAgent, workspaceName }: {
  svgHash: string;
  agentModels?: ApiAgentModelEntry[];
  modelStatuses?: ApiModelStatusEntry[];
  needsInput: boolean;
  humanMessage: string;
  currentAgent: string;
  workspaceName: string;
}) {
  const svgUrl = `/api/v1/workspaces/${encodeURIComponent(workspaceName)}/workflow.svg${svgHash ? `?h=${svgHash}` : ""}`;

  return (
    <Card>
      <CardContent className="space-y-3">
        <div className="flex flex-col lg:flex-row gap-4">
          <div className="flex-1 min-w-0">
            <img
              src={svgUrl}
              alt="Workflow graph"
              className="max-w-full h-auto"
            />
          </div>

          {agentModels && agentModels.length > 0 && (
            <div className="lg:w-80 xl:w-96 shrink-0">
              <AgentModelsTable entries={agentModels} />
            </div>
          )}
        </div>

        {modelStatuses && modelStatuses.length > 0 && (
          <ModelStatusList statuses={modelStatuses} />
        )}

        {needsInput && humanMessage && (
          <div className="mt-3 p-3 border rounded-lg bg-yellow-50">
            <p className="text-sm font-medium">
              <Badge variant="default">{currentAgent}</Badge>
            </p>
            <blockquote className="mt-2 text-sm italic border-l-2 pl-3 text-muted-foreground">
              {humanMessage}
            </blockquote>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function AgentModelsTable({ entries }: { entries: ApiAgentModelEntry[] }) {
  return (
    <div className="text-sm">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="text-muted-foreground">Agent</TableHead>
            <TableHead className="text-muted-foreground">Model(s)</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry) => (
            <TableRow key={entry.agent}>
              <TableCell className="whitespace-nowrap">{entry.agent}</TableCell>
              <TableCell>
                <ul className="list-disc list-inside space-y-0.5">
                  {entry.models.map((model) => {
                    const shortModel = model.split("/").pop() ?? model;
                    return (
                      <li key={model}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="cursor-help">
                              {shortModel}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>{model}</TooltipContent>
                        </Tooltip>
                      </li>
                    );
                  })}
                </ul>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
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
                  className="h-4 w-4 text-muted-foreground transition-transform duration-200 [[open]>&]:rotate-90"
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
                    <Square className="mr-1 h-3 w-3" />
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
      <WorkflowSection
        svgHash={workspace.svgHash}
        agentModels={workspace.agentModels}
        modelStatuses={workspace.modelStatuses}
        needsInput={workspace.needsInput}
        humanMessage={workspace.humanMessage}
        currentAgent={workspace.currentAgent}
        workspaceName={workspaceName}
      />
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
          <EventTimeline events={workspace.events ?? []} />
        </CardContent>
      </Card>
    </div>
  );
}
