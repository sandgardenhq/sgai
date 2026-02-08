import { describe, test, expect, afterEach, beforeEach, spyOn } from "bun:test";
import { render, screen, act, cleanup, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { WizardStep2 } from "./WizardStep2";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockComposeState = {
  workspace: "test-ws",
  state: {
    description: "",
    interactive: "yes",
    completionGate: "",
    agents: [],
    flow: "",
    tasks: "",
  },
  wizard: {
    currentStep: 2,
    techStack: ["go"],
    safetyAnalysis: false,
    interactive: "yes",
  },
  techStackItems: [
    { id: "go", name: "Go", selected: true },
    { id: "react", name: "React", selected: false },
    { id: "python", name: "Python", selected: false },
  ],
};

const mockPreview = {
  content: "# GOAL.md preview",
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
    <MemoryRouter initialEntries={["/compose/step/2?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/step/2" element={<WizardStep2 />} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WizardStep2", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  beforeEach(() => {
    sessionStorage.clear();
  });

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
    sessionStorage.clear();
  });

  test("renders step heading after load", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("Step 2: Tech Stack")).toBeTruthy();
  });

  test("renders tech stack items as buttons", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("Go")).toBeTruthy();
    expect(screen.getByText("React")).toBeTruthy();
    expect(screen.getByText("Python")).toBeTruthy();
  });

  test("shows pre-selected tech stack from server state", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const goButton = screen.getByText("Go").closest("button");
    expect(goButton?.getAttribute("aria-pressed")).toBe("true");

    const reactButton = screen.getByText("React").closest("button");
    expect(reactButton?.getAttribute("aria-pressed")).toBe("false");
  });

  test("toggles tech stack on click", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const reactButton = screen.getByText("React").closest("button")!;
    await act(async () => {
      fireEvent.click(reactButton);
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(reactButton.getAttribute("aria-pressed")).toBe("true");
  });

  test("persists tech stack to sessionStorage", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview, { saved: true }, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const reactButton = screen.getByText("React").closest("button")!;
    await act(async () => {
      fireEvent.click(reactButton);
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    const stored = sessionStorage.getItem("compose-wizard-step-2");
    expect(stored).toBeTruthy();
    const parsed = JSON.parse(stored!);
    expect(parsed.techStack).toContain("react");
    expect(parsed.techStack).toContain("go");
  });

  test("renders navigation buttons", async () => {
    fetchSpy = mockFetchSequence([mockComposeState, mockPreview]);

    await act(async () => {
      renderWithRouter();
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(screen.getByText("Back")).toBeTruthy();
    expect(screen.getByText("Next")).toBeTruthy();
  });
});
