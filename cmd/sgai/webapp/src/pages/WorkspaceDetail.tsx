import { useState, useEffect, Suspense, lazy, useTransition, useRef, useCallback } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { NotYetAvailable } from "@/components/NotYetAvailable";
import { InlineForkEditor } from "@/pages/InlineForkEditor";
import { api } from "@/lib/api";
import { useFactoryState, triggerFactoryRefresh } from "@/lib/factory-state";
import { useAdhocRun } from "@/hooks/useAdhocRun";
import { ChevronRight, Square } from "lucide-react";
import type { ApiWorkspaceEntry, ApiActionEntry } from "@/types";
import { cn } from "@/lib/utils";

const SessionTab = lazy(() => import("./tabs/SessionTab").then((m) => ({ default: m.SessionTab })));
const MessagesTab = lazy(() => import("./tabs/MessagesTab").then((m) => ({ default: m.MessagesTab })));
const LogTab = lazy(() => import("./tabs/LogTab").then((m) => ({ default: m.LogTab })));
const RunTab = lazy(() => import("./tabs/RunTab").then((m) => ({ default: m.RunTab })));
const ChangesTab = lazy(() => import("./tabs/ChangesTab").then((m) => ({ default: m.ChangesTab })));
const CommitsTab = lazy(() => import("./tabs/CommitsTab").then((m) => ({ default: m.CommitsTab })));
const EventsTab = lazy(() => import("./tabs/EventsTab").then((m) => ({ default: m.EventsTab })));
const ForksTab = lazy(() => import("./tabs/ForksTab").then((m) => ({ default: m.ForksTab })));


function parseExecTime(value: string | undefined | null): number | null {
  if (!value) return null;
  const trimmed = value.trim();
  if (!trimmed || trimmed === "-") return null;
  const compact = trimmed.replace(/\s+/g, "");
  const match = compact.match(/^(?:(\d+)m)?(\d+)s$/);
  if (!match) return null;
  const minutes = match[1] ? Number(match[1]) : 0;
  const seconds = Number(match[2]);
  if (Number.isNaN(minutes) || Number.isNaN(seconds)) return null;
  return minutes * 60 + seconds;
}

function formatExecTime(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

function WorkspaceDetailSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 pb-3 border-b">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-6 w-16 rounded" />
        <Skeleton className="h-6 w-20 rounded" />
      </div>
      <Skeleton className="h-10 w-full" />
      <div className="space-y-3">
        <Skeleton className="h-32 w-full rounded-xl" />
        <Skeleton className="h-48 w-full rounded-xl" />
      </div>
    </div>
  );
}

const TABS = [
  { key: "progress", label: "Progress" },
  { key: "log", label: "Log" },
  { key: "changes", label: "Diffs" },
  { key: "commits", label: "Commits" },
  { key: "messages", label: "Messages" },
  { key: "internals", label: "Internals" },
  { key: "run", label: "Run" },
] as const;

const ROOT_TABS = [
  { key: "forks", label: "Forks" },
] as const;

interface TabNavProps {
  workspaceName: string;
  activeTab: string;
  isRoot: boolean;
  hasForks: boolean;
}

