import { useCallback, useEffect, useRef, useState } from "react";
import Editor, { type OnMount } from "@monaco-editor/react";
import type * as MonacoTypes from "monaco-editor";
import { api } from "@/lib/api";
import type { Agent, ApiModelEntry } from "@/types";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ScrollArea } from "@/components/ui/scroll-area";
import { MarkdownContent } from "@/components/MarkdownContent";
import {
  Bold,
  Italic,
  Strikethrough,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  ListChecks,
  Code2,
  Quote,
  Link,
  Image,
  Minus,
  Table,
  Eye,
  Pencil,
} from "lucide-react";

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string | undefined) => void;
  minHeight?: number;
  defaultHeight?: number;
  disabled?: boolean;
  placeholder?: string;
  workspaceName?: string;
  fillHeight?: boolean;
}

type IStandaloneCodeEditor = MonacoTypes.editor.IStandaloneCodeEditor;

interface ToolbarAction {
  icon: React.ReactNode;
  label: string;
  action: (editor: IStandaloneCodeEditor) => void;
}

function wrapSelection(
  editor: IStandaloneCodeEditor,
  prefix: string,
  suffix: string,
) {
  const selection = editor.getSelection();
  if (!selection) return;
  const model = editor.getModel();
  if (!model) return;

  const selectedText = model.getValueInRange(selection);
  const replacement = selectedText
    ? `${prefix}${selectedText}${suffix}`
    : `${prefix}text${suffix}`;

  editor.executeEdits("toolbar", [
    { range: selection, text: replacement },
  ]);

  if (!selectedText) {
    const startCol = selection.startColumn + prefix.length;
    const endCol = startCol + 4;
    editor.setSelection({
      startLineNumber: selection.startLineNumber,
      startColumn: startCol,
      endLineNumber: selection.startLineNumber,
      endColumn: endCol,
    });
  }

  editor.focus();
}

function insertLink(editor: IStandaloneCodeEditor) {
  const selection = editor.getSelection();
  if (!selection) return;
  const model = editor.getModel();
  if (!model) return;
  const selectedText = model.getValueInRange(selection);
  const replacement = selectedText
    ? `[${selectedText}](url)`
    : "[link text](url)";
  editor.executeEdits("toolbar", [
    { range: selection, text: replacement },
  ]);
  editor.focus();
}

function insertAtLineStart(
  editor: IStandaloneCodeEditor,
  prefix: string,
) {
  const selection = editor.getSelection();
  if (!selection) return;
  const model = editor.getModel();
  if (!model) return;

  const lineNumber = selection.startLineNumber;
  const lineContent = model.getLineContent(lineNumber);
  const newContent = `${prefix}${lineContent}`;

  editor.executeEdits("toolbar", [
    {
      range: {
        startLineNumber: lineNumber,
        startColumn: 1,
        endLineNumber: lineNumber,
        endColumn: lineContent.length + 1,
      },
      text: newContent,
    },
  ]);

  editor.focus();
}

function insertAtCursor(
  editor: IStandaloneCodeEditor,
  text: string,
) {
  const selection = editor.getSelection();
  if (!selection) return;

  editor.executeEdits("toolbar", [
    { range: selection, text },
  ]);

  editor.focus();
}

