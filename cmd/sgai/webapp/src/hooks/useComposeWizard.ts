import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { api, ApiError } from "@/lib/api";
import type {
  ApiComposePreviewResponse,
  ApiWizardState,
  ApiComposerState,
  ApiTechStackItem,
  ApiComposerAgentConf,
} from "@/types";

const STORAGE_KEY_PREFIX = "compose-wizard-step-";
const AUTO_SAVE_INTERVAL_MS = 30_000;

export interface WizardStepData {
  description: string;
  techStack: string[];
  safetyAnalysis: boolean;
  interactive: string;
  completionGate: string;
}

function getStepStorageKey(step: number): string {
  return `${STORAGE_KEY_PREFIX}${step}`;
}

export function loadStepFromStorage(step: number): Partial<WizardStepData> | null {
  try {
    const stored = sessionStorage.getItem(getStepStorageKey(step));
    if (stored) {
      return JSON.parse(stored) as Partial<WizardStepData>;
    }
  } catch {
    // Ignore parse errors
  }
  return null;
}

export function saveStepToStorage(step: number, data: Partial<WizardStepData>): void {
  try {
    sessionStorage.setItem(getStepStorageKey(step), JSON.stringify(data));
  } catch {
    // Ignore storage errors
  }
}

export function clearAllWizardStorage(): void {
  for (let i = 1; i <= 4; i++) {
    try {
      sessionStorage.removeItem(getStepStorageKey(i));
    } catch {
      // Ignore
    }
  }
}

function buildWizardStateFromData(data: WizardStepData, step: number): ApiWizardState {
  return {
    currentStep: step,
    description: data.description,
    techStack: data.techStack,
    safetyAnalysis: data.safetyAnalysis,
    interactive: data.interactive,
    completionGate: data.completionGate,
  };
}

const DEFAULT_MODEL = "anthropic/claude-opus-4-6";

interface TechStackMapping {
  agents: string[];
  flow: string[];
}

const TECH_STACK_AGENT_MAP: Record<string, TechStackMapping> = {
  go: {
    agents: ["backend-go-developer", "go-readability-reviewer"],
    flow: [
      '"backend-go-developer" -> "go-readability-reviewer"',
    ],
  },
  htmx: {
    agents: ["htmx-picocss-frontend-developer", "htmx-picocss-frontend-reviewer"],
    flow: ['"htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"'],
  },
  react: {
    agents: ["react-developer", "react-reviewer"],
    flow: ['"react-developer" -> "react-reviewer"'],
  },
  python: {
    agents: ["general-purpose"],
    flow: [],
  },
  typescript: {
    agents: [],
    flow: [],
  },
  shell: {
    agents: ["shell-script-coder", "shell-script-reviewer"],
    flow: ['"shell-script-coder" -> "shell-script-reviewer"'],
  },
  "general-purpose": {
    agents: ["general-purpose"],
    flow: [],
  },
  claudesdk: {
    agents: ["general-purpose", "agent-sdk-verifier-ts", "agent-sdk-verifier-py"],
    flow: [
      '"general-purpose" -> "agent-sdk-verifier-ts"',
      '"general-purpose" -> "agent-sdk-verifier-py"',
    ],
  },
  openaisdk: {
    agents: ["general-purpose", "openai-sdk-verifier-ts", "openai-sdk-verifier-py"],
    flow: [
      '"general-purpose" -> "openai-sdk-verifier-ts"',
      '"general-purpose" -> "openai-sdk-verifier-py"',
    ],
  },
};

function computeAgentsAndFlowFromTechStack(
  techStack: string[],
  safetyAnalysis: boolean,
): { agents: ApiComposerAgentConf[]; flow: string } {
  const agentSet = new Set<string>(["coordinator"]);
  const flowLines: string[] = [];

  for (const tech of techStack) {
    const mapping = TECH_STACK_AGENT_MAP[tech];
    if (!mapping) continue;
    for (const agent of mapping.agents) {
      agentSet.add(agent);
    }
    for (const line of mapping.flow) {
      flowLines.push(line);
    }
  }

  if (safetyAnalysis) {
    agentSet.add("stpa-analyst");
    for (const tech of techStack) {
      const mapping = TECH_STACK_AGENT_MAP[tech];
      if (!mapping) continue;
      const reviewers = mapping.agents.filter(
        (a) => a.includes("reviewer") || a.includes("verifier"),
      );
      for (const reviewer of reviewers) {
        flowLines.push(`"${reviewer}" -> "stpa-analyst"`);
      }
      if (tech === "go") {
        flowLines.push('"backend-go-developer" -> "stpa-analyst"');
      }
      if (tech === "general-purpose") {
        flowLines.push('"general-purpose" -> "stpa-analyst"');
      }
    }
  }

  const agents: ApiComposerAgentConf[] = Array.from(agentSet)
    .sort()
    .map((name) => ({ name, selected: true, model: DEFAULT_MODEL }));

  const uniqueFlowLines = [...new Set(flowLines)];

  return { agents, flow: uniqueFlowLines.join("\n") };
}

