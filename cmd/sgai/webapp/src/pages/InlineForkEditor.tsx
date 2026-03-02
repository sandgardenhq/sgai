import { useState, useCallback, useEffect, useTransition } from "react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { api, ApiError } from "@/lib/api";
import { stripFrontmatter } from "@/lib/markdown-utils";
import { useForkTemplate } from "@/hooks/useForkTemplate";
import { Loader2, GitFork } from "lucide-react";

interface InlineForkEditorProps {
  workspaceName: string;
}

export function InlineForkEditor({ workspaceName }: InlineForkEditorProps) {
  const navigate = useNavigate();
  const templateContent = useForkTemplate(workspaceName);
  const [content, setContent] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSubmitting, startSubmitTransition] = useTransition();

  useEffect(() => {
    if (templateContent) {
      setContent(templateContent);
    }
  }, [templateContent]);

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

  const handleSubmit = useCallback(() => {
    if (isBodyEmpty) {
      setValidationError("Please write a goal description");
      return;
    }
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
  }, [isBodyEmpty, workspaceName, content, navigate]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-semibold mb-1">New Task</h3>
        <p className="text-sm text-muted-foreground">
          Write a GOAL.md for your new task. A fork will be created automatically.
        </p>
      </div>

      {submitError && (
        <Alert className="border-destructive/50 text-destructive">
          <AlertDescription>{submitError}</AlertDescription>
        </Alert>
      )}

      <MarkdownEditor
        value={content}
        onChange={handleContentChange}
        minHeight={300}
        defaultHeight={400}
        disabled={isSubmitting}
        placeholder="Describe the goal for this task..."
        workspaceName={workspaceName}
      />

      {validationError && (
        <p className="text-sm text-destructive" role="alert">
          {validationError}
        </p>
      )}

      <div className="flex justify-end">
        <Button
          type="button"
          onClick={handleSubmit}
          disabled={isSubmitting || isBodyEmpty}
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
      </div>
    </div>
  );
}
