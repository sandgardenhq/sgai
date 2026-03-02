import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, Save, Loader2, Check } from "lucide-react";
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

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        handleSave();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleSave]);

  if (isLoading) {
    return (
      <div className="fixed inset-0 z-[60] flex flex-col bg-background">
        <div className="flex items-center gap-3 px-4 py-2 border-b bg-background">
          <Skeleton className="h-8 w-8" />
          <Skeleton className="h-5 w-32" />
          <div className="ml-auto">
            <Skeleton className="h-8 w-24" />
          </div>
        </div>
        <div className="flex-1 p-4">
          <Skeleton className="h-full w-full" />
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-[60] flex flex-col bg-background">
      <div className="flex items-center gap-3 px-4 py-2 border-b bg-background shrink-0">
        <Link
          to={`/workspaces/${encodeURIComponent(workspaceName)}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1"
          aria-label={`Back to ${workspaceName}`}
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <span className="text-sm font-medium">Edit GOAL.md</span>

        {error ? (
          <Alert className="py-1 px-3 border-destructive/50 text-destructive flex items-center gap-2 h-8">
            <AlertDescription className="text-xs">{error}</AlertDescription>
          </Alert>
        ) : null}

        <div className="ml-auto">
          <Button
            size="sm"
            onClick={handleSave}
            disabled={isSaving || saveSuccess || !content.trim()}
          >
            {saveSuccess ? (
              <>
                <Check className="mr-2 h-4 w-4" />
                Saved!
              </>
            ) : isSaving ? (
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

      <div className="flex-1 min-h-0">
        <MarkdownEditor
          value={content}
          onChange={(v) => setContent(v ?? "")}
          disabled={isSaving || saveSuccess}
          workspaceName={workspaceName}
          fillHeight
        />
      </div>
    </div>
  );
}