const TOOLBAR_ACTIONS: ToolbarAction[] = [
  {
    icon: <Bold className="h-4 w-4" />,
    label: "Bold",
    action: (editor) => wrapSelection(editor, "**", "**"),
  },
  {
    icon: <Italic className="h-4 w-4" />,
    label: "Italic",
    action: (editor) => wrapSelection(editor, "_", "_"),
  },
  {
    icon: <Strikethrough className="h-4 w-4" />,
    label: "Strikethrough",
    action: (editor) => wrapSelection(editor, "~~", "~~"),
  },
  {
    icon: <Heading1 className="h-4 w-4" />,
    label: "Heading 1",
    action: (editor) => insertAtLineStart(editor, "# "),
  },
  {
    icon: <Heading2 className="h-4 w-4" />,
    label: "Heading 2",
    action: (editor) => insertAtLineStart(editor, "## "),
  },
  {
    icon: <Heading3 className="h-4 w-4" />,
    label: "Heading 3",
    action: (editor) => insertAtLineStart(editor, "### "),
  },
  {
    icon: <List className="h-4 w-4" />,
    label: "Bullet List",
    action: (editor) => insertAtLineStart(editor, "- "),
  },
  {
    icon: <ListOrdered className="h-4 w-4" />,
    label: "Numbered List",
    action: (editor) => insertAtLineStart(editor, "1. "),
  },
  {
    icon: <ListChecks className="h-4 w-4" />,
    label: "Checkbox List",
    action: (editor) => insertAtLineStart(editor, "- [ ] "),
  },
  {
    icon: <Code2 className="h-4 w-4" />,
    label: "Code Block",
    action: (editor) => {
      const selection = editor.getSelection();
      if (!selection) return;
      const model = editor.getModel();
      if (!model) return;
      const selectedText = model.getValueInRange(selection);
      const replacement = selectedText
        ? `\`\`\`\n${selectedText}\n\`\`\``
        : "```\ncode\n```";
      editor.executeEdits("toolbar", [
        { range: selection, text: replacement },
      ]);
      editor.focus();
    },
  },
  {
    icon: <Quote className="h-4 w-4" />,
    label: "Blockquote",
    action: (editor) => insertAtLineStart(editor, "> "),
  },
  {
    icon: <Link className="h-4 w-4" />,
    label: "Link",
    action: (editor) => insertLink(editor),
  },
  {
    icon: <Image className="h-4 w-4" />,
    label: "Image",
    action: (editor) =>
      insertAtCursor(editor, "![alt text](image-url)"),
  },
  {
    icon: <Minus className="h-4 w-4" />,
    label: "Horizontal Rule",
    action: (editor) => insertAtCursor(editor, "\n---\n"),
  },
  {
    icon: <Table className="h-4 w-4" />,
    label: "Table",
    action: (editor) =>
      insertAtCursor(
        editor,
        "\n| Header | Header |\n| ------ | ------ |\n| Cell   | Cell   |\n",
      ),
  },
];

