import { useCallback } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { MarkdownContent } from "@/components/MarkdownContent";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ResponseContext } from "@/components/ResponseContext";
import { QuestionBlock } from "@/components/QuestionBlock";
import { useResponseForm } from "@/hooks/useResponseForm";

const STORAGE_PREFIX = "sgai-response-";

function ResponseSkeleton() {
  return (
    <div
      className="max-w-2xl mx-auto space-y-4"
      role="status"
      aria-live="polite"
      aria-labelledby="response-loading-label"
    >
      <span id="response-loading-label" className="sr-only">
        Loading response
      </span>
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-6 w-32" />
      <Skeleton className="h-48 w-full rounded-xl" />
      <Skeleton className="h-24 w-full rounded-xl" />
      <Skeleton className="h-10 w-32" />
    </div>
  );
}

export function ResponseMultiChoice() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const workspaceName = name ?? "";

  const handleQuestionMissing = useCallback(() => {
    if (!workspaceName) {
      navigate("/", { replace: true });
      return;
    }
    navigate(`/workspaces/${encodeURIComponent(workspaceName)}/progress`, { replace: true });
  }, [navigate, workspaceName]);

  const handleSubmitSuccess = useCallback(() => {
    if (!workspaceName) {
      navigate("/", { replace: true });
      return;
    }
    navigate(`/workspaces/${encodeURIComponent(workspaceName)}/progress`, { replace: true });
  }, [navigate, workspaceName]);

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
    active: true,
    onQuestionMissing: handleQuestionMissing,
    onSubmitSuccess: handleSubmitSuccess,
  });

  if (loading) return <ResponseSkeleton />;

  if (!workspaceName) {
    return (
      <div className="max-w-2xl mx-auto">
        <Alert className="mb-4">
          <AlertTitle>Workspace required</AlertTitle>
          <AlertDescription>
            Select a workspace to respond to this agent question.
          </AlertDescription>
        </Alert>
        <Link
          to="/"
          className="text-sm text-primary hover:underline inline-block"
        >
          ← Back to dashboard
        </Link>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-2xl mx-auto">
        <p className="text-sm text-destructive">
          Failed to load question: {error.message}
        </p>
        <Link
          to={`/workspaces/${encodeURIComponent(workspaceName)}/progress`}
          className="text-sm text-primary hover:underline mt-2 inline-block"
        >
          ← Back to workspace
        </Link>
      </div>
    );
  }

  if (!question) return null;

  return (
    <div className="max-w-2xl mx-auto">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <Link
              to={`/workspaces/${encodeURIComponent(workspaceName)}/progress`}
              className="text-sm text-muted-foreground hover:text-foreground no-underline"
            >
              ← Back
            </Link>
            <CardTitle className="text-lg">Response Required</CardTitle>
          </div>
          <Badge variant="secondary" className="w-fit">
            Agent: {question.agentName}
          </Badge>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit}>
            {question.questions && question.questions.length > 0 ? (
              <div className="space-y-6">
              {question.questions.map((q, qIndex) => {
                const questionKey = `${question.questionId}-${q.question}-${q.choices.join("|")}-${
                  q.multiSelect ? "multi" : "single"
                }`;
                return (
                  <QuestionBlock
                    key={questionKey}
                    question={q.question}
                    choices={q.choices}
                    multiSelect={q.multiSelect}
                    questionIndex={qIndex}
                    totalQuestions={question.questions?.length ?? 0}
                    selectedChoices={selections[String(qIndex)] ?? []}
                    onChoiceToggle={handleChoiceToggle}
                  />
                );
              })}
              </div>
            ) : (
              question.message && (
                <MarkdownContent
                  content={question.message}
                  className="mb-4"
                />
              )
            )}

            <div className="mt-6 pt-4 border-t">
              <Label htmlFor="other" className="font-semibold">
                Other (additional comments or alternative answer):
              </Label>
              <Textarea
                id="other"
                value={otherText}
                onChange={(e) => setOtherText(e.target.value)}
                placeholder="Type your custom response here..."
                rows={3}
                className="mt-2"
              />
            </div>

            {submitError && (
              <p className="text-sm text-destructive mt-3">{submitError}</p>
            )}

            <Button
              type="submit"
              disabled={submitting}
              className="mt-4"
            >
              {submitting ? "Sending..." : "Send Response"}
            </Button>
          </form>

          <ResponseContext
            goalContent={workspaceDetail?.goalContent}
            projectMgmtContent={workspaceDetail?.pmContent}
          />
        </CardContent>
      </Card>
    </div>
  );
}
