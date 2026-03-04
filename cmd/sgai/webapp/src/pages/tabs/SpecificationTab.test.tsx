import { describe, it, expect, afterEach, mock } from "bun:test";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { SpecificationTab } from "./SpecificationTab";
import { makeWorkspace } from "@/test-utils";

afterEach(() => {
  cleanup();
});

function renderSpecificationTab(workspaces: ApiWorkspaceEntry[] = [], fetchStatus = "idle") {
  mock.module("@/lib/factory-state", () => ({
    useFactoryState: () => ({ workspaces, fetchStatus, lastFetchedAt: Date.now() }),
    resetFactoryStateStore: () => {},
  }));
  return render(
    <MemoryRouter>
      <SpecificationTab workspaceName="test-project" />
    </MemoryRouter>,
  );
}

describe("SpecificationTab", () => {
  it("renders loading skeleton when fetching with no workspace", () => {
    renderSpecificationTab([], "fetching");
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders null when workspace not found and idle", () => {
    const { container } = renderSpecificationTab([], "idle");
    expect(container.firstChild).toBeNull();
  });

  it("renders GOAL.md content from factory state", () => {
    const workspace = makeWorkspace({
      goalContent: "# Test Goal\n\nBuild amazing things",
      hasProjectMgmt: true,
      pmContent: "## Project Status\n\nIn progress",
    });
    renderSpecificationTab([workspace]);

    expect(screen.getByText("GOAL.md")).toBeDefined();
    expect(screen.getByRole("heading", { name: "Test Goal" })).toBeDefined();
  });

  it("renders PROJECT_MANAGEMENT.md when available", () => {
    const workspace = makeWorkspace({
      goalContent: "# Test Goal\n\nBuild amazing things",
      hasProjectMgmt: true,
      pmContent: "## Project Status\n\nIn progress",
    });
    const { container } = renderSpecificationTab([workspace]);
    const view = within(container);

    expect(view.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
    expect(view.getAllByText("Project Status").length).toBeGreaterThan(0);
    expect(view.getAllByText("In progress").length).toBeGreaterThan(0);
  });

  it("renders empty state when no GOAL.md", () => {
    const workspace = makeWorkspace({ goalContent: "", hasProjectMgmt: false, pmContent: "" });
    renderSpecificationTab([workspace]);

    expect(screen.getByText("No GOAL.md file found")).toBeDefined();
  });

  it("renders error state when fetchStatus is error and no workspace", () => {
    renderSpecificationTab([], "error");
    expect(screen.getByText(/Failed to load specification/i)).toBeDefined();
  });

  it("does not call individual workspace API endpoint", () => {
    const mockFetch = mock(() => Promise.resolve(new Response("{}")));
    globalThis.fetch = mockFetch as unknown as typeof fetch;

    const workspace = makeWorkspace({ goalContent: "# Goal" });
    renderSpecificationTab([workspace]);

    const calledUrls = mockFetch.mock.calls.map((call) => String(call[0]));
    expect(calledUrls.some((url) => url.includes("/api/v1/workspaces/test-project") && !url.includes("/api/v1/state"))).toBe(false);
  });
});
