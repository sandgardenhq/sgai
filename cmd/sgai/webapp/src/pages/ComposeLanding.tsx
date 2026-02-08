import { useState, useEffect } from "react";
import { Link, useSearchParams } from "react-router";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { ArrowLeft, ArrowRight, Pencil, Wand2 } from "lucide-react";
import { api } from "@/lib/api";
import type { ApiComposeTemplateEntry } from "@/types";

export function ComposeLanding() {
  const [searchParams] = useSearchParams();
  const workspace = searchParams.get("workspace") ?? "";
  const [templates, setTemplates] = useState<ApiComposeTemplateEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function loadTemplates() {
      setIsLoading(true);
      try {
        const resp = await api.compose.templates();
        if (!cancelled) {
          setTemplates(resp.templates);
        }
      } catch {
        // Silently handle error
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    loadTemplates();
    return () => { cancelled = true; };
  }, []);

  return (
    <div className="max-w-4xl mx-auto">
      <nav className="mb-6">
        <Link
          to={workspace ? `/workspaces/${encodeURIComponent(workspace)}` : "/"}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          <ArrowLeft className="inline h-4 w-4 mr-1" />
          Back to {workspace || "Dashboard"}
        </Link>
      </nav>

      <div className="text-center mb-8">
        <h1 className="text-2xl font-bold mb-2">Create New GOAL.md</h1>
        <p className="text-muted-foreground">
          Choose a template to get started quickly, or use the guided wizard for a customized setup
        </p>
      </div>

      {/* Template Gallery */}
      <h2 className="text-lg font-semibold mb-4">Quick Start — Pick a Template</h2>
      {isLoading ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-3 mb-8">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} className="h-32 rounded-xl" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-3 mb-8">
          {templates.map((tmpl) => (
            <Link
              key={tmpl.id}
              to={`/compose/template/${encodeURIComponent(tmpl.id)}?workspace=${encodeURIComponent(workspace)}`}
              className="no-underline"
            >
              <Card className="h-full hover:border-primary transition-colors cursor-pointer py-4">
                <CardContent className="text-center px-4 py-0">
                  <div className="text-2xl mb-2">{tmpl.icon}</div>
                  <div className="font-semibold text-sm mb-1 truncate" title={tmpl.name}>
                    {tmpl.name}
                  </div>
                  <p
                    className="text-xs text-muted-foreground line-clamp-2"
                    title={tmpl.description}
                  >
                    {tmpl.description}
                  </p>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}

      <div className="flex items-center gap-4 my-8">
        <Separator className="flex-1" />
        <span className="text-sm text-muted-foreground">OR</span>
        <Separator className="flex-1" />
      </div>

      {/* Guided Wizard */}
      <Card className="border-dashed mb-6 py-6">
        <CardContent className="text-center">
          <Wand2 className="h-8 w-8 mx-auto mb-3 text-muted-foreground" />
          <h3 className="font-semibold mb-1">Guided Wizard</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Answer a few questions and we&apos;ll build the perfect agent team for your project
          </p>
          <Button asChild>
            <Link to={`/compose/step/1?workspace=${encodeURIComponent(workspace)}`}>
              Start Guided Wizard
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </CardContent>
      </Card>

      <div className="flex items-center gap-4 my-8">
        <Separator className="flex-1" />
        <span className="text-sm text-muted-foreground">OR</span>
        <Separator className="flex-1" />
      </div>

      {/* Edit Directly */}
      <Card className="border-dashed py-6">
        <CardContent className="text-center">
          <Pencil className="h-8 w-8 mx-auto mb-3 text-muted-foreground" />
          <h3 className="font-semibold mb-1">Edit GOAL.md Directly</h3>
          <p className="text-sm text-muted-foreground mb-4">
            For advanced users who want full control over their GOAL.md configuration
          </p>
          <Button asChild variant="outline">
            <Link to={workspace ? `/workspaces/${encodeURIComponent(workspace)}/goal/edit` : "#"}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit GOAL.md
            </Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
