import { useState, useEffect } from "react";
import { Link, useParams } from "react-router";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { Snippet } from "@/types";

function SnippetDetailSkeleton() {
  return (
    <div>
      <Skeleton className="mb-2 h-4 w-48" />
      <div className="rounded-xl border p-6">
        <Skeleton className="mb-4 h-7 w-64" />
        <Skeleton className="mb-2 h-4 w-full" />
        <Skeleton className="mb-4 h-4 w-3/4" />
        <Skeleton className="mb-2 h-4 w-24" />
        <Skeleton className="h-40 w-full rounded-md" />
      </div>
    </div>
  );
}

export function SnippetDetail() {
  const { name, lang, fileName } = useParams<{
    name: string;
    lang: string;
    fileName: string;
  }>();
  const workspaceName = name ?? "";
  const snippetLang = lang ?? "";
  const snippetFileName = fileName ?? "";

  const [snippet, setSnippet] = useState<Snippet | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    api.snippets
      .get(snippetLang, snippetFileName, workspaceName)
      .then((response) => {
        if (!cancelled) {
          setSnippet(response);
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
  }, [snippetLang, snippetFileName, workspaceName]);

  return (
    <div>
      <nav className="mb-4 flex items-center gap-2">
        <Link
          to={`/workspaces/${encodeURIComponent(workspaceName)}/snippets`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Snippets
        </Link>
        <span className="font-semibold">{snippetFileName}</span>
      </nav>

      {loading && <SnippetDetailSkeleton />}

      {error && (
        <p className="text-sm text-destructive">
          Failed to load snippet: {error.message}
        </p>
      )}

      {snippet && (
        <div>
          <p className="mb-2 text-sm text-muted-foreground">
            <Link
              to={`/workspaces/${encodeURIComponent(workspaceName)}/snippets`}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              Snippets
            </Link>
            {" / "}
            {snippetLang}
            {" / "}
            {snippetFileName}
          </p>
          <article className="rounded-xl border p-6">
            <h2 className="mb-4 text-xl font-semibold">{snippet.name}</h2>
            {snippet.description && (
              <p className="mb-4 text-sm text-muted-foreground">
                {snippet.description}
              </p>
            )}
            {snippet.whenToUse && (
              <div className="mb-6">
                <strong className="mb-1 block text-sm">When to use:</strong>
                <span className="text-sm text-muted-foreground">
                  {snippet.whenToUse}
                </span>
              </div>
            )}
            <pre className="overflow-x-auto rounded-md bg-muted p-4">
              <code className="text-sm">{snippet.content}</code>
            </pre>
          </article>
        </div>
      )}
    </div>
  );
}
