import { useCallback, useEffect } from "react";
import { useSearchParams } from "react-router";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectOption } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { WizardLayout } from "@/components/WizardLayout";
import { useComposeWizard } from "@/hooks/useComposeWizard";
import { MissingWorkspaceNotice } from "@/components/MissingWorkspaceNotice";

export function WizardStep4() {
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
    goToFinish,
  } = useComposeWizard({ workspace, currentStep: 4 });

  if (!workspace) {
    return <MissingWorkspaceNotice />;
  }

  const handleInteractiveChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      setWizardData((prev) => ({ ...prev, interactive: e.target.value }));
    },
    [setWizardData],
  );

  const handleCompletionGateChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setWizardData((prev) => ({ ...prev, completionGate: e.target.value }));
    },
    [setWizardData],
  );

  // Update preview when settings change
  useEffect(() => {
    if (isLoading) return;
    const timer = setTimeout(() => {
      fetchPreview();
    }, 500);
    return () => clearTimeout(timer);
  }, [wizardData.interactive, wizardData.completionGate, isLoading, fetchPreview]);

  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  return (
    <WizardLayout
      currentStep={4}
      preview={preview}
      draftSavedAt={draftSavedAt}
      isSavingDraft={isSavingDraft}
      onBack={() => goToStep(3)}
      onNext={goToFinish}
      nextLabel="Review & Save"
    >
      <div>
        <h2 className="text-xl font-semibold mb-1">Step 4: Settings</h2>
        <p className="text-sm text-muted-foreground mb-6">
          Configure runtime settings for your workflow.
        </p>

        <div className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="interactive">Interactive Mode</Label>
            <Select
              id="interactive"
              value={wizardData.interactive}
              onChange={handleInteractiveChange}
            >
              <SelectOption value="yes">Yes — Ask for confirmation at each step</SelectOption>
              <SelectOption value="no">No — Run automatically without prompts</SelectOption>
              <SelectOption value="auto">Auto — Decide based on context</SelectOption>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="completiongate">Completion Gate Script</Label>
            <Input
              id="completiongate"
              value={wizardData.completionGate}
              onChange={handleCompletionGateChange}
              placeholder="e.g., make test"
            />
            <p className="text-xs text-muted-foreground">
              A command that must pass before the workflow is considered complete (optional)
            </p>
          </div>
        </div>
      </div>
    </WizardLayout>
  );
}
