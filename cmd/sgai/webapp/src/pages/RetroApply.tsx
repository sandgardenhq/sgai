import { useCallback, useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, Zap, Loader2, CheckCircle2 } from "lucide-react";
import { Link } from "react-router";

interface Suggestion {
  id: string;
  text: string;
  selected: boolean;
  note: string;
}

export function RetroApply(): JSX.Element {
  const { name: workspaceName = "" } = useParams<{ name: string }>();
  const [searchParams] = useSearchParams();
  const session = searchParams.get("session") ?? "";
  const [isLoading, setIsLoading] = useState(true);
  const [isApplying, setIsApplying] = useState(false);
  const [started, setStarted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [improvements, setImprovements] = useState("");
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const hasRequiredParams = Boolean(workspaceName && session);

  useEffect(() => {
    if (!workspaceName || !session) {
      setError("Missing workspace or session");
      setIsLoading(false);
      setIsApplying(false);
      setStarted(false);
      setImprovements("");
      setSuggestions([]);
      return;
    }
    let cancelled = false;

    async function loadRetro() {
      setIsLoading(true);
      setError(null);
      setStarted(false);
      setIsApplying(false);
      setImprovements("");
      setSuggestions([]);
      try {
        const retro = await api.workspaces.retrospectives(workspaceName, session);
        if (!cancelled && retro.details) {
          setImprovements(retro.details.improvementsRaw ?? retro.details.improvements ?? "");
          if (retro.details.isApplying) {
            setIsApplying(true);
            setStarted(true);
          }
          const lines = (retro.details.improvementsRaw ?? "").split("\n");
          const parsed: Suggestion[] = [];
          let idx = 0;
          for (const line of lines) {
            const trimmed = line.trim();
            if (trimmed.startsWith("- ") || trimmed.startsWith("* ")) {
              parsed.push({
                id: String(idx),
                text: trimmed.slice(2).trim(),
                selected: true,
                note: "",
              });
              idx++;
            }
          }
          setSuggestions(parsed);
        } else if (!cancelled) {
          setImprovements("");
          setSuggestions([]);
          setIsApplying(false);
          setStarted(false);
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

  const handleToggleSuggestion = useCallback((id: string, checked: boolean) => {
    setSuggestions((prev) =>
      prev.map((s) => (s.id === id ? { ...s, selected: checked === true } : s)),
    );
  }, []);

  const handleNoteChange = useCallback((id: string, note: string) => {
    setSuggestions((prev) =>
      prev.map((s) => (s.id === id ? { ...s, note } : s)),
    );
  }, []);

  const handleApply = useCallback(async () => {
    if (!workspaceName || !session || isApplying) return;

    const selectedIds = suggestions.filter((s) => s.selected).map((s) => s.id);
    if (selectedIds.length === 0) {
      setError("Please select at least one suggestion to apply");
      return;
    }

    setIsApplying(true);
    setError(null);

    const notes: Record<string, string> = {};
    for (const s of suggestions) {
      if (s.note.trim()) {
        notes[s.id] = s.note.trim();
      }
    }

    try {
      await api.workspaces.retroApply(
        workspaceName,
        session,
        selectedIds,
        Object.keys(notes).length > 0 ? notes : undefined,
      );
      setStarted(true);
      setIsApplying(false);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to apply recommendations");
      }
      setIsApplying(false);
    }
  }, [workspaceName, session, isApplying, suggestions]);

  if (!hasRequiredParams) {
    const backTarget = workspaceName
      ? `/workspaces/${encodeURIComponent(workspaceName)}?tab=retrospectives`
      : "/";
    const backLabel = workspaceName ? "Back to Retrospectives" : "Back to Dashboard";

    return (
      <div className="max-w-2xl mx-auto py-8 space-y-4">
        <Alert className="border-destructive/50 text-destructive">
          <AlertTitle>Missing workspace or session</AlertTitle>
          <AlertDescription>Start from the Retrospectives tab to apply recommendations.</AlertDescription>
        </Alert>
        <Link
          to={backTarget}
          className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1"
        >
          <ArrowLeft className="h-3 w-3" />
          {backLabel}
        </Link>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="max-w-2xl mx-auto py-8 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto py-8">
      <Link
        to={`/workspaces/${encodeURIComponent(workspaceName)}?tab=retrospectives`}
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to Retrospectives
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Apply Retrospective Recommendations</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Select and apply improvement suggestions from session{" "}
        <span className="font-medium text-foreground font-mono">{session}</span>.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {started ? (
        <Alert className="mb-4 border-primary/50 bg-primary/5 text-primary">
          {isApplying ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <CheckCircle2 className="h-4 w-4" />
          )}
          <AlertTitle>{isApplying ? "Applying..." : "Apply Started"}</AlertTitle>
          <AlertDescription>
            {isApplying
              ? "Applying selected recommendations. Check the Retrospectives tab for progress."
              : "Recommendations are being applied. Check the Retrospectives tab for results."}
          </AlertDescription>
        </Alert>
      ) : null}

      {suggestions.length > 0 ? (
        <div className="space-y-3 mb-6">
          {suggestions.map((suggestion) => {
            const checkboxId = `suggestion-${suggestion.id}`;
            const noteId = `suggestion-note-${suggestion.id}`;
            return (
              <div key={suggestion.id} className="border rounded-lg p-3 space-y-2">
                <div className="flex items-start gap-3">
                  <Checkbox
                    id={checkboxId}
                    checked={suggestion.selected}
                    onCheckedChange={(checked) => handleToggleSuggestion(suggestion.id, checked)}
                    disabled={isApplying || started}
                    className="mt-1"
                  />
                  <Label htmlFor={checkboxId} className="text-sm flex-1 cursor-pointer">
                    {suggestion.text}
                  </Label>
                </div>
                {suggestion.selected ? (
                  <div className="space-y-1">
                    <Label htmlFor={noteId} className="sr-only">
                      Note for {suggestion.text}
                    </Label>
                    <Textarea
                      id={noteId}
                      value={suggestion.note}
                      onChange={(e) => handleNoteChange(suggestion.id, e.target.value)}
                      placeholder="Optional note..."
                      rows={2}
                      className="text-sm resize-y"
                      disabled={isApplying || started}
                    />
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      ) : improvements ? (
        <div className="mb-6 space-y-2">
          <Label>Improvement Suggestions</Label>
          <pre className="bg-muted rounded-md p-4 text-sm font-mono overflow-auto max-h-[300px] whitespace-pre-wrap">
            {improvements}
          </pre>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground mb-6">
          No improvement suggestions found. Run analysis first.
        </p>
      )}

      <Button
        onClick={handleApply}
        disabled={isApplying || started || suggestions.filter((s) => s.selected).length === 0}
        className="w-full"
      >
        {isApplying ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Applying...
          </>
        ) : (
          <>
            <Zap className="mr-2 h-4 w-4" />
            Apply Selected Recommendations
          </>
        )}
      </Button>
    </div>
  );
}
