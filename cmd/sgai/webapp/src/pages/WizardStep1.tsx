import { useCallback, useEffect } from "react";
import { useSearchParams } from "react-router";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { WizardLayout } from "@/components/WizardLayout";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { useComposeWizard } from "@/hooks/useComposeWizard";
import { MissingWorkspaceNotice } from "@/components/MissingWorkspaceNotice";

export function WizardStep1() {
  const [searchParams] = useSearchParams();
  const workspace = searchParams.get("workspace") ?? "";

  const {
    wizardData,
    setWizardData,
    preview,
    isLoading,
    draftSavedAt,
    isSavingDraft,
    fetchPreview,
    goToStep,
    goBack,
  } = useComposeWizard({ workspace, currentStep: 1 });

  if (!workspace) {
    return <MissingWorkspaceNotice />;
  }

  const handleDescriptionChange = useCallback(
    (value: string | undefined) => {
      setWizardData((prev) => ({ ...prev, description: value ?? "" }));
    },
    [setWizardData],
  );

  // Debounced preview update on description change
  useEffect(() => {
    if (isLoading) return;
    const timer = setTimeout(() => {
      fetchPreview();
    }, 500);
    return () => clearTimeout(timer);
  }, [wizardData.description, isLoading, fetchPreview]);

  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  return (
    <WizardLayout
      currentStep={1}
      preview={preview}
      draftSavedAt={draftSavedAt}
      isSavingDraft={isSavingDraft}
      onBack={goBack}
      onNext={() => goToStep(2)}
    >
      <div>
        <h2 className="text-xl font-semibold mb-1">Step 1: Project Description</h2>
        <p className="text-sm text-muted-foreground mb-6">
          Describe what you want to build. This will appear at the top of your GOAL.md file.
        </p>

        <div className="space-y-2">
          <Label htmlFor="description">Project Description</Label>
          <MarkdownEditor
            value={wizardData.description}
            onChange={handleDescriptionChange}
            defaultHeight={300}
            placeholder="Example: Build a web application that manages user profiles with authentication, dashboard, and REST API endpoints."
          />
        </div>
      </div>
    </WizardLayout>
  );
}