function TabNav({ workspaceName, activeTab, isRoot, hasForks }: TabNavProps) {
  const tabs = isRoot && hasForks ? ROOT_TABS : TABS;
  const encodedName = encodeURIComponent(workspaceName);

  return (
    <nav className="border-b overflow-x-auto overflow-y-hidden pl-2.5 mb-0">
      <ul className="flex mb-0 whitespace-nowrap min-w-min list-none p-0 m-0">
        {tabs.map((tab) => (
          <li key={tab.key}>
            <Link
              to={`/workspaces/${encodedName}/${tab.key}`}
              aria-current={activeTab === tab.key ? "page" : undefined}
              className={cn(
                "inline-block px-4 py-2 text-sm no-underline transition-colors border-b-2",
                activeTab === tab.key
                  ? "text-primary border-primary"
                  : "text-muted-foreground border-transparent hover:text-foreground"
              )}
            >
              {tab.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}


export function WorkspaceDetail(): JSX.Element | null {
  const { name, "*": tabPath } = useParams<{ name: string; "*": string }>();
  const workspaceName = name ?? "";
  const activeTab = tabPath?.split("/")[0] || "progress";
  const navigate = useNavigate();

  const [actionError, setActionError] = useState<string | null>(null);
  const [runningOverride, setRunningOverride] = useState<boolean | null>(null);
  const previousWorkspaceRef = useRef<string | null>(null);
  const [isStartStopPending, startStartStopTransition] = useTransition();
  const [isSelfDrivePending, startSelfDriveTransition] = useTransition();
  const [isPinPending, startPinTransition] = useTransition();
  const [isEditorPending, startEditorTransition] = useTransition();
  const [isDeletePending, startDeleteTransition] = useTransition();
  const [execTimeSeconds, setExecTimeSeconds] = useState<number | null>(null);

  const { workspaces, fetchStatus } = useFactoryState();

  const detail: ApiWorkspaceEntry | null = workspaces.find((ws) => ws.name === workspaceName) ?? null;
  const loading = fetchStatus === "fetching" && detail === null;
  const error: Error | null = fetchStatus === "error" && detail === null ? new Error("Failed to load workspace state") : null;

  useEffect(() => {
    if (previousWorkspaceRef.current !== workspaceName) {
      previousWorkspaceRef.current = workspaceName;
      setRunningOverride(null);
    }
  }, [workspaceName]);

  useEffect(() => {
    if (runningOverride !== null && detail?.running === runningOverride) {
      setRunningOverride(null);
    }
  }, [detail?.running, runningOverride]);

  useEffect(() => {
    if (!workspaceName) return;
    setActionError(null);
  }, [workspaceName]);

  const totalExecTimeRaw = detail?.totalExecTime;
  const detailRunning = detail?.running;
  useEffect(() => {
    if (!detail) return;
    const parsed = parseExecTime(totalExecTimeRaw ?? "");
    if (parsed !== null) {
      setExecTimeSeconds(parsed);
    } else if (detailRunning) {
      setExecTimeSeconds(0);
    } else {
      setExecTimeSeconds(null);
    }
    if (!detailRunning) return;
    const timer = setInterval(() => {
      setExecTimeSeconds((prev) => (prev === null ? prev : prev + 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [totalExecTimeRaw, detailRunning]);

  const hasForks = (detail?.forks?.length ?? 0) > 0;
  const isForkedRoot = Boolean(detail?.isRoot && hasForks);

  const [actionOutputOpen, setActionOutputOpen] = useState(false);
  const {
    output: actionOutput,
    isRunning: isActionRunning,
    runError: actionRunError,
    startRun: startActionRun,
    stopRun: stopActionRun,
    outputRef: actionOutputRef,
  } = useAdhocRun({ workspaceName, skipModelsFetch: true });

  const handleActionClick = useCallback((action: ApiActionEntry, _forkName?: string) => {
    setActionOutputOpen(true);
    startActionRun(action.prompt, action.model);
  }, [startActionRun]);

  useEffect(() => {
    if (!detail || !detail.isRoot) return;
    if (detail.name !== workspaceName) return;
    const encodedName = encodeURIComponent(detail.name);

    if (hasForks && activeTab !== "forks") {
      navigate(`/workspaces/${encodedName}/forks`, { replace: true });
      return;
    }

    if (!hasForks && activeTab === "forks") {
      navigate(`/workspaces/${encodedName}/progress`, { replace: true });
    }
  }, [detail, hasForks, activeTab, navigate, workspaceName]);

  if (loading && !detail) return <WorkspaceDetailSkeleton />;

  if (error) {
    if (error.message.toLowerCase().includes("workspace not found")) {
      return null;
    }
    return (
      <p className="text-sm text-destructive">
        Failed to load workspace: {error.message}
      </p>
    );
  }

  if (!detail) return null;

  if (!detail.hasSgai && !detail.isRoot) {
    return <NoWorkspaceState name={detail.name} dir={detail.dir} />;
  }

  const effectiveRunning = runningOverride !== null ? runningOverride : (detail.running ?? false);

  const totalExecTime = detail.totalExecTime?.trim() ?? "";
  const fallbackExecTime = totalExecTime && totalExecTime !== "-" ? totalExecTime : "0s";
  const displayExecTime = execTimeSeconds !== null
    ? formatExecTime(execTimeSeconds)
    : fallbackExecTime;
  const agentLabel = detail.currentAgent?.trim();
  const modelLabel = detail.currentModel
    ? detail.currentModel.split("/").pop() ?? detail.currentModel
    : "";
  const agentModelLabel = [agentLabel, modelLabel].filter(Boolean).join(" | ");
  const fullAgentModelLabel = [detail.currentAgent, detail.currentModel].filter(Boolean).join(" | ");
  const statusLine = detail.task?.trim() || detail.status?.trim();
  const showStatusLine = !isForkedRoot && Boolean(agentModelLabel || statusLine);
  const encodedWorkspace = encodeURIComponent(detail.name);
  const selfDriveLabel = effectiveRunning ? "Self-Drive" : "Self-drive";
  const showComposeGoalAction = !effectiveRunning;
  const showEditGoalAction = detail.hasSgai || Boolean(detail.goalContent?.trim());
  const showOpenEditorAction = true;
  const isActionDisabled = effectiveRunning || isStartStopPending || isSelfDrivePending;

  const handleStart = () => {
    if (!workspaceName) return;
    setActionError(null);
    startStartStopTransition(async () => {
      try {
        const result = await api.workspaces.start(workspaceName, false);
        triggerFactoryRefresh();
        if (result.running) {
          setRunningOverride(true);
        }
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to start session");
      }
    });
  };

  const handleStop = () => {
    if (!workspaceName) return;
    setActionError(null);
    setRunningOverride(false);
    startStartStopTransition(async () => {
      try {
        await api.workspaces.stop(workspaceName);
        triggerFactoryRefresh();
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to stop session");
      }
    });
  };

  const handleSelfDrive = () => {
    if (!workspaceName) return;
    setActionError(null);
    startSelfDriveTransition(async () => {
      try {
        await api.workspaces.start(workspaceName, true);
        triggerFactoryRefresh();
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to start self-drive session");
      }
    });
  };

  const handlePinToggle = () => {
    if (!workspaceName) return;
    setActionError(null);
    startPinTransition(async () => {
      try {
        await api.workspaces.togglePin(workspaceName);
        triggerFactoryRefresh();
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to toggle pin");
      }
    });
  };

  const handleOpenEditor = () => {
    if (!workspaceName) return;
    setActionError(null);
    startEditorTransition(async () => {
      try {
        await api.workspaces.openEditor(workspaceName);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to open editor");
      }
    });
  };

  const showDeleteAction = !effectiveRunning && !isForkedRoot;

  const handleDelete = () => {
    if (!workspaceName) return;
    setActionError(null);
    startDeleteTransition(async () => {
      try {
        if (detail.isFork) {
          // Empty string for fork dir: the backend resolveRootForDeleteFork resolves the root from the fork name.
          await api.workspaces.deleteFork(workspaceName, "");
        } else {
          await api.workspaces.deleteWorkspace(workspaceName);
        }
        triggerFactoryRefresh();
        navigate("/");
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to delete workspace");
      }
    });
  };

  return (
    <div className="sticky-header-wrapper">
      <div className="sticky top-0 z-10 bg-background">
        <header className="flex flex-wrap items-start gap-3 mb-3 pb-3 border-b">
          <div className="flex-shrink min-w-0 max-w-fit">
            <Tooltip>
              <TooltipTrigger asChild>
                <h3 className="m-0 text-xl font-semibold whitespace-nowrap overflow-hidden text-ellipsis">
                  {detail.description || detail.name}
                </h3>
              </TooltipTrigger>
              <TooltipContent>{detail.isFork ? detail.name : detail.dir}</TooltipContent>
            </Tooltip>
          </div>

          {!isForkedRoot && (
            <div className="flex items-center gap-2 shrink-0">
              <Tooltip>
                <TooltipTrigger asChild>
                <Badge
                  variant="secondary"
                  className="font-mono"
                  aria-label="Total execution time"
                  tabIndex={0}
                >
                  {displayExecTime}
                </Badge>
                </TooltipTrigger>
                <TooltipContent>Total execution time</TooltipContent>
              </Tooltip>
              <Badge variant={effectiveRunning ? "default" : "secondary"}>
                {effectiveRunning ? "running" : "stopped"}
              </Badge>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2 w-full md:w-auto md:ml-auto mt-2 md:mt-0 justify-start md:justify-end">
              {isForkedRoot ? (
                <>
                  {showOpenEditorAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={handleOpenEditor}
                      disabled={isEditorPending}
                    >
                      Open in Editor
                    </Button>
                  )}
                  <Button
                    type="button"
                    size="sm"
                    variant={detail.pinned ? "secondary" : "outline"}
                    onClick={handlePinToggle}
                    disabled={isPinPending}
                    aria-pressed={detail.pinned}
                  >
                    {detail.pinned ? "Unpin" : "Pin"}
                  </Button>
                </>
              ) : (
                <>
                  {detail?.needsInput && (
                    <Button
                      type="button"
                      size="sm"
                      variant="default"
                      onClick={() => navigate(`/workspaces/${encodedWorkspace}/respond`)}
                    >
                      Respond
                    </Button>
                  )}
                  {detail.continuousMode ? (
                    <>
                      <Button
                        type="button"
                        size="sm"
                        variant={(effectiveRunning && detail.interactiveAuto) ? "default" : "outline"}
                        onClick={handleSelfDrive}
                        disabled={isActionDisabled}
                        aria-pressed={effectiveRunning && detail.interactiveAuto}
                      >
                        Continuous Self-Drive
                      </Button>
                      {effectiveRunning && (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={handleStop}
                          disabled={isStartStopPending}
                        >
                          Stop
                        </Button>
                      )}
                    </>
                  ) : (
                    <>
                      <Button
                        type="button"
                        size="sm"
                        variant={(effectiveRunning && detail.interactiveAuto) ? "default" : "outline"}
                        onClick={handleSelfDrive}
                        disabled={isActionDisabled}
                        aria-pressed={effectiveRunning && detail.interactiveAuto}
                      >
                        {selfDriveLabel}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant={(effectiveRunning && !detail.interactiveAuto) ? "default" : "outline"}
                        onClick={handleStart}
                        disabled={isActionDisabled}
                        aria-pressed={effectiveRunning && !detail.interactiveAuto}
                      >
                        Start
                      </Button>
                      {effectiveRunning && (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={handleStop}
                          disabled={isStartStopPending}
                        >
                          Stop
                        </Button>
                      )}
                    </>
                  )}
                  {showComposeGoalAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/compose?workspace=${encodedWorkspace}`)}
                    >
                      Compose GOAL
                    </Button>
                  )}
                  {showEditGoalAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/workspaces/${encodedWorkspace}/goal/edit`)}
                    >
                      Edit GOAL
                    </Button>
                  )}
                  {showOpenEditorAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={handleOpenEditor}
                      disabled={isEditorPending}
                    >
                      Open in Editor
                    </Button>
                  )}
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => navigate(`/workspaces/${encodedWorkspace}/skills`)}
                  >
                    Skills
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => navigate(`/workspaces/${encodedWorkspace}/snippets`)}
                  >
                    Snippets
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => navigate(`/workspaces/${encodedWorkspace}/agents`)}
                  >
                    Agents
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant={detail.pinned ? "secondary" : "outline"}
                    onClick={handlePinToggle}
                    disabled={isPinPending}
                    aria-pressed={detail.pinned}
                  >
                    {detail.pinned ? "Unpin" : "Pin"}
                  </Button>
                  {showDeleteAction && (
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          disabled={isDeletePending}
                        >
                          Delete
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete workspace</AlertDialogTitle>
                          <AlertDialogDescription>
                            {(!detail.external || detail.isFork)
                              ? <>This will permanently delete the workspace directory from disk. This action cannot be undone.</>
                              : <>This will remove &lsquo;{detail.name}&rsquo; from the interface. The directory and its contents will NOT be deleted.</>
                            }
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={handleDelete}
                            disabled={isDeletePending}
                            className="bg-destructive text-white hover:bg-destructive/90"
                          >
                            {(!detail.external || detail.isFork) ? "Delete" : "Remove"}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  )}
                </>
              )}
          </div>
        </header>

        {showStatusLine && (
          <div className="flex flex-wrap items-center gap-2 mb-2">
            {agentModelLabel && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge variant="secondary" className="font-mono">
                    {agentModelLabel}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent>{fullAgentModelLabel || agentModelLabel}</TooltipContent>
              </Tooltip>
            )}
            {statusLine && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="text-sm text-muted-foreground truncate max-w-[320px] md:max-w-[520px]">
                    {statusLine}
                  </span>
                </TooltipTrigger>
                <TooltipContent>{statusLine}</TooltipContent>
              </Tooltip>
            )}
          </div>
        )}

          {actionError && (
            <p className="text-sm text-destructive mb-2" role="alert">
              {actionError}
            </p>
          )}
          <TabNav
            workspaceName={detail.name}
            activeTab={activeTab}
            isRoot={detail.isRoot}
            hasForks={hasForks}
          />

        </div>

        <div className="pt-4">
          {detail.isRoot && !detail.isFork && (
            <div className="mb-6">
              <InlineForkEditor workspaceName={detail.name} />
            </div>
          )}
          {isForkedRoot && (actionRunError || isActionRunning || actionOutput) ? (
            <div className="space-y-3 mb-4">
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
          ) : null}
          <Suspense fallback={<TabSkeleton />}>
            <TabContent
              activeTab={activeTab}
              workspaceName={detail.name}
              currentModel={detail.currentModel}
              goalContent={detail.goalContent}
              pmContent={detail.pmContent}
              hasProjectMgmt={detail.hasProjectMgmt}
              actions={detail.actions}
              onActionClick={isForkedRoot ? handleActionClick : undefined}
            />
          </Suspense>
        </div>
    </div>
  );
}

function TabSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-24 w-full rounded-xl" />
      <Skeleton className="h-32 w-full rounded-xl" />
    </div>
  );
}

