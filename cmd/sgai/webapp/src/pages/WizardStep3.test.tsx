import { describe, test, expect, afterEach, beforeEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WizardStep3 } from "./WizardStep3";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockComposeState = {
  workspace: "test-ws",
  state: { description: "", interactive: "yes", completionGate: "", agents: [], flow: "", tasks: "" },
  wizard: { currentStep: 3, techStack: [], safetyAnalysis: false, interactive: "yes" },
  techStackItems: [],
};

const mockPreview = { content: "# GOAL.md preview", etag: '"abc123"' };

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
    <MemoryRouter initialEntries={["/compose/step/3?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/step/3" element={<WizardStep3 />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WizardStep3", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  beforeEach(() => { sessionStorage.clear(); });
  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); sessionStorage.clear(); });

  test("renders step heading", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(screen.getByText("Step 3: Safety Analysis")).toBeTruthy();
  });

  test("renders safety toggle switch", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    const toggle = screen.getByRole("switch");
    expect(toggle).toBeTruthy();
    expect(toggle.getAttribute("aria-checked")).toBe("false");
  });

  test("toggle enables safety analysis", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const toggle = screen.getByRole("switch");
    await act(async () => { fireEvent.click(toggle); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(toggle.getAttribute("aria-checked")).toBe("true");
  });

  test("shows STPA info when enabled", async () => {
    const stateWithSafety = {
      ...mockComposeState,
      wizard: { ...mockComposeState.wizard, safetyAnalysis: true },
    };
    fetchSpy = mockFetchSequence([stateWithSafety, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("What STPA provides:")).toBeTruthy();
    expect(screen.getByText("Loss identification")).toBeTruthy();
  });

  test("does not show STPA info when disabled", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.queryByText("What STPA provides:")).toBeNull();
  });

  test("persists safety analysis to sessionStorage", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const toggle = screen.getByRole("switch");
    await act(async () => { fireEvent.click(toggle); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const stored = sessionStorage.getItem("compose-wizard-step-3");
    expect(stored).toBeTruthy();
    const parsed = JSON.parse(stored!);
    expect(parsed.safetyAnalysis).toBe(true);
  });

  test("renders navigation buttons", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(screen.getByText("Back")).toBeTruthy();
    expect(screen.getByText("Next")).toBeTruthy();
  });
});
