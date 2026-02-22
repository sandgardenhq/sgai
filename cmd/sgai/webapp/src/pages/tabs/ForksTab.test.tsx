import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ForksTab } from "./ForksTab";
import { resetDefaultSSEStore } from "@/lib/sse-store";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ApiForksResponse, ApiModelsResponse } from "@/types";

class MockEventSource {
  url: string;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  closed = false;
  constructor(url: string) { this.url = url; }
  addEventListener() {}
  removeEventListener() {}
  close() { this.closed = true; }
}

const originalEventSource = globalThis.EventSource;
const mockFetch = mock(() => Promise.resolve(new Response("{}")));

import { cleanup } from "@testing-library/react";

beforeEach(() => {
  cleanup();
  mockFetch.mockReset();
  globalThis.fetch = mockFetch as unknown as typeof fetch;
  (globalThis as unknown as Record<string, unknown>).EventSource = MockEventSource;
  resetDefaultSSEStore();
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
  resetDefaultSSEStore();
  (globalThis as unknown as Record<string, unknown>).EventSource = originalEventSource;
});

const forksResponse = {
  forks: [
    {
      name: "project-alpha-fork1",
      dir: "/home/user/project-alpha-fork1",
      running: true,
      needsInput: true,
      commitAhead: 3,
      commits: [
        { changeId: "abc12345", commitId: "def67890", timestamp: "2026-02-08 10:00", bookmarks: ["main"], description: "Initial commit" },
        { changeId: "xyz99999", commitId: "uvw11111", timestamp: "2026-02-08 11:00", bookmarks: [], description: "Add feature" },
      ],
    },
    {
      name: "project-alpha-fork2",
      dir: "/home/user/project-alpha-fork2",
      running: false,
      needsInput: false,
      commitAhead: 0,
      commits: [],
    },
  ],
} as ApiForksResponse;

const workspacesResponse = {
  workspaces: [
    {
      name: "test-project",
      dir: "/home/user/test-project",
      running: false,
      needsInput: false,
      inProgress: false,
      pinned: false,
      isRoot: true,
      status: "Stopped",
      hasSgai: true,
      forks: [
        {
          name: "project-alpha-fork1",
          dir: "/home/user/project-alpha-fork1",
          running: true,
          needsInput: true,
          inProgress: true,
          pinned: false,
          isRoot: false,
          status: "Needs Input",
          hasSgai: true,
        },
      ],
    },
  ],
};

const modelsResponse: ApiModelsResponse = {
  models: [
    { id: "anthropic/claude-sonnet-4-20250514", name: "Claude Sonnet" },
    { id: "anthropic/claude-opus-4-20250514", name: "Claude Opus" },
  ],
  defaultModel: "anthropic/claude-sonnet-4-20250514",
};

function mockForksAndWorkspaces() {
  return mockFetch.mockImplementation((input: string | URL | Request) => {
    const url = String(input);
    if (url.includes("/forks")) {
      return Promise.resolve(new Response(JSON.stringify(forksResponse)));
    }
    if (url.endsWith("/api/v1/workspaces")) {
      return Promise.resolve(new Response(JSON.stringify(workspacesResponse)));
    }
    if (url.includes("/api/v1/models")) {
      return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
    }
    if (url.includes("/adhoc")) {
      return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
    }
    return Promise.resolve(new Response("{}"));
  });
}

function renderForksTab() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <ForksTab workspaceName="test-project" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("ForksTab", () => {
  it("renders loading skeleton initially", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}));
    renderForksTab();
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders forks when data loads", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText("project-alpha-fork1")).toBeDefined();
    });

    expect(screen.getByText("project-alpha-fork2")).toBeDefined();
  });

  it("does not render fork status badges", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText("project-alpha-fork1")).toBeDefined();
    });

    expect(screen.queryByText("running")).toBeNull();
    expect(screen.queryByText("stopped")).toBeNull();
  });

  it("renders commit details for forks with commits", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText("Initial commit")).toBeDefined();
    });

    expect(screen.getByText("Add feature")).toBeDefined();
  });

  it("renders respond controls for forks", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText("project-alpha-fork1")).toBeDefined();
    });

    const respondButtons = screen.getAllByRole("button", { name: "Respond" });
    expect(respondButtons.length).toBeGreaterThan(1);
    expect(respondButtons.some((btn) => btn.hasAttribute("disabled"))).toBe(true);
  });

  it("renders empty state when no forks", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = String(input);
      if (url.includes("/forks")) {
        return Promise.resolve(new Response(JSON.stringify({ forks: [] })));
      }
      if (url.endsWith("/api/v1/workspaces")) {
        return Promise.resolve(new Response(JSON.stringify(workspacesResponse)));
      }
      if (url.includes("/api/v1/models")) {
        return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
      }
      if (url.includes("/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
      }
      return Promise.resolve(new Response("{}"));
    });
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText(/No forks yet/)).toBeDefined();
    });
  });

  it("renders error state", async () => {
    mockFetch.mockImplementation(() => Promise.reject(new Error("Network error")));
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText(/Failed to load forks/i)).toBeDefined();
    });
  });

  it("calls forks API on mount", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      const calledUrls = mockFetch.mock.calls.map((call) => (call[0] as string));
      expect(calledUrls.some((url) => url.includes("/api/v1/workspaces/test-project/forks"))).toBe(true);
      expect(calledUrls.some((url) => url.endsWith("/api/v1/workspaces"))).toBe(true);
      expect(calledUrls.some((url) => url.includes("/api/v1/models"))).toBe(true);
    });
  });

  it("renders run box with model selector after forks load", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByLabelText("Model")).toBeDefined();
    });

    expect(screen.getByText("Ad-hoc Prompt")).toBeDefined();
    expect(screen.getByLabelText("Prompt")).toBeDefined();
  });

  it("renders run box submit button", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
    });
  });

  it("renders run box below empty forks state", async () => {
    mockFetch.mockImplementation((input: string | URL | Request) => {
      const url = String(input);
      if (url.includes("/forks")) {
        return Promise.resolve(new Response(JSON.stringify({ forks: [] })));
      }
      if (url.endsWith("/api/v1/workspaces")) {
        return Promise.resolve(new Response(JSON.stringify(workspacesResponse)));
      }
      if (url.includes("/api/v1/models")) {
        return Promise.resolve(new Response(JSON.stringify(modelsResponse)));
      }
      if (url.includes("/adhoc")) {
        return Promise.resolve(new Response(JSON.stringify({ running: false, output: "", message: "" })));
      }
      return Promise.resolve(new Response("{}"));
    });
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText(/No forks yet/)).toBeDefined();
    });

    await waitFor(() => {
      expect(screen.getByText("Ad-hoc Prompt")).toBeDefined();
    });
  });

  it("populates model selector with available models", async () => {
    mockForksAndWorkspaces();
    renderForksTab();

    await waitFor(() => {
      expect(screen.getByText("Claude Sonnet")).toBeDefined();
    });

    expect(screen.getByText("Claude Opus")).toBeDefined();
  });
});
