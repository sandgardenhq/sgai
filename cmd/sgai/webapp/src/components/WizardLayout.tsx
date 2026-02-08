import { Link, useSearchParams } from "react-router";
import { Button } from "@/components/ui/button";
import { ComposePreview } from "@/components/ComposePreview";
import { cn } from "@/lib/utils";
import { ArrowLeft, ArrowRight, Save } from "lucide-react";
import type { ApiComposePreviewResponse } from "@/types";

const TOTAL_STEPS = 4;

interface WizardLayoutProps {
  currentStep: number;
  children: React.ReactNode;
  preview: ApiComposePreviewResponse | null;
  isPreviewLoading?: boolean;
  draftSavedAt: string | null;
  isSavingDraft: boolean;
  onBack: () => void;
  onNext: () => void;
  nextLabel?: string;
  backLabel?: string;
  isFinish?: boolean;
}

export function ProgressDots({ currentStep }: { currentStep: number }) {
  return (
    <div className="flex items-center gap-1.5" role="progressbar" aria-valuenow={currentStep} aria-valuemin={1} aria-valuemax={TOTAL_STEPS}>
      {Array.from({ length: TOTAL_STEPS }, (_, i) => {
        const step = i + 1;
        const isCompleted = step < currentStep;
        const isActive = step === currentStep;

        return (
          <div
            key={step}
            className={cn(
              "h-2.5 w-2.5 rounded-full transition-colors",
              isActive && "bg-primary",
              isCompleted && "bg-primary/60",
              !isActive && !isCompleted && "bg-muted-foreground/20",
            )}
            aria-label={`Step ${step}${isActive ? " (current)" : isCompleted ? " (completed)" : ""}`}
          />
        );
      })}
    </div>
  );
}

export function WizardLayout({
  currentStep,
  children,
  preview,
  isPreviewLoading = false,
  draftSavedAt,
  isSavingDraft,
  onBack,
  onNext,
  nextLabel,
  backLabel,
  isFinish = false,
}: WizardLayoutProps) {
  const [searchParams] = useSearchParams();
  const workspace = searchParams.get("workspace") ?? "";

  const resolvedBackLabel = backLabel ?? (currentStep === 1 ? "Templates" : "Back");
  const resolvedNextLabel = nextLabel ?? (currentStep === TOTAL_STEPS ? "Review & Save" : "Next");

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      {/* Header with nav + progress */}
      <div className="flex items-center justify-between border-b pb-3 mb-4">
        <Link
          to={`/compose?workspace=${encodeURIComponent(workspace)}`}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          ← Back to Templates
        </Link>

        <div className="flex items-center gap-3">
          {draftSavedAt ? (
            <span className="text-xs text-muted-foreground">
              {isSavingDraft ? "Saving draft..." : `Draft saved at ${draftSavedAt}`}
            </span>
          ) : isSavingDraft ? (
            <span className="text-xs text-muted-foreground">Saving draft...</span>
          ) : null}
          <ProgressDots currentStep={currentStep} />
        </div>
      </div>

      {/* Main layout: form + preview */}
      <div className="grid grid-cols-[3fr_2fr] gap-6 flex-1 overflow-hidden">
        {/* Left: wizard step content */}
        <div className="flex flex-col overflow-y-auto pr-2">
          <div className="flex-1">
            {children}
          </div>

          {/* Navigation */}
          <div className="flex justify-between pt-4 border-t mt-auto">
            <Button variant="outline" onClick={onBack} className="min-w-[120px]">
              <ArrowLeft className="mr-2 h-4 w-4" />
              {resolvedBackLabel}
            </Button>

            {isFinish ? (
              <Button onClick={onNext} className="min-w-[120px]">
                <Save className="mr-2 h-4 w-4" />
                {resolvedNextLabel}
              </Button>
            ) : (
              <Button onClick={onNext} className="min-w-[120px]">
                {resolvedNextLabel}
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            )}
          </div>
        </div>

        {/* Right: preview panel */}
        <ComposePreview
          preview={preview}
          isLoading={isPreviewLoading}
        />
      </div>
    </div>
  );
}
