import { useState, useEffect } from "react";
import { Link, useParams, useLocation } from "react-router";
import { Skeleton } from "@/components/ui/skeleton";
import { MarkdownContent } from "@/components/MarkdownContent";
import { api } from "@/lib/api";
import type { Skill } from "@/types";

function SkillDetailSkeleton() {
  return (
    <div>
      <Skeleton className="mb-2 h-4 w-48" />
      <div className="rounded-xl border p-6">
        <Skeleton className="mb-4 h-6 w-64" />
        <Skeleton className="mb-2 h-4 w-full" />
        <Skeleton className="mb-2 h-4 w-full" />
        <Skeleton className="mb-2 h-4 w-3/4" />
        <Skeleton className="mb-4 h-4 w-full" />
        <Skeleton className="mb-2 h-4 w-full" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    </div>
  );
}

export function SkillDetail() {
  const { name } = useParams<{ name: string }>();
  const location = useLocation();
  const workspaceName = name ?? "";

  const prefix = `/workspaces/${name}/skills/`;
  const fullPath = location.pathname.startsWith(prefix)
    ? location.pathname.slice(prefix.length)
    : "";

  const [skill, setSkill] = useState<Skill | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    api.skills
      .get(fullPath, workspaceName)
      .then((response) => {
        if (!cancelled) {
          setSkill(response);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)));
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [fullPath, workspaceName]);

  return (
    <div>
      <nav className="mb-4 flex items-center gap-2">
        <Link
          to={`/workspaces/${encodeURIComponent(workspaceName)}/skills`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Skills
        </Link>
        <span className="font-semibold">{fullPath.split("/").pop()}</span>
      </nav>

      {loading && <SkillDetailSkeleton />}

      {error && (
        <p className="text-sm text-destructive">
          Failed to load skill: {error.message}
        </p>
      )}

      {skill && (
        <div>
          <p className="mb-2 text-sm text-muted-foreground">
            <Link
              to={`/workspaces/${encodeURIComponent(workspaceName)}/skills`}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              Skills
            </Link>
            {" / "}
            {fullPath}
          </p>
          <article className="rounded-xl border p-6">
            <MarkdownContent
              content={skill.rawContent ?? skill.content}
              className="skill-content"
            />
          </article>
        </div>
      )}
    </div>
  );
}
