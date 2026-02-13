import { useCallback, useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ComposePreview } from "@/components/ComposePreview";
import { ProgressDots } from "@/components/WizardLayout";
import { useComposeWizard } from "@/hooks/useComposeWizard";
import { MissingWorkspaceNotice } from "@/components/MissingWorkspaceNotice";
import { Link } from "react-router";
import {
  ArrowLeft,
  Save,
  FileText,
  Users,
  ShieldCheck,
  Terminal,
  CheckCircle2,
  AlertTriangle,
  Loader2,
} from "lucide-react";

export function WizardFinish() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const workspace = searchParams.get("workspace") ?? "";
  const [saveSuccess, setSaveSuccess] = useState(false);

  const {
    wizardData,
    techStackItems,
    preview,
    isLoading,
    isSaving,
    saveError,
    draftSavedAt,
    isSavingDraft,
    fetchPreview,
    saveGoal,
    goToStep,
  } = useComposeWizard({ workspace, currentStep: 4 });

  if (!workspace) {
    return <MissingWorkspaceNotice />;
  }

  // Fetch preview on mount
  useEffect(() => {
    if (!isLoading) {
      fetchPreview();
    }
  }, [isLoading, fetchPreview]);

  const handleSave = useCallback(async () => {
    const success = await saveGoal();
    if (success) {
      setSaveSuccess(true);
      setTimeout(() => {
        navigate(workspace ? `/workspaces/${encodeURIComponent(workspace)}` : "/");
      }, 1500);
    }
  }, [saveGoal, navigate, workspace]);

  const selectedTechNames = techStackItems
    .filter((item) => wizardData.techStack.includes(item.id))
    .map((item) => item.name);

  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      {/* Header */}
      <div className="flex items-center justify-between border-b pb-3 mb-4">
        <Link
          to={`/compose/step/4?workspace=${encodeURIComponent(workspace)}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Settings
        </Link>
        <ProgressDots currentStep={5} />
      </div>

      {/* Success notification */}
      {saveSuccess ? (
        <Alert className="mb-4 border-primary/50 bg-primary/5 text-primary">
          <CheckCircle2 className="h-4 w-4" />
          <AlertTitle>GOAL.md Saved Successfully!</AlertTitle>
          <AlertDescription>
            Redirecting to workspace...
          </AlertDescription>
        </Alert>
      ) : null}

      {/* Error notification */}
      {saveError ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Save Failed</AlertTitle>
          <AlertDescription>{saveError}</AlertDescription>
        </Alert>
      ) : null}

      {/* Main layout */}
      <div className="grid grid-cols-2 gap-6 flex-1 overflow-hidden">
        {/* Left: summary */}
        <div className="overflow-y-auto pr-2 space-y-3">
          <h2 className="text-xl font-semibold mb-4">Review & Save</h2>

          {/* Description */}
          <Card className="py-3">
            <CardContent className="flex items-start gap-3 px-4 py-0">
              <FileText className="h-5 w-5 text-muted-foreground mt-0.5 flex-shrink-0" />
              <div className="min-w-0 flex-1">
                <div className="text-xs text-muted-foreground mb-1">Project Description</div>
                {wizardData.description ? (
                  <div className="font-semibold text-sm truncate" title={wizardData.description}>
                    {wizardData.description}
                  </div>
                ) : (
                  <div className="text-sm italic text-muted-foreground">Not set</div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Tech Stack / Agents */}
          <Card className="py-3">
            <CardContent className="flex items-start gap-3 px-4 py-0">
              <Users className="h-5 w-5 text-muted-foreground mt-0.5 flex-shrink-0" />
              <div className="min-w-0 flex-1">
                <div className="text-xs text-muted-foreground mb-1">Selected Technologies</div>
                {selectedTechNames.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {selectedTechNames.map((name) => (
                      <Badge key={name} variant="secondary" className="text-xs" title={name}>
                        {name}
                      </Badge>
                    ))}
                  </div>
                ) : (
                  <div className="text-sm italic text-muted-foreground">None selected</div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Safety Analysis */}
          <Card className="py-3">
            <CardContent className="flex items-start gap-3 px-4 py-0">
              <ShieldCheck className="h-5 w-5 text-muted-foreground mt-0.5 flex-shrink-0" />
              <div className="min-w-0 flex-1">
                <div className="text-xs text-muted-foreground mb-1">Safety Analysis</div>
                <div className="font-semibold text-sm">
                  {wizardData.safetyAnalysis ? "Enabled" : "Disabled"}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Completion Gate */}
          {wizardData.completionGate ? (
            <Card className="py-3">
              <CardContent className="flex items-start gap-3 px-4 py-0">
                <Terminal className="h-5 w-5 text-muted-foreground mt-0.5 flex-shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="text-xs text-muted-foreground mb-1">Completion Gate</div>
                  <code className="font-semibold text-sm" title={wizardData.completionGate}>
                    {wizardData.completionGate}
                  </code>
                </div>
              </CardContent>
            </Card>
          ) : null}

          {/* Draft saved indicator */}
          {draftSavedAt ? (
            <p className="text-xs text-muted-foreground text-center">
              {isSavingDraft ? "Saving draft..." : `Draft saved at ${draftSavedAt}`}
            </p>
          ) : null}

          {/* Actions */}
          <div className="flex justify-between pt-4 border-t">
            <Button
              variant="outline"
              onClick={() => goToStep(1)}
              className="min-w-[120px]"
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Edit Wizard
            </Button>
            <Button
              onClick={handleSave}
              disabled={isSaving || saveSuccess}
              className="min-w-[140px]"
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

        {/* Right: preview */}
        <ComposePreview
          preview={preview}
          isLoading={isLoading}
          title="Final GOAL.md Preview"
        />
      </div>
    </div>
  );
}
