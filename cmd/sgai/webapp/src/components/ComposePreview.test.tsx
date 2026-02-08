import { describe, test, expect, afterEach } from "bun:test";
import { render, screen, cleanup } from "@testing-library/react";
import { ComposePreview } from "./ComposePreview";

describe("ComposePreview", () => {
  afterEach(cleanup);

  test("renders loading skeleton when isLoading is true", () => {
    render(<ComposePreview preview={null} isLoading={true} />);
    const skeletons = document.querySelectorAll("[data-slot='skeleton']");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders preview content", () => {
    render(
      <ComposePreview
        preview={{ content: "# My GOAL.md\n\nProject description", etag: '"abc"' }}
      />,
    );
    expect(screen.getByText(/My GOAL.md/)).toBeTruthy();
  });

  test("renders default title", () => {
    render(<ComposePreview preview={null} />);
    expect(screen.getByText("GOAL.md Preview")).toBeTruthy();
  });

  test("renders custom title", () => {
    render(<ComposePreview preview={null} title="Final Preview" />);
    expect(screen.getByText("Final Preview")).toBeTruthy();
  });

  test("renders fallback when no preview", () => {
    render(<ComposePreview preview={null} />);
    expect(screen.getByText("No preview available")).toBeTruthy();
  });

  test("renders flow error alert", () => {
    render(
      <ComposePreview
        preview={{ content: "# GOAL.md", flowError: "Invalid flow syntax", etag: '"abc"' }}
      />,
    );
    expect(screen.getByText("Invalid flow syntax")).toBeTruthy();
  });

  test("does not render flow error when absent", () => {
    render(
      <ComposePreview
        preview={{ content: "# GOAL.md", etag: '"abc"' }}
      />,
    );
    expect(screen.queryByRole("alert")).toBeNull();
  });
});
