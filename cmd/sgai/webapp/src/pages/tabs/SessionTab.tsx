import { useState, useTransition, type MouseEvent } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";
import { ChevronRight } from "lucide-react";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import type { ApiAgentCost, ApiStepCost, ApiTodoEntry, ApiActionEntry } from "@/types";

interface SessionTabProps {
  workspaceName: string;
  pmContent?: string;
  hasProjectMgmt?: boolean;
}

export interface ActionBarProps {
  actions: ApiActionEntry[];
  isRunning: boolean;
  onActionClick: (action: ApiActionEntry) => void;
}

export function ActionBar({ actions, isRunning, onActionClick }: ActionBarProps) {
  if (actions.length === 0) return null;

  return (
    <div className="flex flex-nowrap items-center gap-2 overflow-x-auto" role="toolbar" aria-label="Action buttons">
      {actions.map((action) => (
        <Tooltip key={`${action.name}-${action.model}`}>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={isRunning}
              onClick={() => onActionClick(action)}
            >
              {action.name}
            </Button>
          </TooltipTrigger>
          <TooltipContent>{action.description || action.model}</TooltipContent>
        </Tooltip>
      ))}
    </div>
  );
}

function formatCost(cost: number): string {
  return `$${cost.toFixed(4)}`;
}

function formatStepCost(cost: number): string {
  return `$${cost.toFixed(6)}`;
}

function CostSection({ cost }: { cost: { totalCost: number; totalTokens?: { input?: number; output?: number; cacheRead?: number }; byAgent?: ApiAgentCost[] } }) {
  const totalInput = (cost.totalTokens?.input ?? 0) + (cost.totalTokens?.cacheRead ?? 0);

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-base">Cost Tracking</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-4 text-sm">
          <div>
            <span className="text-muted-foreground text-xs">Total Cost</span>
            <div className="font-semibold">{formatCost(cost.totalCost)}</div>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="cursor-help">
                <span className="text-muted-foreground text-xs">Total Input</span>
                <div className="font-semibold">{totalInput.toLocaleString()}</div>
              </div>
            </TooltipTrigger>
            <TooltipContent>Total input = new tokens + cached tokens</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="cursor-help">
                <span className="text-muted-foreground text-xs">New Tokens</span>
                <div className="font-semibold">{(cost.totalTokens?.input ?? 0).toLocaleString()}</div>
              </div>
            </TooltipTrigger>
            <TooltipContent>Newly processed tokens (not from cache)</TooltipContent>
          </Tooltip>
          <div>
            <span className="text-muted-foreground text-xs">Output Tokens</span>
            <div className="font-semibold">{(cost.totalTokens?.output ?? 0).toLocaleString()}</div>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="cursor-help">
                <span className="text-muted-foreground text-xs">Cache Read</span>
                <div className="font-semibold">{(cost.totalTokens?.cacheRead ?? 0).toLocaleString()}</div>
              </div>
            </TooltipTrigger>
            <TooltipContent>Tokens served from prompt cache</TooltipContent>
          </Tooltip>
        </div>

        {cost.byAgent && cost.byAgent.length > 0 && (
          <details className="mt-4">
            <summary className="cursor-pointer text-sm font-medium">
              By Agent ({cost.byAgent.length} agents)
            </summary>
            <div className="mt-2 space-y-2">
              {cost.byAgent.map((agent: ApiAgentCost) => (
                <AgentCostDetail key={agent.agent} agentCost={agent} />
              ))}
            </div>
          </details>
        )}
      </CardContent>
    </Card>
  );
}

