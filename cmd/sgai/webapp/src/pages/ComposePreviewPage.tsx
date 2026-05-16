import { useReducer, useState, useEffect } from "react";
import { useSearchParams, Link } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ArrowLeft, FileText, AlertTriangle, Copy, Check } from "lucide-react";
import { api } from "@/lib/api";
import type { ApiComposePreviewResponse } from "@/types";

export function ComposePreviewPage() {
  const [searchParams] = useSearchParams();
  const workspace = searchParams.get("workspace") ?? "";
  const [{ preview, isLoading }, setPreviewState] = useReducer(
    (_: { preview: ApiComposePreviewResponse | null; isLoading: boolean }, state: { preview: ApiComposePreviewResponse | null; isLoading: boolean }) => state,
    { preview: null, isLoading: true },
  );
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!workspace) return;

    let cancelled = false;

    async function loadPreview() {
      let nextPreview: ApiComposePreviewResponse | null = null;
      try {
        nextPreview = await api.compose.preview(workspace);
      } catch {
        // Silently handle
      }
      if (!cancelled) {
        setPreviewState({ preview: nextPreview, isLoading: false });
      }
    }

    loadPreview();
    return () => { cancelled = true; };
  }, [workspace]);

  const handleCopy = async () => {
    if (!preview?.content) return;
    try {
      await navigator.clipboard.writeText(preview.content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API may not be available
    }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <nav className="mb-6 flex items-center justify-between">
        <Link
          to={`/compose?workspace=${encodeURIComponent(workspace)}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="inline size-4 mr-1" />
          Back to Composer
        </Link>
        <Button variant="outline" size="sm" onClick={handleCopy} disabled={!preview?.content}>
          {copied ? (
            <>
              <Check className="size-4 mr-1" />
              Copied!
            </>
          ) : (
            <>
              <Copy className="size-4 mr-1" />
              Copy
            </>
          )}
        </Button>
      </nav>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="size-5" />
            GOAL.md Preview
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-4 w-3/4" />
            </div>
          ) : (
            <>
              <pre className="text-sm leading-relaxed whitespace-pre-wrap break-words font-mono text-muted-foreground bg-muted/50 p-4 rounded-lg">
                {preview?.content ?? "No preview available"}
              </pre>
              {preview?.flowError ? (
                <Alert className="mt-4 border-destructive/50 text-destructive">
                  <AlertTriangle className="size-4" />
                  <AlertDescription>{preview.flowError}</AlertDescription>
                </Alert>
              ) : null}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
