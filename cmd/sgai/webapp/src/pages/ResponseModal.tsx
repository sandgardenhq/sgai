import { useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { MarkdownContent } from "@/components/MarkdownContent";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { ResponseContext } from "@/components/ResponseContext";
import { QuestionBlock } from "@/components/QuestionBlock";
import { useResponseForm } from "@/hooks/useResponseForm";

const STORAGE_PREFIX = "sgai-response-modal-";

interface ResponseModalProps {
  workspaceName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onResponseSubmitted?: () => void;
}

export function ResponseModal({
  workspaceName,
  open,
  onOpenChange,
  onResponseSubmitted,
}: ResponseModalProps) {
  const handleQuestionMissing = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const handleSubmitSuccess = useCallback(() => {
    onOpenChange(false);
    onResponseSubmitted?.();
  }, [onOpenChange, onResponseSubmitted]);

  const {
    question,
    workspaceDetail,
    loading,
    error,
    submitting,
    submitError,
    selections,
    otherText,
    setOtherText,
    handleChoiceToggle,
    handleSubmit,
  } = useResponseForm({
    workspaceName,
    storagePrefix: STORAGE_PREFIX,
    active: open,
    onQuestionMissing: handleQuestionMissing,
    onSubmitSuccess: handleSubmitSuccess,
  });

  const handleCancel = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Response Required</DialogTitle>
          <DialogDescription>
            {question ? (
              <Badge variant="secondary" className="mt-1">
                Agent: {question.agentName}
              </Badge>
            ) : (
              <span>Loading agent question...</span>
            )}
          </DialogDescription>
        </DialogHeader>

        {loading && (
          <div className="space-y-3 py-4">
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-32 w-full rounded" />
            <Skeleton className="h-16 w-full rounded" />
          </div>
        )}

        {error && (
          <p className="text-sm text-destructive py-4">
            Failed to load question: {error.message}
          </p>
        )}

        {!loading && !error && question && (
          <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
            <ScrollArea className="flex-1 pr-2">
              {question.questions && question.questions.length > 0 ? (
                <div className="space-y-4">
                  {question.questions.map((q, qIndex) => (
                    <QuestionBlock
                      key={qIndex}
                      question={q.question}
                      choices={q.choices}
                      multiSelect={q.multiSelect}
                      questionIndex={qIndex}
                      totalQuestions={question.questions?.length ?? 0}
                      selectedChoices={selections[String(qIndex)] ?? []}
                      onChoiceToggle={handleChoiceToggle}
                      compact
                      idPrefix="modal-"
                    />
                  ))}
                </div>
              ) : (
                question.message && (
                  <MarkdownContent
                    content={question.message}
                    className="mb-4"
                  />
                )
              )}

              <div className="mt-4 pt-3 border-t">
                <Label htmlFor="modal-other" className="font-semibold text-sm">
                  Other (additional comments or alternative answer):
                </Label>
                <Textarea
                  id="modal-other"
                  value={otherText}
                  onChange={(e) => setOtherText(e.target.value)}
                  placeholder="Type your custom response here..."
                  rows={3}
                  className="mt-2"
                />
              </div>

              <ResponseContext
                goalContent={workspaceDetail?.goalContent}
                projectMgmtContent={workspaceDetail?.pmContent}
              />
            </ScrollArea>

            {submitError && (
              <p className="text-sm text-destructive mt-2">{submitError}</p>
            )}

            <DialogFooter className="mt-4 pt-3 border-t">
              <Button
                type="button"
                variant="secondary"
                onClick={handleCancel}
                disabled={submitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? "Sending..." : "Send Response"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
