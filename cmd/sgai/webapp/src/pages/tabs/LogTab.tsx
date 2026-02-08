import { useState, useEffect, useRef } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { api } from "@/lib/api";
import { useSSEEvent } from "@/hooks/useSSE";
import type { ApiLogResponse, ApiLogEntry } from "@/types";

interface LogTabProps {
  workspaceName: string;
}

function LogTabSkeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 10 }, (_, i) => (
        <Skeleton key={i} className="h-5 w-full rounded" />
      ))}
    </div>
  );
}

function LogLine({ line }: { line: ApiLogEntry }) {
  return (
    <div className="font-mono text-xs leading-5 whitespace-pre-wrap break-all">
      {line.prefix && <span className="text-muted-foreground select-none">{line.prefix}</span>}
      <span>{line.text}</span>
    </div>
  );
}

export function LogTab({ workspaceName }: LogTabProps) {
  const [data, setData] = useState<ApiLogResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);
  const scrollRef = useRef<HTMLDivElement>(null);

  const logEvent = useSSEEvent("log:append");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;
    setLoading((prev) => !data ? true : prev);
    setError(null);

    api.workspaces
      .log(workspaceName)
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

  useEffect(() => {
    if (logEvent !== null) {
      setRefreshKey((k) => k + 1);
    }
  }, [logEvent]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [data]);

  if (loading && !data) return <LogTabSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-destructive">
        Failed to load log: {error.message}
      </p>
    );
  }

  if (!data) return null;

  const lines = data.lines ?? [];

  if (lines.length === 0) {
    return <p className="text-sm italic text-muted-foreground">No logs available</p>;
  }

  return (
    <ScrollArea ref={scrollRef} className="max-h-[calc(100vh-16rem)] bg-muted/20 rounded-lg p-3">
      <div id="log-lines">
        {lines.map((line, index) => (
          <LogLine key={index} line={line} />
        ))}
      </div>
    </ScrollArea>
  );
}
