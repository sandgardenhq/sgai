import { describe, it, expect } from "bun:test";
import { buildComposerStateFromData, clearAllWizardStorage, loadStepFromStorage, saveStepToStorage, type WizardStepData } from "../useComposeWizard";
import type { ApiComposeDraftRequest } from "@/types";

function createMemoryStorage(initial: Record<string, string> = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
    removeItem: (key: string) => {
      values.delete(key);
    },
    read: (key: string) => values.get(key),
  };
}

describe("useComposeWizard retrospective state helpers", () => {
  it("loads step 3 settings from session storage", () => {
    const storage = createMemoryStorage({
      "compose-wizard-step-3": JSON.stringify({
        completionGate: "make test",
        retrospective: true,
      }),
    });

    const result = loadStepFromStorage(3, storage);

    expect(result.error).toBeNull();
    expect(result.data).toEqual({
      completionGate: "make test",
      retrospective: true,
    });
  });

  it("saves step 3 settings to session storage", () => {
    const storage = createMemoryStorage();

    const error = saveStepToStorage(3, {
      completionGate: "make test",
      retrospective: true,
    }, storage);

    expect(error).toBeNull();
    expect(JSON.parse(storage.read("compose-wizard-step-3") ?? "{}")).toEqual({
      completionGate: "make test",
      retrospective: true,
    });
  });

  it("ignores and clears stale pre-cleanup step 3 safety analysis storage", () => {
    const storage = createMemoryStorage({
      "compose-wizard-step-3": JSON.stringify({
        safetyAnalysis: true,
      }),
    });

    const result = loadStepFromStorage(3, storage);

    expect(result.error).toBeNull();
    expect(result.data).toBeNull();
    expect(storage.read("compose-wizard-step-3")).toBeUndefined();
  });

  it("clears orphaned pre-renumber step 4 storage with current wizard storage", () => {
    const storage = createMemoryStorage({
      "compose-wizard-step-1": JSON.stringify({ description: "Build a thing" }),
      "compose-wizard-step-2": JSON.stringify({ techStack: ["react"] }),
      "compose-wizard-step-3": JSON.stringify({ completionGate: "make test", retrospective: true }),
      "compose-wizard-step-4": JSON.stringify({ completionGate: "old make test" }),
    });

    const error = clearAllWizardStorage(storage);

    expect(error).toBeNull();
    expect(storage.read("compose-wizard-step-1")).toBeUndefined();
    expect(storage.read("compose-wizard-step-2")).toBeUndefined();
    expect(storage.read("compose-wizard-step-3")).toBeUndefined();
    expect(storage.read("compose-wizard-step-4")).toBeUndefined();
  });

  it("requires compose draft payloads to carry retrospective on state and wizard", () => {
    const wizardData: WizardStepData = {
      description: "Build a thing",
      techStack: ["react"],
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
        currentStep: 3,
        description: wizardData.description,
        techStack: wizardData.techStack,
        completionGate: wizardData.completionGate,
        retrospective: wizardData.retrospective,
      },
    };

    expect(draft.state.retrospective).toBe(true);
    expect(draft.wizard.retrospective).toBe(true);
  });

  it("removes stale tech-derived agents when matching tech selections are deselected", () => {
    const wizardData: WizardStepData = {
      description: "Build a thing",
      techStack: ["go"],
      completionGate: "make test",
      retrospective: false,
    };

    const state = buildComposerStateFromData(wizardData, {
      description: "Build a thing",
      completionGate: "make test",
      retrospective: false,
      agents: [
        { name: "go", selected: true },
        { name: "react", selected: true },
        { name: "manual-reviewer", selected: true },
      ],
      model: "openai/gpt-5.5 (xhigh)",
      tasks: "",
    });

    expect(state.agents).toHaveLength(2);
    expect(state.agents).toContainEqual({ name: "manual-reviewer", selected: true });
    expect(state.agents).toContainEqual({ name: "go", selected: true });
    expect(state.agents).not.toContainEqual({ name: "react", selected: true });
  });
});
