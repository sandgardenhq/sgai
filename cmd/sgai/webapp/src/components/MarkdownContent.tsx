import type {
  AnchorHTMLAttributes,
  HTMLAttributes,
  InputHTMLAttributes,
  TableHTMLAttributes,
  TdHTMLAttributes,
  ThHTMLAttributes,
} from "react";
import { useMemo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkFrontmatter from "remark-frontmatter";
import rehypeRaw from "rehype-raw";
import { parse as parseYaml } from "yaml";
import { cn } from "@/lib/utils";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const FRONTMATTER_RE = /^---\n([\s\S]*?)\n---/;

function extractFrontmatter(content: string): {
  frontmatter: Record<string, unknown> | null;
  body: string;
} {
  const match = FRONTMATTER_RE.exec(content);
  if (!match) {
    return { frontmatter: null, body: content };
  }
  try {
    const parsed = parseYaml(match[1]);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return {
        frontmatter: parsed as Record<string, unknown>,
        body: content.slice(match[0].length),
      };
    }
  } catch {
    // invalid YAML, skip table rendering
  }
  return { frontmatter: null, body: content };
}

function isSimpleValue(value: unknown): boolean {
  return (
    typeof value === "string" &&
    !value.includes("\n") &&
    value.length < 80
  ) || typeof value === "number" || typeof value === "boolean";
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value, null, 2);
}

function FrontmatterTable({ data }: { data: Record<string, unknown> }) {
  const entries = Object.entries(data);
  if (entries.length === 0) {
    return null;
  }

  return (
    <Table data-testid="frontmatter-table">
      <TableHeader>
        <TableRow>
          <TableHead className="w-[140px]">Key</TableHead>
          <TableHead>Value</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map(([key, value]) => (
          <TableRow key={key}>
            <TableCell className="font-medium align-top">{key}</TableCell>
            <TableCell className="whitespace-normal break-words">
              {isSimpleValue(value) ? (
                <span>{formatValue(value)}</span>
              ) : (
                <pre className="whitespace-pre-wrap break-words font-mono text-xs bg-muted/30 rounded p-2 m-0 overflow-x-auto">
                  {formatValue(value)}
                </pre>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

interface MarkdownContentProps {
  content: string;
  className?: string;
}

export function MarkdownContent({ content, className }: MarkdownContentProps) {
  const { frontmatter, body } = useMemo(() => extractFrontmatter(content), [content]);

  const components = {
    h1: (props: HTMLAttributes<HTMLHeadingElement>) => (
      <h1 className="text-2xl font-semibold mt-4 mb-2" {...props} />
    ),
    h2: (props: HTMLAttributes<HTMLHeadingElement>) => (
      <h2 className="text-xl font-semibold mt-4 mb-2" {...props} />
    ),
    h3: (props: HTMLAttributes<HTMLHeadingElement>) => (
      <h3 className="text-lg font-semibold mt-3 mb-2" {...props} />
    ),
    h4: (props: HTMLAttributes<HTMLHeadingElement>) => (
      <h4 className="text-base font-semibold mt-3 mb-1.5" {...props} />
    ),
    p: (props: HTMLAttributes<HTMLParagraphElement>) => (
      <p className="leading-relaxed" {...props} />
    ),
    ul: (props: HTMLAttributes<HTMLUListElement>) => (
      <ul className="list-disc pl-6 space-y-1" {...props} />
    ),
    ol: (props: HTMLAttributes<HTMLOListElement>) => (
      <ol className="list-decimal pl-6 space-y-1" {...props} />
    ),
    li: (props: HTMLAttributes<HTMLLIElement>) => (
      <li className="leading-relaxed" {...props} />
    ),
    blockquote: (props: HTMLAttributes<HTMLQuoteElement>) => (
      <blockquote className="border-l-2 pl-3 text-muted-foreground italic" {...props} />
    ),
    code: ({ className: codeClassName, inline, ...props }: HTMLAttributes<HTMLElement> & { inline?: boolean }) => (
      <code
        className={cn(
          inline
            ? "rounded bg-muted px-1.5 py-0.5 font-mono text-xs"
            : "font-mono text-xs",
          codeClassName,
        )}
        {...props}
      />
    ),
    pre: (props: HTMLAttributes<HTMLPreElement>) => (
      <pre className="overflow-x-auto rounded border bg-muted/30 p-3 text-xs" {...props} />
    ),
    a: (props: AnchorHTMLAttributes<HTMLAnchorElement>) => (
      <a className="text-primary underline underline-offset-4" {...props} />
    ),
    hr: (props: HTMLAttributes<HTMLHRElement>) => (
      <hr className="my-4 border-border" {...props} />
    ),
    table: (props: TableHTMLAttributes<HTMLTableElement>) => (
      <table className="w-full text-sm border border-border" {...props} />
    ),
    thead: (props: HTMLAttributes<HTMLTableSectionElement>) => (
      <thead className="bg-muted/40" {...props} />
    ),
    th: (props: ThHTMLAttributes<HTMLTableCellElement>) => (
      <th className="border border-border px-2 py-1 text-left font-semibold" {...props} />
    ),
    td: (props: TdHTMLAttributes<HTMLTableCellElement>) => (
      <td className="border border-border px-2 py-1 align-top" {...props} />
    ),
    input: ({ type, checked, className: inputClassName, ...props }: InputHTMLAttributes<HTMLInputElement>) => {
      if (type === "checkbox") {
        return (
          <input
            type="checkbox"
            checked={checked}
            readOnly
            disabled
            className={cn("mr-2 align-middle", inputClassName)}
            {...props}
          />
        );
      }

      return <input type={type} className={inputClassName} {...props} />;
    },
  };

  return (
    <div className={cn("space-y-3 text-sm", className)}>
      {frontmatter ? <FrontmatterTable data={frontmatter} /> : null}
      <ReactMarkdown remarkPlugins={[remarkGfm, remarkFrontmatter]} rehypePlugins={[rehypeRaw]} components={components}>
        {body}
      </ReactMarkdown>
    </div>
  );
}
