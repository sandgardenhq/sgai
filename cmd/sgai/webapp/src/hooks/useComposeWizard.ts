import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router";
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
const WIZARD_STEPS = [1, 2, 3, 4] as const;

export interface WizardStepData {
  description: string;
  techStack: string[];
  safetyAnalysis: boolean;
  completionGate: string;
}

interface StorageLoadResult {
  data: Partial<WizardStepData> | null;
  error: string | null;
}

function getStepStorageKey(step: number): string {
  return `${STORAGE_KEY_PREFIX}${step}`;
}

function validateStoredStepData(
  step: number,
  raw: unknown,
): Partial<WizardStepData> | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return null;
  }

  const candidate = raw as Record<string, unknown>;

  switch (step) {
    case 1:
      return typeof candidate.description === "string"
        ? { description: candidate.description }
        : null;
    case 2:
      return Array.isArray(candidate.techStack) &&
        candidate.techStack.every((item) => typeof item === "string")
        ? { techStack: candidate.techStack }
        : null;
    case 3:
      return typeof candidate.safetyAnalysis === "boolean"
        ? { safetyAnalysis: candidate.safetyAnalysis }
        : null;
    case 4:
      return typeof candidate.completionGate === "string"
        ? { completionGate: candidate.completionGate }
        : null;
    default:
      return null;
  }
}

export function loadStepFromStorage(
  step: number,
  storage: Pick<Storage, "getItem"> = sessionStorage,
): StorageLoadResult {
  let stored: string | null;
  try {
    stored = storage.getItem(getStepStorageKey(step));
  } catch (errStorageRead) {
    const message = errStorageRead instanceof Error ? errStorageRead.message : "unknown sessionStorage read error";
    return {
      data: null,
      error: `Failed to read step ${step} from sessionStorage: ${message}`,
    };
  }

  if (!stored) {
    return { data: null, error: null };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(stored);
  } catch (errParseStorage) {
    const message = errParseStorage instanceof Error ? errParseStorage.message : "unknown JSON parse error";
    return {
      data: null,
      error: `Invalid JSON in step ${step} sessionStorage data: ${message}`,
    };
  }

  const validated = validateStoredStepData(step, parsed);
  if (!validated) {
    return {
      data: null,
      error: `Invalid payload shape in step ${step} sessionStorage data`,
    };
  }

  return { data: validated, error: null };
}

export function saveStepToStorage(
  step: number,
  data: Partial<WizardStepData>,
  storage: Pick<Storage, "setItem"> = sessionStorage,
): string | null {
  try {
    storage.setItem(getStepStorageKey(step), JSON.stringify(data));
    return null;
  } catch (errStorageWrite) {
    const message = errStorageWrite instanceof Error ? errStorageWrite.message : "unknown sessionStorage write error";
    return `Failed to save step ${step} to sessionStorage: ${message}`;
  }
}

export function clearAllWizardStorage(
  storage: Pick<Storage, "removeItem"> = sessionStorage,
): string | null {
  for (const step of WIZARD_STEPS) {
    try {
      storage.removeItem(getStepStorageKey(step));
    } catch (errStorageRemove) {
      const message = errStorageRemove instanceof Error ? errStorageRemove.message : "unknown sessionStorage remove error";
      return `Failed to clear step ${step} from sessionStorage: ${message}`;
    }
  }
  return null;
}

function buildWizardStateFromData(data: WizardStepData, step: number): ApiWizardState {
  return {
    currentStep: step,
    description: data.description,
    techStack: data.techStack,
    safetyAnalysis: data.safetyAnalysis,
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
    completionGate: data.completionGate,
    agents: hasUserAgents ? agents : (serverState?.agents ?? []),
    flow: hasUserAgents ? flow : (serverState?.flow ?? ""),
    tasks: serverState?.tasks ?? "",
  };
}

function buildDraftRequest(
  data: WizardStepData,
  currentStep: number,
  serverState: ApiComposerState | null,
): { state: ApiComposerState; wizard: ApiWizardState } {
  return {
    state: buildComposerStateFromData(data, serverState),
    wizard: buildWizardStateFromData(data, currentStep),
  };
}

