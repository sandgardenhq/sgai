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
