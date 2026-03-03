import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api, ApiError } from "@/lib/api";
import { ArrowLeft, FolderInput, Loader2 } from "lucide-react";
import { Link } from "react-router";
import { cn } from "@/lib/utils";
import type { ApiBrowseDirectoryEntry } from "@/types";

const DEBOUNCE_MS = 300;

export function AttachExternal() {
  const navigate = useNavigate();
  const [path, setPath] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [suggestions, setSuggestions] = useState<ApiBrowseDirectoryEntry[]>([]);
  const [isFetchingSuggestions, setIsFetchingSuggestions] = useState(false);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number>(-1);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);

  const fetchSuggestions = useCallback(async (currentPath: string) => {
    if (!currentPath.trim()) {
      setSuggestions([]);
      setShowSuggestions(false);
      return;
    }
    setIsFetchingSuggestions(true);
    try {
      const result = await api.browse.directories(currentPath);
      setSuggestions(result.entries ?? []);
      setShowSuggestions((result.entries ?? []).length > 0);
      setActiveIndex(-1);
    } catch {
      setSuggestions([]);
      setShowSuggestions(false);
      setActiveIndex(-1);
    } finally {
      setIsFetchingSuggestions(false);
    }
  }, []);

  useEffect(() => {
    if (debounceTimerRef.current !== null) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      void fetchSuggestions(path);
    }, DEBOUNCE_MS);

    return () => {
      if (debounceTimerRef.current !== null) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [path, fetchSuggestions]);

  const handleSelectSuggestion = useCallback((entry: ApiBrowseDirectoryEntry) => {
    setPath(entry.path);
    setSuggestions([]);
    setShowSuggestions(false);
    setActiveIndex(-1);
    inputRef.current?.focus();
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (!showSuggestions || suggestions.length === 0) return;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((prev) => (prev < suggestions.length - 1 ? prev + 1 : 0));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((prev) => (prev > 0 ? prev - 1 : suggestions.length - 1));
      } else if (e.key === "Enter" && activeIndex >= 0) {
        e.preventDefault();
        handleSelectSuggestion(suggestions[activeIndex]);
      } else if (e.key === "Escape") {
        setShowSuggestions(false);
        setActiveIndex(-1);
      }
    },
    [showSuggestions, suggestions, activeIndex, handleSelectSuggestion],
  );

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = path.trim();
      if (!trimmed || isSubmitting) return;

      setIsSubmitting(true);
      setError(null);
      setSuggestions([]);
      setShowSuggestions(false);

      try {
        const result = await api.workspaces.attach(trimmed);
        if (result.hasGoal) {
          navigate(`/workspaces/${encodeURIComponent(result.name)}/goal/edit`);
        } else {
          navigate(`/compose?workspace=${encodeURIComponent(result.name)}`);
        }
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Failed to attach workspace");
        }
      } finally {
        setIsSubmitting(false);
      }
    },
    [path, isSubmitting, navigate],
  );

  const handleBlur = useCallback((e: React.FocusEvent) => {
    if (suggestionsRef.current?.contains(e.relatedTarget as Node)) {
      return;
    }
    setShowSuggestions(false);
  }, []);

  const handleFocus = useCallback(() => {
    if (suggestions.length > 0) {
      setShowSuggestions(true);
    }
  }, [suggestions.length]);

  return (
    <div className="max-w-lg mx-auto py-8">
      <Link
        to="/"
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="h-3 w-3" />
        Back to Dashboard
      </Link>

      <h1 className="text-2xl font-semibold mb-2">Attach External Workspace</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Enter the absolute path to an existing directory to attach it as a workspace.
      </p>

      {error ? (
        <Alert className="mb-4 border-destructive/50 text-destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="workspace-path">Directory Path</Label>
          <div className="relative">
            <Input
              id="workspace-path"
              ref={inputRef}
              value={path}
              onChange={(e) => setPath(e.target.value)}
              onFocus={handleFocus}
              onBlur={handleBlur}
              onKeyDown={handleKeyDown}
              placeholder="/home/user/my-project"
              autoFocus
              disabled={isSubmitting}
              autoComplete="off"
              role="combobox"
              aria-expanded={showSuggestions}
              aria-autocomplete="list"
              aria-controls="workspace-path-suggestions"
              aria-activedescendant={activeIndex >= 0 ? `suggestion-${activeIndex}` : undefined}
            />
            {isFetchingSuggestions && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2">
                <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
              </div>
            )}
            {showSuggestions && suggestions.length > 0 && (
              <div
                id="workspace-path-suggestions"
                ref={suggestionsRef}
                className="absolute z-50 mt-1 w-full rounded-md border bg-popover shadow-md"
                role="listbox"
                aria-label="Directory suggestions"
              >
                {suggestions.map((entry, index) => (
                  <button
                    id={`suggestion-${index}`}
                    key={entry.path}
                    type="button"
                    role="option"
                    aria-selected={index === activeIndex}
                    className={cn(
                      "flex w-full items-center gap-2 px-3 py-2 text-sm cursor-pointer text-left",
                      index === activeIndex
                        ? "bg-accent text-accent-foreground"
                        : "hover:bg-accent hover:text-accent-foreground",
                    )}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => handleSelectSuggestion(entry)}
                  >
                    <span className="font-medium">{entry.name}</span>
                    <span className="text-xs text-muted-foreground truncate">{entry.path}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            Enter an absolute path or start typing to see suggestions.
          </p>
        </div>

        <Button
          type="submit"
          disabled={isSubmitting || !path.trim()}
          className="w-full"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Attaching...
            </>
          ) : (
            <>
              <FolderInput className="mr-2 h-4 w-4" />
              Attach Workspace
            </>
          )}
        </Button>
      </form>
    </div>
  );
}
