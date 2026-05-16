import { useReducer, useEffect, useCallback } from "react";
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
import type { SnippetsResponse } from "@/types";

function SnippetListSkeleton() {
  return (
    <div>
      {["language-1", "language-2", "language-3"].map((languageKey) => (
        <div key={languageKey} className="mb-6">
          <Skeleton className="mb-3 h-6 w-40" />
          <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
            {["snippet-1", "snippet-2", "snippet-3", "snippet-4"].map((snippetKey) => (
              <Card key={`${languageKey}-${snippetKey}`}>
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

export function SnippetList() {
  const { name } = useParams<{ name: string }>();
  const workspaceName = name ?? "";

  const [{ data, error, loading, refreshKey }, updateState] = useReducer(
    (
      state: { data: SnippetsResponse | null; error: Error | null; loading: boolean; refreshKey: number },
      update: Partial<{ data: SnippetsResponse | null; error: Error | null; loading: boolean; refreshKey: number }>,
    ) => ({ ...state, ...update }),
    { data: null, error: null, loading: true, refreshKey: 0 },
  );

  useEffect(() => {
    let cancelled = false;
    updateState({ loading: true, error: null });

    api.snippets
      .list(workspaceName)
      .then((response) => {
        if (!cancelled) {
          updateState({ data: response, loading: false });
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          updateState({ error: err instanceof Error ? err : new Error(String(err)), loading: false });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName, refreshKey]);

  const handleRefresh = useCallback(() => {
    updateState({ refreshKey: refreshKey + 1 });
  }, [refreshKey]);

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
          <span className="font-semibold">Snippets</span>
        </div>
        <button
          type="button"
          onClick={handleRefresh}
          className="px-3 py-1 text-sm rounded border hover:bg-muted transition-colors"
        >
          Refresh
        </button>
      </nav>

      {loading && <SnippetListSkeleton />}

      {error && (
        <p className="text-sm text-destructive">
          Failed to load snippets: {error.message}
        </p>
      )}

      {data && (!data.languages || data.languages.length === 0) && (
        <p className="text-sm text-muted-foreground italic">
          No snippets found.
        </p>
      )}

      {data && data.languages && data.languages.length > 0 && (
        <>
          {data.languages.map((lang) => (
            <div key={lang.name} className="mb-6">
              <h3 className="mb-3 border-b border-border pb-2 text-sm font-medium text-muted-foreground lowercase">
                {lang.name}
              </h3>
              <div className="grid grid-cols-[repeat(auto-fit,minmax(300px,1fr))] gap-4">
                {lang.snippets.map((snippet) => (
                  <Link
                    key={snippet.fullPath}
                    to={`/workspaces/${encodeURIComponent(workspaceName)}/snippets/${encodeURIComponent(snippet.language)}/${encodeURIComponent(snippet.fileName)}`}
                    className="no-underline"
                  >
                    <Card className="cursor-pointer transition-colors hover:bg-muted/50">
                      <CardHeader>
                        <CardTitle>{snippet.name}</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <p className="overflow-hidden text-ellipsis whitespace-nowrap text-sm text-muted-foreground">
                              {snippet.description}
                            </p>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p className="max-w-xs">{snippet.description}</p>
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
