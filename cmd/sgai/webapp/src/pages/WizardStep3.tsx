import { useCallback, useEffect } from "react";
import { useSearchParams } from "react-router";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { WizardLayout } from "@/components/WizardLayout";
import { useComposeWizard } from "@/hooks/useComposeWizard";
import { ShieldCheck } from "lucide-react";
import { MissingWorkspaceNotice } from "@/components/MissingWorkspaceNotice";

export function WizardStep3() {
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
  } = useComposeWizard({ workspace, currentStep: 3 });

  if (!workspace) {
    return <MissingWorkspaceNotice />;
  }

  const handleToggleSafety = useCallback(
    (checked: boolean) => {
      setWizardData((prev) => ({ ...prev, safetyAnalysis: checked }));
    },
    [setWizardData],
  );

  // Update preview when safety analysis changes
  useEffect(() => {
    if (isLoading) return;
    const timer = setTimeout(() => {
      fetchPreview();
    }, 300);
    return () => clearTimeout(timer);
  }, [wizardData.safetyAnalysis, isLoading, fetchPreview]);

  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  return (
    <WizardLayout
      currentStep={3}
      preview={preview}
      draftSavedAt={draftSavedAt}
      isSavingDraft={isSavingDraft}
      onBack={() => goToStep(2)}
      onNext={() => goToStep(4)}
    >
      <div>
        <h2 className="text-xl font-semibold mb-1">Step 3: Safety Analysis</h2>
        <p className="text-sm text-muted-foreground mb-6">
          Enable STPA (System Theoretic Process Analysis) for hazard analysis and safety-critical systems.
        </p>

        <div className="flex items-center justify-between p-4 border rounded-lg mb-4">
          <div className="flex-1 mr-4">
            <Label htmlFor="safety-toggle" className="font-semibold cursor-pointer">
              Enable Safety Analysis
            </Label>
            <p className="text-sm text-muted-foreground mt-0.5">
              STPA analyst will identify unsafe control actions and loss scenarios
            </p>
          </div>
          <Switch
            id="safety-toggle"
            checked={wizardData.safetyAnalysis}
            onCheckedChange={handleToggleSafety}
          />
        </div>

        {wizardData.safetyAnalysis ? (
          <Card className="bg-primary/5 border-primary/20 py-4">
            <CardContent className="px-4 py-0">
              <div className="flex items-start gap-3">
                <ShieldCheck className="h-5 w-5 text-primary mt-0.5 flex-shrink-0" />
                <div>
                  <p className="font-semibold text-sm mb-2">What STPA provides:</p>
                  <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
                    <li>Loss identification</li>
                    <li>Hazard analysis</li>
                    <li>Unsafe control action detection</li>
                    <li>Loss scenario modeling</li>
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>
        ) : null}
      </div>
    </WizardLayout>
  );
}
