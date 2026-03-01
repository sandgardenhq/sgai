import { useState, useEffect, useCallback, useMemo, useTransition, type ReactNode } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import sgaiLogo from "@/assets/sgai-logo.svg";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { Loader2, Inbox, Trash2, Plus } from "lucide-react";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { cn } from "@/lib/utils";
import type { ApiWorkspaceEntry } from "@/lib/factory-state";
import type { ApiForkEntry, WorkspaceIndicatorData } from "@/types";

interface DeleteWorkspaceDialogProps {
  workspaceName: string;
  workspaceDir: string;
  isFork: boolean;
  rootWorkspaceName?: string;
  selectedName: string | undefined;
}

function DeleteWorkspaceDialog({
  workspaceName,
  workspaceDir,
  isFork,
  rootWorkspaceName,
  selectedName,
}: DeleteWorkspaceDialogProps) {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [confirmText, setConfirmText] = useState("");
  const [isDeleting, startDeleteTransition] = useTransition();
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const nameMatches = confirmText === workspaceName;

  const handleDelete = useCallback(() => {
    if (!nameMatches) return;
    setDeleteError(null);
    startDeleteTransition(async () => {
      try {
        if (isFork && rootWorkspaceName) {
          await api.workspaces.deleteFork(rootWorkspaceName, workspaceDir);
        } else {
          await api.workspaces.deleteWorkspace(workspaceName);
        }
        setOpen(false);
        if (selectedName === workspaceName) {
          navigate("/");
        }
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Failed to delete workspace");
      }
    });
  }, [nameMatches, isFork, rootWorkspaceName, workspaceDir, workspaceName, selectedName, navigate]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setConfirmText("");
      setDeleteError(null);
    }
  }, []);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); }}
          className="opacity-0 group-hover/row:opacity-100 focus:opacity-100 p-0.5 rounded hover:bg-destructive/20 transition-opacity shrink-0"
          aria-label={`Delete ${workspaceName}`}
        >
          <Trash2 className="h-3 w-3 text-muted-foreground hover:text-destructive" />
        </button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete workspace</DialogTitle>
          <DialogDescription>
            Type &lsquo;{workspaceName}&rsquo; to confirm deletion. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <div className="py-2">
          <Input
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder={workspaceName}
            disabled={isDeleting}
            aria-label="Type workspace name to confirm"
            autoFocus
          />
          {deleteError && (
            <p className="text-sm text-destructive mt-2" role="alert">{deleteError}</p>
          )}
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={!nameMatches || isDeleting}
          >
            {isDeleting ? "Deleting..." : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

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
  workspace: WorkspaceIndicatorData;
}

function WorkspaceIndicators({ workspace }: WorkspaceIndicatorsProps) {
  const isActive = workspace.running;
  const runningLabel = workspace.running ? "Running" : "In progress";

  return (
    <span className="flex items-center gap-1 shrink-0">
      {isActive && (
        <Loader2 className="h-3 w-3 text-primary animate-spin" aria-label={runningLabel} title={runningLabel} />
      )}
      {workspace.needsInput && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Inbox className="h-3 w-3 text-primary" aria-label="Waiting for response" title="Waiting for response" />
          </TooltipTrigger>
          <TooltipContent>Waiting for response</TooltipContent>
        </Tooltip>
      )}
      {workspace.pinned && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="text-[0.7rem] opacity-70">📌</span>
          </TooltipTrigger>
          <TooltipContent>Pinned</TooltipContent>
        </Tooltip>
      )}
    </span>
  );
}

function isForkedMode(workspace: ApiWorkspaceEntry): boolean {
  return workspace.isRoot && (workspace.forks?.length ?? 0) > 0;
}

interface PromotedForkItemProps {
  fork: ApiForkEntry;
  rootWorkspaceName: string;
  selectedName: string | undefined;
}

function PromotedForkItem({ fork, rootWorkspaceName, selectedName }: PromotedForkItemProps) {
  const isSelected = fork.name === selectedName;
  const displayText = fork.goalDescription || fork.name;
  const tooltipText = fork.goalDescription || fork.name;

  return (
    <SidebarMenuItem className="mb-0.5">
      <div className="flex items-center gap-0 group/row">
        <SidebarMenuButton
          asChild
          isActive={isSelected}
          className="flex-1 min-w-0"
        >
          <Link to={`/workspaces/${encodeURIComponent(fork.name)}/progress`}>
            <span className="flex-1 min-w-0 flex flex-col">
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="overflow-hidden text-ellipsis whitespace-nowrap">
                    {displayText}
                  </span>
                </TooltipTrigger>
                <TooltipContent side="right" className="max-w-xs">{tooltipText}</TooltipContent>
              </Tooltip>
            </span>
            <WorkspaceIndicators workspace={fork} />
          </Link>
        </SidebarMenuButton>
        <DeleteWorkspaceDialog
          workspaceName={fork.name}
          workspaceDir={fork.dir}
          isFork
          rootWorkspaceName={rootWorkspaceName}
          selectedName={selectedName}
        />
      </div>
    </SidebarMenuItem>
  );
}

