import { useState, useEffect, useCallback } from "react";
import { Link, useParams } from "react-router";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import type { SkillsResponse } from "@/types";

function SkillListSkeleton() {
  return (
    <div>
      {Array.from({ length: 3 }, (_, i) => (
        <div key={i} className="mb-6">
          <Skeleton className="mb-3 h-6 w-40" />
          <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
            {Array.from({ length: 4 }, (_, j) => (
              <Card key={j}>
                <CardHeader>
                  <Skeleton className="h-5 w-32" />
                </CardHeader>
                <CardContent>
                  <Skeleton className="h-4 w-full" />
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function SkillList() {
  const { name } = useParams<{ name: string }>();
  const workspaceName = name ?? "";

  const [data, setData] = useState<SkillsResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    api.skills
      .list(workspaceName)
      .then((response) => {
        if (!cancelled) {
          setData(response);
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

  const handleRefresh = useCallback(() => {
    setRefreshKey((k) => k + 1);
  }, []);

  return (
    <div>
      <nav className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link
            to={`/workspaces/${encodeURIComponent(workspaceName)}/progress`}
            className="text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            ← Back
          </Link>
          <span className="font-semibold">Skills</span>
        </div>
        <button
          type="button"
          onClick={handleRefresh}
          className="px-3 py-1 text-sm rounded border hover:bg-muted transition-colors"
        >
          Refresh
        </button>
      </nav>

      {loading && <SkillListSkeleton />}

      {error && (
        <p className="text-sm text-destructive">
          Failed to load skills: {error.message}
        </p>
      )}

      {data && (!data.categories || data.categories.length === 0) && (
        <p className="text-sm text-muted-foreground italic">
          No skills found.
        </p>
      )}

      {data && data.categories && data.categories.length > 0 && (
        <>
          {data.categories.map((category) => (
            <div key={category.name} className="mb-6">
              <h3 className="mb-3 border-b border-border pb-2 text-sm font-medium text-muted-foreground lowercase">
                {category.name}
              </h3>
              <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
                {category.skills.map((skill) => (
                  <Link
                    key={skill.fullPath}
                    to={`/workspaces/${encodeURIComponent(workspaceName)}/skills/${skill.fullPath}`}
                    className="no-underline"
                  >
                    <Card className="cursor-pointer transition-colors hover:bg-muted/50">
                      <CardHeader>
                        <CardTitle>{skill.name}</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <p className="overflow-hidden text-ellipsis whitespace-nowrap text-sm text-muted-foreground">
                              {skill.description}
                            </p>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p className="max-w-xs">{skill.description}</p>
                          </TooltipContent>
                        </Tooltip>
                      </CardContent>
                    </Card>
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  );
}
