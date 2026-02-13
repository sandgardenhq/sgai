import { describe, test, expect, afterEach, beforeEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WizardStep1 } from "./WizardStep1";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockComposeState = {
  workspace: "test-ws",
  state: {
    description: "Existing description",
    completionGate: "",
    agents: [],
    flow: "",
    tasks: "",
  },
  wizard: {
    currentStep: 1,
    techStack: [],
    safetyAnalysis: false,
    description: "Existing description",
  },
  techStackItems: [],
};

const mockPreview = {
  content: "# GOAL.md\n\nExisting description",
  etag: '"abc123"',
};

function mockFetchSequence(responses: unknown[]) {
  let callIndex = 0;
  return spyOn(globalThis, "fetch").mockImplementation((_input: string | URL | Request) => {
    const data = responses[callIndex] ?? responses[responses.length - 1];
    callIndex++;
    return Promise.resolve(
      new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
}

function renderWithRouter() {
  return render(
    <MemoryRouter initialEntries={["/compose/step/1?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/step/1" element={<WizardStep1 />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WizardStep1", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  beforeEach(() => {
    sessionStorage.clear();
  });

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
    sessionStorage.clear();
  });

  test("renders loading skeleton initially", async () => {
    fetchSpy = spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));

    await act(async () => {
      renderWithRouter();
    });

    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders step heading after load", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("Step 1: Project Description")).toBeTruthy();
  });

  test("renders textarea with server description", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const textarea = screen.getByRole("textbox");
    expect((textarea as HTMLTextAreaElement).value).toBe("Existing description");
  });

  test("shows progress dots", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const progressbar = document.querySelector("[role='progressbar']");
    expect(progressbar).toBeTruthy();
  });

  test("renders preview panel", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("GOAL.md Preview")).toBeTruthy();
  });

  test("renders navigation buttons", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("Templates")).toBeTruthy();
    expect(screen.getByText("Next")).toBeTruthy();
  });

  test("persists description to sessionStorage on change", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const textarea = screen.getByRole("textbox");
    await act(async () => {
      fireEvent.change(textarea, { target: { value: "New description" } });
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const stored = sessionStorage.getItem("compose-wizard-step-1");
    expect(stored).toBeTruthy();
    const parsed = JSON.parse(stored!);
    expect(parsed.description).toBe("New description");
  });

  test("shows missing workspace notice when workspace param is absent", async () => {
    await act(async () => {
      render(
        <MemoryRouter initialEntries={["/compose/step/1"]}>
          <TooltipProvider>
            <Routes>
              <Route path="compose/step/1" element={<WizardStep1 />} />
            </Routes>
          </TooltipProvider>
        </MemoryRouter>,
      );
    });

    expect(screen.getByText("Workspace required")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Back to Dashboard" })).toBeTruthy();
  });
});
