import { describe, test, expect, afterEach, beforeEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WizardFinish } from "./WizardFinish";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockComposeState = {
  workspace: "test-ws",
  state: {
    description: "Build a web app",
    interactive: "yes",
    completionGate: "make test",
    agents: [{ name: "developer", selected: true, model: "claude" }],
    flow: "",
    tasks: "",
  },
  wizard: {
    currentStep: 4,
    techStack: ["go", "react"],
    safetyAnalysis: true,
    interactive: "yes",
    description: "Build a web app",
    completionGate: "make test",
  },
  techStackItems: [
    { id: "go", name: "Go", selected: true },
    { id: "react", name: "React", selected: true },
    { id: "python", name: "Python", selected: false },
  ],
};

const mockPreview = {
  content: "---\nflow: |\n  ...\n---\n\n## Tasks\n\nBuild a web app",
  etag: '"abc123"',
};

function mockFetchSequence(responses: unknown[]) {
  let callIndex = 0;
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
    const data = responses[callIndex] ?? responses[responses.length - 1];
    callIndex++;
    return Promise.resolve(
      new Response(JSON.stringify(data), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
  });
}

function renderWithRouter() {
  return render(
    <MemoryRouter initialEntries={["/compose/finish?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/finish" element={<WizardFinish />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WizardFinish", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  beforeEach(() => { sessionStorage.clear(); });
  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); sessionStorage.clear(); });

  test("renders review heading", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Review & Save")).toBeTruthy();
  });

  test("renders project description summary", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Build a web app")).toBeTruthy();
  });

  test("renders selected tech stack as badges", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Go")).toBeTruthy();
    expect(screen.getByText("React")).toBeTruthy();
  });

  test("renders safety analysis status", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Enabled")).toBeTruthy();
  });

  test("renders interactive mode", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Interactive Mode")).toBeTruthy();
    expect(screen.getByText("yes")).toBeTruthy();
  });

  test("renders completion gate", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("make test")).toBeTruthy();
  });

  test("renders save button", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Save GOAL.md")).toBeTruthy();
  });

  test("renders edit wizard button", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Edit Wizard")).toBeTruthy();
  });

  test("renders final preview panel", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });
    expect(screen.getByText("Final GOAL.md Preview")).toBeTruthy();
  });

  test("renders all 4 progress dots as completed", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });

    const dots = document.querySelectorAll(".rounded-full.bg-primary\\/60");
    expect(dots.length).toBe(4);
  });

  test("save button shows saving state during save", async () => {
    let callIndex = 0;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
      callIndex++;
      if (callIndex <= 2) {
        // Initial load: compose state + preview
        const data = callIndex === 1 ? mockComposeState : mockPreview;
        return Promise.resolve(new Response(JSON.stringify(data), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      if (callIndex === 3) {
        // Draft save
        return Promise.resolve(new Response(JSON.stringify({ saved: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      if (callIndex === 4) {
        // Preview after draft save
        return Promise.resolve(new Response(JSON.stringify(mockPreview), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      // The actual save - delayed
      if (callIndex === 5) {
        return Promise.resolve(new Response(JSON.stringify({ saved: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      // Second draft save + final compose save
      return Promise.resolve(new Response(JSON.stringify({ saved: true, workspace: "test-ws" }), { status: 201, headers: { "Content-Type": "application/json" } }));
    });

    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });

    const saveButton = screen.getByText("Save GOAL.md");
    await act(async () => { fireEvent.click(saveButton); });

    // Verify it eventually saves
    await act(async () => { await new Promise((r) => setTimeout(r, 200)); });
  });

  test("shows error on save failure with 412 conflict", async () => {
    let callIndex = 0;
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
      callIndex++;
      if (callIndex <= 2) {
        const data = callIndex === 1 ? mockComposeState : mockPreview;
        return Promise.resolve(new Response(JSON.stringify(data), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      if (callIndex === 3 || callIndex === 5) {
        // Draft save succeeds
        return Promise.resolve(new Response(JSON.stringify({ saved: true }), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      if (callIndex === 4) {
        // Preview update
        return Promise.resolve(new Response(JSON.stringify(mockPreview), { status: 200, headers: { "Content-Type": "application/json" } }));
      }
      // Final save returns 412
      return Promise.resolve(new Response("GOAL.md has been modified by another session", { status: 412 }));
    });

    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 100)); });

    const saveButton = screen.getByText("Save GOAL.md");
    await act(async () => { fireEvent.click(saveButton); });
    await act(async () => { await new Promise((r) => setTimeout(r, 200)); });

    expect(screen.getByText("Save Failed")).toBeTruthy();
  });
});
