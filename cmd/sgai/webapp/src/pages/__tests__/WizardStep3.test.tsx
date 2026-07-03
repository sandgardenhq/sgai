import { describe, it, expect, beforeEach, afterEach, mock, spyOn } from "bun:test";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider } from "react-router/dom";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiComposeDraftRequest } from "@/types";

let savedDraft: ApiComposeDraftRequest | null = null;
const saveDraftCalls: ApiComposeDraftRequest[] = [];

const defaultDraft: ApiComposeDraftRequest = {
  state: {
    description: "",
    completionGate: "",
    retrospective: false,
    agents: [],
    model: "openai/gpt-5.5 (xhigh)",
    tasks: "",
  },
  wizard: {
    currentStep: 3,
    description: "",
    techStack: [],
    completionGate: "",
    retrospective: false,
  },
};

const mockGet = mock(() => Promise.resolve({
  workspace: "demo",
  state: savedDraft?.state ?? defaultDraft.state,
  wizard: savedDraft?.wizard ?? defaultDraft.wizard,
  techStackItems: [],
}));

const mockPreview = mock(() => Promise.resolve({
  content: savedDraft?.state.retrospective
    ? "---\nretrospective: true\n---\n# GOAL"
    : "---\n---\n# GOAL",
  etag: "etag-1",
}));

const mockSaveDraft = mock((_workspace: string, draft: ApiComposeDraftRequest) => {
  savedDraft = draft;
  saveDraftCalls.push(draft);
  return Promise.resolve({ saved: true });
});

const mockSaveGoal = mock(() => Promise.resolve({ saved: true, workspace: "demo" }));

