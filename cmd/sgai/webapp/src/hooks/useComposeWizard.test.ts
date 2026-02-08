import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import {
  loadStepFromStorage,
  saveStepToStorage,
  clearAllWizardStorage,
} from "./useComposeWizard";

beforeEach(() => {
  sessionStorage.clear();
});

afterEach(() => {
  sessionStorage.clear();
});

describe("useComposeWizard storage functions", () => {
  it("loadStepFromStorage returns null when no data", () => {
    const result = loadStepFromStorage(1);
    expect(result).toBeNull();
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 1", () => {
    saveStepToStorage(1, { description: "My project" });
    const loaded = loadStepFromStorage(1);
    expect(loaded).not.toBeNull();
    expect(loaded!.description).toBe("My project");
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 2", () => {
    saveStepToStorage(2, { techStack: ["go", "react"] });
    const loaded = loadStepFromStorage(2);
    expect(loaded).not.toBeNull();
    expect(loaded!.techStack).toEqual(["go", "react"]);
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 3", () => {
    saveStepToStorage(3, { safetyAnalysis: true });
    const loaded = loadStepFromStorage(3);
    expect(loaded).not.toBeNull();
    expect(loaded!.safetyAnalysis).toBe(true);
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 4", () => {
    saveStepToStorage(4, { interactive: "auto", completionGate: "make test" });
    const loaded = loadStepFromStorage(4);
    expect(loaded).not.toBeNull();
    expect(loaded!.interactive).toBe("auto");
    expect(loaded!.completionGate).toBe("make test");
  });

  it("each step uses separate storage key", () => {
    saveStepToStorage(1, { description: "Step 1 data" });
    saveStepToStorage(2, { techStack: ["python"] });

    const step1 = loadStepFromStorage(1);
    const step2 = loadStepFromStorage(2);

    expect(step1!.description).toBe("Step 1 data");
    expect(step2!.techStack).toEqual(["python"]);
    expect(step1!.techStack).toBeUndefined();
    expect(step2!.description).toBeUndefined();
  });

  it("clearAllWizardStorage removes all steps", () => {
    saveStepToStorage(1, { description: "data" });
    saveStepToStorage(2, { techStack: ["go"] });
    saveStepToStorage(3, { safetyAnalysis: true });
    saveStepToStorage(4, { interactive: "yes" });

    clearAllWizardStorage();

    expect(loadStepFromStorage(1)).toBeNull();
    expect(loadStepFromStorage(2)).toBeNull();
    expect(loadStepFromStorage(3)).toBeNull();
    expect(loadStepFromStorage(4)).toBeNull();
  });

  it("loadStepFromStorage handles invalid JSON gracefully", () => {
    sessionStorage.setItem("compose-wizard-step-1", "not-json");
    const result = loadStepFromStorage(1);
    expect(result).toBeNull();
  });

  it("saveStepToStorage overwrites existing data", () => {
    saveStepToStorage(1, { description: "first" });
    saveStepToStorage(1, { description: "second" });
    const loaded = loadStepFromStorage(1);
    expect(loaded!.description).toBe("second");
  });
});
