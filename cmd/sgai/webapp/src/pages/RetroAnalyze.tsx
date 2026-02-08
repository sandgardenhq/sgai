import { useCallback, useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router";
import { ArrowLeft, BarChart3, CheckCircle2, Loader2 } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { api, ApiError } from "@/lib/api";

export function RetroAnalyze(): JSX.Element {
  const { name: workspaceName = "", session: sessionParam = "" } = useParams<{ name: string; session?: string }>();
  const [searchParams] = useSearchParams();
  const session = sessionParam || searchParams.get("session") || "";
  const [isLoading, setIsLoading] = useState(true);
  const [isRunning, setIsRunning] = useState(false);
  const [started, setStarted] = useState(false);
  const [autoStartAttempted, setAutoStartAttempted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [goalSummary, setGoalSummary] = useState("");

  useEffect(() => {
    if (!workspaceName || !session) return;
    let cancelled = false;

    async function loadRetro() {
      setIsLoading(true);
      try {
        const retro = await api.workspaces.retrospectives(workspaceName, session);
        if (!cancelled && retro.details) {
          setGoalSummary(retro.details.goalSummary);
          const hasResults = Boolean(retro.details.hasImprovements) || Boolean(retro.details.improvements?.trim());
          if (retro.details.isAnalyzing) {
            setIsRunning(true);
            setStarted(true);
          } else if (hasResults) {
            setStarted(true);
          }
        }
      } catch {
        if (!cancelled) {
          setError("Failed to load retrospective data");
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    loadRetro();
    return () => { cancelled = true; };
  }, [workspaceName, session]);

  const handleAnalyze = useCallback(async () => {
    if (!workspaceName || !session || isRunning) return;

    setIsRunning(true);
    setError(null);

    try {
      await api.workspaces.retroAnalyze(workspaceName, session);
      setStarted(true);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to start analysis");
      }
      setIsRunning(false);
    }
  }, [workspaceName, session, isRunning]);

  useEffect(() => {
    if (!isLoading && !started && !isRunning && !autoStartAttempted && !error && workspaceName && session) {
      setAutoStartAttempted(true);
      handleAnalyze();
    }
  }, [isLoading, started, isRunning, autoStartAttempted, error, workspaceName, session, handleAnalyze]);

  if (isLoading) {
    return (
      <div className="max-w-lg mx-auto py-8 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  return (
    <div className="max-w-lg mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}/retro`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to Retrospectives
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Run Retrospective Analysis</h1>
      <p className="text-sm text-muted-foreground mb-2">
        Analyze session <span className="font-medium text-foreground font-mono">{session}</span>
      </p>
      {goalSummary ? (
        <p className="text-sm text-muted-foreground mb-6 truncate" title={goalSummary}>
          {goalSummary}
        </p>
      ) : null}

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {started && isRunning ? (
        <Alert className="mb-4 border-primary/50 bg-primary/5 text-primary">
          <Loader2 className="h-4 w-4 animate-spin" />
          <AlertTitle>Analysis Running</AlertTitle>
          <AlertDescription>
            The retrospective analysis is running in the background. Check the Retrospectives tab for progress.
          </AlertDescription>
        </Alert>
      ) : null}

      {started && !isRunning ? (
        <Alert className="mb-4 border-primary/50 bg-primary/5 text-primary">
          <CheckCircle2 className="h-4 w-4" />
          <AlertTitle>Analysis Started</AlertTitle>
          <AlertDescription>
            The analysis has been started. Check the Retrospectives tab for results.
          </AlertDescription>
        </Alert>
      ) : null}

      <Button
        onClick={handleAnalyze}
        disabled={isRunning || started}
        className="w-full"
      >
        {isRunning ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Analyzing...
          </>
        ) : (
          <>
            <BarChart3 className="mr-2 h-4 w-4" />
            Start Analysis
          </>
        )}
      </Button>
    </div>
  );
}
