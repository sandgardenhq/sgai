import { useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import { MissingWorkspaceNotice } from "@/components/MissingWorkspaceNotice";
import type { ApiComposeTemplateEntry } from "@/types";

export function ComposeTemplateRedirect() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const workspace = searchParams.get("workspace") ?? "";
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!workspace || !id) return;

    let cancelled = false;

    async function applyTemplate() {
      setError(null);

      try {
        const resp = await api.compose.templates();
        const template = resp.templates.find((entry) => entry.id === id);
        if (!template) {
          throw new Error("Template not found");
        }

        const draft = buildDraftRequest(template);
        await api.compose.saveDraft(workspace, draft);

        if (!cancelled) {
          navigate(`/compose/step/1?workspace=${encodeURIComponent(workspace)}`, { replace: true });
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to apply template");
        }
      }
    }

    applyTemplate();

    return () => {
      cancelled = true;
    };
  }, [workspace, id, navigate]);

  if (!workspace) {
    return <MissingWorkspaceNotice />;
  }

  if (error) {
    return (
      <div className="max-w-lg mx-auto py-8 space-y-4">
        <Alert className="border-destructive/50 text-destructive">
          <AlertTitle>Template unavailable</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
        <Link
          to={`/compose?workspace=${encodeURIComponent(workspace)}`}
          className="text-sm text-primary hover:underline"
        >
          ← Back to templates
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-lg mx-auto py-8 space-y-4" role="status" aria-live="polite">
      <Skeleton className="h-6 w-40" />
      <Skeleton className="h-4 w-64" />
      <Skeleton className="h-32 w-full" />
    </div>
  );
}

function buildDraftRequest(template: ApiComposeTemplateEntry) {
  return {
    state: {
      description: "",
      interactive: template.interactive,
      completionGate: "",
      agents: template.agents,
      flow: template.flow,
      tasks: "",
    },
    wizard: {
      currentStep: 1,
      fromTemplate: template.id,
      description: "",
      techStack: [],
      safetyAnalysis: false,
      interactive: template.interactive,
      completionGate: "",
    },
  };
}
