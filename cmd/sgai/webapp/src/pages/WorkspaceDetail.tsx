import { useState, useEffect, Suspense, lazy, useTransition, useRef, useCallback } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { NotYetAvailable } from "@/components/NotYetAvailable";
import { api } from "@/lib/api";
import { useSSEEvent, useWorkspaceSSEEvent } from "@/hooks/useSSE";
import type { ApiWorkspaceDetailResponse } from "@/types";
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

interface InlineSummaryEditorProps {
  workspaceName: string;
  summary: string | undefined;
  onSaved: (newSummary: string) => void;
}

function InlineSummaryEditor({ workspaceName, summary, onSaved }: InlineSummaryEditorProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(summary ?? "");
  const [isSaving, startSaveTransition] = useTransition();
  const [saveError, setSaveError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!editing) {
      setDraft(summary ?? "");
    }
  }, [summary, editing]);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editing]);

  const handleStartEdit = useCallback(() => {
    setDraft(summary ?? "");
    setSaveError(null);
    setEditing(true);
  }, [summary]);

  const handleCancel = useCallback(() => {
    setEditing(false);
    setDraft(summary ?? "");
    setSaveError(null);
  }, [summary]);

  const handleSave = useCallback(() => {
    const trimmed = draft.trim();
    setSaveError(null);
    startSaveTransition(async () => {
      try {
        await api.workspaces.updateSummary(workspaceName, trimmed);
        onSaved(trimmed);
        setEditing(false);
      } catch (err) {
        setSaveError(err instanceof Error ? err.message : "Failed to save summary");
      }
    });
  }, [draft, workspaceName, onSaved]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        handleSave();
      } else if (e.key === "Escape") {
        e.preventDefault();
        handleCancel();
      }
    },
    [handleSave, handleCancel],
  );

  if (editing) {
    return (
      <div className="flex items-center gap-2 mt-1 max-w-lg">
        <Input
          ref={inputRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Enter a summary..."
          disabled={isSaving}
          className="h-7 text-sm"
          aria-label="Edit workspace summary"
        />
        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={handleSave}
          disabled={isSaving}
          className="h-7 px-2 text-xs shrink-0"
          aria-label="Save summary"
        >
          {isSaving ? "…" : "✓"}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={handleCancel}
          disabled={isSaving}
          className="h-7 px-2 text-xs shrink-0"
          aria-label="Cancel editing"
        >
          ✕
        </Button>
        {saveError && (
          <span className="text-xs text-destructive shrink-0" role="alert">{saveError}</span>
        )}
      </div>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={handleStartEdit}
          className="mt-1 text-sm text-muted-foreground truncate max-w-md cursor-pointer bg-transparent border-0 p-0 text-left hover:text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded-sm"
          aria-label="Click to edit summary"
        >
          {summary ? summary : <span className="italic">No summary yet</span>}
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        {summary ? summary : "Click to add a summary"}
      </TooltipContent>
    </Tooltip>
  );
}

export function WorkspaceDetail(): JSX.Element | null {
  const { name, "*": tabPath } = useParams<{ name: string; "*": string }>();
  const workspaceName = name ?? "";
  const activeTab = tabPath?.split("/")[0] || "progress";
  const navigate = useNavigate();

  const [detail, setDetail] = useState<ApiWorkspaceDetailResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const [actionError, setActionError] = useState<string | null>(null);
  const previousWorkspaceRef = useRef<string | null>(null);
  const [isStartStopPending, startStartStopTransition] = useTransition();
  const [isSelfDrivePending, startSelfDriveTransition] = useTransition();
  const [isPinPending, startPinTransition] = useTransition();
  const [isEditorPending, startEditorTransition] = useTransition();
  const [isOpenCodePending, startOpenCodeTransition] = useTransition();
  const [isResetPending, startResetTransition] = useTransition();
  const [execTimeSeconds, setExecTimeSeconds] = useState<number | null>(null);

  const sessionUpdateEvent = useWorkspaceSSEEvent(workspaceName, "session:update");
  const workspaceUpdateEvent = useSSEEvent("workspace:update");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    const isWorkspaceChange = previousWorkspaceRef.current !== workspaceName;
    if (isWorkspaceChange) {
      previousWorkspaceRef.current = workspaceName;
      setDetail(null);
      setLoading(true);
    } else if (!detail) {
      setLoading(true);
    }
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
    if (!workspaceName) return;
    setActionError(null);
  }, [workspaceName]);

  useEffect(() => {
    if (!workspaceName) return;
    if (sessionUpdateEvent !== null || workspaceUpdateEvent !== null) {
      setRefreshKey((k) => k + 1);
    }
  }, [sessionUpdateEvent, workspaceUpdateEvent, workspaceName]);

  useEffect(() => {
    if (!workspaceName || !detail?.running) return;
    const timer = setInterval(() => {
      setRefreshKey((k) => k + 1);
    }, 3000);
    return () => clearInterval(timer);
  }, [workspaceName, detail?.running]);

  useEffect(() => {
    if (!detail) return;
    const parsed = parseExecTime(detail.totalExecTime ?? "");
    if (parsed !== null) {
      setExecTimeSeconds(parsed);
    } else if (detail.running) {
      setExecTimeSeconds(0);
    } else {
      setExecTimeSeconds(null);
    }
    if (!detail.running) return;
    const timer = setInterval(() => {
      setExecTimeSeconds((prev) => (prev === null ? prev : prev + 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [detail?.totalExecTime, detail?.running]);

  const hasForks = (detail?.forks?.length ?? 0) > 0;
  const isForkedRoot = Boolean(detail?.isRoot && hasForks);
  const isInterrupted = detail ? (detail.status === "working" && !detail.running) : false;

  const handleReset = () => {
    if (!workspaceName) return;
    setActionError(null);
    startResetTransition(async () => {
      try {
        await api.workspaces.reset(workspaceName);
        setRefreshKey((k) => k + 1);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to reset session");
      }
    });
  };

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

  useEffect(() => {
    if (error && error.message.toLowerCase().includes("workspace not found")) {
      navigate("/", { replace: true });
    }
  }, [error, navigate]);

  const handleSummarySaved = useCallback((newSummary: string) => {
    setDetail((prev) => prev ? { ...prev, summary: newSummary, summaryManual: true } : prev);
  }, []);

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
  const selfDriveLabel = detail.running ? "Self-Drive" : "Self-drive";
  const showForkAction = !detail.isFork && !detail.running;
  const showComposeGoalAction = !detail.running;
  const showEditGoalAction = detail.hasSgai || Boolean(detail.goalContent?.trim());
  const showOpenEditorAction = !detail.running;
  const showOpenOpencodeAction = detail.running;
  const isActionDisabled = detail.running || isStartStopPending || isSelfDrivePending;

  const handleStart = () => {
    if (!workspaceName) return;
    setActionError(null);
    startStartStopTransition(async () => {
      try {
        const response = await api.workspaces.start(workspaceName, false);
        setDetail((prev) => prev ? { ...prev, running: response.running } : prev);
        setRefreshKey((k) => k + 1);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to start session");
      }
    });
  };

  const handleStop = () => {
    if (!workspaceName) return;
    setActionError(null);
    startStartStopTransition(async () => {
      try {
        const response = await api.workspaces.stop(workspaceName);
        setDetail((prev) => prev ? { ...prev, running: response.running } : prev);
        setRefreshKey((k) => k + 1);
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
        const response = await api.workspaces.selfdrive(workspaceName);
        setDetail((prev) => prev ? { ...prev, running: response.running, interactiveAuto: response.autoMode } : prev);
        setRefreshKey((k) => k + 1);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to toggle self-drive");
      }
    });
  };

  const handlePinToggle = () => {
    if (!workspaceName) return;
    setActionError(null);
    startPinTransition(async () => {
      try {
        const response = await api.workspaces.togglePin(workspaceName);
        setDetail((prev) => prev ? { ...prev, pinned: response.pinned } : prev);
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

  const handleOpenOpencode = () => {
    if (!workspaceName) return;
    setActionError(null);
    startOpenCodeTransition(async () => {
      try {
        await api.workspaces.openOpencode(workspaceName);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Failed to open OpenCode");
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
                  {detail.isFork ? (
                    <Link
                      to={`/workspaces/${encodeURIComponent(detail.name)}/rename`}
                      className="no-underline text-inherit"
                    >
                      {detail.name} ✏️
                    </Link>
                  ) : (
                    detail.name
                  )}
                </h3>
              </TooltipTrigger>
              <TooltipContent>{detail.dir}</TooltipContent>
            </Tooltip>
            <InlineSummaryEditor
              workspaceName={workspaceName}
              summary={detail.summary}
              onSaved={handleSummarySaved}
            />
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
              <Badge variant={detail.running ? "default" : "secondary"}>
                {detail.running ? "running" : "stopped"}
              </Badge>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2 w-full md:w-auto md:ml-auto mt-2 md:mt-0 justify-start md:justify-end">
              {isForkedRoot ? (
                <>
                  {showForkAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/workspaces/${encodedWorkspace}/fork/new`)}
                    >
                      Fork
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
                  {showOpenOpencodeAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={handleOpenOpencode}
                      disabled={isOpenCodePending}
                    >
                      Open in OpenCode
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
                        variant={(detail.running && detail.interactiveAuto) ? "default" : "outline"}
                        onClick={handleSelfDrive}
                        disabled={isActionDisabled}
                        aria-pressed={detail.running && detail.interactiveAuto}
                      >
                        Continuous Self-Drive
                      </Button>
                      {detail.running && (
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
                        variant={(detail.running && detail.interactiveAuto) ? "default" : "outline"}
                        onClick={handleSelfDrive}
                        disabled={isActionDisabled}
                        aria-pressed={detail.running && detail.interactiveAuto}
                      >
                        {selfDriveLabel}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant={(detail.running && !detail.interactiveAuto) ? "default" : "outline"}
                        onClick={handleStart}
                        disabled={isActionDisabled}
                        aria-pressed={detail.running && !detail.interactiveAuto}
                      >
                        Start
                      </Button>
                      {detail.running && (
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
                  {showForkAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/workspaces/${encodedWorkspace}/fork/new`)}
                    >
                      Fork
                    </Button>
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
                  {showOpenOpencodeAction && (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={handleOpenOpencode}
                      disabled={isOpenCodePending}
                    >
                      Open in OpenCode
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

          {isInterrupted && (
            <Alert className="mt-2 bg-slate-800 text-white border-slate-700 flex items-center justify-between gap-4">
              <AlertDescription className="flex-1 text-white">
                sgai was interrupted while working. Reset state to start fresh.
              </AlertDescription>
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="border-white/50 text-white bg-transparent hover:bg-white/10 hover:text-white shrink-0"
                onClick={handleReset}
                disabled={isResetPending}
              >
                Reset
              </Button>
            </Alert>
          )}
        </div>

        <div className="pt-4">
          <Suspense fallback={<TabSkeleton />}>
            <TabContent
              activeTab={activeTab}
              workspaceName={detail.name}
              currentModel={detail.currentModel}
              goalContent={detail.goalContent}
              pmContent={detail.pmContent}
              hasProjectMgmt={detail.hasProjectMgmt}
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
}: {
  activeTab: string;
  workspaceName: string;
  currentModel?: string;
  goalContent?: string;
  pmContent?: string;
  hasProjectMgmt?: boolean;
}) {
  switch (activeTab) {
    case "progress":
      return <EventsTab workspaceName={workspaceName} goalContent={goalContent} />;
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
      return <ForksTab workspaceName={workspaceName} />;
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
