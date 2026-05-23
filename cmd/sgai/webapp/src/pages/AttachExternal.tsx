import { useCallback, useEffect, useEffectEvent, useReducer, useRef } from "react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { api, ApiError } from "@/lib/api";
import { triggerFactoryRefresh } from "@/lib/factory-state";
import { ArrowLeft, FolderInput, Loader2 } from "lucide-react";
import { Link } from "react-router";
import { cn } from "@/lib/utils";
import type { ApiBrowseDirectoryEntry } from "@/types";

const DEBOUNCE_MS = 300;

export function AttachExternal() {
  const navigate = useNavigate();
  const [{ path, isSubmitting, error, suggestions, isFetchingSuggestions, showSuggestions, activeIndex }, updateState] = useReducer(
    (
      state: {
        path: string;
        isSubmitting: boolean;
        error: string | null;
        suggestions: ApiBrowseDirectoryEntry[];
        isFetchingSuggestions: boolean;
        showSuggestions: boolean;
        activeIndex: number;
      },
      update: Partial<{
        path: string;
        isSubmitting: boolean;
        error: string | null;
        suggestions: ApiBrowseDirectoryEntry[];
        isFetchingSuggestions: boolean;
        showSuggestions: boolean;
        activeIndex: number;
      }>,
    ) => ({ ...state, ...update }),
    {
      path: "",
      isSubmitting: false,
      error: null,
      suggestions: [],
      isFetchingSuggestions: false,
      showSuggestions: false,
      activeIndex: -1,
    },
  );
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLSelectElement>(null);

  const fetchDirectorySuggestions = useEffectEvent(async (currentPath: string) => {
    if (!currentPath.trim()) {
      updateState({ suggestions: [], showSuggestions: false });
      return;
    }
    updateState({ isFetchingSuggestions: true });
    try {
      const result = await api.browse.directories(currentPath);
      const entries = result.entries ?? [];
      updateState({ suggestions: entries, showSuggestions: entries.length > 0, activeIndex: -1 });
    } catch {
      updateState({ suggestions: [], showSuggestions: false, activeIndex: -1 });
    } finally {
      updateState({ isFetchingSuggestions: false });
    }
  });

  useEffect(() => {
    if (debounceTimerRef.current !== null) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      void fetchDirectorySuggestions(path);
    }, DEBOUNCE_MS);

    return () => {
      if (debounceTimerRef.current !== null) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [path]);

  const handleSelectSuggestion = useCallback((entry: ApiBrowseDirectoryEntry) => {
    updateState({ path: entry.path, suggestions: [], showSuggestions: false, activeIndex: -1 });
    inputRef.current?.focus();
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (!showSuggestions || suggestions.length === 0) return;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        updateState({ activeIndex: activeIndex < suggestions.length - 1 ? activeIndex + 1 : 0 });
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        updateState({ activeIndex: activeIndex > 0 ? activeIndex - 1 : suggestions.length - 1 });
      } else if (e.key === "Enter" && activeIndex >= 0) {
        e.preventDefault();
        handleSelectSuggestion(suggestions[activeIndex]);
      } else if (e.key === "Escape") {
        updateState({ showSuggestions: false, activeIndex: -1 });
      }
    },
    [showSuggestions, suggestions, activeIndex, handleSelectSuggestion],
  );

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = path.trim();
      if (!trimmed || isSubmitting) return;

      updateState({ isSubmitting: true, error: null, suggestions: [], showSuggestions: false });

      try {
        const result = await api.workspaces.attach(trimmed);
        triggerFactoryRefresh();
        if (result.hasGoal) {
          navigate(`/workspaces/${encodeURIComponent(result.name)}/goal/edit`);
        } else {
          navigate(`/compose?workspace=${encodeURIComponent(result.name)}`);
        }
      } catch (err) {
        if (err instanceof ApiError) {
          updateState({ error: err.message });
        } else {
          updateState({ error: "Failed to attach workspace" });
        }
      } finally {
        updateState({ isSubmitting: false });
      }
    },
    [path, isSubmitting, navigate],
  );

  const hideSuggestionsAfterFocusLeaves = useCallback((e: React.FocusEvent) => {
    if (suggestionsRef.current?.contains(e.relatedTarget as Node)) {
      return;
    }
    updateState({ showSuggestions: false });
  }, []);

  const revealSuggestionsIfAvailable = useCallback(() => {
    if (suggestions.length > 0) {
      updateState({ showSuggestions: true });
    }
  }, [suggestions.length]);

  return (
    <div className="max-w-lg mx-auto py-8">
      <Link
        to="/"
        className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1 mb-6"
      >
        <ArrowLeft className="size-3" />
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
              onChange={(e) => updateState({ path: e.target.value })}
              onFocus={revealSuggestionsIfAvailable}
              onBlur={hideSuggestionsAfterFocusLeaves}
              onKeyDown={handleKeyDown}
              placeholder="/home/user/my-project"
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
                <Loader2 className="size-3 animate-spin text-muted-foreground" />
              </div>
            )}
            {showSuggestions && suggestions.length > 0 && (
              <select
                id="workspace-path-suggestions"
                ref={suggestionsRef}
                className="absolute z-50 mt-1 w-full rounded-md border bg-popover shadow-md text-sm"
                size={Math.min(suggestions.length, 6)}
                value={activeIndex >= 0 ? suggestions[activeIndex]?.path : ""}
                aria-label="Directory suggestions"
                onMouseDown={(e) => e.preventDefault()}
                onChange={(e) => {
                  const entry = suggestions.find((suggestion) => suggestion.path === e.target.value);
                  if (entry) handleSelectSuggestion(entry);
                }}
              >
                {suggestions.map((entry, index) => (
                  <option
                    id={`suggestion-${index}`}
                    key={entry.path}
                    value={entry.path}
                    className={cn("px-3 py-2", index === activeIndex && "bg-accent text-accent-foreground")}
                  >
                    {entry.name} - {entry.path}
                  </option>
                ))}
              </select>
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
              <Loader2 className="mr-2 size-4 animate-spin" />
              Attaching&hellip;
            </>
          ) : (
            <>
              <FolderInput className="mr-2 size-4" />
              Attach Workspace
            </>
          )}
        </Button>
      </form>
    </div>
  );
}
