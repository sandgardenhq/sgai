import { describe, it, expect, afterEach } from "bun:test";
import { render, screen, cleanup } from "@testing-library/react";
import { NotYetAvailable } from "../NotYetAvailable";

afterEach(() => {
  cleanup();
});

describe("NotYetAvailable", () => {
  it("renders default message without page name", () => {
    render(<NotYetAvailable />);
    expect(screen.getByText("Not Yet Available")).toBeTruthy();
    expect(screen.getByText("This page is not available yet.")).toBeTruthy();
  });

  it("renders with page name prefix", () => {
    render(<NotYetAvailable pageName="Settings" />);
    expect(screen.getByText("Settings — Not Yet Available")).toBeTruthy();
  });

  it("renders without page name prefix when empty", () => {
    render(<NotYetAvailable pageName="" />);
    expect(screen.getByText("Not Yet Available")).toBeTruthy();
  });
});