function buildComposerStateFromData(
  data: WizardStepData,
  serverState: ApiComposerState | null,
): ApiComposerState {
  const { agents, flow } = computeAgentsAndFlowFromTechStack(
    data.techStack,
    data.safetyAnalysis,
  );

  const hasUserAgents = agents.length > 1;

  return {
    description: data.description,
    interactive: data.interactive,
    completionGate: data.completionGate,
    agents: hasUserAgents ? agents : (serverState?.agents ?? []),
    flow: hasUserAgents ? flow : (serverState?.flow ?? ""),
    tasks: serverState?.tasks ?? "",
  };
}

interface UseComposeWizardOptions {
  workspace: string;
  currentStep: number;
}

interface UseComposeWizardReturn {
  wizardData: WizardStepData;
  setWizardData: React.Dispatch<React.SetStateAction<WizardStepData>>;
  techStackItems: ApiTechStackItem[];
  preview: ApiComposePreviewResponse | null;
  isLoading: boolean;
  isSaving: boolean;
  isSavingDraft: boolean;
  saveError: string | null;
  draftSavedAt: string | null;
  etag: string | null;
  isDirty: boolean;
  fetchPreview: () => Promise<void>;
  saveGoal: () => Promise<boolean>;
  saveDraft: () => Promise<void>;
  goToStep: (step: number) => void;
  goToFinish: () => void;
  goBack: () => void;
}

const DEFAULT_WIZARD_DATA: WizardStepData = {
  description: "",
  techStack: [],
  safetyAnalysis: false,
  interactive: "yes",
  completionGate: "",
};

