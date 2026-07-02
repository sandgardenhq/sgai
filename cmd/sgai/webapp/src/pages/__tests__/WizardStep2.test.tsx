import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider } from "react-router/dom";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiComposeDraftRequest } from "@/types";

let savedDraft: ApiComposeDraftRequest | null = null;
const saveDraftCalls: ApiComposeDraftRequest[] = [];
let resolvePreview: ((value: { content: string; etag: string }) => void) | null = null;

const defaultDraft: ApiComposeDraftRequest = {
  state: {
    description: "Build a thing",
    completionGate: "make test",
    retrospective: false,
    agents: [],
    model: "openai/gpt-5.5 (xhigh)",
    tasks: "",
  },
  wizard: {
    currentStep: 2,
    description: "Build a thing",
    techStack: [],
    completionGate: "make test",
    retrospective: false,
  },
};

const mockGet = mock(() => Promise.resolve({
  workspace: "demo",
  state: savedDraft?.state ?? defaultDraft.state,
  wizard: savedDraft?.wizard ?? defaultDraft.wizard,
  techStackItems: [
    { id: "react", name: "React" },
    { id: "go", name: "Go" },
  ],
}));

const mockPreview = mock(() => new Promise<{ content: string; etag: string }>((resolve) => {
  resolvePreview = resolve;
}));

const mockSaveDraft = mock((_workspace: string, draft: ApiComposeDraftRequest) => {
  savedDraft = draft;
  saveDraftCalls.push(draft);
  return Promise.resolve({ saved: true });
});

mock.module("@/lib/api", () => ({
  api: {
    compose: {
      get: mockGet,
      preview: mockPreview,
      saveDraft: mockSaveDraft,
      save: mock(() => Promise.resolve({ saved: true, workspace: "demo" })),
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

const { createMemoryRouter } = await import("react-router");
const { WizardStep2 } = await import("../WizardStep2");

function renderStep2() {
  const router = createMemoryRouter([
    {
      path: "/compose/step/2",
      element: (
        <TooltipProvider>
          <WizardStep2 />
        </TooltipProvider>
      ),
    },
  ], {
    initialEntries: ["/compose/step/2?workspace=demo"],
  });

  return render(<RouterProvider router={router} />);
}

function previewPanel() {
  const heading = screen.getByText("GOAL.md Preview");
  const panel = heading.closest('[data-slot="card"]');
  expect(panel).not.toBeNull();
  return panel as HTMLElement;
}

async function resolveInitialPreview() {
  await waitFor(() => expect(resolvePreview).not.toBeNull());
  resolvePreview?.({ content: "---\nmodel: \"openai/gpt-5.5 (xhigh)\"\n---\n\n# Build a thing", etag: "etag-1" });
}

describe("WizardStep2 tech-stack agent list", () => {
  beforeEach(() => {
    sessionStorage.clear();
    savedDraft = null;
    saveDraftCalls.length = 0;
    resolvePreview = null;
    mockGet.mockClear();
    mockPreview.mockClear();
    mockSaveDraft.mockClear();
  });

  afterEach(() => {
    cleanup();
    sessionStorage.clear();
  });

  it("updates the visible agent list immediately when tech selections change before preview sync resolves", async () => {
    const user = userEvent.setup();
    renderStep2();

    await resolveInitialPreview();
    await screen.findByRole("heading", { name: "Step 2: Tech Stack" });

    await user.click(screen.getByRole("button", { name: "React" }));

    expect(previewPanel().textContent).toContain('agents:\n  - "react"');
  });

  it("persists the same derived agents after server sync", async () => {
    const user = userEvent.setup();
    renderStep2();

    await resolveInitialPreview();
    await screen.findByRole("heading", { name: "Step 2: Tech Stack" });

    await user.click(screen.getByRole("button", { name: "React" }));

    await waitFor(() => {
      const latestDraft = saveDraftCalls.at(-1);
      expect(latestDraft?.wizard.techStack).toEqual(["react"]);
      expect(latestDraft?.state.agents).toEqual([{ name: "react", selected: true }]);
    });
  });
});
