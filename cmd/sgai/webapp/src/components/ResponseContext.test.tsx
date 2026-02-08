import { describe, it, expect } from "bun:test";
import { render, screen } from "@testing-library/react";
import { ResponseContext } from "./ResponseContext";

describe("ResponseContext", () => {
  it("renders nothing when no content provided", () => {
    const { container } = render(<ResponseContext />);
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when both fields are empty strings", () => {
    const { container } = render(
      <ResponseContext goalContent="" projectMgmtContent="" />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders GOAL.md section when goalContent is provided", () => {
    render(
      <ResponseContext goalContent="<p>Goal content here</p>" />,
    );
    expect(screen.getByText("GOAL.md")).toBeDefined();
    expect(screen.getByText("Goal content here")).toBeDefined();
  });

  it("renders PROJECT_MANAGEMENT.md section when projectMgmtContent is provided", () => {
    render(
      <ResponseContext projectMgmtContent="<p>PM content here</p>" />,
    );
    expect(screen.getByText("PROJECT_MANAGEMENT.md")).toBeDefined();
    expect(screen.getByText("PM content here")).toBeDefined();
  });

  it("renders both sections when both contents provided", () => {
    render(
      <ResponseContext
        goalContent="<p>Goal data</p>"
        projectMgmtContent="<p>PM data</p>"
      />,
    );
    expect(screen.getAllByText("GOAL.md").length).toBeGreaterThan(0);
    expect(screen.getAllByText("PROJECT_MANAGEMENT.md").length).toBeGreaterThan(0);
    expect(screen.getByText("Goal data")).toBeDefined();
    expect(screen.getByText("PM data")).toBeDefined();
  });

  it("renders content inside details elements (collapsed by default)", () => {
    const { container } = render(
      <ResponseContext goalContent="<p>Hidden by default</p>" />,
    );
    const details = container.querySelectorAll("details");
    expect(details.length).toBe(1);
    expect((details[0] as HTMLDetailsElement).open).toBe(false);
  });
});
