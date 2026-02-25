import { useState, useEffect, useCallback, useRef } from "react";
import { api, ApiError } from "@/lib/api";
import { useFactoryState } from "@/lib/factory-state";
import type { ApiPendingQuestionResponse, ApiWorkspaceEntry } from "@/types";

export interface StoredResponseState {
  selections: Record<string, string[]>;
  otherText: string;
  questionId: string;
}

function getStorageKey(prefix: string, workspaceName: string): string {
  return `${prefix}${workspaceName}`;
}

export function loadStoredState(prefix: string, workspaceName: string): StoredResponseState | null {
  try {
    const stored = sessionStorage.getItem(getStorageKey(prefix, workspaceName));
    if (stored) {
      return JSON.parse(stored) as StoredResponseState;
    }
  } catch {
    // Ignore parse errors
  }
  return null;
}

export function saveStoredState(prefix: string, workspaceName: string, state: StoredResponseState): void {
  try {
    sessionStorage.setItem(getStorageKey(prefix, workspaceName), JSON.stringify(state));
  } catch {
    // Ignore storage errors
  }
}

export function clearStoredState(prefix: string, workspaceName: string): void {
  try {
    sessionStorage.removeItem(getStorageKey(prefix, workspaceName));
  } catch {
    // Ignore
  }
}

interface UseResponseFormOptions {
  workspaceName: string;
  storagePrefix: string;
  active: boolean;
  onQuestionMissing?: () => void;
  onSubmitSuccess?: () => void;
}

interface UseResponseFormReturn {
  question: ApiPendingQuestionResponse | null;
  workspaceDetail: ApiWorkspaceEntry | null;
  loading: boolean;
  error: Error | null;
  submitting: boolean;
  submitError: string | null;
  selections: Record<string, string[]>;
  otherText: string;
  setOtherText: (text: string) => void;
  handleChoiceToggle: (questionIndex: number, choice: string, multiSelect: boolean) => void;
  handleSubmit: (e: React.FormEvent) => void;
}

export function useResponseForm({
  workspaceName,
  storagePrefix,
  active,
  onQuestionMissing,
  onSubmitSuccess,
}: UseResponseFormOptions): UseResponseFormReturn {
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [selections, setSelections] = useState<Record<string, string[]>>({});
  const [otherText, setOtherText] = useState("");
  const hasUnsavedChangesRef = useRef(false);
  const previousQuestionIdRef = useRef<string | null>(null);

  const { workspaces, fetchStatus } = useFactoryState();
  const workspace = workspaces.find((ws) => ws.name === workspaceName) ?? null;
  const question = workspace?.pendingQuestion ?? null;
  const loading = fetchStatus === "fetching" && workspace === null;
  const error: Error | null = fetchStatus === "error" && workspace === null
    ? new Error("Failed to load workspace state")
    : null;

  useEffect(() => {
    if (!active || !workspaceName) return;

    if (question === null && workspace !== null) {
      onQuestionMissing?.();
      return;
    }

    if (!question) return;

    if (previousQuestionIdRef.current !== question.questionId) {
      previousQuestionIdRef.current = question.questionId;
      const stored = loadStoredState(storagePrefix, workspaceName);
      if (stored && stored.questionId === question.questionId) {
        setSelections(stored.selections);
        setOtherText(stored.otherText);
      } else {
        setSelections({});
        setOtherText("");
      }
    }
  }, [active, workspaceName, storagePrefix, question, workspace, onQuestionMissing]);

  useEffect(() => {
    if (!question) return;

    const hasSelections = Object.values(selections).some((s) => s.length > 0);
    const hasText = otherText.trim().length > 0;
    hasUnsavedChangesRef.current = hasSelections || hasText;

    saveStoredState(storagePrefix, workspaceName, {
      selections,
      otherText,
      questionId: question.questionId,
    });
  }, [selections, otherText, question, workspaceName, storagePrefix]);

  useEffect(() => {
    function handleBeforeUnload(e: BeforeUnloadEvent) {
      if (hasUnsavedChangesRef.current && active) {
        e.preventDefault();
      }
    }

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [active]);

  const handleChoiceToggle = useCallback(
    (questionIndex: number, choice: string, multiSelect: boolean) => {
      setSelections((prev) => {
        const key = String(questionIndex);
        const current = prev[key] ?? [];

        if (multiSelect) {
          const updated = current.includes(choice)
            ? current.filter((c) => c !== choice)
            : [...current, choice];
          return { ...prev, [key]: updated };
        }

        return { ...prev, [key]: [choice] };
      });
    },
    [],
  );

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();

      if (!question || submitting) return;

      setSubmitting(true);
      setSubmitError(null);

      const allSelectedChoices: string[] = [];
      for (const key of Object.keys(selections)) {
        allSelectedChoices.push(...selections[key]);
      }

      try {
        await api.workspaces.respond(workspaceName, {
          questionId: question.questionId,
          answer: otherText.trim(),
          selectedChoices: allSelectedChoices,
        });

        clearStoredState(storagePrefix, workspaceName);
        hasUnsavedChangesRef.current = false;
        onSubmitSuccess?.();
      } catch (err: unknown) {
        if (err instanceof ApiError && err.status === 409) {
          setSubmitError("This question has expired. The agent may have moved on.");
        } else {
          setSubmitError(
            err instanceof Error ? err.message : "Failed to submit response",
          );
        }
      } finally {
        setSubmitting(false);
      }
    },
    [question, submitting, selections, otherText, workspaceName, storagePrefix, onSubmitSuccess],
  );

  return {
    question,
    workspaceDetail: workspace,
    loading,
    error,
    submitting,
    submitError,
    selections,
    otherText,
    setOtherText,
    handleChoiceToggle,
    handleSubmit,
  };
}