export function useComposeWizard({
  workspace,
  currentStep,
}: UseComposeWizardOptions): UseComposeWizardReturn {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [wizardData, setWizardData] = useState<WizardStepData>(DEFAULT_WIZARD_DATA);
  const [techStackItems, setTechStackItems] = useState<ApiTechStackItem[]>([]);
  const [preview, setPreview] = useState<ApiComposePreviewResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [draftSavedAt, setDraftSavedAt] = useState<string | null>(null);
  const [etag, setEtag] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const serverStateRef = useRef<ApiComposerState | null>(null);
  const autoSaveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const initialLoadDoneRef = useRef(false);
  const saveDraftRef = useRef<() => Promise<void>>(() => Promise.resolve());
  const isDirtyRef = useRef(false);

  // Load initial state from server + sessionStorage
  useEffect(() => {
    if (!workspace) return;

    let cancelled = false;

    async function loadState() {
      setIsLoading(true);
      try {
        const [stateResp, previewResp] = await Promise.all([
          api.compose.get(workspace),
          api.compose.preview(workspace),
        ]);

        if (cancelled) return;

        serverStateRef.current = stateResp.state;
        setTechStackItems(stateResp.techStackItems);
        setEtag(previewResp.etag);
        setPreview(previewResp);

        // Merge sessionStorage data per step with server state
        const merged: WizardStepData = {
          description: stateResp.wizard.description ?? stateResp.state.description ?? "",
          techStack: stateResp.wizard.techStack ?? [],
          safetyAnalysis: stateResp.wizard.safetyAnalysis ?? false,
          interactive: stateResp.wizard.interactive ?? stateResp.state.interactive ?? "yes",
          completionGate: stateResp.wizard.completionGate ?? stateResp.state.completionGate ?? "",
        };

        // Override with any sessionStorage persisted data per step (R-14)
        for (let step = 1; step <= 4; step++) {
          const storedStep = loadStepFromStorage(step);
          if (storedStep) {
            if (step === 1 && storedStep.description !== undefined) {
              merged.description = storedStep.description;
            }
            if (step === 2 && storedStep.techStack !== undefined) {
              merged.techStack = storedStep.techStack;
            }
            if (step === 3 && storedStep.safetyAnalysis !== undefined) {
              merged.safetyAnalysis = storedStep.safetyAnalysis;
            }
            if (step === 4) {
              if (storedStep.interactive !== undefined) {
                merged.interactive = storedStep.interactive;
              }
              if (storedStep.completionGate !== undefined) {
                merged.completionGate = storedStep.completionGate;
              }
            }
          }
        }

        setWizardData(merged);
        initialLoadDoneRef.current = true;
      } catch {
        // Silently handle fetch errors — use defaults
        initialLoadDoneRef.current = true;
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    loadState();

    return () => {
      cancelled = true;
    };
  }, [workspace]);

  // Persist current step data to sessionStorage on change (R-14)
  useEffect(() => {
    if (!initialLoadDoneRef.current) return;

    switch (currentStep) {
      case 1:
        saveStepToStorage(1, { description: wizardData.description });
        break;
      case 2:
        saveStepToStorage(2, { techStack: wizardData.techStack });
        break;
      case 3:
        saveStepToStorage(3, { safetyAnalysis: wizardData.safetyAnalysis });
        break;
      case 4:
        saveStepToStorage(4, {
          interactive: wizardData.interactive,
          completionGate: wizardData.completionGate,
        });
        break;
    }

    setIsDirty(true);
  }, [currentStep, wizardData]);

  // beforeunload warning when wizard has unsaved progress (R-9)
  useEffect(() => {
    if (!isDirty) return;

    function handleBeforeUnload(e: BeforeUnloadEvent) {
      e.preventDefault();
    }

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty]);

  const saveDraftInternal = useCallback(async () => {
    if (!workspace || isSavingDraft) return;

    setIsSavingDraft(true);
    try {
      const draftRequest = {
        state: buildComposerStateFromData(wizardData, serverStateRef.current),
        wizard: buildWizardStateFromData(wizardData, currentStep),
      };

      await api.compose.saveDraft(workspace, draftRequest);

      const now = new Date();
      setDraftSavedAt(
        now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      );
      setIsDirty(false);
    } catch {
      // Silently handle draft save errors
    } finally {
      setIsSavingDraft(false);
    }
  }, [workspace, wizardData, currentStep, isSavingDraft]);

  // Keep refs in sync for stable auto-save interval (avoids stale closures)
  useEffect(() => { saveDraftRef.current = saveDraftInternal; }, [saveDraftInternal]);
  useEffect(() => { isDirtyRef.current = isDirty; }, [isDirty]);

  // Auto-save draft every 30s (R-15) — uses refs to avoid stale closures
  useEffect(() => {
    if (!workspace || !initialLoadDoneRef.current) return;

    autoSaveTimerRef.current = setInterval(() => {
      if (!isDirtyRef.current) return;
      saveDraftRef.current();
    }, AUTO_SAVE_INTERVAL_MS);

    return () => {
      if (autoSaveTimerRef.current) {
        clearInterval(autoSaveTimerRef.current);
      }
    };
  }, [workspace]);

  const fetchPreview = useCallback(async () => {
    if (!workspace) return;
    try {
      // Save draft first to update server state, then fetch preview
      const draftRequest = {
        state: buildComposerStateFromData(wizardData, serverStateRef.current),
        wizard: buildWizardStateFromData(wizardData, currentStep),
      };
      await api.compose.saveDraft(workspace, draftRequest);

      const previewResp = await api.compose.preview(workspace);
      setPreview(previewResp);
      setEtag(previewResp.etag);
    } catch {
      // Silently handle preview errors
    }
  }, [workspace, wizardData, currentStep]);

  const saveGoal = useCallback(async (): Promise<boolean> => {
    if (!workspace) return false;

    setIsSaving(true);
    setSaveError(null);

    try {
      // Save draft first to ensure server state is up to date
      const draftRequest = {
        state: buildComposerStateFromData(wizardData, serverStateRef.current),
        wizard: buildWizardStateFromData(wizardData, currentStep),
      };
      await api.compose.saveDraft(workspace, draftRequest);

      // Save with optimistic locking (R-24)
      await api.compose.save(workspace, etag ?? undefined);

      clearAllWizardStorage();
      setIsDirty(false);
      return true;
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setSaveError("GOAL.md has been modified by another session. Please reload and try again.");
      } else if (err instanceof ApiError) {
        setSaveError(err.message);
      } else {
        setSaveError("Failed to save GOAL.md");
      }
      return false;
    } finally {
      setIsSaving(false);
    }
  }, [workspace, wizardData, currentStep, etag]);

  const saveDraft = useCallback(async () => {
    await saveDraftInternal();
  }, [saveDraftInternal]);

  const goToStep = useCallback(
    (step: number) => {
      const wsParam = workspace ? `?workspace=${encodeURIComponent(workspace)}` : "";
      navigate(`/compose/step/${step}${wsParam}`);
    },
    [navigate, workspace],
  );

  const goToFinish = useCallback(() => {
    const wsParam = workspace ? `?workspace=${encodeURIComponent(workspace)}` : "";
    navigate(`/compose/finish${wsParam}`);
  }, [navigate, workspace]);

  const goBack = useCallback(() => {
    if (currentStep > 1) {
      goToStep(currentStep - 1);
    } else {
      const wsParam = workspace ? `?workspace=${encodeURIComponent(workspace)}` : "";
      navigate(`/compose${wsParam}`);
    }
  }, [currentStep, goToStep, navigate, workspace]);

  return {
    wizardData,
    setWizardData,
    techStackItems,
    preview,
    isLoading,
    isSaving,
    isSavingDraft,
    saveError,
    draftSavedAt,
    etag,
    isDirty,
    fetchPreview,
    saveGoal,
    saveDraft,
    goToStep,
    goToFinish,
    goBack,
  };
}
