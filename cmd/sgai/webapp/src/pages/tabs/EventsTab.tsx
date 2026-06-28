import { useState, useEffect, useReducer, useTransition, type MouseEvent } from "react";
import { ChevronRight, Square } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { useAdhocRun } from "@/hooks/useAdhocRun";
import { ActionBar } from "./SessionTab";
import type { ApiEventEntry, ApiActionEntry, ApiTokenUsageResponse, ApiTokenUsageRow } from "@/types";

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

function NeedsInputBanner({ needsInput, humanMessage }: {
  needsInput: boolean;
  humanMessage: string;
}) {
  if (!needsInput || !humanMessage) {
    return null;
  }

  return (
    <Card>
      <CardContent className="p-3 bg-yellow-50">
        <blockquote className="mt-2 text-sm italic border-l-2 pl-3 text-muted-foreground">
          {humanMessage}
        </blockquote>
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

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

function modelDisplay(raw: string): string {
  if (!raw) return "";
  try {
    const desc = JSON.parse(raw) as { id?: string; providerID?: string; variant?: string };
    const parts: string[] = [];
    if (desc.providerID) parts.push(desc.providerID);
    if (desc.id) parts.push(desc.id);
    if (desc.variant && desc.variant !== "default") parts.push(desc.variant);
    return parts.join("/");
  } catch {
    return raw;
  }
}

type TokenUsageState = {
  usage: ApiTokenUsageResponse | null;
  error: string | null;
};

type TokenUsageAction =
  | { type: "loaded"; usage: ApiTokenUsageResponse }
  | { type: "error"; message: string };

const initialTokenUsageState: TokenUsageState = { usage: null, error: null };

function tokenUsageReducer(state: TokenUsageState, action: TokenUsageAction): TokenUsageState {
  switch (action.type) {
    case "loaded":
      return { usage: action.usage, error: null };
    case "error":
      return { ...state, error: action.message };
  }
}

function TokenUsageBox({ workspaceName }: { workspaceName: string }) {
  const [state, dispatch] = useReducer(tokenUsageReducer, initialTokenUsageState);
  const { usage, error } = state;

  useEffect(() => {
    let cancelled = false;

    const fetchUsage = () => {
      api.workspaces.tokenStats(workspaceName)
        .then((data) => {
          if (cancelled) return;
          dispatch({ type: "loaded", usage: data });
        })
        .catch((err) => {
          if (cancelled) return;
          dispatch({ type: "error", message: err instanceof Error ? err.message : "Failed to load token usage" });
        });
    };

    fetchUsage();
    const interval = setInterval(fetchUsage, 15_000);

    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [workspaceName]);

  if (error) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Token Usage</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">{error}</p>
        </CardContent>
      </Card>
    );
  }

  if (!usage) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Token Usage</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-20 w-full rounded" />
        </CardContent>
      </Card>
    );
  }

  if (!usage.rows || usage.rows.length === 0) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Token Usage</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm italic text-muted-foreground">No token usage recorded yet</p>
        </CardContent>
      </Card>
    );
  }

  const t = usage.totals;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Token Usage</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-muted-foreground border-b">
                <th className="py-1 pr-3">Agent</th>
                <th className="py-1 pr-3">Model</th>
                <th className="py-1 pr-3 text-right">Input</th>
                <th className="py-1 pr-3 text-right">Output</th>
                <th className="py-1 pr-3 text-right">Cached In</th>
                <th className="py-1 pr-3 text-right">Cached Out</th>
                <th className="py-1 pr-3 text-right">Other</th>
                <th className="py-1 pr-3 text-right">Reasoning</th>
                <th className="py-1 pr-3 text-right">Total</th>
                <th className="py-1 text-right">Sessions</th>
              </tr>
            </thead>
            <tbody>
              {usage.rows.map((row: ApiTokenUsageRow, i: number) => (
                <tr key={`${row.agent}-${row.model}-${i}`} className="border-b last:border-0">
                  <td className="py-1 pr-3 truncate max-w-[120px]" title={row.agent}>{row.agent}</td>
                  <td className="py-1 pr-3 truncate max-w-[180px]" title={modelDisplay(row.model)}>{modelDisplay(row.model)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(row.input)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(row.output)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(row.cacheRead)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(row.cacheWrite)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(row.other)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(row.reasoning)}</td>
                  <td className="py-1 pr-3 text-right tabular-nums font-semibold">{formatTokens(row.total)}</td>
                  <td className="py-1 text-right tabular-nums">{row.sessionCount}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="font-semibold border-t-2">
                <td className="py-1 pr-3" colSpan={2}>TOTAL</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.input)}</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.output)}</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.cacheRead)}</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.cacheWrite)}</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.other)}</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.reasoning)}</td>
                <td className="py-1 pr-3 text-right tabular-nums">{formatTokens(t.total)}</td>
                <td className="py-1 text-right tabular-nums">{t.sessionCount}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </CardContent>
    </Card>
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
      />
      <TokenUsageBox workspaceName={workspaceName} />
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
          <h3 className="text-base font-semibold mb-2">Events</h3>
          <EventTimeline events={workspace.events ?? []} />
        </CardContent>
      </Card>
    </div>
  );
}
