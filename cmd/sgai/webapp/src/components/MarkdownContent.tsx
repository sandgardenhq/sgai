import type {
  AnchorHTMLAttributes,
  HTMLAttributes,
  InputHTMLAttributes,
  TableHTMLAttributes,
  TdHTMLAttributes,
  ThHTMLAttributes,
} from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkFrontmatter from "remark-frontmatter";
import rehypeRaw from "rehype-raw";
import { cn } from "@/lib/utils";

interface MarkdownContentProps {
  content: string;
  className?: string;
}

export function MarkdownContent({ content, className }: MarkdownContentProps) {
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
      <ReactMarkdown remarkPlugins={[remarkGfm, remarkFrontmatter]} rehypePlugins={[rehypeRaw]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
}
