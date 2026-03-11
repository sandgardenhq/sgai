import { useState, useEffect, useCallback, useMemo, useTransition, type ReactNode, type CSSProperties } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
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
  AlertDialogAction,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
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
import { Loader2, Inbox, Trash2, FolderPlus, Link as LinkIcon } from "lucide-react";
import { api } from "@/lib/api";
import { useFactoryState, triggerFactoryRefresh } from "@/lib/factory-state";
import { useSidebarResize } from "@/hooks/useSidebarResize";
import { cn } from "@/lib/utils";
import type { ApiWorkspaceEntry } from "@/lib/factory-state";

type ForkEntry = NonNullable<ApiWorkspaceEntry["forks"]>[number];

function workspaceToForkEntry(ws: ApiWorkspaceEntry): ForkEntry {
  return {
    name: ws.name,
    dir: ws.dir,
    running: ws.running,
    needsInput: ws.needsInput,
    inProgress: ws.inProgress,
    pinned: ws.pinned,
    description: ws.description,
    commitAhead: 0,
    commits: [],
  };
}

interface DeleteWorkspaceDialogProps {
  workspaceName: string;
  isExternal: boolean;
  isFork: boolean;
  selectedName: string | undefined;
  rootName?: string;
}

function DeleteWorkspaceDialog({
  workspaceName,
  isExternal,
  isFork,
  selectedName,
  rootName,
}: DeleteWorkspaceDialogProps) {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [isDeleting, startDeleteTransition] = useTransition();
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const handleDelete = useCallback(() => {
    setDeleteError(null);
    startDeleteTransition(async () => {
      try {
        if (isFork) {
          await api.workspaces.deleteFork(workspaceName, "");
        } else {
          await api.workspaces.deleteWorkspace(workspaceName);
        }
        triggerFactoryRefresh();
        setOpen(false);
        if (isFork && rootName) {
          navigate(`/workspaces/${encodeURIComponent(rootName)}/forks`);
        } else if (selectedName === workspaceName) {
          navigate("/");
        }
      } catch (err) {
        setDeleteError(err instanceof Error ? err.message : "Failed to delete workspace");
      }
    });
  }, [isFork, workspaceName, selectedName, rootName, navigate]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setDeleteError(null);
    }
  }, []);

  const dialogTitle = isFork ? "Delete fork" : isExternal ? "Detach workspace" : "Delete workspace";
  const dialogDescription = isExternal && !isFork
    ? `This will remove '${workspaceName}' from the interface. The directory and its contents will NOT be deleted.`
    : `This will permanently delete '${workspaceName}' from disk. This action cannot be undone.`;

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          onClick={(e) => { e.stopPropagation(); }}
          className="opacity-0 group-hover/row:opacity-100 focus:opacity-100 h-6 w-6 p-0.5 rounded hover:bg-destructive/20 transition-opacity shrink-0"
          aria-label={`Delete ${workspaceName}`}
        >
          <Trash2 className="h-3 w-3 text-muted-foreground hover:text-destructive" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{dialogTitle}</AlertDialogTitle>
          <AlertDialogDescription>{dialogDescription}</AlertDialogDescription>
        </AlertDialogHeader>
        {deleteError && (
          <p className="text-sm text-destructive" role="alert">{deleteError}</p>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            disabled={isDeleting}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {isDeleting ? "Deleting..." : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
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

interface WorkspaceIndicatorFields {
  running: boolean;
  needsInput: boolean;
  pinned: boolean;
  external?: boolean;
}

interface WorkspaceIndicatorsProps {
  workspace: WorkspaceIndicatorFields;
}

function WorkspaceIndicators({ workspace }: WorkspaceIndicatorsProps) {
  const isActive = workspace.running;
  const runningLabel = workspace.running ? "Running" : "In progress";

  return (
    <span className="flex items-center gap-1 shrink-0">
      {workspace.external && (
        <Tooltip>
          <TooltipTrigger asChild>
            <LinkIcon className="h-3 w-3 text-muted-foreground" aria-label="External workspace" title="External workspace" />
          </TooltipTrigger>
          <TooltipContent>External workspace</TooltipContent>
        </Tooltip>
      )}
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
  fork: ForkEntry;
  selectedName: string | undefined;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
  rootName?: string;
}

function ForkItem({ fork, selectedName, workspaceLookup, rootName }: ForkItemProps) {
  const forkSelected = fork.name === selectedName;
  const forkFullEntry = workspaceLookup.get(fork.name);

  const forkDescription = forkFullEntry?.description || fork.description || null;

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
          isExternal={false}
          isFork
          selectedName={selectedName}
          rootName={rootName}
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
  const fullWorkspace = workspaceLookup.get(workspace.name);
  const forks = fullWorkspace?.forks || workspace.forks || [];
  const hasForks = forks.length > 0;
  const isSelected = workspace.name === selectedName;
  const hasForkSelected = forks.some((f) => f.name === selectedName);
  const [expanded, setExpanded] = useState(() => isSelected || hasForkSelected);

  useEffect(() => {
    if (isSelected || hasForkSelected) {
      setExpanded(true);
    }
  }, [isSelected, hasForkSelected]);

  const isRoot = fullWorkspace?.isRoot ?? workspace.isRoot;
  const showDelete = !isRoot || !hasForks;
  const description = workspace.description || null;

  const displayText = isRoot ? workspace.name : description;
  const tooltipText = workspace.name;

  return (
    <SidebarMenuItem className="mb-0.5">
      <div className="flex items-center gap-0 group/row">
        {hasForks ? (
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setExpanded((prev) => !prev)}
            className="w-5 h-6 p-0 text-xs font-semibold shrink-0 mr-1 bg-muted text-muted-foreground hover:bg-secondary hover:text-secondary-foreground transition-colors self-start mt-1"
            aria-label="Toggle forks"
          >
            {expanded ? "−" : "+"}
          </Button>
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
                <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
                  {displayText ?? <span className="italic text-muted-foreground">No description</span>}
                </span>
              </TooltipTrigger>
              <TooltipContent side="right">
                <div className="max-w-xs">
                  <div>{tooltipText}</div>
                  {description?.endsWith("...") && (
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
            isExternal={workspace.external ?? false}
            isFork={workspace.isFork}
            selectedName={selectedName}
          />
        )}
      </div>

      {hasForks && expanded && (
        <div className="ml-2.5 pl-4 relative before:content-[''] before:absolute before:left-2.5 before:top-0 before:bottom-2 before:w-0.5 before:bg-border before:rounded-sm">
          <SidebarMenu>
            {forks.map((fork) => (
              <ForkItem
                key={fork.name}
                fork={fork}
                selectedName={selectedName}
                workspaceLookup={workspaceLookup}
                rootName={workspace.name}
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
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
}

function InProgressItem({ workspace, selectedName, workspaceLookup }: InProgressItemProps) {
  const isSelected = workspace.name === selectedName;
  const fullWorkspace = workspaceLookup.get(workspace.name);
  const description = workspace.description || null;

  const isRoot = fullWorkspace?.isRoot ?? workspace.isRoot;
  const displayText = isRoot ? workspace.name : description;
  const tooltipText = workspace.name;

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
                {displayText ?? <span className="italic text-muted-foreground">No description</span>}
              </span>
            </TooltipTrigger>
            <TooltipContent side="right">
              <div className="max-w-xs">
                <div>{tooltipText}</div>
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

interface PinnedTreeItemProps {
  workspace: ApiWorkspaceEntry;
  selectedName: string | undefined;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
  pinnedForks: ForkEntry[];
}

function PinnedTreeItem({ workspace, selectedName, workspaceLookup, pinnedForks }: PinnedTreeItemProps) {
  const fullWorkspace = workspaceLookup.get(workspace.name);
  const isSelected = workspace.name === selectedName;
  const hasForkSelected = pinnedForks.some((f) => f.name === selectedName);
  const [expanded, setExpanded] = useState(() => isSelected || hasForkSelected);

  useEffect(() => {
    if (isSelected || hasForkSelected) {
      setExpanded(true);
    }
  }, [isSelected, hasForkSelected]);

  const isRoot = fullWorkspace?.isRoot ?? workspace.isRoot;
  const description = workspace.description || null;
  const displayText = isRoot ? workspace.name : description;
  const tooltipText = workspace.name;

  return (
    <SidebarMenuItem className="mb-0.5">
      <div className="flex items-center gap-0 group/row">
        {pinnedForks.length > 0 ? (
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setExpanded((prev) => !prev)}
            className="w-5 h-6 p-0 text-xs font-semibold shrink-0 mr-1 bg-muted text-muted-foreground hover:bg-secondary hover:text-secondary-foreground transition-colors self-start mt-1"
            aria-label="Toggle forks"
          >
            {expanded ? "−" : "+"}
          </Button>
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
                <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
                  {displayText ?? <span className="italic text-muted-foreground">No description</span>}
                </span>
              </TooltipTrigger>
              <TooltipContent side="right">
                <div className="max-w-xs">
                  <div>{tooltipText}</div>
                  {description?.endsWith("...") && (
                    <div className="text-xs text-muted-foreground mt-1">{workspace.description}</div>
                  )}
                </div>
              </TooltipContent>
            </Tooltip>
            <WorkspaceIndicators workspace={workspace} />
          </Link>
        </SidebarMenuButton>
        <DeleteWorkspaceDialog
          workspaceName={workspace.name}
          isExternal={workspace.external ?? false}
          isFork={workspace.isFork}
          selectedName={selectedName}
        />
      </div>

      {pinnedForks.length > 0 && expanded && (
        <div className="ml-2.5 pl-4 relative before:content-[''] before:absolute before:left-2.5 before:top-0 before:bottom-2 before:w-0.5 before:bg-border before:rounded-sm">
          <SidebarMenu>
            {pinnedForks.map((fork) => (
              <ForkItem
                key={fork.name}
                fork={fork}
                selectedName={selectedName}
                workspaceLookup={workspaceLookup}
                rootName={workspace.name}
              />
            ))}
          </SidebarMenu>
        </div>
      )}
    </SidebarMenuItem>
  );
}

interface OrphanPinnedForkItemProps {
  fork: ApiWorkspaceEntry;
  rootName: string;
  selectedName: string | undefined;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
}

function OrphanPinnedForkItem({ fork, rootName, selectedName, workspaceLookup }: OrphanPinnedForkItemProps) {
  const forkSelected = fork.name === selectedName;
  const forkFullEntry = workspaceLookup.get(fork.name);
  const forkDescription = forkFullEntry?.description || fork.description || fork.name;
  const displayLabel = `${rootName}/${forkDescription}`;

  return (
    <SidebarMenuItem>
      <div className="flex items-center gap-0 group/row">
        <span className="w-5 h-5 inline-block shrink-0 mr-1" />
        <SidebarMenuButton
          asChild
          isActive={forkSelected}
          className={cn(
            "flex-1 min-w-0",
            forkSelected && "border-l-[3px] border-l-primary bg-primary/15 font-medium"
          )}
        >
          <Link to={`/workspaces/${encodeURIComponent(fork.name)}/progress`}>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
                  {displayLabel}
                </span>
              </TooltipTrigger>
              <TooltipContent side="right">
                <div className="max-w-xs">
                  <div className="font-medium">{fork.name}</div>
                  <div className="text-xs text-muted-foreground mt-1">Root: {rootName}</div>
                </div>
              </TooltipContent>
            </Tooltip>
            <WorkspaceIndicators workspace={fork} />
          </Link>
        </SidebarMenuButton>
        <DeleteWorkspaceDialog
          workspaceName={fork.name}
          isExternal={false}
          isFork
          selectedName={selectedName}
          rootName={rootName}
        />
      </div>
    </SidebarMenuItem>
  );
}

interface PinnedSectionProps {
  workspaces: ApiWorkspaceEntry[];
  selectedName: string | undefined;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
  forkParentLookup: Map<string, string>;
}

function PinnedSection({ workspaces, selectedName, workspaceLookup, forkParentLookup }: PinnedSectionProps) {
  const pinned = useMemo(() => {
    return workspaces.filter((w) => w.pinned);
  }, [workspaces]);

  const pinnedRootsAndForks = useMemo(() => {
    const pinnedForks = pinned.filter((w) => w.isFork);
    const pinnedRoots = pinned.filter((w) => !w.isFork);
    const pinnedRootNames = new Set(pinnedRoots.map((r) => r.name));

    const forkGroups = new Map<string, ForkEntry[]>();
    const orphanForks: Array<{ fork: ApiWorkspaceEntry; rootName: string }> = [];

    for (const fork of pinnedForks) {
      const parentName = forkParentLookup.get(fork.name);
      if (parentName && pinnedRootNames.has(parentName)) {
        const existing = forkGroups.get(parentName) || [];
        existing.push(workspaceToForkEntry(fork));
        forkGroups.set(parentName, existing);
      } else {
        orphanForks.push({ fork, rootName: parentName || fork.name });
      }
    }

    return { pinnedRoots, forkGroups, orphanForks };
  }, [pinned, forkParentLookup]);

  if (pinned.length === 0) return null;

  const { pinnedRoots, forkGroups, orphanForks } = pinnedRootsAndForks;

  return (
    <div className="mb-3 pb-2 border-b" role="region" aria-label="Pinned">
      <SidebarMenu>
        {pinnedRoots.map((root) => (
          <PinnedTreeItem
            key={root.name}
            workspace={root}
            selectedName={selectedName}
            workspaceLookup={workspaceLookup}
            pinnedForks={forkGroups.get(root.name) || []}
          />
        ))}
        {orphanForks.map(({ fork, rootName }) => (
          <OrphanPinnedForkItem
            key={fork.name}
            fork={fork}
            rootName={rootName}
            selectedName={selectedName}
            workspaceLookup={workspaceLookup}
          />
        ))}
      </SidebarMenu>
    </div>
  );
}

interface InProgressSectionProps {
  workspaces: ApiWorkspaceEntry[];
  selectedName: string | undefined;
  workspaceLookup: Map<string, ApiWorkspaceEntry>;
}

function InProgressSection({ workspaces, selectedName, workspaceLookup }: InProgressSectionProps) {
  const inProgress = useMemo(() => {
    return workspaces.filter((w) => (w.inProgress || w.running) && !w.pinned);
  }, [workspaces]);

  if (inProgress.length === 0) return null;

  return (
    <div className="mb-3 pb-2 border-b" role="region" aria-label="In progress">
      <SidebarMenu>
        {inProgress.map((w) => (
          <InProgressItem key={w.name} workspace={w} selectedName={selectedName} workspaceLookup={workspaceLookup} />
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
      const existing = map.get(w.name);
      if (!existing || (w.isRoot && !existing.isRoot)) {
        map.set(w.name, w);
      }
    }
    return map;
  }, [workspaces]);

  const forkParentLookup = useMemo(() => {
    const lookup = new Map<string, string>();
    for (const w of workspaces) {
      if (w.forks) {
        for (const fork of w.forks) {
          lookup.set(fork.name, w.name);
        }
      }
    }
    return lookup;
  }, [workspaces]);

  const deduplicatedWorkspaces = useMemo(() => {
    return deduplicateWorkspacesByName(workspaces);
  }, [workspaces]);

  return (
    <>
      <PinnedSection
        workspaces={deduplicatedWorkspaces}
        selectedName={selectedName}
        workspaceLookup={workspaceLookup}
        forkParentLookup={forkParentLookup}
      />
      <InProgressSection
        workspaces={deduplicatedWorkspaces}
        selectedName={selectedName}
        workspaceLookup={workspaceLookup}
      />
      <SidebarMenu>
        {deduplicatedWorkspaces.length > 0 ? (
          deduplicatedWorkspaces.filter((w) => !w.isFork).map((workspace) => (
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

interface WorkspaceStatusEntry {
  name: string;
  running: boolean;
  needsInput: boolean;
}

function deduplicateByName<T extends { name: string }>(workspaces: T[]): T[] {
  const seen = new Set<string>();
  return workspaces.filter((w) => {
    if (seen.has(w.name)) return false;
    seen.add(w.name);
    return true;
  });
}

function deduplicateWorkspacesByName(workspaces: ApiWorkspaceEntry[]): ApiWorkspaceEntry[] {
  const map = new Map<string, ApiWorkspaceEntry>();
  for (const w of workspaces) {
    const existing = map.get(w.name);
    if (!existing || (w.isRoot && !existing.isRoot)) {
      map.set(w.name, w);
    }
  }
  return Array.from(map.values());
}

function collectAllWorkspaces(workspaces: ApiWorkspaceEntry[]): WorkspaceStatusEntry[] {
  const all: WorkspaceStatusEntry[] = [];
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
            <Button
              variant="ghost"
              size="icon"
              onClick={handleInboxClick}
              aria-label={needsInputCount === 1
                ? "1 workspace waiting for response"
                : `${needsInputCount} workspaces waiting for response`}
              className="relative inline-flex items-center text-primary bg-transparent border-0 p-0 h-auto w-auto"
            >
              <Inbox className="h-4 w-4" />
              <Badge
                variant="destructive"
                className="absolute -top-2 -right-2.5 h-4 min-w-4 px-1 text-[0.6rem] leading-none flex items-center justify-center rounded-full"
              >
                {needsInputCount}
              </Badge>
            </Button>
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

  const handleNewWorkspace = useCallback(() => {
    navigate("/workspaces/new");
  }, [navigate]);

  const handleAttachExternal = useCallback(() => {
    navigate("/workspaces/attach");
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
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" className="w-full">
                [ + ]
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="top" align="center" className="w-48">
              <DropdownMenuItem onClick={handleNewWorkspace}>
                <FolderPlus className="mr-2 h-4 w-4" />
                New Workspace
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleAttachExternal}>
                <LinkIcon className="mr-2 h-4 w-4" />
                Attach External
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
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

  const sidebarStyle = useMemo(
    () => ({ "--sidebar-width": `${sidebarWidth}px` } as CSSProperties),
    [sidebarWidth]
  );

  return (
    <SidebarProvider style={sidebarStyle}>
      <div className="flex min-h-[calc(100vh-4rem)] w-full">
        <DashboardContent onSidebarResizeMouseDown={handleMouseDown}>{children}</DashboardContent>
      </div>
    </SidebarProvider>
  );
}
