import { describe, it, expect } from "bun:test";
import { loadStepFromStorage, saveStepToStorage, type WizardStepData } from "../useComposeWizard";
import type { ApiComposeDraftRequest } from "@/types";

function createMemoryStorage(initial: Record<string, string> = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
    read: (key: string) => values.get(key),
  };
}

describe("useComposeWizard retrospective state helpers", () => {
  it("loads step 4 retrospective from session storage", () => {
    const storage = createMemoryStorage({
      "compose-wizard-step-4": JSON.stringify({
        completionGate: "make test",
        retrospective: true,
      }),
    });

    const result = loadStepFromStorage(4, storage);

    expect(result.error).toBeNull();
    expect(result.data).toEqual({
      completionGate: "make test",
      retrospective: true,
    });
  });

  it("saves step 4 retrospective to session storage", () => {
    const storage = createMemoryStorage();

    const error = saveStepToStorage(4, {
      completionGate: "make test",
      retrospective: true,
    }, storage);

    expect(error).toBeNull();
    expect(JSON.parse(storage.read("compose-wizard-step-4") ?? "{}")).toEqual({
      completionGate: "make test",
      retrospective: true,
    });
  });

  it("requires compose draft payloads to carry retrospective on state and wizard", () => {
    const wizardData: WizardStepData = {
      description: "Build a thing",
      techStack: ["react"],
      safetyAnalysis: false,
      completionGate: "",
      retrospective: true,
    };

    const draft: ApiComposeDraftRequest = {
      state: {
        description: wizardData.description,
        completionGate: wizardData.completionGate,
        retrospective: wizardData.retrospective,
        agents: [],
        model: "openai/gpt-5.5 (xhigh)",
        tasks: "",
      },
      wizard: {
        currentStep: 4,
        description: wizardData.description,
        techStack: wizardData.techStack,
        safetyAnalysis: wizardData.safetyAnalysis,
        completionGate: wizardData.completionGate,
        retrospective: wizardData.retrospective,
      },
    };

    expect(draft.state.retrospective).toBe(true);
    expect(draft.wizard.retrospective).toBe(true);
  });
});
