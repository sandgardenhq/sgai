import { useState, useTransition, type MouseEvent } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/components/MarkdownContent";
import { ChevronRight } from "lucide-react";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import type { ApiTodoEntry, ApiActionEntry } from "@/types";

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
    <div className="flex flex-wrap items-center gap-2" role="toolbar" aria-label="Action buttons">
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
  const [pmOpenError, setPmOpenError] = useState<string | null>(null);
  const [isPmOpenPending, startPmOpenTransition] = useTransition();

  const { workspaces } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);

  const projectTodos = workspace?.projectTodos ?? [];
  const agentTodos = workspace?.agentTodos ?? [];

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
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Tasks</CardTitle>
        </CardHeader>
        <CardContent>
          <TasksSection projectTodos={projectTodos} agentTodos={agentTodos} />
        </CardContent>
      </Card>

      {hasProjectMgmt && (
        <details className="group">
          <summary className="cursor-pointer font-semibold text-sm mb-2 flex items-center gap-2 list-none [&::-webkit-details-marker]:hidden">
            <ChevronRight
              className="size-4 text-muted-foreground transition-transform duration-200 group-open:rotate-90"
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
