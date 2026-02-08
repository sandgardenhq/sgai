import { describe, it, expect } from "bun:test";
import { render, screen } from "@testing-library/react";
import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
  it("renders instruction to select a workspace", () => {
    render(<EmptyState />);
    expect(screen.getByText(/Select a workspace to view its details/i)).toBeDefined();
  });

  it("has centered layout", () => {
    const { container } = render(<EmptyState />);
    const wrapper = container.firstElementChild;
    expect(wrapper?.className).toContain("flex");
    expect(wrapper?.className).toContain("items-center");
    expect(wrapper?.className).toContain("justify-center");
  });
});