function AgentCostDetail({ agentCost }: { agentCost: ApiAgentCost }) {
  return (
    <details className="ml-2">
      <summary className="cursor-pointer text-sm flex items-center gap-2">
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="font-medium truncate max-w-[200px]">{agentCost.agent}</span>
          </TooltipTrigger>
          <TooltipContent>{agentCost.agent}</TooltipContent>
        </Tooltip>
        <span className="text-muted-foreground">{formatCost(agentCost.cost)}</span>
        <span className="text-xs text-muted-foreground">({agentCost.steps?.length ?? 0} steps)</span>
      </summary>
      {agentCost.steps && agentCost.steps.length > 0 && (
        <div className="ml-4 mt-1 space-y-0.5">
          {agentCost.steps.map((step: ApiStepCost) => (
            <div key={`${step.stepId}-${step.timestamp}`} className="flex items-center gap-2 text-xs text-muted-foreground">
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="truncate max-w-[150px] cursor-help">{step.stepId}</span>
                </TooltipTrigger>
                <TooltipContent>{step.stepId}</TooltipContent>
              </Tooltip>
              <span>{formatStepCost(step.cost)}</span>
              <span>({step.tokens.input} in, {step.tokens.output} out)</span>
            </div>
          ))}
        </div>
      )}
    </details>
  );
}

function TodoStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "pending":
      return <span>{"○"}</span>;
    case "in_progress":
      return <span>{"◐"}</span>;
    case "completed":
      return <span>{"●"}</span>;
    case "cancelled":
      return <span>{"✕"}</span>;
    default:
      return <span>{"○"}</span>;
  }
}

function TodoList({ todos, emptyMessage }: { todos: ApiTodoEntry[]; emptyMessage: string }) {
  if (!todos || todos.length === 0) {
    return <p className="text-sm italic text-muted-foreground">{emptyMessage}</p>;
  }

  return (
    <ScrollArea className="max-h-[300px]">
      <ul className="space-y-1.5">
        {todos.map((todo) => (
          <li key={`${todo.id}-${todo.content}-${todo.status}-${todo.priority}`} className="flex items-start gap-2 text-sm">
            <TodoStatusIcon status={todo.status} />
            <span className="flex-1">
              {todo.content}
              <span className="text-xs text-muted-foreground ml-1">({todo.priority})</span>
            </span>
          </li>
        ))}
      </ul>
    </ScrollArea>
  );
}

function TasksSection({ projectTodos, agentTodos }: { projectTodos: ApiTodoEntry[]; agentTodos: ApiTodoEntry[] }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Project TODO</CardTitle>
        </CardHeader>
        <CardContent>
          <TodoList todos={projectTodos ?? []} emptyMessage="No project todos" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Agent TODO</CardTitle>
        </CardHeader>
        <CardContent>
          <TodoList todos={agentTodos ?? []} emptyMessage="No active agent todos" />
        </CardContent>
      </Card>
    </div>
  );
}

