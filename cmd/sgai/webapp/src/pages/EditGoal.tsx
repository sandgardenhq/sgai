import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, Save, Loader2, CheckCircle2 } from "lucide-react";
import { Link } from "react-router";

export function EditGoal(): JSX.Element {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [content, setContent] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const redirectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (redirectTimeoutRef.current) {
        clearTimeout(redirectTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!workspaceName) return;
    let cancelled = false;

    async function loadGoal() {
      setIsLoading(true);
      try {
        const goal = await api.workspaces.getGoal(workspaceName);
        if (!cancelled) {
          setContent(goal.content);
        }
      } catch {
        if (!cancelled) {
          setError("Failed to load GOAL.md");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    loadGoal();
    return () => { cancelled = true; };
  }, [workspaceName]);

  const handleSave = useCallback(async () => {
    if (!workspaceName || isSaving || !content.trim()) return;

    setIsSaving(true);
    setError(null);

    try {
      await api.workspaces.updateGoal(workspaceName, content);
      setSaveSuccess(true);
      if (redirectTimeoutRef.current) {
        clearTimeout(redirectTimeoutRef.current);
      }
      redirectTimeoutRef.current = setTimeout(() => {
        redirectTimeoutRef.current = null;
        navigate(`/workspaces/${encodeURIComponent(workspaceName)}`);
      }, 1000);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to save GOAL.md");
      }
    } finally {
      setIsSaving(false);
    }
  }, [workspaceName, isSaving, content, navigate]);

  if (isLoading) {
    return (
      <div className="max-w-3xl mx-auto py-8 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to {workspaceName}
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Edit GOAL.md</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Edit the GOAL.md content for <span className="font-medium text-foreground">{workspaceName}</span>.
      </p>

      {saveSuccess ? (
        <Alert className="mb-4 border-primary/50 bg-primary/5 text-primary">
          <CheckCircle2 className="h-4 w-4" />
          <AlertTitle>Saved!</AlertTitle>
          <AlertDescription>Redirecting to workspace...</AlertDescription>
        </Alert>
      ) : null}

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="goal-content">GOAL.md Content</Label>
          <MarkdownEditor
            value={content}
            onChange={(v) => setContent(v ?? "")}
            defaultHeight={500}
            disabled={isSaving || saveSuccess}
          />
        </div>

        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            onClick={() => navigate(`/workspaces/${encodeURIComponent(workspaceName)}`)}
            disabled={isSaving}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={isSaving || saveSuccess || !content.trim()}
          >
            {isSaving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                Save GOAL.md
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