mock.module("@/lib/api", () => ({
  api: {
    compose: {
      get: mockGet,
      preview: mockPreview,
      saveDraft: mockSaveDraft,
      save: mockSaveGoal,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

mock.module("@/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ value, onChange }: { value: string; onChange: (value: string | undefined) => void }) => (
    <textarea
      aria-label="Project Description"
      value={value}
      onChange={(event) => onChange(event.currentTarget.value)}
    />
  ),
}));

const { createMemoryRouter } = await import("react-router");

const { WizardStep1 } = await import("../WizardStep1");
const { WizardStep2 } = await import("../WizardStep2");
const { WizardStep3 } = await import("../WizardStep3");
const { WizardFinish } = await import("../WizardFinish");

const hookOrderErrorPattern = /change in the order of Hooks|Rendered more hooks|Rendered fewer hooks/i;

function renderStep3() {
  const router = createMemoryRouter([
    {
      path: "/compose/step/3",
      element: <WizardStep3 />,
    },
  ], {
    initialEntries: ["/compose/step/3?workspace=demo"],
  });

  return render(
    <RouterProvider router={router} />,
  );
}

function renderWizardPageWithoutWorkspace(path: string, element: React.ReactNode) {
  const router = createMemoryRouter([
    {
      path: "/",
      element: <h1>Dashboard</h1>,
    },
    {
      path,
      element: (
        <TooltipProvider>
          {element}
        </TooltipProvider>
      ),
    },
  ], {
    initialEntries: [path],
  });

  const view = render(
    <RouterProvider router={router} />,
  );

  return { router, view };
}

async function expectMissingWorkspaceNavigation(path: string, element: React.ReactNode) {
  const consoleError = spyOn(console, "error").mockImplementation(() => {});

  try {
    const { view } = renderWizardPageWithoutWorkspace(path, element);
    const user = userEvent.setup();

    expect(screen.getByRole("alert")).not.toBeNull();
    expect(screen.getByText("Workspace required")).not.toBeNull();

    await user.click(screen.getByRole("button", { name: "Back to Dashboard" }));
    expect(await screen.findByRole("heading", { name: "Dashboard" })).not.toBeNull();
    await waitFor(() => {
      expect(screen.queryByRole("alert")).toBeNull();
    });

    const hookOrderMessages: string[] = [];
    for (const call of consoleError.mock.calls) {
      for (const value of call) {
        const message = String(value);
        if (hookOrderErrorPattern.test(message)) {
          hookOrderMessages.push(message);
        }
      }
    }
    expect(hookOrderMessages).toEqual([]);

    view.unmount();
  } finally {
    consoleError.mockRestore();
    cleanup();
  }
}

function renderFinish() {
  const router = createMemoryRouter([
    {
      path: "/compose/finish",
      element: <WizardFinish />,
    },
    {
      path: "/workspaces/:workspace",
      element: <div>Workspace page</div>,
    },
  ], {
    initialEntries: ["/compose/finish?workspace=demo"],
  });

  return render(
    <RouterProvider router={router} />,
  );
}

async function waitForSettings() {
  await screen.findByRole("heading", { name: "Step 3: Settings" });
}

describe("WizardStep3 retrospective opt-in", () => {
  beforeEach(() => {
    document.body.style.cssText = "pointer-events: auto;";
    sessionStorage.clear();
    savedDraft = null;
    saveDraftCalls.length = 0;
    mockGet.mockClear();
    mockPreview.mockClear();
    mockSaveDraft.mockClear();
    mockSaveGoal.mockClear();
  });

  afterEach(() => {
    cleanup();
    sessionStorage.clear();
  });

  it("renders the retrospective switch off by default with accessible copy", async () => {
    renderStep3();

    await waitForSettings();

    const retrospectiveSwitch = screen.getByRole("switch", {
      name: "Run a retrospective after completion",
    });

    expect(retrospectiveSwitch.getAttribute("aria-checked")).toBe("false");
    expect(screen.getByText("Capture lessons and factory improvements after the build finishes. Optional and off by default.")).not.toBeNull();
  });

  it("persists the retrospective switch in session storage across remounts", async () => {
    const firstRender = renderStep3();
    await waitForSettings();

    await userEvent.click(screen.getByRole("switch", { name: "Run a retrospective after completion" }));

    await waitFor(() => {
      expect(JSON.parse(sessionStorage.getItem("compose-wizard-step-3") ?? "{}")).toMatchObject({
        retrospective: true,
      });
    });

    firstRender.unmount();
    renderStep3();
    await waitForSettings();

    expect(screen.getByRole("switch", { name: "Run a retrospective after completion" }).getAttribute("aria-checked")).toBe("true");
  });

  it("refreshes preview and draft payload from the retrospective switch state", async () => {
    renderStep3();
    await waitForSettings();

    const previewPanel = screen.getByText("GOAL.md Preview").closest("div")?.parentElement;
    expect(previewPanel).not.toBeNull();
    expect(within(previewPanel as HTMLElement).queryByText(/retrospective: true/)).toBeNull();

    await userEvent.click(screen.getByRole("switch", { name: "Run a retrospective after completion" }));

    await waitFor(() => {
      const latestDraft = saveDraftCalls.at(-1);
      expect(latestDraft?.state.retrospective).toBe(true);
      expect(latestDraft?.wizard.retrospective).toBe(true);
    });

    await waitFor(() => {
      expect(screen.getByText(/retrospective: true/)).not.toBeNull();
    });

    await userEvent.click(screen.getByRole("switch", { name: "Run a retrospective after completion" }));

    await waitFor(() => {
      const latestDraft = saveDraftCalls.at(-1);
      expect(latestDraft?.state.retrospective).toBe(false);
      expect(latestDraft?.wizard.retrospective).toBe(false);
    });

    await waitFor(() => {
      expect(screen.queryByText(/retrospective: true/)).toBeNull();
    });
  });

  it("keeps wizard pages stable when users leave the missing-workspace state", async () => {
    await expectMissingWorkspaceNavigation(
      "/compose/step/1",
      <WizardStep1 key="wizard-step-1" />,
    );
    await expectMissingWorkspaceNavigation(
      "/compose/step/2",
      <WizardStep2 key="wizard-step-2" />,
    );
    await expectMissingWorkspaceNavigation(
      "/compose/step/3",
      <WizardStep3 key="wizard-step-3" />,
    );
    await expectMissingWorkspaceNavigation(
      "/compose/finish",
      <WizardFinish key="wizard-finish" />,
    );
  });

  it("saves the final GOAL.md with the current retrospective opt-in state", async () => {
    savedDraft = {
      state: {
        description: "Build the thing",
        completionGate: "make test",
        retrospective: true,
        agents: [],
        model: "openai/gpt-5.5 (xhigh)",
        tasks: "",
      },
      wizard: {
        currentStep: 3,
        description: "Build the thing",
        techStack: [],
        completionGate: "make test",
        retrospective: true,
      },
    };

    renderFinish();

    await screen.findByRole("heading", { name: "Review & Save" });
    expect(screen.getByText("Enabled")).not.toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Save GOAL.md" }));

    await waitFor(() => {
      expect(mockSaveGoal).toHaveBeenCalledWith("demo", "etag-1");
    });
    const latestDraft = saveDraftCalls.at(-1);
    expect(latestDraft?.state.retrospective).toBe(true);
    expect(latestDraft?.wizard.retrospective).toBe(true);
  });
});
