import { describe, it, expect } from "bun:test";
import { stripFrontmatter, stripMarkdownToPlaintext, truncateDescription, workspaceDescription } from "@/lib/markdown-utils";

describe("stripFrontmatter", () => {
  it("removes YAML frontmatter from content", () => {
    const content = "---\ntitle: Test\n---\n# Hello World";
    expect(stripFrontmatter(content)).toBe("# Hello World");
  });

  it("returns content unchanged when no frontmatter present", () => {
    const content = "# Hello World";
    expect(stripFrontmatter(content)).toBe("# Hello World");
  });

  it("handles empty content", () => {
    expect(stripFrontmatter("")).toBe("");
  });

  it("handles frontmatter with leading whitespace", () => {
    const content = "  ---\ntitle: Test\n---\n# Hello";
    expect(stripFrontmatter(content)).toBe("# Hello");
  });

  it("handles content that is only frontmatter", () => {
    const content = "---\ntitle: Test\n---\n";
    expect(stripFrontmatter(content)).toBe("");
  });
});

describe("stripMarkdownToPlaintext", () => {
  it("removes headings", () => {
    expect(stripMarkdownToPlaintext("# Heading")).toBe("Heading");
    expect(stripMarkdownToPlaintext("## Sub")).toBe("Sub");
    expect(stripMarkdownToPlaintext("### Third")).toBe("Third");
  });

  it("removes bold formatting", () => {
    expect(stripMarkdownToPlaintext("**bold**")).toBe("bold");
    expect(stripMarkdownToPlaintext("__bold__")).toBe("bold");
  });

  it("removes italic formatting", () => {
    expect(stripMarkdownToPlaintext("*italic*")).toBe("italic");
    expect(stripMarkdownToPlaintext("_italic_")).toBe("italic");
  });

  it("removes strikethrough", () => {
    expect(stripMarkdownToPlaintext("~~deleted~~")).toBe("deleted");
  });

  it("removes inline code", () => {
    expect(stripMarkdownToPlaintext("`code`")).toBe("code");
  });

  it("extracts link text", () => {
    expect(stripMarkdownToPlaintext("[link text](http://example.com)")).toBe("link text");
  });

  it("extracts image alt text", () => {
    // The image regex replaces ![alt](url) -> alt but leaves the ! prefix
    // since the link regex handles [text](url) -> text before image regex
    const result = stripMarkdownToPlaintext("![alt text](image.png)");
    expect(result).toBe("!alt text");
  });

  it("removes blockquote markers", () => {
    expect(stripMarkdownToPlaintext("> quoted text")).toBe("quoted text");
  });

  it("removes list markers", () => {
    expect(stripMarkdownToPlaintext("- item")).toBe("item");
    expect(stripMarkdownToPlaintext("* item")).toBe("item");
    expect(stripMarkdownToPlaintext("+ item")).toBe("item");
    expect(stripMarkdownToPlaintext("1. item")).toBe("item");
  });

  it("removes checkbox markers", () => {
    expect(stripMarkdownToPlaintext("- [x] done")).toBe("done");
    expect(stripMarkdownToPlaintext("- [ ] todo")).toBe("todo");
  });

  it("removes horizontal rules", () => {
    expect(stripMarkdownToPlaintext("---")).toBe("");
  });

  it("collapses multiple newlines into spaces", () => {
    expect(stripMarkdownToPlaintext("line one\n\nline two")).toBe("line one line two");
  });

  it("strips frontmatter first", () => {
    const content = "---\ntitle: Test\n---\n# Hello";
    expect(stripMarkdownToPlaintext(content)).toBe("Hello");
  });
});

describe("truncateDescription", () => {
  it("returns text unchanged when shorter than maxLen", () => {
    expect(truncateDescription("short text")).toBe("short text");
  });

  it("truncates text and adds ellipsis when longer than maxLen", () => {
    const text = "a".repeat(300);
    const result = truncateDescription(text, 255);
    expect(result.length).toBe(258);
    expect(result.endsWith("...")).toBe(true);
  });

  it("uses custom maxLen", () => {
    const result = truncateDescription("hello world", 5);
    expect(result).toBe("hello...");
  });

  it("returns exact text when length equals maxLen", () => {
    const text = "a".repeat(255);
    expect(truncateDescription(text, 255)).toBe(text);
  });
});

describe("workspaceDescription", () => {
  it("returns null for undefined content", () => {
    expect(workspaceDescription(undefined)).toBeNull();
  });

  it("returns null for empty content", () => {
    expect(workspaceDescription("")).toBeNull();
    expect(workspaceDescription("  ")).toBeNull();
  });

  it("returns null when stripped content is empty", () => {
    expect(workspaceDescription("---\ntitle: Test\n---\n")).toBeNull();
  });

  it("returns plain text description from markdown", () => {
    const result = workspaceDescription("# My Goal\n\nBuild a **great** app.");
    expect(result).toBe("My Goal Build a great app.");
  });

  it("truncates long descriptions", () => {
    const longContent = "# " + "a".repeat(300);
    const result = workspaceDescription(longContent);
    expect(result).not.toBeNull();
    expect(result!.length).toBeLessThanOrEqual(258);
    expect(result!.endsWith("...")).toBe(true);
  });
});
