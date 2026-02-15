import { useState, useEffect, useCallback, useMemo, useRef, type ReactNode } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Loader2, Menu, X, Inbox } from "lucide-react";
import { api } from "@/lib/api";
import { useSSEEvent } from "@/hooks/useSSE";
import { cn } from "@/lib/utils";
import type { ApiWorkspaceEntry } from "@/types";

function WorkspaceTreeSkeleton() {
  return (
    <div className="space-y-2 p-2">
      {Array.from({ length: 5 }, (_, i) => (
        <Skeleton key={i} className="h-8 w-full rounded" />
      ))}
    </div>
  );
}

interface WorkspaceIndicatorsProps {
  workspace: ApiWorkspaceEntry;
}

function WorkspaceIndicators({ workspace }: WorkspaceIndicatorsProps) {
  const isActive = workspace.running;
  const runningLabel = workspace.running ? "Running" : "In progress";

  return (
    <span className="flex items-center gap-1 shrink-0">
      {workspace.pinned && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="text-[0.7rem] opacity-70">📌</span>
          </TooltipTrigger>
          <TooltipContent>Pinned</TooltipContent>
        </Tooltip>
      )}
      {workspace.needsInput && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Inbox className="h-3 w-3 text-primary" aria-label="Waiting for response" title="Waiting for response" />
          </TooltipTrigger>
          <TooltipContent>Waiting for response</TooltipContent>
        </Tooltip>
      )}
      {isActive && (
        <Loader2 className="h-3 w-3 text-primary animate-spin" aria-label={runningLabel} title={runningLabel} />
      )}
    </span>
  );
}

interface WorkspaceTreeItemProps {
  workspace: ApiWorkspaceEntry;
  selectedName: string | undefined;
}