export function SessionTab({ workspaceName, pmContent, hasProjectMgmt }: SessionTabProps) {
  const [steerMessage, setSteerMessage] = useState("");
  const [steerError, setSteerError] = useState<string | null>(null);
  const [steerSuccess, setSteerSuccess] = useState(false);
  const [isSteering, startSteerTransition] = useTransition();
  const [pmOpenError, setPmOpenError] = useState<string | null>(null);
  const [isPmOpenPending, startPmOpenTransition] = useTransition();
  const [startAppError, setStartAppError] = useState<string | null>(null);
  const [isStartAppPending, startStartAppTransition] = useTransition();

  const { workspaces } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);

  const agentSequence = workspace?.agentSequence ?? [];
  const cost = workspace?.cost;
  const modelStatuses = workspace?.modelStatuses;
  const projectTodos = workspace?.projectTodos ?? [];
  const agentTodos = workspace?.agentTodos ?? [];

  const handleStartApplication = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!workspaceName || isStartAppPending) return;
    setStartAppError(null);
    startStartAppTransition(async () => {
      try {
        await api.workspaces.start(workspaceName, false);
      } catch (err) {
        setStartAppError(err instanceof Error ? err.message : "Failed to start application");
      }
    });
  };

  const handleSteerSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!workspaceName || !steerMessage.trim()) return;

    setSteerError(null);
    setSteerSuccess(false);
    startSteerTransition(async () => {
      try {
        const response = await api.workspaces.steer(workspaceName, steerMessage.trim());
        if (response.success) {
          setSteerSuccess(true);
          setSteerMessage("");
        } else {
          setSteerError(response.message || "Failed to submit steering message");
        }
      } catch (err) {
        setSteerError(err instanceof Error ? err.message : "Failed to submit steering message");
      }
    });
  };

  const handleOpenProjectManagement = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!workspaceName || isPmOpenPending) return;
    setPmOpenError(null);
    startPmOpenTransition(async () => {
      try {
        await api.workspaces.openEditorProjectManagement(workspaceName);
      } catch (err) {
        setPmOpenError(err instanceof Error ? err.message : "Failed to open PROJECT_MANAGEMENT.md");
      }
    });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={handleStartApplication}
          disabled={isStartAppPending}
        >
          Start Application
        </Button>
      </div>
      {startAppError && (
        <p className="text-sm text-destructive" role="alert">{startAppError}</p>
      )}

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Steer Next Turn</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSteerSubmit} className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="steer-message">Instruction</Label>
              <Textarea
                id="steer-message"
                value={steerMessage}
                onChange={(event) => setSteerMessage(event.target.value)}
                placeholder="Enter re-steering instruction..."
                rows={4}
                className="resize-y"
                disabled={isSteering}
              />
            </div>
            {steerError && (
              <p className="text-sm text-destructive">{steerError}</p>
            )}
            {steerSuccess && !steerError && (
              <p className="text-sm text-primary">Steering instruction sent.</p>
            )}
            <Button type="submit" disabled={isSteering || !steerMessage.trim()}>
              Submit
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Tasks</CardTitle>
        </CardHeader>
        <CardContent>
          <TasksSection projectTodos={projectTodos} agentTodos={agentTodos} />
        </CardContent>
      </Card>

      {cost && <CostSection cost={cost} />}

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Agent Sequence</CardTitle>
        </CardHeader>
        <CardContent>
          {agentSequence && agentSequence.length > 0 ? (
            <ScrollArea className="max-h-[300px]">
              <ol className="list-decimal list-inside space-y-1 text-sm">
                {agentSequence.map((entry) => (
                  <li key={`${entry.agent}-${entry.elapsedTime}`} className="flex items-center gap-2">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className={entry.isCurrent ? "font-bold" : ""}>
                          {entry.agent}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>{entry.agent}</TooltipContent>
                    </Tooltip>
                    <span className="text-xs text-muted-foreground">
                      ({entry.elapsedTime})
                    </span>
                  </li>
                ))}
              </ol>
            </ScrollArea>
          ) : (
            <p className="text-sm italic text-muted-foreground">No agent sequence yet</p>
          )}
        </CardContent>
      </Card>

      {modelStatuses && modelStatuses.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Model Consensus</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-1 text-sm">
              {modelStatuses.map((ms) => (
                <li key={`${ms.modelId}-${ms.status}`} className="flex items-center gap-2">
                  <span>
                    {ms.status === "model-working" ? "◐" : ms.status === "model-done" ? "●" : "✕"}
                  </span>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="truncate max-w-[200px] cursor-help">
                        {ms.modelId.split("/").pop() ?? ms.modelId}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>{ms.modelId}</TooltipContent>
                  </Tooltip>
                  <Badge variant="secondary" className="text-xs">{ms.status}</Badge>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {hasProjectMgmt && (
        <details className="group">
          <summary className="cursor-pointer font-semibold text-sm mb-2 flex items-center gap-2 list-none [&::-webkit-details-marker]:hidden">
            <ChevronRight
              className="h-4 w-4 text-muted-foreground transition-transform duration-200 group-open:rotate-90"
              aria-hidden="true"
            />
            <span>PROJECT_MANAGEMENT.md</span>
            <span className="ml-auto">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                title="Open PROJECT_MANAGEMENT.md in editor"
                aria-label="Open PROJECT_MANAGEMENT.md in editor"
                onClick={handleOpenProjectManagement}
                disabled={isPmOpenPending}
              >
                📝
              </Button>
            </span>
          </summary>
          {pmContent ? (
            <MarkdownContent
              content={pmContent}
              className="p-4 border rounded-lg bg-muted/20"
            />
          ) : (
            <p className="text-sm italic text-muted-foreground p-4">No content available</p>
          )}
          {pmOpenError && (
            <p className="text-xs text-destructive mt-2" role="alert">
              {pmOpenError}
            </p>
          )}
        </details>
      )}
    </div>
  );
}
