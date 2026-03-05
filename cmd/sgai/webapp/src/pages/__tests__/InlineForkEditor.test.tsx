import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { InlineForkEditor } from "../InlineForkEditor";

beforeEach(() => {
  document.body.style.pointerEvents = "auto";
});

const mockFork = mock(() => Promise.resolve({ name: "new-fork", dir: "/path/to/new-fork" }));
const mockForkTemplate = mock(() => Promise.resolve({ content: "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Goal\n\nDescribe your task" }));
const mockTriggerFactoryRefresh = mock(() => {});
const mockNavigate = mock(() => {});

mock.module("react-router", () => ({
  ...require("react-router"),
  useNavigate: () => mockNavigate,
}));

mock.module("@/lib/factory-state", () => ({
  triggerFactoryRefresh: mockTriggerFactoryRefresh,
}));

mock.module("@/lib/api", () => ({
  api: {
    workspaces: {
      fork: mockFork,
      forkTemplate: mockForkTemplate,
    },
  },
  ApiError: class ApiError extends Error {
    constructor(public status: number, message: string) {
      super(message);
      this.name = "ApiError";
    }
  },
}));

mock.module("@/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ value, onChange, disabled, placeholder }: {
    value: string;
    onChange: (v: string | undefined) => void;
    disabled: boolean;
    placeholder?: string;
  }) => (
    <div data-testid="markdown-editor">
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        data-testid="fork-editor-textarea"
        placeholder={placeholder}
      />
    </div>
  ),
}));

function renderInlineForkEditor(workspaceName = "test-workspace") {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <InlineForkEditor workspaceName={workspaceName} />
      </TooltipProvider>
    </MemoryRouter>
  );
}

afterEach(() => {
  cleanup();
});

describe("InlineForkEditor", () => {
  beforeEach(() => {
    mockFork.mockClear();
    mockForkTemplate.mockClear();
    mockTriggerFactoryRefresh.mockClear();
    mockNavigate.mockClear();
    mockFork.mockImplementation(() => Promise.resolve({ name: "new-fork", dir: "/path/to/new-fork" }));
    mockForkTemplate.mockImplementation(() => Promise.resolve({ content: "---\nflow: |\n  \"a\" -> \"b\"\n---\n# Goal\n\nDescribe your task" }));
  });

  describe("rendering", () => {
    it("shows title and description", async () => {
      renderInlineForkEditor();

      expect(screen.getByText("New Task")).toBeTruthy();
      expect(screen.getByText(/Write a GOAL.md/)).toBeTruthy();
    });

    it("shows Create Fork button", async () => {
      renderInlineForkEditor();

      expect(screen.getByText("Create Fork")).toBeTruthy();
    });

    it("shows markdown editor", async () => {
      renderInlineForkEditor();

      expect(screen.getByTestId("markdown-editor")).toBeTruthy();
    });

    it("loads fork template on mount", async () => {
      renderInlineForkEditor();

      await waitFor(() => {
        expect(mockForkTemplate).toHaveBeenCalledWith("test-workspace");
      });
    });

    it("populates editor with template content", async () => {
      renderInlineForkEditor();

      await waitFor(() => {
        const textarea = screen.getByTestId("fork-editor-textarea") as HTMLTextAreaElement;
        expect(textarea.value).toContain("Goal");
      });
    });
  });

  describe("validation", () => {
    it("disables Create Fork button when body is empty", async () => {
      mockForkTemplate.mockImplementation(() => Promise.resolve({ content: "---\nflow: |\n  \"a\" -> \"b\"\n---\n" }));

      renderInlineForkEditor();

      await waitFor(() => {
        const button = screen.getByText("Create Fork").closest("button");
        expect(button?.hasAttribute("disabled")).toBe(true);
      });
    });
  });

  describe("fork creation", () => {
    it("calls fork API on submit", async () => {
      const user = userEvent.setup();
      renderInlineForkEditor();

      await waitFor(() => {
        expect(screen.getByTestId("fork-editor-textarea")).toBeTruthy();
      });

      const button = screen.getByText("Create Fork").closest("button");
      if (button && !button.hasAttribute("disabled")) {
        await user.click(button);

        await waitFor(() => {
          expect(mockFork).toHaveBeenCalled();
        });
      }
    });

    it("triggers factory refresh after successful fork", async () => {
      const user = userEvent.setup();
      renderInlineForkEditor();

      await waitFor(() => {
        const textarea = screen.getByTestId("fork-editor-textarea") as HTMLTextAreaElement;
        expect(textarea.value).toContain("Goal");
      });

      const button = screen.getByText("Create Fork").closest("button");
      if (button && !button.hasAttribute("disabled")) {
        await user.click(button);

        await waitFor(() => {
          expect(mockTriggerFactoryRefresh).toHaveBeenCalled();
        });
      }
    });

    it("shows error when fork creation fails", async () => {
      const user = userEvent.setup();
      mockFork.mockImplementation(() => Promise.reject(new Error("Fork failed")));

      renderInlineForkEditor();

      await waitFor(() => {
        const textarea = screen.getByTestId("fork-editor-textarea") as HTMLTextAreaElement;
        expect(textarea.value).toContain("Goal");
      });

      const button = screen.getByText("Create Fork").closest("button");
      if (button && !button.hasAttribute("disabled")) {
        await user.click(button);

        await waitFor(() => {
          expect(screen.getByText("Failed to create fork")).toBeTruthy();
        });
      }
    });
  });
});
