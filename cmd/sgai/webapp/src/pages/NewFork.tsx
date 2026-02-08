import { useCallback, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, GitFork, Loader2 } from "lucide-react";
import { Link } from "react-router";

export function NewFork() {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [forkName, setForkName] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = forkName.trim();
      if (!trimmed || isSubmitting || !workspaceName) return;

      setIsSubmitting(true);
      setError(null);

      try {
        const result = await api.workspaces.fork(workspaceName, trimmed);
        navigate(`/workspaces/${encodeURIComponent(result.name)}`);
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Failed to create fork");
        }
      } finally {
        setIsSubmitting(false);
      }
    },
    [forkName, isSubmitting, workspaceName, navigate],
  );

  return (
    <div className="max-w-lg mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to {workspaceName}
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Fork Workspace</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Create a fork of <span className="font-medium text-foreground">{workspaceName}</span> to work on changes independently.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="fork-name">Fork Name</Label>
          <Input
            id="fork-name"
            value={forkName}
            onChange={(e) => setForkName(e.target.value)}
            placeholder="my-feature-branch"
            autoFocus
            disabled={isSubmitting}
          />
          <p className="text-xs text-muted-foreground">
            Use lowercase letters, numbers, and hyphens only.
          </p>
        </div>

        <Button
          type="submit"
          disabled={isSubmitting || !forkName.trim()}
          className="w-full"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creating Fork...
            </>
          ) : (
            <>
              <GitFork className="mr-2 h-4 w-4" />
              Create Fork
            </>
          )}
        </Button>
      </form>
    </div>
  );
}
