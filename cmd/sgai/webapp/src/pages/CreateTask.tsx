import { useCallback, useEffect, useMemo, useRef, useState, useTransition } from "react";
import { useNavigate, useParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { api, ApiError } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import { Loader2, Rocket } from "lucide-react";

const DEFAULT_FRONTMATTER = `---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "react-developer" -> "react-reviewer"
  "general-purpose"
  "project-critic-council"
  "skill-writer"
  "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
---

`;

function extractFrontmatter(content: string): string {
  const match = content.match(/^(---\n[\s\S]*?\n---)\n?/);
  if (match) {
    return match[1] + "\n\n";
  }
  return "";
}

export function CreateTask(): JSX.Element {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const { workspaces } = useFactoryState();

  const workspace = useMemo(
    () => workspaces.find((w) => w.name === workspaceName),
    [workspaces, workspaceName],
  );

  const storageKey = `create-task-draft-${workspaceName}`;

  const initialContent = useMemo(() => {
    const rawGoal = workspace?.rawGoalContent ?? workspace?.goalContent ?? "";
    const frontmatter = extractFrontmatter(rawGoal);
    return frontmatter || DEFAULT_FRONTMATTER;
  }, [workspace]);

  const [content, setContent] = useState(() => {
    const saved = sessionStorage.getItem(storageKey);
    return saved ?? initialContent;
  });
  const hasUserEdited = useRef(false);
  const [isCreating, startCreateTransition] = useTransition();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!hasUserEdited.current && initialContent !== DEFAULT_FRONTMATTER) {
      setContent(initialContent);
    }
  }, [initialContent]);

  const handleContentChange = useCallback((v: string | undefined) => {
    const newContent = v ?? "";
    hasUserEdited.current = true;
    setContent(newContent);
    sessionStorage.setItem(storageKey, newContent);
  }, [storageKey]);

  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (content !== initialContent) {
        e.preventDefault();
      }
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [content, initialContent]);

  const handleCreate = useCallback(() => {
    if (!workspaceName || !content.trim()) return;
    setError(null);

    startCreateTransition(async () => {
      try {
        const result = await api.workspaces.forkWithGoal(workspaceName, content);
        sessionStorage.removeItem(storageKey);
        navigate(`/workspaces/${encodeURIComponent(result.name)}/progress`);
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Failed to create task");
        }
      }
    });
  }, [workspaceName, content, storageKey, navigate]);

  return (
    <div className="max-w-4xl mx-auto py-8 px-4">
      <h1 className="text-2xl font-semibold mb-2">New Task</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Write a GOAL.md for a new task. The frontmatter configures the agent workflow.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="space-y-4">
        <MarkdownEditor
          value={content}
          onChange={handleContentChange}
          defaultHeight={500}
          disabled={isCreating}
          placeholder="Describe what you want to accomplish..."
        />

        <div className="flex justify-end">
          <Button
            onClick={handleCreate}
            disabled={isCreating || !content.trim()}
            size="lg"
          >
            {isCreating ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Rocket className="mr-2 h-4 w-4" />
                Create
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
