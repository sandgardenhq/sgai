import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { useFactoryState } from "@/lib/factory-state";
import type { ApiDiffLine } from "@/types";

interface ChangesTabProps {
  workspaceName: string;
}

function ChangesTabSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-12 w-full rounded" />
      <Skeleton className="h-64 w-full rounded-xl" />
    </div>
  );
}

function statLineStyle(line: ApiDiffLine): string {
  const trimmed = line.text.trimStart();
  if (trimmed.startsWith("(") || trimmed.includes("changed")) {
    return "text-muted-foreground text-xs italic";
  }
  return "text-sm";
}

function StatLineRow({ line }: { line: ApiDiffLine }) {
  return (
    <div
      className={`font-mono leading-6 whitespace-pre px-3 ${statLineStyle(line)}`}
      data-line-number={line.lineNumber}
    >
      {line.text}
    </div>
  );
}

export function ChangesTab({ workspaceName }: ChangesTabProps) {
  const { workspaces, fetchStatus } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);
  const changes = workspace?.changes;

  if (fetchStatus === "fetching" && !workspace) return <ChangesTabSkeleton />;

  if (!workspace) {
    if (fetchStatus === "error") {
      return (
        <p className="text-sm text-destructive">
          Failed to load changes
        </p>
      );
    }
    return null;
  }

  const diffLines = changes?.diffLines ?? [];

  const handleViewFullDiff = () => {
    window.open(`/workspace/${encodeURIComponent(workspaceName)}/diff`, "_blank");
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Diff Stat</CardTitle>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleViewFullDiff}
            >
              View Full Diff
            </Button>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {diffLines.length > 0 ? (
            <div className="py-2">
              {diffLines.map((line) => (
                <StatLineRow key={line.lineNumber} line={line} />
              ))}
            </div>
          ) : (
            <p className="text-sm italic text-muted-foreground px-6 pb-4">No changes to display</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
