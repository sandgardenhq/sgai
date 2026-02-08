import { useCallback, useState } from "react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, FolderPlus, Loader2 } from "lucide-react";
import { Link } from "react-router";

export function NewWorkspace() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = name.trim();
      if (!trimmed || isSubmitting) return;

      setIsSubmitting(true);
      setError(null);

      try {
        const result = await api.workspaces.create(trimmed);
        navigate(`/compose?workspace=${encodeURIComponent(result.name)}`);
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Failed to create workspace");
        }
      } finally {
        setIsSubmitting(false);
      }
    },
    [name, isSubmitting, navigate],
  );

  return (
    <div className="max-w-lg mx-auto py-8">
      <Link
        to="/"
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to Dashboard
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Create New Workspace</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Create a new workspace directory for your project.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="workspace-name">Workspace Name</Label>
          <Input
            id="workspace-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-project"
            autoFocus
            disabled={isSubmitting}
          />
          <p className="text-xs text-muted-foreground">
            Use lowercase letters, numbers, and hyphens only.
          </p>
        </div>

        <Button
          type="submit"
          disabled={isSubmitting || !name.trim()}
          className="w-full"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creating...
            </>
          ) : (
            <>
              <FolderPlus className="mr-2 h-4 w-4" />
              Create Workspace
            </>
          )}
        </Button>
      </form>
    </div>
  );
}
