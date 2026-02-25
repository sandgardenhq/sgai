import { useEffect, useRef } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useFactoryState } from "@/lib/factory-state";
import type { ApiLogEntry } from "@/types";

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
  const scrollRef = useRef<HTMLDivElement>(null);

  const { workspaces, fetchStatus } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName);
  const lines = workspace?.log ?? [];

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [lines]);

  if (fetchStatus === "fetching" && !workspace) return <LogTabSkeleton />;

  if (!workspace) {
    if (fetchStatus === "error") {
      return (
        <p className="text-sm text-destructive">
          Failed to load log
        </p>
      );
    }
    return null;
  }

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