export function MarkdownEditor({
  value,
  onChange,
  minHeight = 200,
  defaultHeight,
  disabled = false,
  placeholder,
  workspaceName,
  fillHeight = false,
}: MarkdownEditorProps) {
  const editorRef = useRef<IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<typeof MonacoTypes | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const toolbarRef = useRef<HTMLDivElement>(null);
  const modeBarRef = useRef<HTMLDivElement>(null);
  const completionDisposableRef = useRef<MonacoTypes.IDisposable | null>(null);
  const agentsRef = useRef<Agent[]>([]);
  const modelsRef = useRef<ApiModelEntry[]>([]);

  const baseHeight = defaultHeight ?? minHeight;
  const [editorHeight, setEditorHeight] = useState(baseHeight);
  const [mode, setMode] = useState<"write" | "preview">("write");
  const [monacoReady, setMonacoReady] = useState(false);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const observer = new ResizeObserver(() => {
      const toolbar = toolbarRef.current;
      const modeBar = modeBarRef.current;
      const toolbarHeight = toolbar ? toolbar.offsetHeight : 0;
      const modeBarHeight = modeBar ? modeBar.offsetHeight : 0;
      const available = container.clientHeight - toolbarHeight - modeBarHeight;
      setEditorHeight(Math.max(available, minHeight));
    });

    observer.observe(container);

    return () => {
      observer.disconnect();
    };
  }, [minHeight, fillHeight]);

  useEffect(() => {
    if (!workspaceName) return;
    let cancelled = false;

    Promise.all([
      api.agents.list(workspaceName),
      api.models.list(workspaceName),
    ]).then(
      ([agentsResult, modelsResult]) => {
        if (cancelled) return;
        agentsRef.current = agentsResult.agents ?? [];
        modelsRef.current = modelsResult.models ?? [];
      },
      () => {},
    );

    return () => { cancelled = true; };
  }, [workspaceName]);

  const handleMount: OnMount = useCallback(
    (editor, monaco) => {
      editorRef.current = editor;
      monacoRef.current = monaco;
      setMonacoReady(true);

      editor.addAction({
        id: "markdown-bold",
        label: "Bold",
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyB],
        run: (ed) => wrapSelection(ed, "**", "**"),
      });

      editor.addAction({
        id: "markdown-italic",
        label: "Italic",
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyI],
        run: (ed) => wrapSelection(ed, "_", "_"),
      });

      editor.addAction({
        id: "markdown-link",
        label: "Insert Link",
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyK],
        run: (ed) => insertLink(ed),
      });

      editor.addAction({
        id: "select-all",
        label: "Select All",
        keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyA],
        run: (ed) => {
          const model = ed.getModel();
          if (model) {
            ed.setSelection(model.getFullModelRange());
          }
        },
      });

      if (placeholder) {
        const model = editor.getModel();
        if (model && !model.getValue()) {
          const decorations = [
            {
              range: {
                startLineNumber: 1,
                startColumn: 1,
                endLineNumber: 1,
                endColumn: 1,
              },
              options: {
                after: {
                  content: placeholder,
                  inlineClassName: "monaco-placeholder",
                },
              },
            },
          ];
          let decorationIds = editor.deltaDecorations([], decorations);

          editor.onDidChangeModelContent(() => {
            const currentValue = model.getValue();
            if (currentValue) {
              decorationIds = editor.deltaDecorations(decorationIds, []);
            } else {
              decorationIds = editor.deltaDecorations(
                decorationIds,
                decorations,
              );
            }
          });
        }
      }
    },
    [placeholder],
  );

  useEffect(() => {
    const monaco = monacoRef.current;
    if (!workspaceName || !monaco || !monacoReady) return;

    // The completion provider is registered globally for the "markdown" language. The cleanup
    // disposes it on unmount/re-render, but if two editors mount simultaneously they would share
    // the provider. Currently only one editor renders at a time (EditGoal is full-screen,
    // InlineForkEditor is on workspace detail), so this is safe.
    completionDisposableRef.current?.dispose();
    completionDisposableRef.current = monaco.languages.registerCompletionItemProvider("markdown", {
      triggerCharacters: ['"', "'", " ", ":"],
      provideCompletionItems: (model, position) => {
        const content = model.getValue();
        const lines = content.split("\n");

        const firstLineIsFrontmatter = lines[0]?.trim() === "---";
        if (!firstLineIsFrontmatter) return { suggestions: [] };

        let closingIndex = -1;
        for (let i = 1; i < lines.length; i++) {
          if (lines[i]?.trim() === "---") {
            closingIndex = i;
            break;
          }
        }
        if (closingIndex === -1) return { suggestions: [] };

        const lineIndex = position.lineNumber - 1;
        if (lineIndex <= 0 || lineIndex >= closingIndex) return { suggestions: [] };

        let currentSection = "";
        for (let i = lineIndex; i >= 1; i--) {
          const line = lines[i] ?? "";
          const sectionMatch = line.match(/^(\w[\w-]*):/);
          if (sectionMatch) {
            currentSection = sectionMatch[1] ?? "";
            break;
          }
        }

        const currentLine = lines[lineIndex] ?? "";
        const textBeforeCursor = currentLine.substring(0, position.column - 1);

        const quoteMatch = textBeforeCursor.match(/["'][^"']*$/);
        const replaceStart = quoteMatch
          ? position.column - quoteMatch[0].length
          : position.column;
        const range: MonacoTypes.IRange = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: replaceStart,
          endColumn: position.column,
        };

        const suggestions: MonacoTypes.languages.CompletionItem[] = [];

        if (currentSection === "flow") {
          for (const agent of agentsRef.current) {
            suggestions.push({
              label: `"${agent.name}"`,
              kind: monaco.languages.CompletionItemKind.Value,
              detail: agent.description,
              insertText: `"${agent.name}"`,
              filterText: agent.name,
              range,
            });
          }
        } else if (currentSection === "models") {
          const colonPos = textBeforeCursor.indexOf(":");
          const isAfterColon = colonPos >= 0 && position.column - 1 > colonPos;

          if (isAfterColon) {
            for (const m of modelsRef.current) {
              suggestions.push({
                label: `"${m.id}"`,
                kind: monaco.languages.CompletionItemKind.Variable,
                detail: m.name,
                insertText: `"${m.id}"`,
                filterText: m.id,
                range,
              });
            }
          } else {
            for (const agent of agentsRef.current) {
              suggestions.push({
                label: `"${agent.name}"`,
                kind: monaco.languages.CompletionItemKind.Value,
                detail: agent.description,
                insertText: `"${agent.name}":`,
                filterText: agent.name,
                range,
              });
            }
          }
        }

        return { suggestions };
      },
    });

    return () => {
      completionDisposableRef.current?.dispose();
      completionDisposableRef.current = null;
    };
  }, [workspaceName, monacoReady]);

  const handleToolbarAction = useCallback(
    (action: (editor: IStandaloneCodeEditor) => void) => {
      if (editorRef.current && !disabled) {
        action(editorRef.current);
      }
    },
    [disabled],
  );

  return (
    <div
      ref={containerRef}
      className={fillHeight ? "flex flex-col h-full overflow-hidden" : "border rounded-md"}
      style={fillHeight ? undefined : { minHeight: `${minHeight}px`, resize: "vertical", overflow: "hidden" }}
      data-testid="markdown-editor"
    >
      <div
        ref={modeBarRef}
        className="flex gap-1 p-1 border-b bg-muted/30"
      >
        <Button
          type="button"
          variant={mode === "write" ? "secondary" : "ghost"}
          size="sm"
          className="h-7 px-3 text-xs"
          onClick={() => setMode("write")}
          aria-pressed={mode === "write"}
        >
          <Pencil className="h-3.5 w-3.5 mr-1" />
          Write
        </Button>
        <Button
          type="button"
          variant={mode === "preview" ? "secondary" : "ghost"}
          size="sm"
          className="h-7 px-3 text-xs"
          onClick={() => setMode("preview")}
          aria-pressed={mode === "preview"}
        >
          <Eye className="h-3.5 w-3.5 mr-1" />
          Preview
        </Button>
      </div>

      {mode === "write" ? (
        <>
          <div ref={toolbarRef} className="flex flex-wrap gap-0.5 p-1 border-b bg-muted/50">
            {TOOLBAR_ACTIONS.map((item) => (
              <Tooltip key={item.label}>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    disabled={disabled}
                    onClick={() => handleToolbarAction(item.action)}
                    aria-label={item.label}
                  >
                    {item.icon}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{item.label}</TooltipContent>
              </Tooltip>
            ))}
          </div>

          <Editor
            height={`${editorHeight}px`}
            language="markdown"
            value={value}
            onChange={onChange}
            onMount={handleMount}
            options={{
              wordWrap: "on",
              automaticLayout: true,
              minimap: { enabled: false },
              scrollBeyondLastLine: false,
              lineNumbers: "off",
              glyphMargin: false,
              folding: false,
              renderLineHighlight: "none",
              overviewRulerBorder: false,
              hideCursorInOverviewRuler: true,
              readOnly: disabled,
              domReadOnly: disabled,
              padding: { top: 8, bottom: 8 },
              quickSuggestions: workspaceName ? { other: true, strings: true } : false,
              wordBasedSuggestions: "off" as const,
              suggestOnTriggerCharacters: !!workspaceName,
              acceptSuggestionOnEnter: workspaceName ? "on" : "off",
            }}
          />
        </>
      ) : (
        <ScrollArea
          style={{ height: `${editorHeight}px` }}
          className="p-4"
        >
          {value ? (
            <MarkdownContent content={value} />
          ) : (
            <p className="text-muted-foreground text-sm italic">
              Nothing to preview
            </p>
          )}
        </ScrollArea>
      )}
    </div>
  );
}
