import { describe, it, expect } from "bun:test";
import {
  stripFrontmatter,
  stripMarkdownToPlaintext,
  truncateDescription,
  workspaceDescription,
} from "./markdown-utils";

describe("stripFrontmatter", () => {
  it("removes YAML frontmatter", () => {
    const input = `---
title: Hello
---
Content here`;
    expect(stripFrontmatter(input)).toBe("Content here");
  });

  it("returns content unchanged when no frontmatter", () => {
    expect(stripFrontmatter("Just plain text")).toBe("Just plain text");
  });

  it("trims leading whitespace", () => {
    expect(stripFrontmatter("  \n  Hello")).toBe("Hello");
  });
});

describe("stripMarkdownToPlaintext", () => {
  it("strips heading markers", () => {
    expect(stripMarkdownToPlaintext("# Title")).toBe("Title");
    expect(stripMarkdownToPlaintext("## Subtitle")).toBe("Subtitle");
    expect(stripMarkdownToPlaintext("### H3")).toBe("H3");
  });

  it("strips bold markers", () => {
    expect(stripMarkdownToPlaintext("**bold text**")).toBe("bold text");
    expect(stripMarkdownToPlaintext("__also bold__")).toBe("also bold");
  });

  it("strips italic markers", () => {
    expect(stripMarkdownToPlaintext("*italic text*")).toBe("italic text");
    expect(stripMarkdownToPlaintext("_also italic_")).toBe("also italic");
  });

  it("strips strikethrough markers", () => {
    expect(stripMarkdownToPlaintext("~~deleted~~")).toBe("deleted");
  });

  it("strips inline code markers", () => {
    expect(stripMarkdownToPlaintext("`code`")).toBe("code");
  });

  it("strips links keeping text", () => {
    expect(stripMarkdownToPlaintext("[click here](http://example.com)")).toBe(
      "click here"
    );
  });

  it("strips images keeping alt text", () => {
    expect(stripMarkdownToPlaintext("![alt text](http://img.png)")).toBe(
      "!alt text"
    );
  });

  it("strips blockquote markers", () => {
    expect(stripMarkdownToPlaintext("> quoted text")).toBe("quoted text");
  });

  it("strips unordered list markers", () => {
    expect(stripMarkdownToPlaintext("- item one")).toBe("item one");
    expect(stripMarkdownToPlaintext("* item two")).toBe("item two");
    expect(stripMarkdownToPlaintext("+ item three")).toBe("item three");
  });

  it("strips ordered list markers", () => {
    expect(stripMarkdownToPlaintext("1. first")).toBe("first");
    expect(stripMarkdownToPlaintext("10. tenth")).toBe("tenth");
  });

  it("strips unchecked checkbox markers", () => {
    expect(stripMarkdownToPlaintext("- [ ] todo item")).toBe("todo item");
  });

  it("strips checked checkbox markers", () => {
    expect(stripMarkdownToPlaintext("- [x] done item")).toBe("done item");
  });

  it("strips horizontal rules", () => {
    expect(stripMarkdownToPlaintext("---")).toBe("");
    expect(stripMarkdownToPlaintext("-----")).toBe("");
  });

  it("collapses multiple newlines into spaces", () => {
    expect(stripMarkdownToPlaintext("line one\n\nline two")).toBe(
      "line one line two"
    );
  });

  it("converts single newlines to spaces", () => {
    expect(stripMarkdownToPlaintext("line one\nline two")).toBe(
      "line one line two"
    );
  });

  it("strips checkbox markers from mixed prose and checkboxes", () => {
    const input = `# Build a feature

Build a feature to optimize database queries.

- [ ] Profile existing queries
- [ ] Add indexes
- [x] Review schema`;
    const result = stripMarkdownToPlaintext(input);
    expect(result).toBe(
      "Build a feature Build a feature to optimize database queries. Profile existing queries Add indexes Review schema"
    );
    expect(result).not.toContain("[ ]");
    expect(result).not.toContain("[x]");
  });

  it("strips checkbox markers without leaving bracket artifacts", () => {
    const input = `Description text

- [ ] First task
- [x] Second task
- [ ] Third task`;
    const result = stripMarkdownToPlaintext(input);
    expect(result).not.toContain("[");
    expect(result).not.toContain("]");
    expect(result).toContain("First task");
    expect(result).toContain("Second task");
    expect(result).toContain("Third task");
  });

  it("handles frontmatter + checkboxes together", () => {
    const input = `---
title: My Goal
---
# Goal

- [ ] Do something
- [x] Already done`;
    const result = stripMarkdownToPlaintext(input);
    expect(result).toBe("Goal Do something Already done");
    expect(result).not.toContain("[ ]");
    expect(result).not.toContain("[x]");
  });

  it("handles only checkboxes with no prose", () => {
    const input = `- [ ] Task A
- [x] Task B`;
    const result = stripMarkdownToPlaintext(input);
    expect(result).toBe("Task A Task B");
  });
});

describe("truncateDescription", () => {
  it("returns text unchanged when under max length", () => {
    expect(truncateDescription("short text")).toBe("short text");
  });

  it("returns text unchanged at exactly max length", () => {
    const text = "a".repeat(255);
    expect(truncateDescription(text)).toBe(text);
  });

  it("truncates and adds ellipsis when over max length", () => {
    const text = "a".repeat(256);
    const result = truncateDescription(text);
    expect(result).toBe("a".repeat(255) + "...");
    expect(result.length).toBe(258);
  });

  it("respects custom max length", () => {
    const result = truncateDescription("hello world", 5);
    expect(result).toBe("hello...");
  });
});

describe("workspaceDescription", () => {
  it("returns null for undefined input", () => {
    expect(workspaceDescription(undefined)).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(workspaceDescription("")).toBeNull();
  });

  it("returns null for whitespace-only string", () => {
    expect(workspaceDescription("   \n  ")).toBeNull();
  });

  it("returns null when markdown strips to empty", () => {
    expect(workspaceDescription("---\n---\n")).toBeNull();
  });

  it("returns stripped and truncated description", () => {
    const input = `---
flow: coordinator
---
# Improve UX

Build a better interface for users.

- [ ] Add dark mode
- [x] Fix layout`;
    const result = workspaceDescription(input);
    expect(result).not.toBeNull();
    expect(result).not.toContain("[ ]");
    expect(result).not.toContain("[x]");
    expect(result).toContain("Improve UX");
    expect(result).toContain("Build a better interface");
  });

  it("truncates long descriptions to 255 chars with ellipsis", () => {
    const longContent = "A".repeat(300);
    const result = workspaceDescription(longContent);
    expect(result).not.toBeNull();
    expect(result!.length).toBe(258);
    expect(result!.endsWith("...")).toBe(true);
  });
});