function workspaceSearchParam(workspace: string): string {
  return workspace ? `?workspace=${encodeURIComponent(workspace)}` : "";
}

function getErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    return err.message;
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return fallback;
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

function serializeWizardData(data: WizardStepData): string {
  return JSON.stringify({
    description: data.description,
    techStack: data.techStack,
    safetyAnalysis: data.safetyAnalysis,
    completionGate: data.completionGate,
  });
}

const DEFAULT_WIZARD_DATA: WizardStepData = {
  description: "",
  techStack: [],
  safetyAnalysis: false,
  completionGate: "",
};

export function useComposeWizard({
  workspace,
  currentStep,
}: UseComposeWizardOptions): UseComposeWizardReturn {
  const navigate = useNavigate();
  const [wizardData, setWizardData] = useState<WizardStepData>(DEFAULT_WIZARD_DATA);
  const [techStackItems, setTechStackItems] = useState<ApiTechStackItem[]>([]);
  const [preview, setPreview] = useState<ApiComposePreviewResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isInitialLoadDone, setIsInitialLoadDone] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [draftSavedAt, setDraftSavedAt] = useState<string | null>(null);
  const [etag, setEtag] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const serverStateRef = useRef<ApiComposerState | null>(null);
  const autoSaveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const saveDraftRef = useRef<() => Promise<void>>(() => Promise.resolve());
  const isDirtyRef = useRef(false);
  const cleanBaselineRef = useRef(serializeWizardData(DEFAULT_WIZARD_DATA));
  const wizardDataRef = useRef(wizardData);

  // Load initial state from server + sessionStorage
  useEffect(() => {
    if (!workspace) return;

    let cancelled = false;
    setIsInitialLoadDone(false);

    async function loadState() {
      setIsLoading(true);
      setSaveError(null);
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
          completionGate: stateResp.wizard.completionGate ?? stateResp.state.completionGate ?? "",
        };

        // Override with any sessionStorage persisted data per step (R-14)
        const storageErrors: string[] = [];
        for (const step of WIZARD_STEPS) {
          const storedStepResult = loadStepFromStorage(step);
          if (storedStepResult.error) {
            storageErrors.push(storedStepResult.error);
            continue;
          }

          const storedStep = storedStepResult.data;
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
              if (storedStep.completionGate !== undefined) {
                merged.completionGate = storedStep.completionGate;
              }
            }
          }
        }

        setWizardData(merged);
        cleanBaselineRef.current = serializeWizardData(merged);
        setIsDirty(false);
        if (storageErrors.length > 0) {
          setSaveError(storageErrors[0]);
        }
      } catch (err) {
        setSaveError(getErrorMessage(err, "Failed to load compose wizard state"));
      } finally {
        if (!cancelled) {
          setIsLoading(false);
          setIsInitialLoadDone(true);
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
    if (!isInitialLoadDone) return;

    let storageError: string | null = null;

    switch (currentStep) {
      case 1:
        storageError = saveStepToStorage(1, { description: wizardData.description });
        break;
      case 2:
        storageError = saveStepToStorage(2, { techStack: wizardData.techStack });
        break;
      case 3:
        storageError = saveStepToStorage(3, { safetyAnalysis: wizardData.safetyAnalysis });
        break;
      case 4:
        storageError = saveStepToStorage(4, {
          completionGate: wizardData.completionGate,
        });
        break;
    }

    if (storageError) {
      setSaveError(storageError);
    }

    setIsDirty(serializeWizardData(wizardData) !== cleanBaselineRef.current);
  }, [currentStep, isInitialLoadDone, wizardData]);

  // beforeunload warning when wizard has unsaved progress (R-9)
  useEffect(() => {
    if (!isDirty) return;

    function handleBeforeUnload(e: BeforeUnloadEvent) {
      e.preventDefault();
      e.returnValue = "";
    }

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty]);

  const saveDraftInternal = useCallback(async () => {
    if (!workspace || isSavingDraft) return;

    const savedSnapshot = serializeWizardData(wizardData);

    setIsSavingDraft(true);
    try {
      const draftRequest = buildDraftRequest(
        wizardData,
        currentStep,
        serverStateRef.current,
      );

      await api.compose.saveDraft(workspace, draftRequest);

      const now = new Date();
      setDraftSavedAt(
        now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      );
      cleanBaselineRef.current = savedSnapshot;
      const latestSnapshot = serializeWizardData(wizardDataRef.current);
      setIsDirty(latestSnapshot !== savedSnapshot);
      setSaveError(null);
    } catch (err) {
      setSaveError(getErrorMessage(err, "Failed to save draft"));
    } finally {
      setIsSavingDraft(false);
    }
  }, [workspace, wizardData, currentStep, isSavingDraft]);

  // Keep refs in sync for stable auto-save interval (avoids stale closures)
  useEffect(() => {
    saveDraftRef.current = saveDraftInternal;
  }, [saveDraftInternal]);

  useEffect(() => {
    isDirtyRef.current = isDirty;
  }, [isDirty]);

  useEffect(() => {
    wizardDataRef.current = wizardData;
  }, [wizardData]);

  // Auto-save draft every 30s (R-15) — uses refs to avoid stale closures
  useEffect(() => {
    if (!workspace || !isInitialLoadDone) return;

    autoSaveTimerRef.current = setInterval(() => {
      if (!isDirtyRef.current) return;
      saveDraftRef.current();
    }, AUTO_SAVE_INTERVAL_MS);

    return () => {
      if (autoSaveTimerRef.current) {
        clearInterval(autoSaveTimerRef.current);
      }
    };
  }, [workspace, isInitialLoadDone]);

  const fetchPreview = useCallback(async () => {
    if (!workspace) return;
    try {
      // Save draft first to update server state, then fetch preview
      const draftRequest = buildDraftRequest(
        wizardData,
        currentStep,
        serverStateRef.current,
      );
      await api.compose.saveDraft(workspace, draftRequest);

      const previewResp = await api.compose.preview(workspace);
      setPreview(previewResp);
      setEtag(previewResp.etag);
      setSaveError(null);
    } catch (err) {
      setSaveError(getErrorMessage(err, "Failed to refresh preview"));
    }
  }, [workspace, wizardData, currentStep]);

  const saveGoal = useCallback(async (): Promise<boolean> => {
    if (!workspace) return false;

    setIsSaving(true);
    setSaveError(null);

    try {
      // Save draft first to ensure server state is up to date
      const draftRequest = buildDraftRequest(
        wizardData,
        currentStep,
        serverStateRef.current,
      );
      await api.compose.saveDraft(workspace, draftRequest);

      // Save with optimistic locking (R-24)
      await api.compose.save(workspace, etag ?? undefined);

      const clearStorageError = clearAllWizardStorage();
      if (clearStorageError) {
        setSaveError(clearStorageError);
      }
      const savedSnapshot = serializeWizardData(wizardData);
      cleanBaselineRef.current = savedSnapshot;
      const latestSnapshot = serializeWizardData(wizardDataRef.current);
      setIsDirty(latestSnapshot !== savedSnapshot);
      return true;
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        setSaveError("GOAL.md has been modified by another session. Please reload and try again.");
      } else {
        setSaveError(getErrorMessage(err, "Failed to save GOAL.md"));
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
      navigate(`/compose/step/${step}${workspaceSearchParam(workspace)}`);
    },
    [navigate, workspace],
  );

  const goToFinish = useCallback(() => {
    navigate(`/compose/finish${workspaceSearchParam(workspace)}`);
  }, [navigate, workspace]);

  const goBack = useCallback(() => {
    if (currentStep > 1) {
      goToStep(currentStep - 1);
    } else {
      navigate(`/compose${workspaceSearchParam(workspace)}`);
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
