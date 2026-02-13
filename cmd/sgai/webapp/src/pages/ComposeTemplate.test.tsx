import { describe, test, expect, afterEach, spyOn } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { ComposeTemplateRedirect } from "./ComposeTemplate";
import { TooltipProvider } from "@/components/ui/tooltip";

const templatesResponse = {
  templates: [
    {
      id: "basic",
      name: "Basic",
      description: "desc",
      icon: "📦",
      agents: [],
      flow: "flow",
    },
  ],
};

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/compose/template/basic?workspace=test-ws"]}>
      <TooltipProvider>
        <Routes>
          <Route path="compose/template/:id" element={<ComposeTemplateRedirect />} />
          <Route path="compose/step/1" element={<div>Wizard Step 1</div>} />
        </Routes>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("ComposeTemplateRedirect", () => {
  let fetchSpy: ReturnType<typeof spyOn>;

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
  });

  test("applies template and navigates to step 1", async () => {
    let draftBody = "";
    fetchSpy = spyOn(globalThis, "fetch").mockImplementation((input, init) => {
      const url = String(input);
      if (url.includes("/api/v1/compose/templates")) {
        return Promise.resolve(
          new Response(JSON.stringify(templatesResponse), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.includes("/api/v1/compose/draft")) {
        draftBody = String(init?.body ?? "");
        return Promise.resolve(
          new Response(JSON.stringify({ saved: true }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(new Response("{}"));
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Wizard Step 1")).toBeTruthy();
    });

    expect(draftBody).toContain("\"fromTemplate\":\"basic\"");
    expect(draftBody).toContain("\"flow\":\"flow\"");
  });
});