function TabContent({
  activeTab,
  workspaceName,
  currentModel,
  goalContent,
  pmContent,
  hasProjectMgmt,
  actions,
  onActionClick,
}: {
  activeTab: string;
  workspaceName: string;
  currentModel?: string;
  goalContent?: string;
  pmContent?: string;
  hasProjectMgmt?: boolean;
  actions?: ApiActionEntry[];
  onActionClick?: (action: ApiActionEntry, forkName: string) => void;
}) {
  switch (activeTab) {
    case "progress":
      return <EventsTab workspaceName={workspaceName} goalContent={goalContent} actions={actions} />;
    case "log":
      return <LogTab workspaceName={workspaceName} />;
    case "changes":
      return <ChangesTab workspaceName={workspaceName} />;
    case "commits":
      return <CommitsTab workspaceName={workspaceName} />;
    case "messages":
      return <MessagesTab workspaceName={workspaceName} />;
    case "internals":
      return <SessionTab workspaceName={workspaceName} pmContent={pmContent} hasProjectMgmt={hasProjectMgmt} />;

    case "run":
      return <RunTab workspaceName={workspaceName} currentModel={currentModel} />;
    case "forks":
      return <ForksTab workspaceName={workspaceName} actions={actions} onActionClick={onActionClick} />;
    default:
      return <NotYetAvailable pageName={`${activeTab.charAt(0).toUpperCase() + activeTab.slice(1)} Tab`} />;
  }
}

function NoWorkspaceState({ name, dir }: { name: string; dir: string }) {
  return (
    <div>
      <div className="sticky top-0 z-10 bg-background">
        <header className="flex items-center gap-3 mb-3 pb-3 border-b">
          <h3 className="m-0 text-xl font-semibold" title={dir}>{name}</h3>
          <Badge variant="secondary">no workspace</Badge>
        </header>
      </div>
      <div className="text-center py-8 text-muted-foreground italic">
        <p>No workspace configured for this directory.</p>
        <Link
          to={`/workspaces/${encodeURIComponent(name)}/goal/edit`}
          className="inline-block mt-4 px-4 py-2 text-sm rounded border hover:bg-muted transition-colors no-underline"
        >
          Edit GOAL
        </Link>
      </div>
    </div>
  );
}
