import { useState, useEffect, useCallback, useMemo, useTransition, type ReactNode, type CSSProperties } from "react";
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
  SidebarRail,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { Loader2, Inbox, Trash2 } from "lucide-react";
import { api } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { useSidebarResize } from "@/hooks/useSidebarResize";
import { cn } from "@/lib/utils";
import type { ApiWorkspaceEntry } from "@/lib/factory-state";

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
  workspace: ApiWorkspaceEntry;
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

interface ForkItemProps {
  fork: ApiWorkspaceEntry["forks"] extends (infer F)[] | undefined ? F : never;
  selectedName: string | undefined;
  rootWorkspaceName: string;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
}

function ForkItem({ fork, selectedName, rootWorkspaceName, workspaceLookup }: ForkItemProps) {
  const forkSelected = fork.name === selectedName;
  const forkFullEntry = workspaceLookup.get(fork.name);

  const forkDescription = useMemo(() => {
    return forkFullEntry?.description || fork.description || null;
  }, [forkFullEntry?.description, fork.description]);

  return (
    <SidebarMenuItem>
      <div className="flex items-center gap-0 group/row">
        <SidebarMenuButton
          asChild
          isActive={forkSelected}
          className={cn(
            "flex-1 min-w-0 relative before:content-[''] before:absolute before:left-[-0.875rem] before:top-1/2 before:w-3.5 before:h-0.5 before:bg-border before:rounded-sm",
            forkSelected && "border-l-[3px] border-l-primary bg-primary/15 font-medium"
          )}
        >
          <Link to={`/workspaces/${encodeURIComponent(fork.name)}/progress`}>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
                  {forkDescription ?? <span className="italic text-muted-foreground">No description</span>}
                </span>
              </TooltipTrigger>
              <TooltipContent side="right">
                <div className="max-w-xs">
                  <div className="font-medium">{fork.name}</div>
                  {forkDescription?.endsWith("...") && (
                    <div className="text-xs text-muted-foreground mt-1">{forkFullEntry?.description || fork.description}</div>
                  )}
                </div>
              </TooltipContent>
            </Tooltip>
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

interface WorkspaceTreeItemProps {
  workspace: ApiWorkspaceEntry;
  selectedName: string | undefined;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
}

function WorkspaceTreeItem({ workspace, selectedName, workspaceLookup }: WorkspaceTreeItemProps) {
  const isSelected = workspace.name === selectedName;
  const hasForks = workspace.forks && workspace.forks.length > 0;
  const hasForkSelected = workspace.forks?.some((f) => f.name === selectedName) ?? false;
  const [expanded, setExpanded] = useState(isSelected || hasForkSelected);

  useEffect(() => {
    if (isSelected || hasForkSelected) {
      setExpanded(true);
    }
  }, [isSelected, hasForkSelected]);

  const showDelete = !workspace.isRoot || !hasForks;
  const description = workspace.description || null;

  const isRootWithForks = workspace.isRoot && hasForks;
  const displayText = isRootWithForks ? workspace.name : description;
  const tooltipText = isRootWithForks ? (description ?? workspace.name) : workspace.name;

  return (
    <SidebarMenuItem className="mb-0.5">
      <div className="flex items-center gap-0 group/row">
        {hasForks ? (
          <button
            type="button"
            onClick={() => setExpanded((prev) => !prev)}
            className="w-5 h-5 inline-flex items-center justify-center rounded text-xs font-semibold shrink-0 mr-1 bg-muted text-muted-foreground hover:bg-secondary hover:text-secondary-foreground transition-colors self-start mt-1"
            aria-label="Toggle forks"
          >
            {expanded ? "−" : "+"}
          </button>
        ) : (
          <span className="w-5 h-5 inline-block shrink-0 mr-1" />
        )}
        <SidebarMenuButton
          asChild
          isActive={isSelected}
          className={cn(
            "flex-1 min-w-0",
            isSelected && "border-l-[3px] border-l-primary bg-primary/15 font-medium"
          )}
        >
          <Link to={`/workspaces/${encodeURIComponent(workspace.name)}/progress`}>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className={cn("flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap", isRootWithForks && "font-semibold")}>
                  {displayText ?? <span className="italic text-muted-foreground">No description</span>}
                </span>
              </TooltipTrigger>
              <TooltipContent side="right">
                <div className="max-w-xs">
                  <div>{tooltipText}</div>
                  {description?.endsWith("...") && !isRootWithForks && (
                    <div className="text-xs text-muted-foreground mt-1">{workspace.description}</div>
                  )}
                </div>
              </TooltipContent>
            </Tooltip>
            <WorkspaceIndicators workspace={workspace} />
          </Link>
        </SidebarMenuButton>
        {showDelete && (
          <DeleteWorkspaceDialog
            workspaceName={workspace.name}
            workspaceDir={workspace.dir}
            isFork={workspace.isFork}
            selectedName={selectedName}
          />
        )}
      </div>

      {hasForks && expanded && (
        <div className="ml-2.5 pl-4 relative before:content-[''] before:absolute before:left-2.5 before:top-0 before:bottom-2 before:w-0.5 before:bg-border before:rounded-sm">
          <SidebarMenu>
            {workspace.forks?.map((fork) => (
              <ForkItem
                key={fork.name}
                fork={fork}
                selectedName={selectedName}
                rootWorkspaceName={workspace.name}
                workspaceLookup={workspaceLookup}
              />
            ))}
          </SidebarMenu>
        </div>
      )}
    </SidebarMenuItem>
  );
}

interface InProgressItemProps {
  workspace: ApiWorkspaceEntry;
  selectedName: string | undefined;
}

function InProgressItem({ workspace, selectedName }: InProgressItemProps) {
  const isSelected = workspace.name === selectedName;
  const description = workspace.description || null;

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        isActive={isSelected}
        className={cn(
          "ml-2 mb-0.5",
          !isSelected && "hover:bg-destructive/10",
          isSelected && "border-l-[3px] border-l-primary bg-primary/15 font-medium"
        )}
      >
        <Link
          to={workspace.needsInput
            ? `/workspaces/${encodeURIComponent(workspace.name)}/respond`
            : `/workspaces/${encodeURIComponent(workspace.name)}/progress`
          }
        >
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
                {description ?? <span className="italic text-muted-foreground">No description</span>}
              </span>
            </TooltipTrigger>
            <TooltipContent side="right">
              <div className="max-w-xs">
                <div className="font-medium">{workspace.name}</div>
                {description?.endsWith("...") && (
                  <div className="text-xs text-muted-foreground mt-1">{workspace.description}</div>
                )}
              </div>
            </TooltipContent>
          </Tooltip>
          <WorkspaceIndicators workspace={workspace} />
        </Link>
      </SidebarMenuButton>
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
    <div className="mb-3 pb-2 border-b" role="region" aria-label="In progress">
      <SidebarMenu>
        {inProgress.map((w) => (
          <InProgressItem key={w.name} workspace={w} selectedName={selectedName} />
        ))}
      </SidebarMenu>
    </div>
  );
}

interface WorkspaceListProps {
  workspaces: ApiWorkspaceEntry[];
  selectedName: string | undefined;
}

function WorkspaceList({ workspaces, selectedName }: WorkspaceListProps) {
  const workspaceLookup = useMemo(() => {
    const map = new Map<string, ApiWorkspaceEntry>();
    for (const w of workspaces) {
      map.set(w.name, w);
    }
    return map;
  }, [workspaces]);

  return (
    <>
      <InProgressSection workspaces={workspaces} selectedName={selectedName} />
      <SidebarMenu>
        {workspaces.length > 0 ? (
          workspaces.filter((w) => !w.isFork).map((workspace) => (
            <WorkspaceTreeItem
              key={workspace.name}
              workspace={workspace}
              selectedName={selectedName}
              workspaceLookup={workspaceLookup}
            />
          ))
        ) : (
          <p className="text-sm text-muted-foreground italic p-2">No workspaces found.</p>
        )}
      </SidebarMenu>
    </>
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
  return deduplicateByName(all);
}

interface SidebarHeaderIndicatorsProps {
  workspaces: ApiWorkspaceEntry[];
}

function SidebarHeaderIndicators({ workspaces }: SidebarHeaderIndicatorsProps) {
  const navigate = useNavigate();
  const allWorkspaces = useMemo(() => collectAllWorkspaces(workspaces), [workspaces]);

  const needsInputCount = useMemo(
    () => allWorkspaces.filter((w) => w.needsInput).length,
    [allWorkspaces],
  );

  const hasAnyRunning = useMemo(
    () => allWorkspaces.some((w) => w.running || w.needsInput),
    [allWorkspaces],
  );

  const handleInboxClick = useCallback(() => {
    const firstNeedsInput = allWorkspaces.find((w) => w.needsInput);
    if (firstNeedsInput) {
      navigate(`/workspaces/${encodeURIComponent(firstNeedsInput.name)}/respond`);
    }
  }, [allWorkspaces, navigate]);

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
  onSidebarResizeMouseDown: (e: React.MouseEvent) => void;
}

function DashboardContent({ children, onSidebarResizeMouseDown }: DashboardContentProps): JSX.Element {
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
              <WorkspaceList workspaces={workspaces} selectedName={selectedName} />
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
        <SidebarRail />
        <div
          className="absolute inset-y-0 right-0 z-30 hidden w-1.5 cursor-col-resize bg-transparent hover:bg-primary/20 transition-colors md:block"
          onMouseDown={onSidebarResizeMouseDown}
          aria-hidden="true"
        />
      </Sidebar>

      <div className="flex-1 flex flex-col min-w-0">
        <MobileHeader workspaces={workspaces} loading={loading} error={error} />
        <div className="hidden md:flex items-center gap-2 pl-2 pt-2">
          <SidebarTrigger />
        </div>
        <main className="flex-1 overflow-auto pt-2 md:pt-0 md:pl-4">
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
  const { sidebarWidth, handleMouseDown } = useSidebarResize();

  return (
    <SidebarProvider
      style={{ "--sidebar-width": `${sidebarWidth}px` } as CSSProperties}
    >
      <div className="flex min-h-[calc(100vh-4rem)] w-full">
        <DashboardContent onSidebarResizeMouseDown={handleMouseDown}>{children}</DashboardContent>
      </div>
    </SidebarProvider>
  );
}
