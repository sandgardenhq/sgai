import { describe, it, expect, afterEach } from "bun:test";
import { render, screen, cleanup } from "@testing-library/react";
import { MarkdownContent } from "../MarkdownContent";

afterEach(() => {
  cleanup();
});

describe("MarkdownContent", () => {
  describe("basic rendering", () => {
    it("renders plain text content", () => {
      render(<MarkdownContent content="Hello World" />);
      expect(screen.getByText("Hello World")).toBeTruthy();
    });

    it("renders heading text", () => {
      render(<MarkdownContent content={"# Heading One"} />);
      expect(screen.getByText("Heading One")).toBeTruthy();
    });

    it("renders bold text", () => {
      render(<MarkdownContent content="**bold text**" />);
      expect(screen.getByText("bold text")).toBeTruthy();
    });

    it("renders italic text", () => {
      render(<MarkdownContent content="*italic text*" />);
      expect(screen.getByText("italic text")).toBeTruthy();
    });

    it("renders links", () => {
      render(<MarkdownContent content="[link text](http://example.com)" />);
      const link = screen.getByText("link text");
      expect(link).toBeTruthy();
      expect(link.closest("a")?.getAttribute("href")).toBe("http://example.com");
    });

    it("renders blockquotes", () => {
      render(<MarkdownContent content="> Quoted text" />);
      expect(screen.getByText("Quoted text")).toBeTruthy();
    });

    it("renders inline code", () => {
      render(<MarkdownContent content="Use `code` here" />);
      expect(screen.getByText("code")).toBeTruthy();
    });

    it("renders horizontal rules", () => {
      const { container } = render(<MarkdownContent content={"Text above\n\n---\n\nText below"} />);
      const hrs = container.querySelectorAll("hr");
      expect(hrs.length).toBeGreaterThan(0);
    });

    it("applies heading styling classes", () => {
      const { container } = render(<MarkdownContent content={"# Title"} />);
      const h1 = container.querySelector("h1");
      expect(h1).toBeTruthy();
      expect(h1?.className).toContain("font-semibold");
    });
  });

  describe("frontmatter handling", () => {
    it("renders frontmatter as table when present", () => {
      const content = "---\ntitle: Test\nauthor: User\n---\n\nBody content";
      render(<MarkdownContent content={content} />);
      expect(screen.getByTestId("frontmatter-table")).toBeTruthy();
      expect(screen.getByText("title")).toBeTruthy();
      expect(screen.getByText("Test")).toBeTruthy();
    });

    it("renders body content after frontmatter", () => {
      const content = "---\ntitle: Test\n---\n\nBody content here";
      render(<MarkdownContent content={content} />);
      expect(screen.getByText("Body content here")).toBeTruthy();
    });

    it("renders without frontmatter table when no frontmatter", () => {
      render(<MarkdownContent content="Just a paragraph" />);
      expect(screen.queryByTestId("frontmatter-table")).toBeNull();
    });

    it("handles invalid YAML frontmatter gracefully", () => {
      const content = "---\n[invalid yaml\n---\nBody";
      render(<MarkdownContent content={content} />);
      expect(screen.queryByTestId("frontmatter-table")).toBeNull();
    });

    it("renders complex frontmatter values as pre-formatted text", () => {
      const content = '---\nflow: |\n  "a" -> "b"\n  "b" -> "c"\n---\n\nBody';
      render(<MarkdownContent content={content} />);
      expect(screen.getByTestId("frontmatter-table")).toBeTruthy();
    });

    it("renders boolean and number values in frontmatter", () => {
      const content = "---\nenabled: true\ncount: 42\n---\n\nBody";
      render(<MarkdownContent content={content} />);
      expect(screen.getByText("true")).toBeTruthy();
      expect(screen.getByText("42")).toBeTruthy();
    });

    it("renders empty frontmatter table when all entries are empty", () => {
      // Frontmatter with no entries doesn't render the table
      const content = "---\n---\nBody";
      render(<MarkdownContent content={content} />);
      // Empty object from YAML returns null, so no table should render
      expect(screen.queryByTestId("frontmatter-table")).toBeNull();
    });
  });

  describe("className prop", () => {
    it("applies custom className", () => {
      const { container } = render(
        <MarkdownContent content="Test" className="custom-class" />
      );
      const div = container.firstChild as HTMLElement;
      expect(div.className).toContain("custom-class");
    });

    it("always includes base classes", () => {
      const { container } = render(<MarkdownContent content="Test" />);
      const div = container.firstChild as HTMLElement;
      expect(div.className).toContain("space-y-3");
      expect(div.className).toContain("text-sm");
    });
  });

  describe("GFM support", () => {
    it("renders strikethrough text", () => {
      render(<MarkdownContent content="~~deleted text~~" />);
      expect(screen.getByText("deleted text")).toBeTruthy();
    });
  });

  describe("component rendering", () => {
    it("renders with ReactMarkdown components", () => {
      const { container } = render(<MarkdownContent content="**bold** and *italic*" />);
      const strong = container.querySelector("strong");
      const em = container.querySelector("em");
      expect(strong).toBeTruthy();
      expect(em).toBeTruthy();
    });

    it("renders code elements with styling", () => {
      render(<MarkdownContent content="`inline code`" />);
      const code = screen.getByText("inline code");
      expect(code.tagName.toLowerCase()).toBe("code");
    });

    it("renders link with proper classes", () => {
      render(<MarkdownContent content="[click me](http://example.com)" />);
      const link = screen.getByText("click me");
      expect(link.className).toContain("text-primary");
    });
  });
});
