import { describe, test, expect, afterEach } from "bun:test";
import { render, screen, cleanup } from "@testing-library/react";
import { MarkdownContent } from "./MarkdownContent";

describe("MarkdownContent", () => {
  afterEach(cleanup);

  test("renders markdown body without frontmatter", () => {
    render(<MarkdownContent content="# Hello World" />);
    expect(screen.getByText("Hello World")).toBeTruthy();
  });

  test("renders frontmatter table when content has frontmatter", () => {
    const content = "---\ntitle: My Project\nversion: 1\n---\n\n# Hello";
    render(<MarkdownContent content={content} />);

    const table = document.querySelector("[data-testid='frontmatter-table']");
    expect(table).toBeTruthy();
    expect(screen.getByText("Key")).toBeTruthy();
    expect(screen.getByText("Value")).toBeTruthy();
    expect(screen.getByText("title")).toBeTruthy();
    expect(screen.getByText("My Project")).toBeTruthy();
    expect(screen.getByText("version")).toBeTruthy();
  });

  test("does not render frontmatter table when no frontmatter", () => {
    render(<MarkdownContent content="# Just Markdown\n\nSome content" />);

    const table = document.querySelector("[data-testid='frontmatter-table']");
    expect(table).toBeNull();
    expect(screen.getByText(/Just Markdown/)).toBeTruthy();
  });

  test("renders simple string values inline", () => {
    const content = '---\nname: simple-value\n---\n\nBody';
    render(<MarkdownContent content={content} />);

    expect(screen.getByText("simple-value").tagName).toBe("SPAN");
  });

  test("renders boolean values inline", () => {
    const content = "---\nenabled: true\n---\n\nBody";
    render(<MarkdownContent content={content} />);

    expect(screen.getByText("true").tagName).toBe("SPAN");
  });

  test("renders number values inline", () => {
    const content = "---\ncount: 42\n---\n\nBody";
    render(<MarkdownContent content={content} />);

    expect(screen.getByText("42").tagName).toBe("SPAN");
  });

  test("renders multi-line string values as pre-formatted text", () => {
    const content = '---\nflow: |\n  line1\n  line2\n---\n\nBody';
    render(<MarkdownContent content={content} />);

    const preElements = document.querySelectorAll("pre");
    const found = Array.from(preElements).some(
      (pre) => pre.textContent?.includes("line1") && pre.textContent?.includes("line2"),
    );
    expect(found).toBe(true);
  });

  test("renders nested object values as pre-formatted text", () => {
    const content = '---\nmodels:\n  coordinator: model-a\n  developer: model-b\n---\n\nBody';
    render(<MarkdownContent content={content} />);

    const preElements = document.querySelectorAll("pre");
    const found = Array.from(preElements).some(
      (pre) => pre.textContent?.includes("coordinator"),
    );
    expect(found).toBe(true);
  });

  test("renders markdown body below the frontmatter table", () => {
    const content = "---\ntitle: Project\n---\n\n# Heading\n\nParagraph text";
    render(<MarkdownContent content={content} />);

    expect(document.querySelector("[data-testid='frontmatter-table']")).toBeTruthy();
    expect(screen.getByText("Heading")).toBeTruthy();
    expect(screen.getByText("Paragraph text")).toBeTruthy();
  });

  test("handles invalid YAML gracefully", () => {
    const content = "---\n: invalid: yaml: [[\n---\n\n# Heading";
    render(<MarkdownContent content={content} />);

    const table = document.querySelector("[data-testid='frontmatter-table']");
    expect(table).toBeNull();
    expect(screen.getByText("Heading")).toBeTruthy();
  });

  test("handles empty frontmatter", () => {
    const content = "---\n\n---\n\n# Heading";
    render(<MarkdownContent content={content} />);

    const table = document.querySelector("[data-testid='frontmatter-table']");
    expect(table).toBeNull();
  });

  test("renders real-world GOAL.md frontmatter", () => {
    const content = [
      "---",
      "flow: |",
      '  "backend-go-developer" -> "go-readability-reviewer"',
      '  "react-developer" -> "react-reviewer"',
      "models:",
      '  "coordinator": "anthropic/claude-opus-4-6 (max)"',
      '  "backend-go-developer": "anthropic/claude-opus-4-6"',
      "---",
      "",
      "# Build the project",
      "",
      "- [ ] Task one",
    ].join("\n");

    render(<MarkdownContent content={content} />);

    expect(document.querySelector("[data-testid='frontmatter-table']")).toBeTruthy();
    expect(screen.getByText("flow")).toBeTruthy();
    expect(screen.getByText("models")).toBeTruthy();
    expect(screen.getByText("Build the project")).toBeTruthy();
    expect(screen.getByText("Task one")).toBeTruthy();
  });
});
