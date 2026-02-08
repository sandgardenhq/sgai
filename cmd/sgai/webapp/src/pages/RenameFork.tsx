import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, Pencil, Loader2 } from "lucide-react";
import { Link } from "react-router";
import type { ApiWorkspaceDetailResponse } from "@/types";

export function RenameFork() {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [newName, setNewName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detail, setDetail] = useState<ApiWorkspaceDetailResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    if (!workspaceName) return;
    let cancelled = false;
    setIsLoading(true);
    setLoadError(null);

    api.workspaces
      .get(workspaceName)
      .then((response) => {
        if (!cancelled) {
          setDetail(response);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setLoadError("Failed to load workspace details");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [workspaceName]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = newName.trim();
      if (!trimmed || isSubmitting || !workspaceName) return;

      setIsSubmitting(true);
      setError(null);

      try {
        const result = await api.workspaces.rename(workspaceName, trimmed);
        navigate(`/workspaces/${encodeURIComponent(result.name)}`);
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Failed to rename fork");
        }
      } finally {
        setIsSubmitting(false);
      }
    },
    [newName, isSubmitting, workspaceName, navigate],
  );

  if (isLoading) {
    return (
      <div className="max-w-lg mx-auto py-8 space-y-4">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="max-w-lg mx-auto py-8">
        <Link
          to={`/workspaces/${encodeURIComponent(workspaceName)}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
        >
          <ArrowLeft className="h-3 w-3" />
          Back to {workspaceName}
        </Link>
        <Alert className="border-destructive/50 text-destructive">
          <AlertTitle>Unable to load workspace</AlertTitle>
          <AlertDescription>{loadError}</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!detail) return null;

  if (!detail.isFork) {
    return (
      <div className="max-w-lg mx-auto py-8">
        <Link
          to={`/workspaces/${encodeURIComponent(workspaceName)}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
        >
          <ArrowLeft className="h-3 w-3" />
          Back to {workspaceName}
        </Link>
        <Alert className="border-destructive/50 text-destructive">
          <AlertTitle>Only forks can be renamed.</AlertTitle>
          <AlertDescription>
            This workspace is not a fork. Return to the workspace page to continue.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="max-w-lg mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to {workspaceName}
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Rename Fork</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Rename <span className="font-medium text-foreground">{workspaceName}</span> to a new name.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="new-name">New Name</Label>
          <Input
            id="new-name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="new-fork-name"
            autoFocus
            disabled={isSubmitting}
          />
          <p className="text-xs text-muted-foreground">
            Use lowercase letters, numbers, and hyphens only.
          </p>
        </div>

        <Button
          type="submit"
          disabled={isSubmitting || !newName.trim()}
          className="w-full"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Renaming...
            </>
          ) : (
            <>
              <Pencil className="mr-2 h-4 w-4" />
              Rename Fork
            </>
          )}
        </Button>
      </form>
    </div>
  );
}
