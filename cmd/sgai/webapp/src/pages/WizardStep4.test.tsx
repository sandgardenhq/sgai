import { describe, test, expect, afterEach, beforeEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WizardStep4 } from "./WizardStep4";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockComposeState = {
  workspace: "test-ws",
  state: { description: "", interactive: "yes", completionGate: "make test", agents: [], flow: "", tasks: "" },
  wizard: { currentStep: 4, techStack: [], safetyAnalysis: false, interactive: "yes", completionGate: "make test" },
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
    <MemoryRouter initialEntries={["/compose/step/4?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/step/4" element={<WizardStep4 />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WizardStep4", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  beforeEach(() => { sessionStorage.clear(); });
  afterEach(() => { cleanup(); fetchSpy?.mockRestore(); sessionStorage.clear(); });

  test("renders step heading", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });
    expect(screen.getByText("Step 4: Settings")).toBeTruthy();
  });

  test("renders interactive mode select", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const select = screen.getByLabelText("Interactive Mode");
    expect(select).toBeTruthy();
    expect((select as HTMLSelectElement).value).toBe("yes");
  });

  test("renders completion gate input with server value", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const input = screen.getByLabelText("Completion Gate Script");
    expect(input).toBeTruthy();
    expect((input as HTMLInputElement).value).toBe("make test");
  });

  test("changes interactive mode", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const select = screen.getByLabelText("Interactive Mode");
    await act(async () => {
      fireEvent.change(select, { target: { value: "auto" } });
    });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect((select as HTMLSelectElement).value).toBe("auto");
  });

  test("changes completion gate", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const input = screen.getByLabelText("Completion Gate Script");
    await act(async () => {
      fireEvent.change(input, { target: { value: "npm run test" } });
    });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect((input as HTMLInputElement).value).toBe("npm run test");
  });

  test("persists settings to sessionStorage", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const select = screen.getByLabelText("Interactive Mode");
    await act(async () => {
      fireEvent.change(select, { target: { value: "no" } });
    });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    const stored = sessionStorage.getItem("compose-wizard-step-4");
    expect(stored).toBeTruthy();
    const parsed = JSON.parse(stored!);
    expect(parsed.interactive).toBe("no");
  });

  test("renders navigation with Review & Save button", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);
    await act(async () => { renderWithRouter(); });
    await act(async () => { await new Promise((r) => setTimeout(r, 50)); });

    expect(screen.getByText("Back")).toBeTruthy();
    expect(screen.getByText("Review & Save")).toBeTruthy();
  });
});