interface ForkedModeHeaderProps {
  workspace: ApiWorkspaceEntry;
  selectedName: string | undefined;
}

function ForkedModeHeader({ workspace, selectedName }: ForkedModeHeaderProps) {
  const isSelected = workspace.name === selectedName;

  return (
    <SidebarMenuItem className="mb-1">
      <SidebarMenuButton
        asChild
        isActive={isSelected}
        className="flex-1 min-w-0"
      >
        <Link to={`/workspaces/${encodeURIComponent(workspace.name)}/create-task`}>
          <span className="flex-1 min-w-0 flex items-center gap-1">
            <Plus className="h-3.5 w-3.5 shrink-0" />
            <span className="text-sm">New task</span>
          </span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

interface WorkspaceTreeItemProps {
  workspace: ApiWorkspaceEntry;
  selectedName: string | undefined;
}

function StandaloneWorkspaceItem({ workspace, selectedName }: WorkspaceTreeItemProps) {
  const isSelected = workspace.name === selectedName;

  return (
    <SidebarMenuItem className="mb-0.5">
      <div className="flex items-center gap-0 group/row">
        <SidebarMenuButton
          asChild
          isActive={isSelected}
          className="flex-1 min-w-0"
        >
          <Link to={`/workspaces/${encodeURIComponent(workspace.name)}/progress`}>
            <span className="flex-1 min-w-0 flex flex-col">
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="overflow-hidden text-ellipsis whitespace-nowrap">
                    {workspace.name}
                  </span>
                </TooltipTrigger>
                <TooltipContent side="right">{workspace.name}</TooltipContent>
              </Tooltip>
              {workspace.summary && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="text-[0.65rem] text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap leading-tight">
                      {workspace.summary}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="right" className="max-w-xs">{workspace.summary}</TooltipContent>
                </Tooltip>
              )}
            </span>
            <WorkspaceIndicators workspace={workspace} />
          </Link>
        </SidebarMenuButton>
        <DeleteWorkspaceDialog
          workspaceName={workspace.name}
          workspaceDir={workspace.dir}
          isFork={workspace.isFork}
          selectedName={selectedName}
        />
      </div>
    </SidebarMenuItem>
  );
}

interface InProgressSectionProps {
  workspaces: ApiWorkspaceEntry[];
  selectedName: string | undefined;
}

function InProgressSection({ workspaces, selectedName }: InProgressSectionProps) {
  const inProgress = deduplicateByName(workspaces.filter((w) => w.inProgress || w.running));

  if (inProgress.length === 0) return null;

  return (
    <div className="mb-3 pb-2 border-b">
      <SidebarMenu>
        {inProgress.map((w) => {
          const isSelected = w.name === selectedName;
          const displayText = w.goalDescription || w.name;
          return (
            <SidebarMenuItem key={w.name}>
              <SidebarMenuButton
                asChild
                isActive={isSelected}
                className={cn(
                  "ml-2 mb-0.5",
                  !isSelected && "hover:bg-destructive/10"
                )}
              >
                <Link
                  to={w.needsInput
                    ? `/workspaces/${encodeURIComponent(w.name)}/respond`
                    : `/workspaces/${encodeURIComponent(w.name)}/progress`
                  }
                >
                  <span className="flex-1 min-w-0 flex flex-col">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="overflow-hidden text-ellipsis whitespace-nowrap">
                          {displayText}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="right" className="max-w-xs">{displayText}</TooltipContent>
                    </Tooltip>
                  </span>
                  <WorkspaceIndicators workspace={w} />
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          );
        })}
      </SidebarMenu>
    </div>
  );
}

function deduplicateByName(workspaces: ApiWorkspaceEntry[]): ApiWorkspaceEntry[] {
  const seen = new Set<string>();
  return workspaces.filter((w) => {
    if (seen.has(w.name)) return false;
    seen.add(w.name);
    return true;
  });
}

interface SidebarEntry {
  name: string;
  running: boolean;
  needsInput: boolean;
}

function collectAllWorkspaces(workspaces: ApiWorkspaceEntry[]): SidebarEntry[] {
  const all: SidebarEntry[] = [];
  const seen = new Set<string>();
  for (const w of workspaces) {
    if (!seen.has(w.name)) {
      seen.add(w.name);
      all.push(w);
    }
    if (w.forks) {
      for (const fork of w.forks) {
        if (!seen.has(fork.name)) {
          seen.add(fork.name);
          all.push(fork);
        }
      }
    }
  }
  return all;
}

function SidebarHeaderIndicators({ workspaces }: { workspaces: ApiWorkspaceEntry[] }) {
  const navigate = useNavigate();
  const allEntries = useMemo(() => collectAllWorkspaces(workspaces), [workspaces]);

  const needsInputCount = useMemo(
    () => allEntries.filter((w) => w.needsInput).length,
    [allEntries],
  );

  const hasAnyRunning = useMemo(
    () => allEntries.some((w) => w.running || w.needsInput),
    [allEntries],
  );

  const handleInboxClick = useCallback(() => {
    const firstNeedsInput = allEntries.find((w) => w.needsInput);
    if (firstNeedsInput) {
      navigate(`/workspaces/${encodeURIComponent(firstNeedsInput.name)}/respond`);
    }
  }, [allEntries, navigate]);

  return (
    <div className="flex items-center gap-2">
      {needsInputCount > 0 && (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              onClick={handleInboxClick}
              aria-label={needsInputCount === 1
                ? "1 workspace waiting for response"
                : `${needsInputCount} workspaces waiting for response`}
              className="relative inline-flex items-center text-primary cursor-pointer bg-transparent border-0 p-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded-sm"
            >
              <Inbox className="h-4 w-4" />
              <Badge
                variant="destructive"
                className="absolute -top-2 -right-2.5 h-4 min-w-4 px-1 text-[0.6rem] leading-none flex items-center justify-center rounded-full"
              >
                {needsInputCount}
              </Badge>
            </button>
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

function MobileHeader({ workspaces, loading, error }: { workspaces: ApiWorkspaceEntry[]; loading: boolean; error: Error | null }) {
  return (
    <div className="flex items-center gap-2 pb-3 md:hidden">
      <SidebarTrigger />
      <span className="text-sm font-semibold">Workspaces</span>
      <span className="flex-1 flex justify-center">
        <img src={sgaiLogo} alt="SGAI" className="h-[28px] w-auto" />
      </span>
      {!loading && !error && (
        <SidebarHeaderIndicators workspaces={workspaces} />
      )}
    </div>
  );
}

interface DashboardContentProps {
  children: ReactNode;
}

function DashboardContent({ children }: DashboardContentProps): JSX.Element {
  const { name: selectedName } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const { setOpenMobile } = useSidebar();

  const { workspaces, fetchStatus } = useFactoryState();
  const loading = fetchStatus === "fetching" && workspaces.length === 0;
  const error = fetchStatus === "error" && workspaces.length === 0
    ? new Error("Failed to load workspaces")
    : null;

  useEffect(() => {
    if (selectedName) {
      setOpenMobile(false);
    }
  }, [selectedName, setOpenMobile]);

  const handleCreateWorkspace = useCallback(() => {
    navigate("/workspaces/new");
  }, [navigate]);

  return (
    <>
      <Sidebar side="left" collapsible="offcanvas">
        <SidebarHeader className="px-3 py-2">
          <div>
            <img src={sgaiLogo} alt="SGAI" className="h-[35px] w-auto" />
          </div>
          <Separator />
          <div className="flex items-center justify-between pt-2">
            <span className="text-sm font-semibold">Workspaces</span>
            {!loading && !error && <SidebarHeaderIndicators workspaces={workspaces} />}
          </div>
        </SidebarHeader>
        <Separator />
        <SidebarContent>
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
                <SidebarMenu>
                  {workspaces.length > 0 ? (
                    workspaces.filter((w) => !w.isFork).map((workspace) => {
                      if (isForkedMode(workspace)) {
                        return (
                          <div key={workspace.name}>
                            <ForkedModeHeader workspace={workspace} selectedName={selectedName} />
                            {workspace.forks?.map((fork) => (
                              <PromotedForkItem
                                key={fork.name}
                                fork={fork}
                                rootWorkspaceName={workspace.name}
                                selectedName={selectedName}
                              />
                            ))}
                          </div>
                        );
                      }
                      return (
                        <StandaloneWorkspaceItem
                          key={workspace.name}
                          workspace={workspace}
                          selectedName={selectedName}
                        />
                      );
                    })
                  ) : (
                    <p className="text-sm text-muted-foreground italic p-2">No workspaces found.</p>
                  )}
                </SidebarMenu>
              </>
            )}
          </ScrollArea>
        </SidebarContent>
        <Separator />
        <SidebarFooter className="p-2">
          <Button
            variant="outline"
            className="w-full"
            onClick={handleCreateWorkspace}
          >
            [ + ]
          </Button>
        </SidebarFooter>
      </Sidebar>

      <div className="flex-1 flex flex-col min-w-0">
        <MobileHeader workspaces={workspaces} loading={loading} error={error} />
        <main className="flex-1 overflow-auto pt-4 md:pt-0 md:pl-4">
          {children}
        </main>
      </div>
    </>
  );
}

interface DashboardProps {
  children: ReactNode;
}

export function Dashboard({ children }: DashboardProps): JSX.Element {
  return (
    <SidebarProvider>
      <div className="flex min-h-[calc(100vh-4rem)] w-full">
        <DashboardContent>{children}</DashboardContent>
      </div>
    </SidebarProvider>
  );
}
