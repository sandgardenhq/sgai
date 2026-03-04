import { useState, useCallback, useEffect, useTransition } from "react";
import { useNavigate, useParams, Link } from "react-router";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { api, ApiError } from "@/lib/api";
import { stripFrontmatter } from "@/lib/markdown-utils";
import { ArrowLeft, GitFork, Loader2 } from "lucide-react";

export function NewFork() {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [content, setContent] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSubmitting, startSubmitTransition] = useTransition();

  useEffect(() => {
    if (!workspaceName) return;
    let cancelled = false;
    api.workspaces.forkTemplate(workspaceName).then(
      (result) => { if (!cancelled && result.content) setContent(result.content); },
      () => {},
    );
    return () => { cancelled = true; };
  }, [workspaceName]);

  const bodyText = stripFrontmatter(content).trim();
  const isBodyEmpty = bodyText.length === 0;

  const handleContentChange = useCallback((value: string | undefined) => {
    const newValue = value ?? "";
    setContent(newValue);
    const newBody = stripFrontmatter(newValue).trim();
    if (newBody.length > 0) {
      setValidationError(null);
    }
  }, []);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (isBodyEmpty) {
        setValidationError("Please write a goal description");
        return;
      }
      if (isSubmitting || !workspaceName) return;
      setValidationError(null);
      setSubmitError(null);
      startSubmitTransition(async () => {
        try {
          const result = await api.workspaces.fork(workspaceName, content);
          navigate(`/workspaces/${encodeURIComponent(result.name)}/progress`);
        } catch (err) {
          if (err instanceof ApiError) {
            setSubmitError(err.message);
          } else {
            setSubmitError("Failed to create fork");
          }
        }
      });
    },
    [isBodyEmpty, isSubmitting, workspaceName, content, navigate],
  );

  return (
    <div className="max-w-2xl mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to {workspaceName}
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Fork Workspace</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Write a GOAL.md for the new fork of{" "}
        <span className="font-medium text-foreground">{workspaceName}</span>.
        The fork name will be generated automatically.
      </p>

      {submitError ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{submitError}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <MarkdownEditor
          value={content}
          onChange={handleContentChange}
          minHeight={200}
          defaultHeight={300}
          disabled={isSubmitting}
          placeholder="Describe the goal for this fork..."
          workspaceName={workspaceName}
        />

        {validationError ? (
          <p className="text-sm text-destructive" role="alert">
            {validationError}
          </p>
        ) : null}

        <Button
          type="submit"
          disabled={isSubmitting || isBodyEmpty}
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
