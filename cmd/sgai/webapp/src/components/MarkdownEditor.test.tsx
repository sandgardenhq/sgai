import { describe, test, expect, afterEach, mock, beforeEach } from "bun:test";
import React from "react";
import { render, screen, cleanup, fireEvent, act } from "@testing-library/react";
import { TooltipProvider } from "@/components/ui/tooltip";

const mockAddAction = mock(() => {});
const mockSetSelection = mock(() => {});
const mockGetFullModelRange = mock(() => ({
  startLineNumber: 1,
  startColumn: 1,
  endLineNumber: 5,
  endColumn: 10,
}));
const mockGetModel = mock(() => ({
  getFullModelRange: mockGetFullModelRange,
  getValue: mock(() => "test content"),
  getValueInRange: mock(() => ""),
  getLineContent: mock(() => ""),
}));
const mockDeltaDecorations = mock(() => []);
const mockOnDidChangeModelContent = mock(() => {});
const mockFocus = mock(() => {});
const mockGetSelection = mock(() => ({
  startLineNumber: 1,
  startColumn: 1,
  endLineNumber: 1,
  endColumn: 1,
}));
const mockExecuteEdits = mock(() => {});

let capturedOnMount: ((editor: unknown, monaco: unknown) => void) | null = null;

mock.module("@monaco-editor/react", () => ({
  default: (props: { onMount?: (editor: unknown, monaco: unknown) => void; value?: string; onChange?: (v: string | undefined) => void }) => {
    capturedOnMount = props.onMount ?? null;
    return React.createElement("div", { "data-testid": "mock-monaco-editor" }, props.value);
  },
}));

function createMockEditor() {
  return {
    addAction: mockAddAction,
    setSelection: mockSetSelection,
    getModel: mockGetModel,
    deltaDecorations: mockDeltaDecorations,
    onDidChangeModelContent: mockOnDidChangeModelContent,
    focus: mockFocus,
    getSelection: mockGetSelection,
    executeEdits: mockExecuteEdits,
  };
}

function createMockMonaco() {
  return {
    KeyMod: { CtrlCmd: 2048 },
    KeyCode: {
      KeyB: 32,
      KeyI: 39,
      KeyK: 41,
      KeyA: 31,
    },
  };
}

function renderEditor(props?: Partial<{ value: string; onChange: (v: string | undefined) => void; disabled: boolean }>) {
  const { MarkdownEditor } = require("./MarkdownEditor");
  return render(
    <TooltipProvider>
      <MarkdownEditor
        value={props?.value ?? "# Hello World"}
        onChange={props?.onChange ?? (() => {})}
        disabled={props?.disabled}
      />
    </TooltipProvider>,
  );
}

describe("MarkdownEditor", () => {
  beforeEach(() => {
    mockAddAction.mockClear();
    mockSetSelection.mockClear();
    mockGetFullModelRange.mockClear();
    mockGetModel.mockClear();
    capturedOnMount = null;
  });

  afterEach(cleanup);

  test("renders write/preview mode buttons", () => {
    renderEditor();
    expect(screen.getByText("Write")).toBeTruthy();
    expect(screen.getByText("Preview")).toBeTruthy();
  });

  test("renders toolbar actions in write mode", () => {
    renderEditor();
    expect(screen.getByLabelText("Bold")).toBeTruthy();
    expect(screen.getByLabelText("Italic")).toBeTruthy();
    expect(screen.getByLabelText("Link")).toBeTruthy();
  });

  test("registers select-all action on mount", () => {
    renderEditor();
    expect(capturedOnMount).toBeTruthy();

    const mockEditor = createMockEditor();
    const mockMonaco = createMockMonaco();

    capturedOnMount!(mockEditor, mockMonaco);

    const actionCalls = mockAddAction.mock.calls;
    const selectAllAction = actionCalls.find(
      (call: unknown[]) => (call[0] as { id: string }).id === "select-all",
    );
    expect(selectAllAction).toBeTruthy();

    const action = selectAllAction![0] as {
      id: string;
      label: string;
      keybindings: number[];
      run: (ed: unknown) => void;
    };
    expect(action.label).toBe("Select All");
    expect(action.keybindings).toEqual([2048 | 31]);
  });

  test("select-all action selects the full model range", () => {
    renderEditor();
    expect(capturedOnMount).toBeTruthy();

    const mockEditor = createMockEditor();
    const mockMonaco = createMockMonaco();

    capturedOnMount!(mockEditor, mockMonaco);

    const actionCalls = mockAddAction.mock.calls;
    const selectAllAction = actionCalls.find(
      (call: unknown[]) => (call[0] as { id: string }).id === "select-all",
    );
    expect(selectAllAction).toBeTruthy();

    const action = selectAllAction![0] as { run: (ed: unknown) => void };
    action.run(mockEditor);

    expect(mockGetModel).toHaveBeenCalled();
    expect(mockGetFullModelRange).toHaveBeenCalled();
    expect(mockSetSelection).toHaveBeenCalledWith({
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: 5,
      endColumn: 10,
    });
  });

  test("select-all action handles missing model gracefully", () => {
    renderEditor();
    expect(capturedOnMount).toBeTruthy();

    const mockEditor = createMockEditor();
    mockEditor.getModel = mock(() => null);
    const mockMonaco = createMockMonaco();

    capturedOnMount!(mockEditor, mockMonaco);

    const actionCalls = mockAddAction.mock.calls;
    const selectAllAction = actionCalls.find(
      (call: unknown[]) => (call[0] as { id: string }).id === "select-all",
    );

    const action = selectAllAction![0] as { run: (ed: unknown) => void };
    action.run(mockEditor);

    expect(mockSetSelection).not.toHaveBeenCalled();
  });

  test("registers bold, italic, link, and select-all actions", () => {
    renderEditor();
    expect(capturedOnMount).toBeTruthy();

    const mockEditor = createMockEditor();
    const mockMonaco = createMockMonaco();

    capturedOnMount!(mockEditor, mockMonaco);

    const actionIds = mockAddAction.mock.calls.map(
      (call: unknown[]) => (call[0] as { id: string }).id,
    );
    expect(actionIds).toContain("markdown-bold");
    expect(actionIds).toContain("markdown-italic");
    expect(actionIds).toContain("markdown-link");
    expect(actionIds).toContain("select-all");
  });

  test("switches to preview mode and shows content", async () => {
    renderEditor({ value: "# Hello World" });

    const previewButton = screen.getByText("Preview");
    await act(async () => {
      fireEvent.click(previewButton);
    });

    expect(screen.getByText("Hello World")).toBeTruthy();
  });

  test("shows nothing to preview when value is empty", async () => {
    renderEditor({ value: "" });

    const previewButton = screen.getByText("Preview");
    await act(async () => {
      fireEvent.click(previewButton);
    });

    expect(screen.getByText("Nothing to preview")).toBeTruthy();
  });
});
