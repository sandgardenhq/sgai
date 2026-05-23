import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/components/ui/sidebar";
import { WizardStep3 } from "../WizardStep3";

function defaultComposeGetResponse() {
  return {
  workspace: "test-workspace",
  state: {
    description: "Build safer controls",
    completionGate: "make test",
    agents: [],
    flow: "",
    tasks: "",
  },
  wizard: {
    currentStep: 3,
    description: "Build safer controls",
    techStack: ["react"],
    safetyAnalysis: false,
    completionGate: "make test",
  },
  techStackItems: [
    { id: "react", name: "React", selected: true },
  ],
  };
}

let composeGetResponse = defaultComposeGetResponse();

let composePreviewResponse = {
  content: "# Preview",
  etag: "test-etag",
};

const mockComposeGet = mock(() => Promise.resolve(composeGetResponse));

const mockComposePreview = mock(() => Promise.resolve(
  mockComposeSaveDraft.mock.calls.length > 0
    ? { content: "# Preview", etag: "test-etag" }
    : composePreviewResponse,
));

const mockComposeSaveDraft = mock(() => Promise.resolve({ saved: true }));

mock.module("@/lib/api", () => ({
  api: {
    compose: {
      get: mockComposeGet,
      preview: mockComposePreview,
      saveDraft: mockComposeSaveDraft,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

function renderWizardStep3() {
  return render(
    <MemoryRouter initialEntries={["/compose/step/3?workspace=test-workspace"]}>
      <TooltipProvider>
        <SidebarProvider>
          <Routes>
            <Route path="/compose/step/3" element={<WizardStep3 />} />
          </Routes>
        </SidebarProvider>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

async function latestSavedDraftRequest() {
  await waitFor(() => {
    expect(mockComposeSaveDraft.mock.calls.length).toBeGreaterThan(0);
  });
  const latestCall = mockComposeSaveDraft.mock.calls.at(-1);
  expect(latestCall).toBeDefined();
  return latestCall?.[1];
}

function agentModelsByName(agents: Array<{ name: string; model: string }>) {
  return Object.fromEntries(agents.map((agent) => [agent.name, agent.model]));
}

describe("WizardStep3", () => {
  beforeEach(() => {
    sessionStorage.clear();
    composeGetResponse = defaultComposeGetResponse();
    composePreviewResponse = {
      content: "# Preview",
      etag: "test-etag",
    };
    mockComposeGet.mockClear();
    mockComposePreview.mockClear();
    mockComposeSaveDraft.mockClear();
  });

  afterEach(() => {
    cleanup();
    sessionStorage.clear();
  });

  it("describes safety analysis as coordinator STPA skill behavior", async () => {
    renderWizardStep3();

    expect(await screen.findByText("Enable Safety Analysis")).toBeTruthy();
    expect(screen.getByText(/coordinator loads the STPA skill/i)).toBeTruthy();
    expect(screen.queryByText(/STPA analyst/i)).toBeNull();
  });

  it("creates React stack agents with xhigh coordinator and low React worker model defaults", async () => {
    renderWizardStep3();

    const draftRequest = await latestSavedDraftRequest();
    const modelByAgent = agentModelsByName(draftRequest?.state.agents ?? []);

    expect(modelByAgent).toEqual({
      coordinator: "openai/gpt-5.5 (xhigh)",
      react: "openai/gpt-5.5 (low)",
    });
  });

  it("creates multi-agent stack agents with xhigh coordinator and low worker model defaults", async () => {
    composeGetResponse = {
      ...defaultComposeGetResponse(),
      wizard: {
        currentStep: 3,
        description: "Build SDK verifier workflow",
        techStack: ["claudesdk", "openaisdk", "shell"],
        safetyAnalysis: false,
        completionGate: "make test",
      },
      techStackItems: [
        { id: "claudesdk", name: "Claude SDK", selected: true },
        { id: "openaisdk", name: "OpenAI SDK", selected: true },
        { id: "shell", name: "Shell", selected: true },
      ],
    };

    renderWizardStep3();

    const draftRequest = await latestSavedDraftRequest();
    const modelByAgent = agentModelsByName(draftRequest?.state.agents ?? []);

    expect(modelByAgent.coordinator).toBe("openai/gpt-5.5 (xhigh)");
    for (const [agentName, model] of Object.entries(modelByAgent)) {
      if (agentName !== "coordinator") {
        expect(model).toBe("openai/gpt-5.5 (low)");
      }
    }
    expect(Object.keys(modelByAgent).sort()).toEqual([
      "agent-sdk-verifier-py",
      "agent-sdk-verifier-ts",
      "coordinator",
      "general-purpose",
      "openai-sdk-verifier-py",
      "openai-sdk-verifier-ts",
      "shell-script",
    ]);
  });

  it("preserves custom TypeScript-only server state while sanitizing retired agents", async () => {
    composeGetResponse = {
      ...defaultComposeGetResponse(),
      state: {
        description: "Preserve custom TypeScript workflow",
        completionGate: "make test",
        agents: [
          { name: "coordinator", selected: true, model: "custom/coordinator-model" },
          { name: "custom-reviewer", selected: true, model: "custom/reviewer-model" },
          { name: "stpa-analyst", selected: true, model: "custom/stpa-model" },
        ],
        flow: '"coordinator" -> "custom-reviewer"\n"coordinator" -> "stpa-analyst"',
        tasks: "keep this task plan",
      },
      wizard: {
        currentStep: 3,
        description: "Preserve custom TypeScript workflow",
        techStack: ["typescript"],
        safetyAnalysis: false,
        completionGate: "make test",
      },
      techStackItems: [
        { id: "typescript", name: "TypeScript", selected: true },
      ],
    };

    renderWizardStep3();

    const draftRequest = await latestSavedDraftRequest();

    expect(draftRequest?.state.agents).toEqual([
      { name: "coordinator", selected: true, model: "custom/coordinator-model" },
      { name: "custom-reviewer", selected: true, model: "custom/reviewer-model" },
    ]);
    expect(draftRequest?.state.flow).toBe('"coordinator" -> "custom-reviewer"');
    expect(draftRequest?.state.tasks).toBe("keep this task plan");
    expect(JSON.stringify(draftRequest)).not.toContain("stpa-analyst");
  });

  it("merges preserved server agents with generated React stack agents while keeping explicit models", async () => {
    composeGetResponse = {
      ...defaultComposeGetResponse(),
      state: {
        description: "Preserve custom React workflow",
        completionGate: "make test",
        agents: [
          { name: "coordinator", selected: true, model: "custom/coordinator-model" },
          { name: "custom-reviewer", selected: true, model: "custom/reviewer-model" },
          { name: "stpa-analyst", selected: true, model: "custom/stpa-model" },
        ],
        flow: '"coordinator" -> "custom-reviewer"\n"coordinator" -> "stpa-analyst"',
        tasks: "keep generated stack task plan",
      },
      wizard: {
        currentStep: 3,
        description: "Preserve custom React workflow",
        techStack: ["react"],
        safetyAnalysis: false,
        completionGate: "make test",
      },
      techStackItems: [
        { id: "react", name: "React", selected: true },
      ],
    };

    renderWizardStep3();

    const draftRequest = await latestSavedDraftRequest();

    expect(draftRequest?.state.agents).toEqual([
      { name: "coordinator", selected: true, model: "custom/coordinator-model" },
      { name: "custom-reviewer", selected: true, model: "custom/reviewer-model" },
      { name: "react", selected: true, model: "openai/gpt-5.5 (low)" },
    ]);
    expect(draftRequest?.state.flow).toBe('"coordinator" -> "custom-reviewer"\n"react"');
    expect(draftRequest?.state.tasks).toBe("keep generated stack task plan");
    expect(JSON.stringify(draftRequest)).not.toContain("stpa-analyst");
  });

  it("does not add stpa-analyst to draft agents or flow when safety analysis is enabled", async () => {
    const user = userEvent.setup();
    renderWizardStep3();

    const toggle = await screen.findByRole("switch", { name: /enable safety analysis/i });
    let disabledSafetyDraftRequest:
      | { state: { agents: unknown; flow: unknown }; wizard: { safetyAnalysis: boolean } }
      | undefined;
    await waitFor(() => {
      expect(mockComposeSaveDraft.mock.calls.length).toBeGreaterThan(0);
      const latestCall = mockComposeSaveDraft.mock.calls.at(-1);
      expect(latestCall).toBeDefined();
      disabledSafetyDraftRequest = latestCall?.[1];
      expect(disabledSafetyDraftRequest?.wizard.safetyAnalysis).toBe(false);
    });

    await user.click(toggle);

    await waitFor(() => {
      expect(mockComposeSaveDraft.mock.calls.length).toBeGreaterThan(0);
      const latestCall = mockComposeSaveDraft.mock.calls.at(-1);
      expect(latestCall).toBeDefined();
      const draftRequest = latestCall?.[1];
      expect(JSON.stringify(draftRequest)).not.toContain("stpa-analyst");
      expect(draftRequest?.state.agents).toEqual(disabledSafetyDraftRequest?.state.agents);
      expect(draftRequest?.state.flow).toBe(disabledSafetyDraftRequest?.state.flow);
      expect(draftRequest?.wizard.safetyAnalysis).toBe(true);
    });
  });

  it("removes retired stpa-analyst from preserved server draft state", async () => {
    composePreviewResponse = {
      content: "stale stpa-analyst preview",
      etag: "stale-etag",
    };
    composeGetResponse = {
      ...defaultComposeGetResponse(),
      state: {
        description: "Build safer controls",
        completionGate: "make test",
        agents: [
          { name: "coordinator", selected: true, model: "openai/gpt-5.5" },
          { name: "stpa-analyst", selected: true, model: "openai/gpt-5.5" },
        ],
        flow: 'coordinator -> stpa-analyst',
        tasks: "",
      },
      wizard: {
        currentStep: 3,
        description: "Build safer controls",
        techStack: ["typescript"],
        safetyAnalysis: false,
        completionGate: "make test",
      },
      techStackItems: [
        { id: "typescript", name: "TypeScript", selected: true },
      ],
    };

    const user = userEvent.setup();
    renderWizardStep3();

    const toggle = await screen.findByRole("switch", { name: /enable safety analysis/i });
    await waitFor(() => {
      expect(screen.queryByText(/stale stpa-analyst preview/i)).toBeNull();
    });
    await user.click(toggle);

    await waitFor(() => {
      expect(mockComposeSaveDraft.mock.calls.length).toBeGreaterThan(0);
      const latestCall = mockComposeSaveDraft.mock.calls.at(-1);
      expect(latestCall).toBeDefined();
      const draftRequest = latestCall?.[1];
      expect(JSON.stringify(draftRequest)).not.toContain("stpa-analyst");
      expect(draftRequest?.state.agents.map((agent) => agent.name)).toEqual(["coordinator"]);
      expect(draftRequest?.state.flow).toBe("");
      expect(draftRequest?.wizard.safetyAnalysis).toBe(true);
    });
  });
});
