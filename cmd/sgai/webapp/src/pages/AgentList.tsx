import { useState, useEffect, useCallback } from "react";
import { Link, useParams } from "react-router";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { AgentsResponse } from "@/types";

function AgentListSkeleton() {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
      {Array.from({ length: 6 }, (_, i) => (
        <Card key={i}>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-4 w-full" />
            <Skeleton className="mt-2 h-4 w-3/4" />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export function AgentList() {
  const { name } = useParams<{ name: string }>();
  const workspaceName = name ?? "";

  const [data, setData] = useState<AgentsResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    api.agents
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
          <span className="font-semibold">Agents</span>
        </div>
        <button
          type="button"
          onClick={handleRefresh}
          className="px-3 py-1 text-sm rounded border hover:bg-muted transition-colors"
        >
          Refresh
        </button>
      </nav>

      {loading && <AgentListSkeleton />}

      {error && (
        <p className="text-sm text-destructive">
          Failed to load agents: {error.message}
        </p>
      )}

      {data && (!data.agents || data.agents.length === 0) && (
        <p className="text-sm text-muted-foreground italic">
          No agents found.
        </p>
      )}

      {data && data.agents && data.agents.length > 0 && (
        <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
          {data.agents.map((agent) => (
            <Card key={agent.name}>
              <CardHeader>
                <CardTitle>{agent.name}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {agent.description}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