function WorkspaceTreeItem({ workspace, selectedName }: WorkspaceTreeItemProps) {
  const isSelected = workspace.name === selectedName;
  const hasForks = workspace.forks && workspace.forks.length > 0;
  const hasForkSelected = workspace.forks?.some((f) => f.name === selectedName) ?? false;
  const [expanded, setExpanded] = useState(isSelected || hasForkSelected);

  useEffect(() => {
    if (isSelected || hasForkSelected) {
      setExpanded(true);
    }
  }, [isSelected, hasForkSelected]);

  return (
    <div className="mb-0.5">
      <div className="flex items-center gap-0">
        {hasForks ? (
          <button
            type="button"
            onClick={() => setExpanded((prev) => !prev)}
            className="w-5 h-5 inline-flex items-center justify-center rounded text-xs font-semibold shrink-0 mr-1 bg-muted text-muted-foreground hover:bg-secondary hover:text-secondary-foreground transition-colors"
            aria-label="Toggle forks"
          >
            {expanded ? "−" : "+"}
          </button>
        ) : (
          <span className="w-5 h-5 inline-block shrink-0 mr-1" />
        )}
        <Link
          to={`/workspaces/${encodeURIComponent(workspace.name)}/progress`}
          className={cn(
            "flex items-center gap-1 flex-1 min-w-0 px-2 py-1.5 rounded text-sm no-underline transition-colors",
            isSelected
              ? "bg-primary/10 border-l-[3px] border-primary pl-[calc(0.5rem-3px)] font-semibold text-primary"
              : "hover:bg-muted"
          )}
        >
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
                {workspace.name}
              </span>
            </TooltipTrigger>
            <TooltipContent side="right">{workspace.name}</TooltipContent>
          </Tooltip>
          <WorkspaceIndicators workspace={workspace} />
        </Link>
      </div>

      {hasForks && expanded && (
        <div className="ml-2.5 pl-4 relative before:content-[''] before:absolute before:left-2.5 before:top-0 before:bottom-2 before:w-0.5 before:bg-border before:rounded-sm">
          {workspace.forks?.map((fork) => {
            const forkSelected = fork.name === selectedName;
            return (
              <Link
                key={fork.name}
                to={`/workspaces/${encodeURIComponent(fork.name)}/progress`}
                className={cn(
                  "flex items-center gap-1 py-1 px-2 rounded text-sm no-underline transition-colors relative",
                  "before:content-[''] before:absolute before:left-[-0.875rem] before:top-1/2 before:w-3.5 before:h-0.5 before:bg-border before:rounded-sm",
                  forkSelected
                    ? "bg-primary/10 border-l-[3px] border-primary pl-[calc(0.5rem-3px)] font-semibold text-primary"
                    : "hover:bg-muted"
                )}
              >
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
                      {fork.name}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="right">{fork.name}</TooltipContent>
                </Tooltip>
                <WorkspaceIndicators workspace={fork} />
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

interface InProgressSectionProps {
  workspaces: ApiWorkspaceEntry[];
  selectedName: string | undefined;
}

function InProgressSection({ workspaces, selectedName }: InProgressSectionProps) {
  const inProgress = workspaces.flatMap((w) => {
    const items: ApiWorkspaceEntry[] = [];
    if (w.inProgress || w.running) items.push(w);
    if (w.forks) {
      for (const fork of w.forks) {
        if (fork.inProgress || fork.running) items.push(fork);
      }
    }
    return items;
  });

  if (inProgress.length === 0) return null;

  return (
    <div className="mb-3 pb-2 border-b">
      {inProgress.map((w) => {
        const isSelected = w.name === selectedName;
        return (
            <Link
              key={w.name}
              to={w.needsInput
              ? `/workspaces/${encodeURIComponent(w.name)}/respond`
              : `/workspaces/${encodeURIComponent(w.name)}/progress`
            }
            className={cn(
              "flex items-center gap-1 py-1 px-2 ml-2 rounded text-sm no-underline transition-colors mb-0.5",
              isSelected
                ? "bg-primary/10 border-l-[3px] border-primary pl-[calc(0.5rem-3px)] font-semibold text-primary"
                : "hover:bg-destructive/10"
            )}
          >
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
                  {w.name}
                </span>
              </TooltipTrigger>
              <TooltipContent side="right">{w.name}</TooltipContent>
            </Tooltip>
            <WorkspaceIndicators workspace={w} />
          </Link>
        );
      })}
    </div>
  );
}

function collectAllWorkspaces(workspaces: ApiWorkspaceEntry[]): ApiWorkspaceEntry[] {
  const all: ApiWorkspaceEntry[] = [];
  for (const w of workspaces) {
    all.push(w);
    if (w.forks) {
      for (const fork of w.forks) {
        all.push(fork);
      }
    }
  }
  return all;
}

interface SidebarHeaderIndicatorsProps {
  workspaces: ApiWorkspaceEntry[];
}

function SidebarHeaderIndicators({ workspaces }: SidebarHeaderIndicatorsProps) {
  const allWorkspaces = useMemo(() => collectAllWorkspaces(workspaces), [workspaces]);

  const needsInputCount = useMemo(
    () => allWorkspaces.filter((w) => w.needsInput).length,
    [allWorkspaces],
  );

  const hasAnyRunning = useMemo(
    () => allWorkspaces.some((w) => w.running || w.needsInput),
    [allWorkspaces],
  );

  return (
    <div className="flex items-center gap-2">
      {needsInputCount > 0 && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="relative inline-flex items-center text-primary cursor-help">
              <Inbox className="h-4 w-4" />
              <Badge
                variant="destructive"
                className="absolute -top-2 -right-2.5 h-4 min-w-4 px-1 text-[0.6rem] leading-none flex items-center justify-center rounded-full"
              >
                {needsInputCount}
              </Badge>
            </span>
          </TooltipTrigger>
          <TooltipContent>
            {needsInputCount === 1
              ? "1 workspace waiting for response"
              : `${needsInputCount} workspaces waiting for response`}
          </TooltipContent>
        </Tooltip>
      )}
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="text-sm cursor-help" aria-label={hasAnyRunning ? "Some factories running" : "All factories stopped"}>
            {hasAnyRunning ? "●" : "○"}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          {hasAnyRunning ? "Some factories are running" : "All factories stopped"}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

interface DashboardProps {
  children: ReactNode;
}

export function Dashboard({ children }: DashboardProps): JSX.Element {
  const { name: selectedName } = useParams<{ name: string }>();
  const navigate = useNavigate();

  const [workspaces, setWorkspaces] = useState<ApiWorkspaceEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const hasLoadedOnce = useRef(false);

  const workspaceUpdateEvent = useSSEEvent("workspace:update");

  useEffect(() => {
    let cancelled = false;
    if (!hasLoadedOnce.current) {
      setLoading(true);
    }
    setError(null);

    api.workspaces
      .list()
      .then((response) => {
        if (!cancelled) {
          setWorkspaces(response.workspaces ?? []);
          setLoading(false);
          hasLoadedOnce.current = true;
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
  }, [refreshKey]);

  useEffect(() => {
    if (workspaceUpdateEvent === null) return;
    const timeoutId = setTimeout(() => {
      setRefreshKey((k) => k + 1);
    }, 100);
    return () => clearTimeout(timeoutId);
  }, [workspaceUpdateEvent]);

  useEffect(() => {
    if (selectedName) {
      setIsSidebarOpen(false);
    }
  }, [selectedName]);

  const handleCreateWorkspace = useCallback(() => {
    navigate("/workspaces/new");
  }, [navigate]);

  return (
    <div className="relative flex flex-col md:flex-row gap-0 h-[calc(100vh-4rem)]">
      {!isSidebarOpen && (
        <div className="flex items-center gap-2 pb-3 md:hidden">
          <Button
            type="button"
            size="icon"
            variant="outline"
            aria-label="Open workspace list"
            onClick={() => setIsSidebarOpen(true)}
          >
            <Menu className="h-4 w-4" />
          </Button>
          <span className="text-sm font-semibold">Workspaces</span>
          {!loading && !error && (
            <span className="ml-auto">
              <SidebarHeaderIndicators workspaces={workspaces} />
            </span>
          )}
        </div>
      )}

      {isSidebarOpen && (
        <button
          type="button"
          aria-label="Close workspace list"
          className="absolute inset-0 z-10 bg-black/40 md:hidden"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      <aside
        className={cn(
          "absolute inset-y-0 left-0 z-20 w-[85vw] max-w-[320px] bg-background border-r flex flex-col transition-transform",
          "md:static md:z-auto md:w-[280px] md:min-w-[240px] md:max-w-[320px] md:translate-x-0 md:border-b-0",
          isSidebarOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className="flex items-center justify-between px-2 py-2 md:hidden">
          <span className="text-sm font-semibold">Workspaces</span>
          <div className="flex items-center gap-2">
            {!loading && !error && <SidebarHeaderIndicators workspaces={workspaces} />}
            <Button
              type="button"
              size="icon"
              variant="ghost"
              aria-label="Close workspace list"
              onClick={() => setIsSidebarOpen(false)}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
        <Separator className="md:hidden" />
        <div className="hidden md:flex items-center justify-between px-3 py-2">
          <span className="text-sm font-semibold">Workspaces</span>
          {!loading && !error && <SidebarHeaderIndicators workspaces={workspaces} />}
        </div>
        <Separator className="hidden md:block" />
        <ScrollArea className="flex-1 px-1 py-2">
          {loading && <WorkspaceTreeSkeleton />}

          {error && (
            <p className="text-sm text-destructive p-2">
              Failed to load workspaces: {error.message}
            </p>
          )}

          {!loading && !error && (
            <>
              <InProgressSection workspaces={workspaces} selectedName={selectedName} />
              {workspaces.length > 0 ? (
                workspaces.map((workspace) => (
                  <WorkspaceTreeItem
                    key={workspace.name}
                    workspace={workspace}
                    selectedName={selectedName}
                  />
                ))
              ) : (
                <p className="text-sm text-muted-foreground italic p-2">No workspaces found.</p>
              )}
            </>
          )}
        </ScrollArea>
        <Separator />
        <div className="p-2">
          <Button
            variant="outline"
            className="w-full"
            onClick={handleCreateWorkspace}
          >
            [ + ]
          </Button>
        </div>
      </aside>

      <main className="flex-1 overflow-auto pt-4 md:pt-0 md:pl-4">
        {children}
      </main>
    </div>
  );
}
