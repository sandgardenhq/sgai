import { Skeleton } from "@/components/ui/skeleton";
import { MarkdownContent } from "@/components/MarkdownContent";
import { useFactoryState } from "@/lib/factory-state";

interface SpecificationTabProps {
  workspaceName: string;
}

function SpecificationTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-6 w-32" />
      <Skeleton className="h-48 w-full rounded-xl" />
      <Skeleton className="h-6 w-48" />
      <Skeleton className="h-32 w-full rounded-xl" />
    </div>
  );
}

export function SpecificationTab({ workspaceName }: SpecificationTabProps) {
  const { workspaces, fetchStatus } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);

  if (fetchStatus === "fetching" && !workspace) return <SpecificationTabSkeleton />;

  if (!workspace) {
    if (fetchStatus === "error") {
      return (
        <p className="text-sm text-destructive">
          Failed to load specification
        </p>
      );
    }
    return null;
  }

  return (
    <div className="space-y-4">
      {workspace.goalContent ? (
        <details open>
          <summary className="cursor-pointer font-semibold text-sm mb-2">
            GOAL.md
          </summary>
          <MarkdownContent
            content={workspace.goalContent}
            className="p-4 border rounded-lg bg-muted/20"
          />
        </details>
      ) : (
        <div className="text-center py-8 text-muted-foreground italic">
          <p>No GOAL.md file found</p>
        </div>
      )}

      {workspace.hasProjectMgmt && workspace.pmContent && (
        <details open>
          <summary className="cursor-pointer font-semibold text-sm mb-2">
            PROJECT_MANAGEMENT.md
          </summary>
          <MarkdownContent
            content={workspace.pmContent}
            className="p-4 border rounded-lg bg-muted/20"
          />
        </details>
      )}
    </div>
  );
}
