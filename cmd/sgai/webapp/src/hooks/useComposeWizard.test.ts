import { createElement, type ReactNode } from "react";
import { describe, it, expect, beforeEach, afterEach, spyOn } from "bun:test";
import { renderHook, act, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import {
  useComposeWizard,
  loadStepFromStorage,
  saveStepToStorage,
  clearAllWizardStorage,
} from "./useComposeWizard";

beforeEach(() => {
  sessionStorage.clear();
});

afterEach(() => {
  cleanup();
  sessionStorage.clear();
});

function jsonResponse(data: unknown): Response {
  return new Response(JSON.stringify(data), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function RouterWrapper({ children }: { children: ReactNode }) {
  return createElement(
    MemoryRouter,
    { initialEntries: ["/compose/step/1?workspace=test-ws"] },
    children,
  );
}

describe("useComposeWizard storage functions", () => {
  it("loadStepFromStorage returns null when no data", () => {
    const result = loadStepFromStorage(1);
    expect(result.data).toBeNull();
    expect(result.error).toBeNull();
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 1", () => {
    const saveError = saveStepToStorage(1, { description: "My project" });
    expect(saveError).toBeNull();
    const loaded = loadStepFromStorage(1);
    expect(loaded.error).toBeNull();
    expect(loaded.data).not.toBeNull();
    expect(loaded.data!.description).toBe("My project");
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 2", () => {
    const saveError = saveStepToStorage(2, { techStack: ["go", "react"] });
    expect(saveError).toBeNull();
    const loaded = loadStepFromStorage(2);
    expect(loaded.error).toBeNull();
    expect(loaded.data).not.toBeNull();
    expect(loaded.data!.techStack).toEqual(["go", "react"]);
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 3", () => {
    const saveError = saveStepToStorage(3, { safetyAnalysis: true });
    expect(saveError).toBeNull();
    const loaded = loadStepFromStorage(3);
    expect(loaded.error).toBeNull();
    expect(loaded.data).not.toBeNull();
    expect(loaded.data!.safetyAnalysis).toBe(true);
  });

  it("saveStepToStorage and loadStepFromStorage round-trip for step 4", () => {
    const saveError = saveStepToStorage(4, { completionGate: "make test" });
    expect(saveError).toBeNull();
    const loaded = loadStepFromStorage(4);
    expect(loaded.error).toBeNull();
    expect(loaded.data).not.toBeNull();
    expect(loaded.data!.completionGate).toBe("make test");
  });

  it("each step uses separate storage key", () => {
    saveStepToStorage(1, { description: "Step 1 data" });
    saveStepToStorage(2, { techStack: ["python"] });

    const step1 = loadStepFromStorage(1);
    const step2 = loadStepFromStorage(2);

    expect(step1.data!.description).toBe("Step 1 data");
    expect(step2.data!.techStack).toEqual(["python"]);
    expect(step1.data!.techStack).toBeUndefined();
    expect(step2.data!.description).toBeUndefined();
  });

  it("clearAllWizardStorage removes all steps", () => {
    saveStepToStorage(1, { description: "data" });
    saveStepToStorage(2, { techStack: ["go"] });
    saveStepToStorage(3, { safetyAnalysis: true });
    saveStepToStorage(4, { completionGate: "make test" });

    const clearError = clearAllWizardStorage();
    expect(clearError).toBeNull();

    expect(loadStepFromStorage(1).data).toBeNull();
    expect(loadStepFromStorage(2).data).toBeNull();
    expect(loadStepFromStorage(3).data).toBeNull();
    expect(loadStepFromStorage(4).data).toBeNull();
  });

  it("loadStepFromStorage handles invalid JSON gracefully", () => {
    sessionStorage.setItem("compose-wizard-step-1", "not-json");
    const result = loadStepFromStorage(1);
    expect(result.data).toBeNull();
    expect(result.error).toContain("Invalid JSON in step 1 sessionStorage data");
  });

  it("loadStepFromStorage rejects invalid payload shape", () => {
    sessionStorage.setItem("compose-wizard-step-2", JSON.stringify({ techStack: "go" }));
    const result = loadStepFromStorage(2);
    expect(result.data).toBeNull();
    expect(result.error).toBe("Invalid payload shape in step 2 sessionStorage data");
  });

  it("loadStepFromStorage surfaces sessionStorage get errors", () => {
    const result = loadStepFromStorage(1, {
      getItem: () => {
        throw new Error("storage unavailable");
      },
    });
    expect(result.data).toBeNull();
    expect(result.error).not.toBeNull();
    expect(result.error).toContain("Failed to read step 1 from sessionStorage");
  });

  it("saveStepToStorage surfaces sessionStorage set errors", () => {
    const saveError = saveStepToStorage(
      1,
      { description: "blocked" },
      {
        setItem: () => {
          throw new Error("quota exceeded");
        },
      },
    );
    expect(saveError).not.toBeNull();
    expect(saveError).toContain("Failed to save step 1 to sessionStorage");
  });

  it("clearAllWizardStorage surfaces sessionStorage remove errors", () => {
    const clearError = clearAllWizardStorage({
      removeItem: () => {
        throw new Error("remove blocked");
      },
    });
    expect(clearError).not.toBeNull();
    expect(clearError).toContain("Failed to clear step 1 from sessionStorage");
  });

  it("saveStepToStorage overwrites existing data", () => {
    saveStepToStorage(1, { description: "first" });
    saveStepToStorage(1, { description: "second" });
    const loaded = loadStepFromStorage(1);
    expect(loaded.data!.description).toBe("second");
  });

  it("ignores malformed session payloads during merge and surfaces saveError", async () => {
    sessionStorage.setItem("compose-wizard-step-2", JSON.stringify({ techStack: "invalid" }));

    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: {
            description: "",
            completionGate: "",
            agents: [],
            flow: "",
            tasks: "",
          },
          wizard: {
            currentStep: 1,
            description: "",
            techStack: ["go"],
            safetyAnalysis: false,
            completionGate: "",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(result.current.wizardData.techStack).toEqual(["go"]);
    expect(result.current.saveError).toBe("Invalid payload shape in step 2 sessionStorage data");

    fetchSpy.mockRestore();
  });

  it("surfaces load failures in saveError", async () => {
    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.reject(new Error("failed to load compose state"));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(result.current.isLoading).toBe(false);
    expect(result.current.saveError).toBe("failed to load compose state");

    fetchSpy.mockRestore();
  });

  it("surfaces draft save failures in saveError", async () => {
    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: { description: "", completionGate: "", agents: [], flow: "", tasks: "" },
          wizard: {
            currentStep: 1,
            description: "",
            techStack: [],
            safetyAnalysis: false,
            completionGate: "",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      if (url.includes("/api/v1/compose/draft?workspace=") && init?.method === "POST") {
        return Promise.reject(new Error("failed to save draft"));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    await act(async () => {
      await result.current.saveDraft();
    });

    expect(result.current.saveError).toBe("failed to save draft");

    fetchSpy.mockRestore();
  });

  it("surfaces preview refresh failures in saveError", async () => {
    let previewRequestCount = 0;
    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: { description: "", completionGate: "", agents: [], flow: "", tasks: "" },
          wizard: {
            currentStep: 1,
            description: "",
            techStack: [],
            safetyAnalysis: false,
            completionGate: "",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        previewRequestCount += 1;
        if (previewRequestCount > 1) {
          return Promise.reject(new Error("failed to refresh preview"));
        }
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      if (url.includes("/api/v1/compose/draft?workspace=") && init?.method === "POST") {
        return Promise.resolve(jsonResponse({}));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    await act(async () => {
      await result.current.fetchPreview();
    });

    expect(result.current.saveError).toBe("failed to refresh preview");

    fetchSpy.mockRestore();
  });

  it("keeps isDirty true when user edits during in-flight draft save", async () => {
    let draftResolve: (() => void) | null = null;
    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: { description: "", completionGate: "", agents: [], flow: "", tasks: "" },
          wizard: {
            currentStep: 1,
            description: "",
            techStack: [],
            safetyAnalysis: false,
            completionGate: "",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      if (url.includes("/api/v1/compose/draft?workspace=") && init?.method === "POST") {
        return new Promise<Response>((resolve) => {
          draftResolve = () => resolve(jsonResponse({}));
        });
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    act(() => {
      result.current.setWizardData((prev) => ({ ...prev, description: "first edit" }));
    });

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(result.current.isDirty).toBe(true);

    let savePromise: Promise<void> | null = null;
    act(() => {
      savePromise = result.current.saveDraft();
    });

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(result.current.isSavingDraft).toBe(true);

    act(() => {
      result.current.setWizardData((prev) => ({ ...prev, description: "second edit during save" }));
    });

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    await act(async () => {
      draftResolve!();
      await savePromise;
    });

    expect(result.current.isDirty).toBe(true);
    expect(result.current.isSavingDraft).toBe(false);

    fetchSpy.mockRestore();
  });

  it("starts auto-save interval after load and saves when dirty", async () => {
    let draftSaveCount = 0;
    let autoSaveCallback: (() => void) | null = null;

    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: { description: "", completionGate: "", agents: [], flow: "", tasks: "" },
          wizard: {
            currentStep: 1,
            description: "",
            techStack: [],
            safetyAnalysis: false,
            completionGate: "",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      if (url.includes("/api/v1/compose/draft?workspace=") && init?.method === "POST") {
        draftSaveCount++;
        return Promise.resolve(jsonResponse({}));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const setIntervalSpy = spyOn(globalThis, "setInterval").mockImplementation((handler: TimerHandler) => {
      autoSaveCallback = handler as () => void;
      return 1 as unknown as ReturnType<typeof setInterval>;
    });

    const clearIntervalSpy = spyOn(globalThis, "clearInterval").mockImplementation(() => {});

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(setIntervalSpy).toHaveBeenCalledTimes(1);
    expect(autoSaveCallback).not.toBeNull();

    act(() => {
      result.current.setWizardData((previous) => ({
        ...previous,
        description: "updated",
      }));
    });

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(draftSaveCount).toBe(0);

    await act(async () => {
      autoSaveCallback?.();
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(draftSaveCount).toBe(1);

    fetchSpy.mockRestore();
    setIntervalSpy.mockRestore();
    clearIntervalSpy.mockRestore();
  });

  it("keeps initial hydrated state clean", async () => {
    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: {
            description: "server description",
            completionGate: "make test",
            agents: [],
            flow: "",
            tasks: "",
          },
          wizard: {
            currentStep: 1,
            description: "server description",
            techStack: ["go"],
            safetyAnalysis: false,
            completionGate: "make test",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(result.current.isDirty).toBe(false);

    const unloadEvent = new Event("beforeunload", { cancelable: true });
    const dispatchResult = window.dispatchEvent(unloadEvent);
    expect(dispatchResult).toBe(true);
    expect(unloadEvent.defaultPrevented).toBe(false);

    fetchSpy.mockRestore();
  });

  it("activates dirty state and beforeunload protection only after edits", async () => {
    const fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input: string | URL | Request, init?: RequestInit) => {
      const url = input.toString();
      if (url.includes("/api/v1/compose?workspace=") && !init?.method) {
        return Promise.resolve(jsonResponse({
          workspace: "test-ws",
          state: { description: "", completionGate: "", agents: [], flow: "", tasks: "" },
          wizard: {
            currentStep: 1,
            description: "",
            techStack: [],
            safetyAnalysis: false,
            completionGate: "",
          },
          techStackItems: [],
        }));
      }
      if (url.includes("/api/v1/compose/preview?workspace=")) {
        return Promise.resolve(jsonResponse({ content: "# preview", etag: '"etag"' }));
      }
      return Promise.resolve(jsonResponse({}));
    });

    const { result } = renderHook(
      () => useComposeWizard({ workspace: "test-ws", currentStep: 1 }),
      { wrapper: RouterWrapper },
    );

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(result.current.isDirty).toBe(false);

    act(() => {
      result.current.setWizardData((previous) => ({
        ...previous,
        description: "changed",
      }));
    });

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(result.current.isDirty).toBe(true);

    const unloadEvent = new Event("beforeunload", { cancelable: true }) as BeforeUnloadEvent;
    Object.defineProperty(unloadEvent, "returnValue", {
      value: undefined,
      writable: true,
      configurable: true,
    });

    const dispatchResult = window.dispatchEvent(unloadEvent);
    expect(dispatchResult).toBe(false);
    expect(unloadEvent.defaultPrevented).toBe(true);
    expect(unloadEvent.returnValue).toBe("");

    fetchSpy.mockRestore();
  });
});
