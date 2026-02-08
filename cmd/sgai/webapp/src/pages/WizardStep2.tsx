import { useCallback, useEffect } from "react";
import { useSearchParams } from "react-router";
import { Skeleton } from "@/components/ui/skeleton";
import { WizardLayout } from "@/components/WizardLayout";
import { useComposeWizard } from "@/hooks/useComposeWizard";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MissingWorkspaceNotice } from "@/components/MissingWorkspaceNotice";

export function WizardStep2() {
  const [searchParams] = useSearchParams();
  const workspace = searchParams.get("workspace") ?? "";

  const {
    wizardData,
    setWizardData,
    techStackItems,
    preview,
    isLoading,
    draftSavedAt,
    isSavingDraft,
    fetchPreview,
    goToStep,
    goBack,
  } = useComposeWizard({ workspace, currentStep: 2 });

  if (!workspace) {
    return <MissingWorkspaceNotice />;
  }

  const handleToggleTech = useCallback(
    (techId: string) => {
      setWizardData((prev) => {
        const isSelected = prev.techStack.includes(techId);
        return {
          ...prev,
          techStack: isSelected
            ? prev.techStack.filter((id) => id !== techId)
            : [...prev.techStack, techId],
        };
      });
    },
    [setWizardData],
  );

  // Update preview when tech stack changes
  useEffect(() => {
    if (isLoading) return;
    const timer = setTimeout(() => {
      fetchPreview();
    }, 300);
    return () => clearTimeout(timer);
  }, [wizardData.techStack, isLoading, fetchPreview]);

  if (isLoading) {
    return (
      <div className="space-y-4 p-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <div className="grid grid-cols-3 gap-3">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <WizardLayout
      currentStep={2}
      preview={preview}
      draftSavedAt={draftSavedAt}
      isSavingDraft={isSavingDraft}
      onBack={() => goToStep(1)}
      onNext={() => goToStep(3)}
    >
      <div>
        <h2 className="text-xl font-semibold mb-1">Step 2: Tech Stack</h2>
        <p className="text-sm text-muted-foreground mb-6">
          Select the technologies you&apos;ll be using. Each choice automatically adds the right developer and reviewer agents.
        </p>

        <div className="grid grid-cols-3 gap-3">
          {techStackItems.map((item) => {
            const isSelected = wizardData.techStack.includes(item.id);
            return (
              <Tooltip key={item.id}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => handleToggleTech(item.id)}
                    className={cn(
                      "flex items-center justify-center p-3 rounded-lg border-2 transition-colors cursor-pointer text-center min-h-[3.5rem]",
                      isSelected
                        ? "border-primary bg-primary/10"
                        : "border-border hover:border-primary/50",
                    )}
                    aria-pressed={isSelected}
                  >
                    <span className="font-semibold text-sm truncate">
                      {item.name}
                    </span>
                  </button>
                </TooltipTrigger>
                <TooltipContent side="bottom">
                  <p>{item.name}</p>
                </TooltipContent>
              </Tooltip>
            );
          })}
        </div>
      </div>
    </WizardLayout>
  );
}
