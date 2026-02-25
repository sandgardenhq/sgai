import { useState, useEffect } from "react";
import { useParams } from "react-router";
import { cn } from "@/lib/utils";
import { api } from "@/lib/api";

interface ParsedDiffLine {
  lineNumber: number;
  text: string;
}

function diffLineColor(text: string): string {
  const trimmed = text.trimStart();
  if (trimmed.startsWith("@@")) {
    return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50 font-semibold";
  }
  if (trimmed.startsWith("+++ ") || trimmed.startsWith("--- ")) {
    return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50";
  }
  if (trimmed.startsWith("diff ") || trimmed.startsWith("index ") || trimmed.startsWith("new file") || trimmed.startsWith("deleted file")) {
    return "border-l-4 border-yellow-400 text-yellow-800 bg-yellow-50 font-semibold";
  }
  if (trimmed.startsWith("+") && !trimmed.startsWith("+++")) {
    return "border-l-4 border-green-500 text-green-700 bg-green-50";
  }
  if (trimmed.startsWith("-") && !trimmed.startsWith("---")) {
    return "border-l-4 border-red-500 text-red-700 bg-red-50";
  }
  return "border-l-4 border-transparent";
}

function parseDiff(rawDiff: string): ParsedDiffLine[] {
  if (!rawDiff) return [];
  return rawDiff.split("\n").map((text, idx) => ({
    lineNumber: idx + 1,
    text,
  }));
}

type LoadState = "loading" | "loaded" | "error";

export function FullDiffPage() {
  const { name: workspaceName } = useParams<{ name: string }>();
  const [diffLines, setDiffLines] = useState<ParsedDiffLine[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");

  useEffect(() => {
    if (!workspaceName) return;

    let cancelled = false;

    api.workspaces.getDiff(workspaceName)
      .then((data) => {
        if (!cancelled) {
          setDiffLines(parseDiff(data.diff ?? ""));
          setLoadState("loaded");
        }
      })
      .catch(() => {
        if (!cancelled) {
          setLoadState("error");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName]);

  const pageTitle = workspaceName ? `Full Diff — ${workspaceName}` : "Full Diff";

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-10 border-b bg-background px-4 py-3">
        <h1 className="text-base font-semibold truncate">{pageTitle}</h1>
      </header>

      <main className="p-0">
        {loadState === "loading" && (
          <p className="text-sm text-muted-foreground p-6">Loading diff…</p>
        )}
        {loadState === "error" && (
          <p className="text-sm text-destructive p-6">Failed to load diff.</p>
        )}
        {loadState === "loaded" && diffLines.length === 0 && (
          <p className="text-sm italic text-muted-foreground p-6">No diff to display.</p>
        )}
        {loadState === "loaded" && diffLines.length > 0 && (
          <div className="divide-y divide-border/30">
            {diffLines.map((line) => (
              <div
                key={line.lineNumber}
                className={cn("font-mono text-xs leading-5 whitespace-pre-wrap break-all px-2", diffLineColor(line.text))}
                data-line-number={line.lineNumber}
              >
                {line.text}
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
